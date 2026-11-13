package fs003

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/haydary1986/turath-research/backend/internal/service/activity"
)

var testMeta = Meta{
	Organization: "جامعة التراث – كلية الهندسة",
	Period:       "لشهر تشرين الثاني عام 2026",
	Coordinator:  "د. سارة علي",
}

// roundTrip builds the workbook, writes it out and opens the bytes again, as a client would.
func roundTrip(t *testing.T, rows map[string][]Row) *excelize.File {
	t.Helper()
	book, err := Build(testMeta, rows)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := book.Write(&buf); err != nil {
		t.Fatal(err)
	}
	_ = book.Close()
	opened, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	return opened
}

func value(t *testing.T, f *excelize.File, sheet, ref string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, ref)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTemplateSheetsFollowCategories(t *testing.T) {
	f := roundTrip(t, nil)
	sheets := f.GetSheetList()
	if len(sheets) != 13 {
		t.Fatalf("13 sheets expected, got %d", len(sheets))
	}
	// Sheet names start with their form number; there is no sheet 8.
	prefixes := []string{"1-", "2-", "3-", "4-", "5-", "6-", "7-", "9-", "10-", "11-", "1٢-", "13-", "14-"}
	for i, p := range prefixes {
		if !strings.HasPrefix(sheets[i], p) {
			t.Errorf("sheet %d = %q, want prefix %q", i, sheets[i], p)
		}
	}
	if got := SheetFor(f, activity.CategoryLocalAgreements); got != sheets[10] {
		t.Errorf("local agreements sheet = %q", got)
	}
}

func TestHeadersAndCoordinator(t *testing.T) {
	f := roundTrip(t, nil)
	for _, category := range activity.Categories {
		sheet := SheetFor(f, category)
		title := value(t, f, sheet, TitleCell)
		if !strings.HasPrefix(title, "استمارة تقدم عمل الجامعة او الهيئة جامعة التراث – كلية الهندسة") ||
			!strings.Contains(title, "لشهر تشرين الثاني عام 2026") || strings.Contains(title, "____") {
			t.Errorf("%s: title %q", sheet, title)
		}
		if value(t, f, sheet, "B9") != "ت" {
			t.Errorf("%s: header row moved", sheet)
		}
	}
	// The coordinator sits in F3, or G3 on the two agreement sheets.
	if got := value(t, f, SheetFor(f, activity.CategoryWorkshops), "F3"); got != "اسم المنسق : د. سارة علي" {
		t.Errorf("F3 = %q", got)
	}
	agreements := SheetFor(f, activity.CategoryInternationalAgreements)
	if got := value(t, f, agreements, "G3"); got != "اسم المنسق : د. سارة علي" {
		t.Errorf("G3 = %q", got)
	}
	if strings.TrimSpace(value(t, f, agreements, "G2")) != "FS 003" {
		t.Error("form code must stay untouched")
	}
}

func TestFirstDataRowsAndTotals(t *testing.T) {
	f := roundTrip(t, map[string][]Row{
		activity.CategoryWorkshops: {
			{Title: "ورشة الذكاء الاصطناعي", Detail: "قاعة المؤتمرات", Date: "02/11/2026", Notes: "حضوري"},
			{Title: "ندوة الطاقة", Detail: "كلية الهندسة", Date: "05/11/2026"},
		},
		activity.CategoryInfrastructure: {
			{Title: "مختبر حاسوب", Count: 2, Date: "01/11/2026"},
			{Title: "قاعة دراسية", Count: 3, Date: "03/11/2026"},
		},
	})
	sheet := SheetFor(f, activity.CategoryWorkshops)
	want := map[string]string{
		"B10": "1", "C10": "ورشة الذكاء الاصطناعي", "D10": "قاعة المؤتمرات", "E10": "02/11/2026", "F10": "حضوري",
		"B11": "2", "C11": "ندوة الطاقة", "D11": "كلية الهندسة",
		"C12": "", "B30": "المجموع ", "D30": "2",
	}
	for ref, w := range want {
		if got := value(t, f, sheet, ref); got != w {
			t.Errorf("%s = %q, want %q", ref, got, w)
		}
	}
	infra := SheetFor(f, activity.CategoryInfrastructure)
	if value(t, f, infra, "D10") != "2" || value(t, f, infra, "D11") != "3" || value(t, f, infra, "D30") != "5" {
		t.Error("infrastructure writes counts and sums them in the total row")
	}
	empty := SheetFor(f, activity.CategoryPatents)
	if value(t, f, empty, "C10") != "" || value(t, f, empty, "D30") != "0" {
		t.Error("an empty sheet keeps its blank rows and totals zero")
	}
}

func TestRowInsertionBeyondTemplate(t *testing.T) {
	const n = TemplateDataRows + 5
	rows := make([]Row, n)
	for i := range rows {
		rows[i] = Row{Title: fmt.Sprintf("اتفاقية %d", i+1), Detail: "جامعة", Date: "01/11/2026"}
	}
	f := roundTrip(t, map[string][]Row{activity.CategoryInternationalAgreements: rows})
	sheet := SheetFor(f, activity.CategoryInternationalAgreements)

	last := FirstDataRow + n - 1
	if got := value(t, f, sheet, fmt.Sprintf("C%d", last)); got != fmt.Sprintf("اتفاقية %d", n) {
		t.Errorf("last row C%d = %q", last, got)
	}
	if got := value(t, f, sheet, fmt.Sprintf("B%d", last)); got != fmt.Sprint(n) {
		t.Errorf("rows stay numbered: B%d = %q", last, got)
	}
	total := TotalRow(n)
	if total != last+1 {
		t.Fatalf("total row %d should follow the last data row %d", total, last)
	}
	if value(t, f, sheet, fmt.Sprintf("B%d", total)) != "المجموع " || value(t, f, sheet, fmt.Sprintf("D%d", total)) != fmt.Sprint(n) {
		t.Error("total row moved down with its label and count")
	}
	merged, _ := f.GetMergeCells(sheet)
	found := false
	for _, m := range merged {
		if m.GetStartAxis() == fmt.Sprintf("B%d", total) && m.GetEndAxis() == fmt.Sprintf("C%d", total) {
			found = true
		}
	}
	if !found {
		t.Error("the merged «المجموع» cell must follow the inserted rows")
	}

	// Inserted rows copy a template data row's style; the last row keeps its bottom border.
	srcStyle, _ := f.GetCellStyle(sheet, fmt.Sprintf("C%d", styleSourceRow))
	insStyle, _ := f.GetCellStyle(sheet, fmt.Sprintf("C%d", lastTemplateRow+2))
	if srcStyle == 0 || insStyle != srcStyle {
		t.Errorf("inserted row style %d, want %d", insStyle, srcStyle)
	}
	srcHeight, _ := f.GetRowHeight(sheet, styleSourceRow)
	insHeight, _ := f.GetRowHeight(sheet, lastTemplateRow+2)
	if insHeight != srcHeight {
		t.Errorf("inserted row height %v, want %v", insHeight, srcHeight)
	}

	for _, dn := range f.GetDefinedName() {
		if dn.Name == "_xlnm.Print_Area" && dn.Scope == sheet && !strings.HasSuffix(dn.RefersTo, fmt.Sprintf("$%d", total)) {
			t.Errorf("print area %q does not reach the total row %d", dn.RefersTo, total)
		}
	}
	other := SheetFor(f, activity.CategoryWorkshops)
	if value(t, f, other, "B30") != "المجموع " {
		t.Error("other sheets keep the template layout")
	}
}

func TestPeriodPhrase(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse(isoDate, s); return v }
	cases := []struct{ from, to, want string }{
		{"2026-11-01", "2026-11-30", "لشهر تشرين الثاني عام 2026"},
		{"2026-02-01", "2026-02-28", "لشهر شباط عام 2026"},
		{"2026-01-01", "2026-12-31", "لعام 2026"},
		{"2026-01-01", "2026-03-31", "للمدة من 01/01/2026 إلى 31/03/2026"},
		{"2026-11-01", "2026-11-29", "للمدة من 01/11/2026 إلى 29/11/2026"},
	}
	for _, c := range cases {
		if got := PeriodPhrase(d(c.from), d(c.to)); got != c.want {
			t.Errorf("%s..%s = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	if FormatDate("2026-03-05") != "05/03/2026" || FormatDate("garbage") != "garbage" {
		t.Error("date formatting")
	}
	if Baghdad == nil || time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).In(Baghdad).Hour() != 3 {
		t.Error("Baghdad is UTC+3")
	}
}

func TestTitleSkipsBlankParts(t *testing.T) {
	if got := Title(Meta{Organization: " جامعة التراث "}); got != "استمارة تقدم عمل الجامعة او الهيئة جامعة التراث" {
		t.Errorf("title = %q", got)
	}
}
