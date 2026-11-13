package journalcheck

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/haydary1986/turath-research/backend/internal/apiclient"
)

const (
	openAlexBase   = "https://api.openalex.org"
	crossrefBase   = "https://api.crossref.org"
	doajBase       = "https://doaj.org/api/search/journals"
	listsBase      = "https://raw.githubusercontent.com/stop-predatory-journals/stop-predatory-journals.github.io/master/_data/"
	lookupsAgent   = "turath-research/1.0"
	lookupTimeout  = 10 * time.Second
	listTimeout    = 45 * time.Second
	nameCandidates = "10"
)

var listFiles = map[string]string{
	ListPredatoryJournal:   "journals.csv",
	ListPredatoryPublisher: "publishers.csv",
	ListHijacked:           "hijacked.csv",
}

// HTTPLookups implements Lookups against the live services, one rate-limited
// client per host so an outage at one cannot trip the others' breakers.
type HTTPLookups struct {
	openAlex, crossref, doaj, lists *apiclient.Client
	apiKey, mailto                  string
}

// NewHTTPLookups builds the live client; apiKey and mailto may be empty.
func NewHTTPLookups(apiKey, mailto string) *HTTPLookups {
	agent := lookupsAgent
	if mailto != "" {
		agent += " (mailto:" + mailto + ")"
	}
	return &HTTPLookups{
		openAlex: apiclient.NewClient(lookupTimeout, 5, agent),
		crossref: apiclient.NewClient(lookupTimeout, 5, agent),
		doaj:     apiclient.NewClient(lookupTimeout, 2, agent),
		lists:    apiclient.NewClient(listTimeout, 2, agent),
		apiKey:   apiKey, mailto: mailto,
	}
}

func (h *HTTPLookups) openAlexParams(params url.Values) string {
	if h.apiKey != "" {
		params.Set("api_key", h.apiKey)
	}
	if h.mailto != "" {
		params.Set("mailto", h.mailto)
	}
	if len(params) == 0 {
		return ""
	}
	return "?" + params.Encode()
}

func notFoundOr(err error) error {
	if apiclient.IsNotFound(err) {
		return ErrNotFound
	}
	return err
}

// OpenAlexByISSN implements Lookups.
func (h *HTTPLookups) OpenAlexByISSN(ctx context.Context, issn string) (SourceEvidence, error) {
	body, err := h.openAlex.Get(ctx, openAlexBase+"/sources/issn:"+url.PathEscape(issn)+h.openAlexParams(url.Values{}), nil)
	if err != nil {
		return SourceEvidence{}, notFoundOr(err)
	}
	return ParseOpenAlexSource(body)
}

// OpenAlexByName implements Lookups.
func (h *HTTPLookups) OpenAlexByName(ctx context.Context, name string) ([]SourceEvidence, error) {
	params := url.Values{"search": {name}, "per_page": {nameCandidates}}
	body, err := h.openAlex.Get(ctx, openAlexBase+"/sources"+h.openAlexParams(params), nil)
	if err != nil {
		return nil, err
	}
	return ParseOpenAlexSearch(body)
}

// CrossrefByISSN implements Lookups.
func (h *HTTPLookups) CrossrefByISSN(ctx context.Context, issn string) (SourceEvidence, error) {
	body, err := h.crossref.Get(ctx, crossrefBase+"/journals/"+url.PathEscape(issn), nil)
	if err != nil {
		return SourceEvidence{}, notFoundOr(err)
	}
	return ParseCrossrefJournal(body)
}

// DOAJByISSN implements Lookups.
func (h *HTTPLookups) DOAJByISSN(ctx context.Context, issn string) (SourceEvidence, error) {
	body, err := h.doaj.Get(ctx, doajBase+"/issn:"+url.PathEscape(issn), nil)
	if err != nil {
		return SourceEvidence{}, notFoundOr(err)
	}
	evidence, err := ParseDOAJSearch(body)
	if err != nil {
		return SourceEvidence{}, err
	}
	if evidence.Status == StatusNotFound {
		return SourceEvidence{}, ErrNotFound
	}
	return evidence, nil
}

// FetchList implements Lookups.
func (h *HTTPLookups) FetchList(ctx context.Context, listType string) ([]byte, error) {
	file, ok := listFiles[listType]
	if !ok {
		return nil, fmt.Errorf("unknown list type %q", listType)
	}
	return h.lists.Get(ctx, listsBase+file, nil)
}
