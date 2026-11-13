package sync

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
)

// Codes stored in researchers.last_sync_error and localized by the client.
const (
	codeNoORCID             = "no_orcid"
	codeORCIDNotFound       = "orcid_not_found"
	codeORCIDUnavailable    = "orcid_unavailable"
	codeOpenAlexUnavailable = "openalex_unavailable"
	codeSourcesUnavailable  = "sources_unavailable"
	codePartialSave         = "partial_save"
	codeSaveFailed          = "save_failed"
	codeSuperseded          = "superseded"
	codeOpenAlexTruncated   = "openalex_truncated"

	// Identity codes ask staff to look: the sync refused to trust a source.
	CodeORCIDNameMismatch    = "orcid_name_mismatch"
	CodeORCIDNameUnverified  = "orcid_name_unverified"
	CodeOpenAlexNameMismatch = "openalex_name_mismatch"

	maxDOILength = 255
)

var (
	openAlexAuthorID = regexp.MustCompile(`^A[0-9]{1,20}$`)
	doiPrefix        = regexp.MustCompile(`^(https?://(dx\.)?doi\.org/|doi:\s*)`)
)

type importedWork struct {
	DOI             string
	Title           string
	PublicationDate string
	CitationCount   int
	Type            string
	Journal         string
	Language        string
	AbstractEn      string
	// FromOpenAlex marks a citation count worth recording: ORCID carries none.
	FromOpenAlex bool
}

type syncOutcome struct {
	Code      string
	Succeeded bool // at least one source answered, so last_synced_at advances
}

// outcomeOf summarizes a run. Partial answers count as success with a visible code.
func outcomeOf(orcidErr, openalexErr error, failedRows int) syncOutcome {
	switch {
	case orcidErr != nil && openalexErr != nil:
		if apiclient.IsNotFound(orcidErr) {
			return syncOutcome{Code: codeORCIDNotFound}
		}
		return syncOutcome{Code: codeSourcesUnavailable}
	case orcidErr != nil:
		if apiclient.IsNotFound(orcidErr) {
			return syncOutcome{Code: codeORCIDNotFound, Succeeded: true}
		}
		return syncOutcome{Code: codeORCIDUnavailable, Succeeded: true}
	case openalexErr != nil:
		return syncOutcome{Code: codeOpenAlexUnavailable, Succeeded: true}
	case failedRows > 0:
		return syncOutcome{Code: codePartialSave, Succeeded: true}
	}
	return syncOutcome{Succeeded: true}
}

var errSuperseded = errors.New("superseded by a newer sync")

// normalizeDOI reduces any spelling to the bare lowercase form; "" when unusable.
// ORCID and OpenAlex spell the same DOI differently, so dedupe depends on this.
func normalizeDOI(raw string) string {
	doi := strings.ToLower(strings.TrimSpace(raw))
	doi = doiPrefix.ReplaceAllString(doi, "")
	if !strings.HasPrefix(doi, "10.") || !strings.Contains(doi, "/") || len(doi) > maxDOILength {
		return ""
	}
	return doi
}

func titleKey(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

// mergeWorks dedupes by DOI, then by title. Works are taken in the given order, so
// earlier lists win on conflicting values and later ones only fill the gaps.
func mergeWorks(lists ...[]importedWork) []importedWork {
	merged := []importedWork{}
	byDOI := map[string]int{}
	byTitle := map[string]int{}
	for _, list := range lists {
		for _, w := range list {
			key := titleKey(w.Title)
			if key == "" {
				continue
			}
			index, found := -1, false
			if w.DOI != "" {
				index, found = byDOI[w.DOI]
			}
			if !found {
				// Equal titles with two different DOIs are distinct records (preprint and article).
				if i, ok := byTitle[key]; ok && (w.DOI == "" || merged[i].DOI == "") {
					index, found = i, true
				}
			}
			if !found {
				merged = append(merged, w)
				index = len(merged) - 1
			} else {
				merged[index] = fillGaps(merged[index], w)
			}
			if merged[index].DOI != "" {
				byDOI[merged[index].DOI] = index
			}
			if _, taken := byTitle[key]; !taken {
				byTitle[key] = index
			}
		}
	}
	return merged
}

func fillGaps(base, extra importedWork) importedWork {
	first := func(a, b string) string {
		if a != "" {
			return a
		}
		return b
	}
	base.DOI = first(base.DOI, extra.DOI)
	base.PublicationDate = first(base.PublicationDate, extra.PublicationDate)
	base.Type = first(base.Type, extra.Type)
	base.Journal = first(base.Journal, extra.Journal)
	base.Language = first(base.Language, extra.Language)
	base.AbstractEn = first(base.AbstractEn, extra.AbstractEn)
	base.CitationCount = max(base.CitationCount, extra.CitationCount)
	base.FromOpenAlex = base.FromOpenAlex || extra.FromOpenAlex
	return base
}

func workFromOpenAlex(w apiclient.OpenAlexWork) importedWork {
	work := importedWork{
		DOI:             normalizeDOI(w.DOI),
		Title:           strings.TrimSpace(w.DisplayName),
		PublicationDate: validDate(w.PublicationDate),
		CitationCount:   w.CitedByCount,
		Type:            w.Type,
		Language:        w.Language,
		AbstractEn:      apiclient.ReconstructAbstract(w.AbstractInvIdx),
		FromOpenAlex:    true,
	}
	if w.PrimaryLocation != nil && w.PrimaryLocation.Source != nil {
		work.Journal = w.PrimaryLocation.Source.DisplayName
	}
	return work
}

func workFromORCID(s apiclient.ORCIDWorkSummary) importedWork {
	work := importedWork{DOI: normalizeDOI(apiclient.ExtractDOI(s)), Type: s.Type}
	if s.Title != nil {
		work.Title = strings.TrimSpace(s.Title.Title.Value)
	}
	if s.JournalTitle != nil {
		work.Journal = s.JournalTitle.Value
	}
	if d := s.PublicationDate; d != nil && d.Year != nil {
		month, day := "01", "01"
		if d.Month != nil {
			month = d.Month.Value
			if d.Day != nil {
				day = d.Day.Value
			}
		}
		work.PublicationDate = validDate(d.Year.Value + "-" + month + "-" + day)
	}
	return work
}

func validDate(value string) string {
	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return ""
	}
	return value
}

// openAlexShortID turns https://openalex.org/A123 into A123; "" when malformed.
func openAlexShortID(id string) string {
	short := id[strings.LastIndex(id, "/")+1:]
	if !openAlexAuthorID.MatchString(short) {
		return ""
	}
	return short
}

// publicationType maps ORCID and OpenAlex vocabularies onto the publication_type enum.
func publicationType(raw string) string {
	switch strings.ReplaceAll(strings.ToLower(raw), "_", "-") {
	case "conference-paper", "proceedings-article":
		return "conference_paper"
	case "book-chapter":
		return "book_chapter"
	case "book", "monograph", "edited-book":
		return "book"
	case "preprint", "posted-content":
		return "preprint"
	case "dissertation", "dissertation-thesis", "thesis":
		return "thesis"
	case "patent":
		return "patent"
	case "report":
		return "report"
	}
	return "journal_article"
}

// splitTitle files a title under the column for its script.
func splitTitle(title string) (ar, en *string) {
	for _, r := range title {
		if unicode.Is(unicode.Arabic, r) {
			return &title, nil
		}
	}
	return nil, &title
}
