package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

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

	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		model.RespondError(w, http.StatusUnsupportedMediaType, "unsupported_file_type")
		return
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		model.RespondError(w, http.StatusUnprocessableEntity, "empty_sheet")
		return
	}
	rows, err := book.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		model.RespondError(w, http.StatusUnprocessableEntity, "empty_sheet")
		return
	}

	header, missing := importer.MapHeader(rows[0])
	if len(missing) > 0 {
		model.RespondJSON(w, http.StatusUnprocessableEntity, importReport{
			DryRun: true, MissingColumns: missing,
		})
		return
	}
	valid, problems := importer.ParseRows(header, rows[1:], 2)

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
		if err := h.applyRow(ctx, orgs, row, &report); err != nil {
			report.Problems = append(report.Problems, importer.Problem{Line: row.Line, Code: "save_failed"})
		}
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

func (c *orgCache) created() (int, int) { return c.newColleges, c.newDepartments }

func orgKey(parts ...string) string { return strings.ToLower(strings.Join(parts, "|")) }

func (h *AdminImportHandler) ensureCollege(ctx context.Context, c *orgCache, name string) (*string, error) {
	if name == "" {
		return nil, nil
	}
	if id, ok := c.colleges[orgKey(name)]; ok {
		return &id, nil
	}
	var id string
	err := h.db.QueryRow(ctx, `SELECT id FROM colleges WHERE lower(trim(name_ar)) = lower(trim($1))`, name).Scan(&id)
	if isNoRows(err) {
		err = h.db.QueryRow(ctx, `
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

func (h *AdminImportHandler) ensureDepartment(ctx context.Context, c *orgCache, name string, collegeID *string) (*string, error) {
	if name == "" || collegeID == nil {
		return nil, nil
	}
	key := orgKey(*collegeID, name)
	if id, ok := c.departments[key]; ok {
		return &id, nil
	}
	var id string
	err := h.db.QueryRow(ctx, `
		SELECT id FROM departments WHERE college_id = $1 AND lower(trim(name_ar)) = lower(trim($2))
	`, *collegeID, name).Scan(&id)
	if isNoRows(err) {
		err = h.db.QueryRow(ctx,
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

func (h *AdminImportHandler) applyRow(ctx context.Context, orgs *orgCache, row importer.Row, report *importReport) error {
	collegeID, err := h.ensureCollege(ctx, orgs, row.College)
	if err != nil {
		return err
	}
	departmentID, err := h.ensureDepartment(ctx, orgs, row.Department, collegeID)
	if err != nil {
		return err
	}

	var researcherID string
	var inserted bool
	err = h.db.QueryRow(ctx, `
		INSERT INTO researchers (name_ar, name_en, email, title_ar, department_id, orcid_id, status)
		VALUES ($1, NULLIF($2, ''), $3, NULLIF($4, ''), $5, NULLIF($6, ''), 'active')
		ON CONFLICT (email) DO UPDATE SET
			name_ar       = EXCLUDED.name_ar,
			name_en       = COALESCE(EXCLUDED.name_en, researchers.name_en),
			title_ar      = COALESCE(EXCLUDED.title_ar, researchers.title_ar),
			department_id = COALESCE(EXCLUDED.department_id, researchers.department_id),
			orcid_id      = COALESCE(EXCLUDED.orcid_id, researchers.orcid_id),
			updated_at    = NOW()
		RETURNING id, (xmax = 0)
	`, row.NameAr, row.NameEn, row.Email, row.TitleAr, departmentID, row.ORCID).Scan(&researcherID, &inserted)
	if err != nil {
		return err
	}
	if inserted {
		report.Created++
	} else {
		report.Updated++
	}

	// Link an existing portal account with the same university email.
	tag, err := h.db.Exec(ctx, `
		UPDATE users SET researcher_id = $1,
			college_id = COALESCE(users.college_id, $2), department_id = COALESCE(users.department_id, $3)
		WHERE lower(email) = $4 AND researcher_id IS NULL
	`, researcherID, collegeID, departmentID, row.Email)
	if err == nil && tag.RowsAffected() > 0 {
		report.UsersLinked++
	}
	return nil
}
