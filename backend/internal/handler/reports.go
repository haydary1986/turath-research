package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/portal"
)

// Arabic labels for raw enum values — the workbook goes to the ministry and
// must never show internal English codes.
var reportStatusAr = map[string]string{
	"draft":               "مسودة",
	"pending_dean":        "بانتظار موافقة العميد",
	"pending":             "لدى الشؤون العلمية",
	"pending_vp":          "بانتظار تأكيد المساعد العلمي",
	"returned":            "معاد للتعديل",
	"approved":            "معتمدة",
	"rejected":            "مرفوضة",
	"verified":            "مستوفية للرصانة",
	"not_eligible":        "غير مستوفية",
	"awaiting_signatures": "بانتظار التواقيع",
	"completed":           "مكتملة",
	"issued":              "صادر",
}

var activityStatusAr = map[string]string{
	"pending":  "بانتظار اعتماد الشؤون العلمية",
	"approved": "معتمدة",
	"returned": "معادة إلى المنظّم للتعديل",
	"rejected": "مرفوضة",
}

var activityTypeAr = map[string]string{
	"seminar":    "ندوة",
	"workshop":   "ورشة عمل",
	"panel":      "حلقة نقاشية",
	"lecture":    "محاضرة",
	"course":     "دورة تدريبية",
	"conference": "مؤتمر",
	"other":      "أخرى",
}

// activityCategoryAr names the FS 003 sheets as the ministry form does.
var activityCategoryAr = map[string]string{
	"infrastructure":           "البنى التحتية",
	"workshops":                "الورش والندوات والحلقات النقاشية",
	"courses":                  "الدورات",
	"local_research":           "البحوث المنشورة محليا",
	"indexed_research":         "البحوث المنشورة في المستوعبات العالمية",
	"international_research":   "البحوث المنشورة دوليا",
	"books":                    "الكتب المؤلفة والمترجمة",
	"patents":                  "براءات الاختراع",
	"conferences":              "المؤتمرات العلمية",
	"international_agreements": "الاتفاقيات الدولية",
	"local_agreements":         "الاتفاقيات المحلية",
	"fellowships":              "الزمالات البحثية والدراسة ما بعد الدكتوراه والتفرغ العلمي",
	"community_service":        "خدمة المجتمع",
}

var attendanceModeAr = map[string]string{
	"in_person": "حضوري",
	"online":    "إلكتروني",
	"hybrid":    "مدمج",
}

var extractedAr = map[string]string{
	"not_extracted":    "غير مستل",
	"extracted_master": "مستل من رسالة ماجستير",
	"extracted_phd":    "مستل من أطروحة دكتوراه",
}

var workTypeAr = map[string]string{"individual": "فردي", "joint": "مشترك"}

func labelOr(m map[string]string, key string) string {
	if key == "" {
		return ""
	}
	if v, ok := m[key]; ok {
		return v
	}
	return key
}

type reportSheet struct {
	name    string
	headers []string
	query   string
	// transform rewrites raw enum columns into Arabic, keyed by column index.
	transform map[int]func(string) string
}

const (
	reportTimeout = 2 * time.Minute
	reportDate    = "2006-01-02"

	// Reporting days are Baghdad days whatever the server's time zone is. The status test keeps
	// the cast away from unvalidated drafts (SQL does not guarantee WHERE evaluation order);
	// statuses past submission passed requireDate.
	reportSubmittedDate   = `(r.submitted_at AT TIME ZONE 'Asia/Baghdad')::date`
	reportPublicationDate = `COALESCE(CASE
		WHEN r.status IN ('verified', 'awaiting_signatures', 'completed')
		 AND r.details->>'publication_date' ~ '^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$'
		THEN (r.details->>'publication_date')::date
	END, ` + reportSubmittedDate + `)`
)

type reportPeriod struct {
	start, end time.Time
	label      string // shown inside the workbook
	slug       string // identifies the period in the file name
}

func (p reportPeriod) filename() string { return "turath-report-" + p.slug + ".xlsx" }

// reportRange resolves the requested period: explicit from/to dates win, otherwise the
// whole requested (or current) year. It returns "field:code" problems for unusable input.
func reportRange(q url.Values, now time.Time) (reportPeriod, []string) {
	var problems []string
	year := now.Year()
	if raw := q.Get("year"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 2000 || parsed > 2100 {
			problems = append(problems, "year:invalid")
		}
		year = parsed
	}

	rawFrom, rawTo := q.Get("from"), q.Get("to")
	if rawFrom == "" && rawTo == "" {
		return reportPeriod{
			start: time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC),
			label: strconv.Itoa(year),
			slug:  strconv.Itoa(year),
		}, problems
	}

	from, errFrom := time.Parse(reportDate, rawFrom)
	if errFrom != nil {
		problems = append(problems, "from:invalid_date")
	}
	to, errTo := time.Parse(reportDate, rawTo)
	if errTo != nil {
		problems = append(problems, "to:invalid_date")
	}
	if errFrom == nil && errTo == nil && to.Before(from) {
		problems = append(problems, "to:invalid")
	}
	return reportPeriod{
		start: from,
		end:   to,
		label: rawFrom + " — " + rawTo,
		slug:  rawFrom + "_" + rawTo,
	}, problems
}

// AnnualReport builds the ministry-style workbook (staff only): a statistics
// summary per college, then one RTL sheet per register holding the APPROVED
// records of the period — publications, appreciation letters, research plan,
// scientific activities.
func (h *PortalHandler) AnnualReport(w http.ResponseWriter, r *http.Request) {
	period, problems := reportRange(r.URL.Query(), time.Now())
	if len(problems) > 0 {
		model.RespondError(w, http.StatusBadRequest, "validation:"+strings.Join(problems, ","))
		return
	}
	start, end, periodLabel := period.start, period.end, period.label
	ctx, cancel := longRunningContext(w, r, reportTimeout)
	defer cancel()

	book := excelize.NewFile()
	defer book.Close() //nolint:errcheck // in-memory workbook

	sheets := reportSheets()

	titleStyle, _ := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "0F766E"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	headerStyle, _ := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 12},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"0F766E"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	totalStyle, _ := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	styles := sheetStyles{title: titleStyle, header: headerStyle, total: totalStyle}

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
		if err := h.fillSheet(ctx, book, sheet, start, end, periodLabel, styles); err != nil {
			serverError(w, "Failed to build report", err)
			return
		}
	}

	if err := h.addSummarySheet(ctx, book, start, end, periodLabel, styles); err != nil {
		serverError(w, "Failed to build report", err)
		return
	}
	if idx, err := book.GetSheetIndex("الملخص الإحصائي"); err == nil {
		book.SetActiveSheet(idx)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, period.filename()))
	if err := book.Write(w); err != nil {
		// Headers are already sent; a JSON error would corrupt the file. Log only.
		slog.Error("portal: report stream interrupted", "error", err)
	}
}

// reportSheets defines the registers of the workbook. Query columns line up with headers one to one.
func reportSheets() []reportSheet {
	return []reportSheet{
		{
			name: "النشر العلمي",
			headers: []string{
				"رقم الطلب", "الباحث", "الكلية", "القسم", "عنوان البحث", "المجلة", "الناشر",
				"المستوعب", "التصنيف", "تاريخ القبول", "تاريخ النشر", "المجلد", "العدد", "الصفحات",
				"ترتيب الباحث", "عدد الباحثين", "الباحثون المشاركون", "مستل", "نسبة الاستلال", "DOI", "حالة الطلب",
			},
			query: `SELECT COALESCE(r.request_no, ''), COALESCE(u.name_ar, u.email), COALESCE(c.name_ar, ''), COALESCE(d.name_ar, ''),
			        COALESCE(r.title_en, r.title_ar, ''), COALESCE(r.details->>'journal_name', ''), COALESCE(r.details->>'publisher', ''),
			        COALESCE(r.details->>'index_db', ''), COALESCE(r.details->>'quartile', ''),
			        COALESCE(r.details->>'acceptance_date', ''), COALESCE(r.details->>'publication_date', ''),
			        COALESCE(r.details->>'volume', ''), COALESCE(r.details->>'issue', ''), COALESCE(r.details->>'pages', ''),
			        COALESCE(r.details->>'author_order', ''), COALESCE(r.details->>'authors_count', ''),
			        COALESCE(r.details->>'co_authors', ''), COALESCE(r.details->>'extracted_thesis', ''),
			        COALESCE(r.details->>'plagiarism_pct', ''), COALESCE(r.doi, ''), r.status
			 FROM portal_requests r
			 JOIN users u ON u.id = r.applicant_id
			 LEFT JOIN colleges c ON c.id = r.college_id
			 LEFT JOIN departments d ON d.id = r.department_id
			 WHERE r.kind = 'journal_verification'
			   AND r.status IN ('verified', 'awaiting_signatures', 'completed')
			   AND ` + reportPublicationDate + ` BETWEEN $1 AND $2
			 ORDER BY c.name_ar, r.request_no`,
			transform: map[int]func(string) string{
				7:  portal.IndexDBLabel,
				17: func(s string) string { return labelOr(extractedAr, s) },
				20: func(s string) string { return labelOr(reportStatusAr, s) },
			},
		},
		{
			name: "كتب الشكر والتقدير",
			headers: []string{
				"العدد الرسمي (صادرة الديوان)", "الرقم الداخلي", "التاريخ", "المستلم", "الكلية", "القسم", "عنوان البحث",
			},
			query: `SELECT COALESCE(l.diwan_no, ''), l.letter_no, to_char(l.issued_date, 'YYYY-MM-DD'), l.recipient_name,
			        l.college_name, l.department_name, COALESCE(r.title_en, r.title_ar, '')
			 FROM issued_letters l
			 JOIN portal_requests r ON r.id = l.request_id
			 WHERE l.issued_date BETWEEN $1 AND $2
			 ORDER BY l.letter_year, l.letter_seq`,
		},
		{
			name: "الخطة البحثية",
			headers: []string{
				"رقم الطلب", "الباحث", "الكلية", "القسم", "العنوان (عربي)", "العنوان (إنجليزي)",
				"نوع البحث", "الإنجاز المتوقع", "الحالة",
			},
			query: `SELECT COALESCE(r.request_no, ''), COALESCE(u.name_ar, u.email), COALESCE(c.name_ar, ''), COALESCE(d.name_ar, ''),
			        COALESCE(r.title_ar, ''), COALESCE(r.title_en, ''),
			        COALESCE(r.details->>'work_type', ''), COALESCE(r.details->>'expected_completion', ''), r.status
			 FROM portal_requests r
			 JOIN users u ON u.id = r.applicant_id
			 LEFT JOIN colleges c ON c.id = r.college_id
			 LEFT JOIN departments d ON d.id = r.department_id
			 WHERE r.kind = 'research_plan' AND r.status = 'approved'
			   AND ` + reportSubmittedDate + ` BETWEEN $1 AND $2
			 ORDER BY c.name_ar, r.request_no`,
			transform: map[int]func(string) string{
				6: func(s string) string { return labelOr(workTypeAr, s) },
				8: func(s string) string { return labelOr(reportStatusAr, s) },
			},
		},
		{
			name: "النشاطات العلمية",
			headers: []string{
				"التصنيف (استمارة FS 003)", "النوع", "العنوان", "المحاضر", "الكلية", "القسم", "التاريخ", "المكان", "نمط الإقامة",
				"الفئة المستهدفة", "الجهة المنظمة", "عدد الحضور", "عدد كتاب الإقامة", "الحالة",
			},
			query: `SELECT a.category, a.activity_type, a.title_ar, COALESCE(a.presenter, ''), COALESCE(c.name_ar, ''), COALESCE(d.name_ar, ''),
			        to_char(a.activity_date, 'YYYY-MM-DD'), COALESCE(a.location, ''), COALESCE(a.attendance_mode, ''),
			        COALESCE(a.target_audience, ''), COALESCE(a.organizer, ''), a.attendees_count::text,
			        COALESCE(a.approval_no, ''), a.status
			 FROM activities a
			 LEFT JOIN colleges c ON c.id = a.college_id
			 LEFT JOIN departments d ON d.id = a.department_id
			 WHERE a.status = 'approved' AND a.activity_date BETWEEN $1 AND $2
			 ORDER BY c.name_ar, a.activity_date`,
			transform: map[int]func(string) string{
				0:  func(s string) string { return labelOr(activityCategoryAr, s) },
				1:  func(s string) string { return labelOr(activityTypeAr, s) },
				8:  func(s string) string { return labelOr(attendanceModeAr, s) },
				13: func(s string) string { return labelOr(activityStatusAr, s) },
			},
		},
	}
}

type sheetStyles struct {
	title, header, total int
}

func setSheetRTL(book *excelize.File, sheet string) error {
	rtl := true
	return book.SetSheetView(sheet, 0, &excelize.ViewOptions{RightToLeft: &rtl})
}

// writeSheetTitle puts the university banner + period on the first row, merged
// across the table width, and returns nothing: headers go on row 2.
func writeSheetTitle(book *excelize.File, sheet, subtitle string, width int, style int) {
	endCell, _ := excelize.CoordinatesToCellName(width, 1)
	_ = book.MergeCell(sheet, "A1", endCell)
	_ = book.SetCellValue(sheet, "A1", "جامعة التراث — قسم الشؤون العلمية · "+subtitle)
	_ = book.SetCellStyle(sheet, "A1", endCell, style)
	_ = book.SetRowHeight(sheet, 1, 24)
}

func (h *PortalHandler) fillSheet(ctx context.Context, book *excelize.File, sheet reportSheet, start, end time.Time, periodLabel string, styles sheetStyles) error {
	if err := setSheetRTL(book, sheet.name); err != nil {
		return err
	}
	writeSheetTitle(book, sheet.name, sheet.name+" · "+periodLabel, len(sheet.headers), styles.title)
	for col, title := range sheet.headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		if err := book.SetCellValue(sheet.name, cell, title); err != nil {
			return err
		}
	}
	endHeader, _ := excelize.CoordinatesToCellName(len(sheet.headers), 2)
	_ = book.SetCellStyle(sheet.name, "A2", endHeader, styles.header)
	lastCol, _ := excelize.ColumnNumberToName(len(sheet.headers))
	_ = book.SetColWidth(sheet.name, "A", lastCol, 22)

	rows, err := h.db.Query(ctx, sheet.query, start, end)
	if err != nil {
		return err
	}
	defer rows.Close()

	rowIdx := 3
	values := make([]string, len(sheet.headers))
	scan := make([]any, len(sheet.headers))
	for i := range values {
		scan[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scan...); err != nil {
			return err
		}
		for col, v := range values {
			if fn, ok := sheet.transform[col]; ok {
				v = fn(v)
			}
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx)
			if err := book.SetCellValue(sheet.name, cell, v); err != nil {
				return err
			}
		}
		rowIdx++
	}
	if err := rows.Err(); err != nil {
		return err
	}

	totalCell, _ := excelize.CoordinatesToCellName(1, rowIdx)
	_ = book.SetCellValue(sheet.name, totalCell, fmt.Sprintf("المجموع: %d", rowIdx-3))
	_ = book.SetCellStyle(sheet.name, totalCell, totalCell, styles.total)
	return nil
}

// addSummarySheet prepends the per-college statistics the ministry asks for first.
func (h *PortalHandler) addSummarySheet(ctx context.Context, book *excelize.File, start, end time.Time, periodLabel string, styles sheetStyles) error {
	const name = "الملخص الإحصائي"
	if _, err := book.NewSheet(name); err != nil {
		return err
	}
	if err := book.MoveSheet(name, book.GetSheetName(0)); err != nil {
		return err
	}
	if err := setSheetRTL(book, name); err != nil {
		return err
	}

	type row struct {
		pubs, letters, plans, activities int
	}
	byCollege := map[string]*row{}
	get := func(college string) *row {
		if college == "" {
			college = "بدون كلية"
		}
		if _, ok := byCollege[college]; !ok {
			byCollege[college] = &row{}
		}
		return byCollege[college]
	}

	collect := func(query string, assign func(*row, int)) error {
		rows, err := h.db.Query(ctx, query, start, end)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var college string
			var n int
			if err := rows.Scan(&college, &n); err != nil {
				return err
			}
			assign(get(college), n)
		}
		return rows.Err()
	}

	queries := []struct {
		query  string
		assign func(*row, int)
	}{
		{`SELECT COALESCE(c.name_ar, ''), COUNT(*) FROM portal_requests r LEFT JOIN colleges c ON c.id = r.college_id
		  WHERE r.kind = 'journal_verification' AND r.status IN ('verified', 'awaiting_signatures', 'completed')
		    AND ` + reportPublicationDate + ` BETWEEN $1 AND $2
		  GROUP BY 1`, func(r *row, n int) { r.pubs = n }},
		{`SELECT l.college_name, COUNT(*) FROM issued_letters l WHERE l.issued_date BETWEEN $1 AND $2 GROUP BY 1`,
			func(r *row, n int) { r.letters = n }},
		{`SELECT COALESCE(c.name_ar, ''), COUNT(*) FROM portal_requests r LEFT JOIN colleges c ON c.id = r.college_id
		  WHERE r.kind = 'research_plan' AND r.status = 'approved' AND ` + reportSubmittedDate + ` BETWEEN $1 AND $2
		  GROUP BY 1`, func(r *row, n int) { r.plans = n }},
		{`SELECT COALESCE(c.name_ar, ''), COUNT(*) FROM activities a LEFT JOIN colleges c ON c.id = a.college_id
		  WHERE a.status = 'approved' AND a.activity_date BETWEEN $1 AND $2
		  GROUP BY 1`, func(r *row, n int) { r.activities = n }},
	}
	for _, q := range queries {
		if err := collect(q.query, q.assign); err != nil {
			return err
		}
	}

	headers := []string{"الكلية", "البحوث المنشورة (المستوفية)", "كتب الشكر الصادرة", "بحوث الخطة المعتمدة", "النشاطات المعتمدة"}
	writeSheetTitle(book, name, "الملخص الإحصائي · "+periodLabel, len(headers), styles.title)
	for col, title := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		_ = book.SetCellValue(name, cell, title)
	}
	endHeader, _ := excelize.CoordinatesToCellName(len(headers), 2)
	_ = book.SetCellStyle(name, "A2", endHeader, styles.header)
	_ = book.SetColWidth(name, "A", "E", 26)

	colleges := make([]string, 0, len(byCollege))
	for college := range byCollege {
		colleges = append(colleges, college)
	}
	sort.Strings(colleges)

	rowIdx := 3
	var sum row
	for _, college := range colleges {
		r := byCollege[college]
		cells := []any{college, r.pubs, r.letters, r.plans, r.activities}
		for col, v := range cells {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx)
			_ = book.SetCellValue(name, cell, v)
		}
		sum.pubs += r.pubs
		sum.letters += r.letters
		sum.plans += r.plans
		sum.activities += r.activities
		rowIdx++
	}
	totals := []any{"المجموع", sum.pubs, sum.letters, sum.plans, sum.activities}
	for col, v := range totals {
		cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx)
		_ = book.SetCellValue(name, cell, v)
	}
	startTotal, _ := excelize.CoordinatesToCellName(1, rowIdx)
	endTotal, _ := excelize.CoordinatesToCellName(len(totals), rowIdx)
	_ = book.SetCellStyle(name, startTotal, endTotal, styles.total)
	return nil
}
