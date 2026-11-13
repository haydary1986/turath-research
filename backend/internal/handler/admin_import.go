package handler

import (
	"context"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/importer"
)

const maxImportBytes = 10 << 20

// AdminImportHandler bulk-imports researchers from an Excel sheet (admin only).
type AdminImportHandler struct {
	db *pgxpool.Pool
}

func NewAdminImportHandler(db *pgxpool.Pool) *AdminImportHandler {
	return &AdminImportHandler{db: db}
}

type importReport struct {
	DryRun             bool               `json:"dry_run"`
	ValidRows          int                `json:"valid_rows"`
	Created            int                `json:"created"`
	Updated            int                `json:"updated"`
	UsersLinked        int                `json:"users_linked"`
	CollegesCreated    int                `json:"colleges_created"`
	DepartmentsCreated int                `json:"departments_created"`
	Problems           []importer.Problem `json:"problems"`
	MissingColumns     []string           `json:"missing_columns,omitempty"`
	Preview            []importer.Row     `json:"preview,omitempty"`
}

// ImportResearchers accepts an .xlsx file (multipart "file"). With ?dry_run=true it
// only validates and previews. Rows are applied one by one so a single bad row
// fails alone and appears in the report instead of aborting the whole file.
func (h *AdminImportHandler) ImportResearchers(w http.ResponseWriter, r *http.Request) {
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

	sheet, ok := readImportSheet(w, data, importer.MaxRows)
	if !ok {
		return
	}
	rows := sheet.rows

	header, missing := importer.MapHeader(rows[0])
	if len(missing) > 0 {
		model.RespondJSON(w, http.StatusUnprocessableEntity, importReport{
			DryRun: true, MissingColumns: missing,
		})
		return
	}
	valid, problems := importer.ParseRows(header, rows[1:], 2)

	if sheet.truncated && len(valid) < importer.MaxRows {
		problems = append(problems, importer.Problem{Line: len(rows) + 1, Code: "too_many_rows"})
	}

	report := importReport{ValidRows: len(valid), Problems: problems}
	if report.Problems == nil {
		report.Problems = []importer.Problem{}
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

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	orgs := newOrgCache()
	for _, row := range valid {
		staged := orgs.clone()
		result, err := h.applyRow(ctx, staged, row)
		if err != nil {
			report.Problems = append(report.Problems, importer.Problem{Line: row.Line, Code: importFailureCode(row.Line, err)})
			continue
		}
		orgs = staged
		report.addResult(row.Line, result)
	}
	report.CollegesCreated, report.DepartmentsCreated = orgs.created()
	model.RespondJSON(w, http.StatusOK, report)
}

// orgCache avoids re-resolving the same college/department name on every row.
type orgCache struct {
	colleges, departments       map[string]string
	newColleges, newDepartments int
}

func newOrgCache() *orgCache {
	return &orgCache{colleges: map[string]string{}, departments: map[string]string{}}
}

// clone lets a row work on a copy, so what a rolled-back row created is forgotten with it.
func (c *orgCache) clone() *orgCache {
	return &orgCache{
		colleges: maps.Clone(c.colleges), departments: maps.Clone(c.departments),
		newColleges: c.newColleges, newDepartments: c.newDepartments,
	}
}

func (c *orgCache) created() (int, int) { return c.newColleges, c.newDepartments }

func orgKey(parts ...string) string { return strings.ToLower(strings.Join(parts, "|")) }

func ensureCollege(ctx context.Context, tx pgx.Tx, c *orgCache, name string) (*string, error) {
	if name == "" {
		return nil, nil
	}
	if id, ok := c.colleges[orgKey(name)]; ok {
		return &id, nil
	}
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM colleges WHERE lower(trim(name_ar)) = lower(trim($1))`, name).Scan(&id)
	if isNoRows(err) {
		err = tx.QueryRow(ctx, `
			INSERT INTO colleges (name_ar, institution_id)
			SELECT $1, id FROM institutions ORDER BY created_at LIMIT 1
			RETURNING id`, name).Scan(&id)
		if err == nil {
			c.newColleges++
		}
	}
	if err != nil {
		return nil, err
	}
	c.colleges[orgKey(name)] = id
	return &id, nil
}

func ensureDepartment(ctx context.Context, tx pgx.Tx, c *orgCache, name string, collegeID *string) (*string, error) {
	if name == "" || collegeID == nil {
		return nil, nil
	}
	key := orgKey(*collegeID, name)
	if id, ok := c.departments[key]; ok {
		return &id, nil
	}
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id FROM departments WHERE college_id = $1 AND lower(trim(name_ar)) = lower(trim($2))
	`, *collegeID, name).Scan(&id)
	if isNoRows(err) {
		err = tx.QueryRow(ctx,
			`INSERT INTO departments (name_ar, college_id) VALUES ($1, $2) RETURNING id`,
			name, *collegeID).Scan(&id)
		if err == nil {
			c.newDepartments++
		}
	}
	if err != nil {
		return nil, err
	}
	c.departments[key] = id
	return &id, nil
}
