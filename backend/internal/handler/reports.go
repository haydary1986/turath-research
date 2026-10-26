package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

// AnnualReport builds the ministry-style yearly workbook (staff only):
// one RTL sheet per register — publications, appreciation letters,
// research plan, scientific activities.
func (h *PortalHandler) AnnualReport(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil || year < 2000 || year > 2100 {
		year = time.Now().Year()
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	book := excelize.NewFile()
	defer book.Close() //nolint:errcheck // in-memory workbook

	sheets := []struct {
		name    string
		headers []string
		query   string
	}{
		{
			"النشر العلمي",
			[]string{"رقم الطلب", "الباحث", "الكلية", "القسم", "عنوان البحث", "المجلة", "الناشر", "المستوعب", "التصنيف", "تاريخ النشر", "DOI", "حالة الطلب"},
			`SELECT COALESCE(r.request_no, ''), COALESCE(u.name_ar, u.email), COALESCE(c.name_ar, ''), COALESCE(d.name_ar, ''),
			        COALESCE(r.title_en, r.title_ar, ''), COALESCE(r.details->>'journal_name', ''), COALESCE(r.details->>'publisher', ''),
			        COALESCE(r.details->>'index_db', ''), COALESCE(r.details->>'quartile', ''), COALESCE(r.details->>'publication_date', ''),
			        COALESCE(r.doi, ''), r.status
			 FROM portal_requests r
			 JOIN users u ON u.id = r.applicant_id
			 LEFT JOIN colleges c ON c.id = r.college_id
			 LEFT JOIN departments d ON d.id = r.department_id
			 WHERE r.kind = 'journal_verification' AND r.status NOT IN ('draft')
			   AND EXTRACT(YEAR FROM r.submitted_at) = $1
			 ORDER BY c.name_ar, r.request_no`,
		},
		{
			"كتب الشكر والتقدير",
			[]string{"العدد", "التاريخ", "المستلم", "الكلية", "القسم", "عنوان البحث"},
			`SELECT l.letter_no, to_char(l.issued_date, 'YYYY-MM-DD'), l.recipient_name,
			        l.college_name, l.department_name, COALESCE(r.title_en, r.title_ar, '')
			 FROM issued_letters l
			 JOIN portal_requests r ON r.id = l.request_id
			 WHERE l.letter_year = $1
			 ORDER BY l.letter_seq`,
		},
		{
			"الخطة البحثية",
			[]string{"رقم الطلب", "الباحث", "الكلية", "القسم", "العنوان (عربي)", "العنوان (إنجليزي)", "نوع البحث", "الإنجاز المتوقع", "الحالة"},
			`SELECT COALESCE(r.request_no, ''), COALESCE(u.name_ar, u.email), COALESCE(c.name_ar, ''), COALESCE(d.name_ar, ''),
			        COALESCE(r.title_ar, ''), COALESCE(r.title_en, ''),
			        COALESCE(r.details->>'work_type', ''), COALESCE(r.details->>'expected_completion', ''), r.status
			 FROM portal_requests r
			 JOIN users u ON u.id = r.applicant_id
			 LEFT JOIN colleges c ON c.id = r.college_id
			 LEFT JOIN departments d ON d.id = r.department_id
			 WHERE r.kind = 'research_plan' AND r.status NOT IN ('draft')
			   AND EXTRACT(YEAR FROM r.submitted_at) = $1
			 ORDER BY c.name_ar, r.request_no`,
		},
		{
			"النشاطات العلمية",
			[]string{"النوع", "العنوان", "المحاضر", "الكلية", "القسم", "التاريخ", "المكان", "عدد الحضور", "عدد كتاب الإقامة", "الحالة"},
			`SELECT a.activity_type, a.title_ar, COALESCE(a.presenter, ''), COALESCE(c.name_ar, ''), COALESCE(d.name_ar, ''),
			        to_char(a.activity_date, 'YYYY-MM-DD'), COALESCE(a.location, ''), a.attendees_count::text,
			        COALESCE(a.approval_no, ''), a.status
			 FROM activities a
			 LEFT JOIN colleges c ON c.id = a.college_id
			 LEFT JOIN departments d ON d.id = a.department_id
			 WHERE EXTRACT(YEAR FROM a.activity_date) = $1
			 ORDER BY c.name_ar, a.activity_date`,
		},
	}

	headerStyle, _ := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 12},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"0F766E"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	for i, sheet := range sheets {
		if i == 0 {
			if err := book.SetSheetName("Sheet1", sheet.name); err != nil {
				serverError(w, "Failed to build report", err)
				return
			}
		} else if _, err := book.NewSheet(sheet.name); err != nil {
			serverError(w, "Failed to build report", err)
			return
		}
		if err := h.fillSheet(ctx, book, sheet.name, sheet.headers, sheet.query, year, headerStyle); err != nil {
			serverError(w, "Failed to build report", err)
			return
		}
	}
	book.SetActiveSheet(0)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="turath-report-%d.xlsx"`, year))
	if err := book.Write(w); err != nil {
		// Headers are already sent; a JSON error would corrupt the file. Log only.
		slog.Error("portal: report stream interrupted", "error", err)
	}
}

func (h *PortalHandler) fillSheet(ctx context.Context, book *excelize.File, sheet string, headers []string, query string, year, headerStyle int) error {
	rtl := true
	if err := book.SetSheetView(sheet, 0, &excelize.ViewOptions{RightToLeft: &rtl}); err != nil {
		return err
	}
	for col, title := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := book.SetCellValue(sheet, cell, title); err != nil {
			return err
		}
	}
	endHeader, _ := excelize.CoordinatesToCellName(len(headers), 1)
	_ = book.SetCellStyle(sheet, "A1", endHeader, headerStyle)
	_ = book.SetColWidth(sheet, "A", "L", 22)

	rows, err := h.db.Query(ctx, query, year)
	if err != nil {
		return err
	}
	defer rows.Close()

	rowIdx := 2
	values := make([]string, len(headers))
	scan := make([]any, len(headers))
	for i := range values {
		scan[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scan...); err != nil {
			return err
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx)
			if err := book.SetCellValue(sheet, cell, v); err != nil {
				return err
			}
		}
		rowIdx++
	}
	return rows.Err()
}
