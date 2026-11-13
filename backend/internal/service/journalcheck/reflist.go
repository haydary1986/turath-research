package journalcheck

import (
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

// maxListRows is far above any real list and bounds what a bad upstream file can cost.
const maxListRows = 100_000

// ReferenceSource attributes rows parsed from the public lists.
const ReferenceSource = "stop-predatory-journals"

// minAliasRunes keeps fragments and acronyms out of matching: "AES" or "AAAS" name
// many unrelated bodies, so a short alias would brand them all as predatory.
const minAliasRunes = 6

// RefKey folds a name for list matching: screening.NameKey plus the "&"/"and"
// and leading "the" spellings that differ between the lists and the indexes.
func RefKey(name string) string {
	key := screening.NameKey(strings.ReplaceAll(cleanText(name), "&", " and "))
	return strings.TrimPrefix(key, "the ")
}

func cleanText(s string) string {
	s = strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\x00", "")
	return strings.TrimSpace(html.UnescapeString(s))
}

// sharedHosts serve many unrelated journals, so a listed journal on one of them
// says nothing about its neighbours.
var sharedHosts = map[string]struct{}{
	"sites.google.com": {}, "google.com": {}, "journals.sfu.ca": {}, "wordpress.com": {},
	"blogspot.com": {}, "weebly.com": {}, "wixsite.com": {}, "github.io": {},
	"academia.edu": {}, "researchgate.net": {}, "facebook.com": {}, "linkedin.com": {},
	"doi.org": {}, "archive.org": {}, "web.archive.org": {}, "worldcat.org": {},
}

func isSharedHost(host string) bool {
	if _, ok := sharedHosts[host]; ok {
		return true
	}
	labels := strings.Split(host, ".")
	for i, label := range labels {
		last := i == len(labels)-1
		secondLast := i == len(labels)-2
		switch {
		case (label == "edu" || label == "gov") && (last || secondLast):
			return true
		case label == "ac" && secondLast:
			return true
		}
	}
	return false
}

// SafeURL returns an absolute http(s) form of raw, or "" for anything else.
// Scheme-less domains are accepted; javascript: and relative paths are not.
func SafeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "/") || strings.ContainsAny(raw, " \t\r\n") {
		return ""
	}
	if !strings.Contains(raw, "://") {
		if strings.Contains(strings.SplitN(raw, "/", 2)[0], ":") {
			return ""
		}
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.Contains(u.Hostname(), ".") {
		return ""
	}
	return u.String()
}

// URLHost returns the lowercase host of raw without any www prefix, or "".
func URLHost(raw string) string {
	safe := SafeURL(raw)
	if safe == "" {
		return ""
	}
	u, err := url.Parse(safe)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

// HostCandidates lists host and its parent domains, so a subdomain of a listed
// publisher still matches. Single-label suffixes are excluded.
func HostCandidates(host string) []string {
	if host == "" {
		return nil
	}
	labels := strings.Split(host, ".")
	out := make([]string, 0, len(labels))
	for i := 0; i < len(labels)-1; i++ {
		out = append(out, strings.Join(labels[i:], "."))
	}
	return out
}

func listHost(raw string) string {
	host := URLHost(raw)
	if isSharedHost(host) {
		return ""
	}
	return host
}

// ParseList parses one reference CSV. An error means the caller must keep its
// previous data: a changed header or an empty file is never a valid empty list.
func ParseList(listType string, data []byte, fetchedAt time.Time) ([]RefEntry, error) {
	rows, err := readCSV(data)
	if err != nil {
		return nil, err
	}
	var entries []RefEntry
	switch listType {
	case ListPredatoryJournal, ListPredatoryPublisher:
		entries, err = parseNamedRows(listType, rows, fetchedAt)
	case ListHijacked:
		entries, err = parseHijackedRows(rows, fetchedAt)
	default:
		return nil, fmt.Errorf("unknown list type %q", listType)
	}
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("list has no usable rows")
	}
	return entries, nil
}

func readCSV(data []byte) ([]map[string]string, error) {
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\uFEFF")))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	var rows []map[string]string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		if len(rows) >= maxListRows {
			return nil, fmt.Errorf("list exceeds %d rows", maxListRows)
		}
		row := make(map[string]string, len(header))
		for i, column := range header {
			if i < len(record) {
				row[strings.ToLower(strings.TrimSpace(column))] = cleanText(record[i])
			}
		}
		rows = append(rows, row)
	}
}

// nameKeys lists what a row can be matched by: its full name, each part of a
// composite "A/B" name, and its abbreviation, the last two only where long enough
// to identify a journal. A lowercase part means "and/or" prose rather than a second name.
func nameKeys(name, abbr string) []string {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if first, _ := utf8.DecodeRuneInString(trimmed); unicode.IsLower(first) {
			parts = nil
			break
		}
	}
	keys := []string{RefKey(name)}
	if keys[0] == "" {
		return nil
	}
	for _, alias := range append(parts, abbr) {
		key := RefKey(alias)
		if utf8.RuneCountInString(key) >= minAliasRunes && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func requireColumns(rows []map[string]string, columns ...string) error {
	if len(rows) == 0 {
		return errors.New("list has no rows")
	}
	for _, column := range columns {
		if _, ok := rows[0][column]; !ok {
			return fmt.Errorf("missing column %q", column)
		}
	}
	return nil
}

func parseNamedRows(listType string, rows []map[string]string, fetchedAt time.Time) ([]RefEntry, error) {
	if err := requireColumns(rows, "url", "name"); err != nil {
		return nil, err
	}
	out := make([]RefEntry, 0, len(rows))
	for _, row := range rows {
		for _, key := range nameKeys(row["name"], row["abbr"]) {
			out = append(out, RefEntry{
				ListType: listType, NameKey: key, Name: row["name"],
				URL: SafeURL(row["url"]), URLHost: listHost(row["url"]),
				Source: ReferenceSource, FetchedAt: fetchedAt,
			})
		}
	}
	return out, nil
}

func parseHijackedRows(rows []map[string]string, fetchedAt time.Time) ([]RefEntry, error) {
	if err := requireColumns(rows, "hijacked", "hijackedurl", "althijackedurl", "authentic", "authenticurl"); err != nil {
		return nil, err
	}
	out := make([]RefEntry, 0, len(rows))
	for _, row := range rows {
		for _, key := range nameKeys(row["hijacked"], row["hijackedabbr"]) {
			base := RefEntry{
				ListType: ListHijacked, NameKey: key, Name: row["hijacked"],
				AuthenticName: row["authentic"], AuthenticURL: SafeURL(row["authenticurl"]),
				Source: ReferenceSource, FetchedAt: fetchedAt,
			}
			primary := base
			primary.URL, primary.URLHost = SafeURL(row["hijackedurl"]), listHost(row["hijackedurl"])
			out = append(out, primary)
			if alt := SafeURL(row["althijackedurl"]); alt != "" {
				second := base
				second.URL, second.URLHost = alt, listHost(alt)
				out = append(out, second)
			}
		}
	}
	return out, nil
}
