// Package fs003 fills the ministry form FS 003 «استمارة تقدم عمل الجامعة او الهيئة»
// (مكتب الوكيل الاداري / قسم البرنامج الحكومي) from the official blank workbook.
//
// The template has 13 sheets, one per category, in the order of activity.Categories
// (the form has no sheet 8). Every sheet shares one layout:
//
//	C2 header · F2/G2 form code · F3/G3 «اسم المنسق :» · C5 the form title line ·
//	C7 section title · row 9 headers ت | title | category column | date | notes ·
//	rows 10–29 numbered data rows · row 30 «المجموع» (B30:C30 merged).
package fs003

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/haydary1986/turath-research/backend/internal/service/activity"
)

//go:embed template.xlsx
var templateXLSX []byte

// Layout of every sheet of the template.
const (
	FirstDataRow     = 10
	TemplateDataRows = 20
	TitleCell        = "C5"

	firstCol        = 2 // B: the row number «ت»
	lastStyledCol   = 7 // G: the outer border column of the wide sheets
	styleSourceRow  = FirstDataRow + TemplateDataRows - 2
	lastTemplateRow = FirstDataRow + TemplateDataRows - 1
	coordinatorRow  = 3
	coordinatorMark = "اسم المنسق"
	formTitle       = "استمارة تقدم عمل الجامعة او الهيئة"
)

// Row is one line of a sheet. Infrastructure uses Count for its «العدد» column;
// every other sheet uses Detail (venue, journal, publisher, counter-party…).
type Row struct {
	Title  string
	Detail string
	Count  int
	Date   string // already formatted for display
	Notes  string
}

// Meta is what the form's header asks for.
type Meta struct {
	Organization string // university, or university and college
	Period       string // e.g. «لشهر تشرين الثاني عام 2026»
	Coordinator  string // «اسم المنسق»
}

// Title is the filled form line of C5.
func Title(meta Meta) string {
	parts := []string{formTitle}
	for _, p := range []string{meta.Organization, meta.Period} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " ")
}

// TotalRow returns the row of «المجموع» once a sheet holds n records.
func TotalRow(n int) int {
	return lastTemplateRow + 1 + extraRows(n)
}

func extraRows(n int) int {
	if n > TemplateDataRows {
		return n - TemplateDataRows
	}
	return 0
}

// Build opens the embedded template and fills it. rows is keyed by category.
func Build(meta Meta, rows map[string][]Row) (*excelize.File, error) {
	book, err := excelize.OpenReader(bytes.NewReader(templateXLSX))
	if err != nil {
		return nil, fmt.Errorf("fs003: open template: %w", err)
	}
	sheets := book.GetSheetList()
	if len(sheets) != len(activity.Categories) {
		_ = book.Close()
		return nil, fmt.Errorf("fs003: template has %d sheets, want %d", len(sheets), len(activity.Categories))
	}
	for i, category := range activity.Categories {
		if err := fillSheet(book, sheets[i], category, meta, rows[category]); err != nil {
			_ = book.Close()
			return nil, fmt.Errorf("fs003: sheet %q: %w", sheets[i], err)
		}
	}
	book.SetActiveSheet(0)
	return book, nil
}

// SheetFor returns the template's sheet name of a category (for tests and callers that read back).
func SheetFor(book *excelize.File, category string) string {
	sheets := book.GetSheetList()
	for i, c := range activity.Categories {
		if c == category && i < len(sheets) {
			return sheets[i]
		}
	}
	return ""
}

func fillSheet(book *excelize.File, sheet, category string, meta Meta, rows []Row) error {
	if err := book.SetCellStr(sheet, TitleCell, Title(meta)); err != nil {
		return err
	}
	if err := setCoordinator(book, sheet, meta.Coordinator); err != nil {
		return err
	}
	if extra := extraRows(len(rows)); extra > 0 {
		if err := insertDataRows(book, sheet, extra); err != nil {
			return err
		}
	}
	total := 0
	for i, row := range rows {
		r := FirstDataRow + i
		if err := writeRow(book, sheet, r, i+1, category, row); err != nil {
			return err
		}
		if category == activity.CategoryInfrastructure {
			total += row.Count
		}
	}
	if category != activity.CategoryInfrastructure {
		total = len(rows)
	}
	return book.SetCellInt(sheet, cell(4, TotalRow(len(rows))), int64(total))
}

func writeRow(book *excelize.File, sheet string, r, number int, category string, row Row) error {
	if err := book.SetCellInt(sheet, cell(firstCol, r), int64(number)); err != nil {
		return err
	}
	if err := book.SetCellStr(sheet, cell(3, r), row.Title); err != nil {
		return err
	}
	var err error
	if category == activity.CategoryInfrastructure {
		err = book.SetCellInt(sheet, cell(4, r), int64(row.Count))
	} else {
		err = book.SetCellStr(sheet, cell(4, r), row.Detail)
	}
	if err != nil {
		return err
	}
	if err := book.SetCellStr(sheet, cell(5, r), row.Date); err != nil {
		return err
	}
	return book.SetCellStr(sheet, cell(6, r), row.Notes)
}

// setCoordinator fills «اسم المنسق :», which sits in F3 or (on the wider sheets) G3.
func setCoordinator(book *excelize.File, sheet, name string) error {
	for col := firstCol; col <= lastStyledCol+1; col++ {
		ref := cell(col, coordinatorRow)
		value, err := book.GetCellValue(sheet, ref)
		if err != nil {
			return err
		}
		if strings.Contains(value, coordinatorMark) {
			return book.SetCellStr(sheet, ref, coordinatorMark+" : "+strings.TrimSpace(name))
		}
	}
	return fmt.Errorf("coordinator cell not found")
}

// insertDataRows adds rows above the last template row (which carries the table's
// bottom border), copying a regular data row's cell styles and height.
func insertDataRows(book *excelize.File, sheet string, extra int) error {
	if err := book.InsertRows(sheet, lastTemplateRow, extra); err != nil {
		return err
	}
	height, err := book.GetRowHeight(sheet, styleSourceRow)
	if err != nil {
		return err
	}
	first, last := lastTemplateRow, lastTemplateRow+extra-1
	for col := firstCol; col <= lastStyledCol; col++ {
		style, err := book.GetCellStyle(sheet, cell(col, styleSourceRow))
		if err != nil {
			return err
		}
		if err := book.SetCellStyle(sheet, cell(col, first), cell(col, last), style); err != nil {
			return err
		}
	}
	for r := first; r <= last; r++ {
		if err := book.SetRowHeight(sheet, r, height); err != nil {
			return err
		}
	}
	return extendPrintArea(book, sheet, extra)
}

// extendPrintArea grows the sheet's print area by the inserted rows when excelize did not.
func extendPrintArea(book *excelize.File, sheet string, extra int) error {
	for _, dn := range book.GetDefinedName() {
		if dn.Name != "_xlnm.Print_Area" || dn.Scope != sheet {
			continue
		}
		want := lastTemplateRow + 1 + extra
		if strings.HasSuffix(dn.RefersTo, fmt.Sprintf("$%d", want)) {
			return nil
		}
		old := fmt.Sprintf("$%d", lastTemplateRow+1)
		if !strings.HasSuffix(dn.RefersTo, old) {
			return nil // an unexpected print area is left alone rather than guessed at
		}
		updated := dn
		updated.RefersTo = strings.TrimSuffix(dn.RefersTo, old) + fmt.Sprintf("$%d", want)
		if err := book.DeleteDefinedName(&excelize.DefinedName{Name: dn.Name, Scope: dn.Scope}); err != nil {
			return err
		}
		return book.SetDefinedName(&updated)
	}
	return nil
}

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
