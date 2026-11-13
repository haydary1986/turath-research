package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/screening"
)

// Bulk import of the journal watchlist: ministry accreditation lists and
// predatory-journal lists come as spreadsheets with thousands of rows, so
// row-by-row manual entry would leave the watchlist empty in practice.

type screeningImportRow struct {
	Line           int    `json:"line"`
	Name           string `json:"name"`
	ISSN           string `json:"issn"`
	ISSNOnline     string `json:"issn_online"`
	Publisher      string `json:"publisher"`
	Classification string `json:"classification"`
	Source         string `json:"source"`
	Note           string `json:"note"`
}

type screeningImportProblem struct {
	Line int    `json:"line"`
	Code string `json:"code"`
}

type screeningImportReport struct {
	DryRun         bool                     `json:"dry_run"`
	ValidRows      int                      `json:"valid_rows"`
	Created        int                      `json:"created"`
	Updated        int                      `json:"updated"`
	Problems       []screeningImportProblem `json:"problems"`
	MissingColumns []string                 `json:"missing_columns,omitempty"`
	Preview        []screeningImportRow     `json:"preview,omitempty"`
}

const (
	maxScreeningRows = 20000
	importTimeout    = 2 * time.Minute
)

// Header aliases, matched on a lowercased/trimmed cell with ة folded to ه.
var screeningHeaderAliases = map[string]string{
	"name":           "name",
	"journal":        "name",
	"journal name":   "name",
	"اسم المجله":     "name",
	"المجله":         "name",
	"issn":           "issn",
	"issn ورقي":      "issn",
	"print issn":     "issn",
	"issn online":    "issn_online",
	"eissn":          "issn_online",
	"e-issn":         "issn_online",
	"issn الكتروني":  "issn_online",
	"الكتروني issn":  "issn_online",
	"publisher":      "publisher",
	"الناشر":         "publisher",
	"classification": "classification",
	"التصنيف":        "classification",
	"الحاله":         "classification",
	"source":         "source",
	"المصدر":         "source",
	"القائمه":        "source",
	"note":           "note",
	"notes":          "note",
	"ملاحظات":        "note",
	"ملاحظه":         "note",
}

var screeningClassAliases = map[string]string{
	"accredited": screening.ClassAccredited,
	"معتمده":     screening.ClassAccredited,
	"معتمد":      screening.ClassAccredited,
	"رصينه":      screening.ClassAccredited,
	"predatory":  screening.ClassPredatory,
	"مفترسه":     screening.ClassPredatory,
	"مفترس":      screening.ClassPredatory,
	"suspended":  screening.ClassSuspended,
	"موقوفه":     screening.ClassSuspended,
	"موقوف":      screening.ClassSuspended,
	"معلقه":      screening.ClassSuspended,
}

func normalizeScreeningKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ة", "ه")
	s = strings.ReplaceAll(s, "أ", "ا")
	s = strings.ReplaceAll(s, "إ", "ا")
	s = strings.ReplaceAll(s, "آ", "ا")
	return strings.Join(strings.Fields(s), " ")
}

func mapScreeningHeader(cells []string) (map[int]string, []string) {
	header := map[int]string{}
	seen := map[string]bool{}
	for i, cell := range cells {
		if field, ok := screeningHeaderAliases[normalizeScreeningKey(cell)]; ok && !seen[field] {
			header[i] = field
			seen[field] = true
		}
	}
	var missing []string
	for _, required := range []string{"name", "classification"} {
		if !seen[required] {
			missing = append(missing, required)
		}
	}
	return header, missing
}

func parseScreeningRows(header map[int]string, rows [][]string, firstLine int) ([]screeningImportRow, []screeningImportProblem) {
	var valid []screeningImportRow
	var problems []screeningImportProblem
	for idx, cells := range rows {
		line := firstLine + idx
		if len(valid) >= maxScreeningRows {
			problems = append(problems, screeningImportProblem{Line: line, Code: "too_many_rows"})
			break
		}
		row := screeningImportRow{Line: line}
		empty := true
		for i, field := range header {
			if i >= len(cells) {
				continue
			}
			value := strings.TrimSpace(cells[i])
			if value != "" {
				empty = false
			}
			switch field {
			case "name":
				row.Name = value
			case "issn":
				row.ISSN = value
			case "issn_online":
				row.ISSNOnline = value
			case "publisher":
				row.Publisher = value
			case "classification":
				row.Classification = value
			case "source":
				row.Source = value
			case "note":
				row.Note = value
			}
		}
		if empty {
			continue
		}
		if row.Name == "" {
			problems = append(problems, screeningImportProblem{Line: line, Code: "name_required"})
			continue
		}
		class, ok := screeningClassAliases[normalizeScreeningKey(row.Classification)]
		if !ok {
			problems = append(problems, screeningImportProblem{Line: line, Code: "classification_invalid"})
			continue
		}
		row.Classification = class
		if row.ISSN != "" && screening.NormalizeISSN(row.ISSN) == "" {
			problems = append(problems, screeningImportProblem{Line: line, Code: "issn_malformed"})
			continue
		}
		if row.ISSNOnline != "" && screening.NormalizeISSN(row.ISSNOnline) == "" {
			problems = append(problems, screeningImportProblem{Line: line, Code: "issn_malformed"})
			continue
		}
		valid = append(valid, row)
	}
	return valid, problems
}

// ImportScreening accepts an .xlsx watchlist (multipart "file", staff only).
// With ?dry_run=true it only validates and previews. A journal that carries
// both a print and an online ISSN is stored as two entries so either one matches.
func (h *PortalHandler) ImportScreening(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	ctx, cancel := longRunningContext(w, r, importTimeout)
	defer cancel()

	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes+multipartMemory)
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		model.RespondError(w, http.StatusRequestEntityTooLarge, "file_too_large")
		return
	}
	defer r.MultipartForm.RemoveAll() //nolint:errcheck // temp-file cleanup

	file, _, err := r.FormFile("file")
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
	if err != nil || len(data) > maxImportBytes {
		model.RespondError(w, http.StatusRequestEntityTooLarge, "file_too_large")
		return
	}

	sheet, ok := readImportSheet(w, data, maxScreeningRows)
	if !ok {
		return
	}
	rows := sheet.rows

	header, missing := mapScreeningHeader(rows[0])
	if len(missing) > 0 {
		model.RespondJSON(w, http.StatusOK, screeningImportReport{
			DryRun: true, MissingColumns: missing, Problems: []screeningImportProblem{},
		})
		return
	}
	valid, problems := parseScreeningRows(header, rows[1:], 2)
	if sheet.truncated && len(valid) < maxScreeningRows {
		problems = append(problems, screeningImportProblem{Line: len(rows) + 1, Code: "too_many_rows"})
	}

	report := screeningImportReport{ValidRows: len(valid), Problems: problems}
	if report.Problems == nil {
		report.Problems = []screeningImportProblem{}
	}

	if r.URL.Query().Get("dry_run") == "true" {
		report.DryRun = true
		if len(valid) > 20 {
			report.Preview = valid[:20]
		} else {
			report.Preview = valid
		}
		model.RespondJSON(w, http.StatusOK, report)
		return
	}

	if err := h.saveScreeningRows(ctx, valid, user.UserID, &report); err != nil {
		respondImportFailure(w, err)
		return
	}
	model.RespondJSON(w, http.StatusOK, report)
}

// screeningRowError names the spreadsheet line that aborted the import.
type screeningRowError struct {
	line int
	err  error
}

func (e screeningRowError) Error() string { return fmt.Sprintf("line %d: %v", e.line, e.err) }
func (e screeningRowError) Unwrap() error { return e.err }

// saveScreeningRows stores every row or none of them.
func (h *PortalHandler) saveScreeningRows(ctx context.Context, rows []screeningImportRow, actorID string, report *screeningImportReport) error {
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, row := range rows {
		if err := applyScreeningRow(ctx, tx, row, actorID, report); err != nil {
			return screeningRowError{line: row.Line, err: err}
		}
	}
	return tx.Commit(ctx)
}

// respondImportFailure blames the line for bad data (Postgres classes 22 and 23) and the server otherwise.
func respondImportFailure(w http.ResponseWriter, err error) {
	var rowErr screeningRowError
	if !errors.As(err, &rowErr) {
		serverError(w, "Failed to save import", err)
		return
	}
	slog.Error("portal: watchlist import rolled back", "line", rowErr.line, "error", err)
	status := http.StatusInternalServerError
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (strings.HasPrefix(pgErr.Code, "22") || strings.HasPrefix(pgErr.Code, "23")) {
		status = http.StatusUnprocessableEntity
	}
	model.RespondError(w, status, fmt.Sprintf("import_failed_line:%d", rowErr.line))
}

func applyScreeningRow(ctx context.Context, tx pgx.Tx, row screeningImportRow, actorID string, report *screeningImportReport) error {
	issns := []string{}
	if issn := screening.NormalizeISSN(row.ISSN); issn != "" {
		issns = append(issns, issn)
	}
	if issn := screening.NormalizeISSN(row.ISSNOnline); issn != "" && !strings.EqualFold(issn, row.ISSN) {
		duplicate := false
		for _, existing := range issns {
			if existing == issn {
				duplicate = true
			}
		}
		if !duplicate {
			issns = append(issns, issn)
		}
	}
	nameKey := screening.NameKey(row.Name)

	if len(issns) == 0 {
		return upsertScreeningByName(ctx, tx, row, nameKey, actorID, report)
	}
	for _, issn := range issns {
		var inserted bool
		err := tx.QueryRow(ctx, `
			INSERT INTO journal_screening (issn, name, name_key, publisher, classification, note, source, created_by)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), NULLIF($7, ''), $8)
			ON CONFLICT (issn) WHERE issn IS NOT NULL DO UPDATE SET
				name = EXCLUDED.name, name_key = EXCLUDED.name_key,
				publisher = COALESCE(EXCLUDED.publisher, journal_screening.publisher),
				classification = EXCLUDED.classification,
				note = COALESCE(EXCLUDED.note, journal_screening.note),
				source = COALESCE(EXCLUDED.source, journal_screening.source),
				updated_at = NOW()
			RETURNING (xmax = 0)
		`, issn, row.Name, nameKey, row.Publisher, row.Classification, row.Note, row.Source, actorID).Scan(&inserted)
		if err != nil {
			return err
		}
		if inserted {
			report.Created++
		} else {
			report.Updated++
		}
	}
	return nil
}

func upsertScreeningByName(ctx context.Context, tx pgx.Tx, row screeningImportRow, nameKey, actorID string, report *screeningImportReport) error {
	var id string
	err := tx.QueryRow(ctx,
		`SELECT id FROM journal_screening WHERE issn IS NULL AND name_key = $1 LIMIT 1`, nameKey).Scan(&id)
	switch {
	case isNoRows(err):
		_, err = tx.Exec(ctx, `
			INSERT INTO journal_screening (issn, name, name_key, publisher, classification, note, source, created_by)
			VALUES (NULL, $1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), NULLIF($6, ''), $7)
		`, row.Name, nameKey, row.Publisher, row.Classification, row.Note, row.Source, actorID)
		if err == nil {
			report.Created++
		}
		return err
	case err != nil:
		return err
	default:
		_, err = tx.Exec(ctx, `
			UPDATE journal_screening SET
				name = $2, publisher = COALESCE(NULLIF($3, ''), publisher), classification = $4,
				note = COALESCE(NULLIF($5, ''), note), source = COALESCE(NULLIF($6, ''), source), updated_at = NOW()
			WHERE id = $1
		`, id, row.Name, row.Publisher, row.Classification, row.Note, row.Source)
		if err == nil {
			report.Updated++
		}
		return err
	}
}
