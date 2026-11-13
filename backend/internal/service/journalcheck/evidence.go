package journalcheck

import (
	"encoding/json"
	"fmt"

	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

type openAlexSource struct {
	DisplayName     string   `json:"display_name"`
	AlternateTitles []string `json:"alternate_titles"`
	ISSNL           string   `json:"issn_l"`
	ISSN            []string `json:"issn"`
	Type            string   `json:"type"`
	IsInDOAJ        *bool    `json:"is_in_doaj"`
	Host            string   `json:"host_organization_name"`
	HomepageURL     string   `json:"homepage_url"`
	WorksCount      *int     `json:"works_count"`
	SummaryStats    struct {
		HIndex        *int     `json:"h_index"`
		MeanCitedness *float64 `json:"2yr_mean_citedness"`
	} `json:"summary_stats"`
}

func (s openAlexSource) evidence() SourceEvidence {
	return SourceEvidence{
		Status: StatusFound, Title: cleanText(s.DisplayName), AlternateTitles: cleanAll(s.AlternateTitles),
		Publisher: cleanText(s.Host), ISSNs: normalizeISSNs(append([]string{s.ISSNL}, s.ISSN...)),
		Type: cleanText(s.Type), InDOAJ: s.IsInDOAJ, HIndex: s.SummaryStats.HIndex,
		MeanCitedness: s.SummaryStats.MeanCitedness, WorksCount: s.WorksCount, URL: SafeURL(s.HomepageURL),
	}
}

// ParseOpenAlexSource parses GET /sources/issn:{issn}.
func ParseOpenAlexSource(data []byte) (SourceEvidence, error) {
	var source openAlexSource
	if err := json.Unmarshal(data, &source); err != nil {
		return SourceEvidence{}, fmt.Errorf("openalex source: %w", err)
	}
	if cleanText(source.DisplayName) == "" {
		return SourceEvidence{}, fmt.Errorf("openalex source: no display_name")
	}
	return source.evidence(), nil
}

// ParseOpenAlexSearch parses GET /sources?filter=display_name.search:{name}.
func ParseOpenAlexSearch(data []byte) ([]SourceEvidence, error) {
	var page struct {
		Results *[]openAlexSource `json:"results"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("openalex search: %w", err)
	}
	if page.Results == nil {
		return nil, fmt.Errorf("openalex search: no results field")
	}
	out := make([]SourceEvidence, 0, len(*page.Results))
	for _, source := range *page.Results {
		out = append(out, source.evidence())
	}
	return out, nil
}

// ResolveByName accepts only exact folded-name candidates. Two different exact
// matches stay ambiguous: predatory clones copy names, so guessing could pin the
// evidence on the wrong journal.
func ResolveByName(candidates []SourceEvidence, name string) SourceEvidence {
	key := RefKey(name)
	var exact []SourceEvidence
	for _, candidate := range candidates {
		if titleMatchesExactly(candidate, key) {
			exact = append(exact, candidate)
		}
	}
	switch len(exact) {
	case 0:
		return SourceEvidence{Status: StatusNotFound}
	case 1:
		return exact[0]
	default:
		return SourceEvidence{Status: statusAmbiguous}
	}
}

func titleMatchesExactly(e SourceEvidence, key string) bool {
	if key == "" {
		return false
	}
	if RefKey(e.Title) == key {
		return true
	}
	for _, title := range e.AlternateTitles {
		if RefKey(title) == key {
			return true
		}
	}
	return false
}

// ParseCrossrefJournal parses GET /journals/{issn}.
func ParseCrossrefJournal(data []byte) (SourceEvidence, error) {
	var response struct {
		Message *struct {
			Title     string   `json:"title"`
			Publisher string   `json:"publisher"`
			ISSN      []string `json:"ISSN"`
			ISSNType  []struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"issn-type"`
			Counts struct {
				TotalDOIs *int `json:"total-dois"`
			} `json:"counts"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return SourceEvidence{}, fmt.Errorf("crossref journal: %w", err)
	}
	if response.Message == nil || cleanText(response.Message.Title) == "" {
		return SourceEvidence{}, fmt.Errorf("crossref journal: no title")
	}
	message := response.Message
	out := SourceEvidence{
		Status: StatusFound, Title: cleanText(message.Title), Publisher: cleanText(message.Publisher),
		ISSNs: normalizeISSNs(message.ISSN), TotalDOIs: message.Counts.TotalDOIs,
	}
	for _, typed := range message.ISSNType {
		switch typed.Type {
		case "print":
			out.PrintISSN = screening.NormalizeISSN(typed.Value)
		case "electronic":
			out.OnlineISSN = screening.NormalizeISSN(typed.Value)
		}
	}
	return out, nil
}

// ParseDOAJSearch parses GET /api/search/journals/issn:{issn}. Zero results is a
// valid "not in DOAJ" answer, reported as StatusNotFound.
func ParseDOAJSearch(data []byte) (SourceEvidence, error) {
	var response struct {
		Results *[]struct {
			BibJSON struct {
				Title     string `json:"title"`
				Alternate string `json:"alternative_title"`
				EISSN     string `json:"eissn"`
				PISSN     string `json:"pissn"`
				Publisher struct {
					Name string `json:"name"`
				} `json:"publisher"`
				Ref struct {
					Journal string `json:"journal"`
				} `json:"ref"`
			} `json:"bibjson"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return SourceEvidence{}, fmt.Errorf("doaj search: %w", err)
	}
	if response.Results == nil {
		return SourceEvidence{}, fmt.Errorf("doaj search: no results field")
	}
	if len(*response.Results) == 0 {
		return SourceEvidence{Status: StatusNotFound}, nil
	}
	bib := (*response.Results)[0].BibJSON
	inDOAJ := true
	return SourceEvidence{
		Status: StatusFound, Title: cleanText(bib.Title), AlternateTitles: cleanAll([]string{bib.Alternate}),
		Publisher: cleanText(bib.Publisher.Name), ISSNs: normalizeISSNs([]string{bib.PISSN, bib.EISSN}),
		PrintISSN: screening.NormalizeISSN(bib.PISSN), OnlineISSN: screening.NormalizeISSN(bib.EISSN),
		InDOAJ: &inDOAJ, URL: SafeURL(bib.Ref.Journal),
	}, nil
}

func cleanAll(values []string) []string {
	var out []string
	for _, value := range values {
		if cleaned := cleanText(value); cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}

func normalizeISSNs(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		issn := screening.NormalizeISSN(value)
		if issn == "" {
			continue
		}
		if _, dup := seen[issn]; !dup {
			seen[issn] = struct{}{}
			out = append(out, issn)
		}
	}
	return out
}
