package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuri/excelize/v2"
)

func workbookBytes(t *testing.T, fill func(f *excelize.File)) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	fill(f)
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func numberedRows(rows int) func(*excelize.File) {
	return func(f *excelize.File) {
		for i := 1; i <= rows; i++ {
			if err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i), fmt.Sprintf("v%d", i)); err != nil {
				panic(err)
			}
		}
	}
}

// withPadding re-zips a workbook with a highly compressible extra member, the shape of a zip bomb.
func withPadding(t *testing.T, data []byte, size int64) []byte {
	t.Helper()
	source, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	target := zip.NewWriter(&out)
	for _, member := range source.File {
		if err := target.Copy(member); err != nil {
			t.Fatal(err)
		}
	}
	padding, err := target.Create("xl/media/padding.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(padding, zeros{}, size); err != nil {
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestReadImportSheetRejectsZipBomb(t *testing.T) {
	bomb := withPadding(t, workbookBytes(t, numberedRows(2)), maxUnzippedBytes+1)
	if len(bomb) > maxImportBytes {
		t.Fatalf("the test file must pass the upload limit, got %d bytes", len(bomb))
	}
	rec := httptest.NewRecorder()
	if _, ok := readImportSheet(rec, bomb, 10); ok {
		t.Fatal("a file unpacking past the limit was accepted")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestReadImportSheetStopsAtTheRowCap(t *testing.T) {
	rec := httptest.NewRecorder()
	sheet, ok := readImportSheet(rec, workbookBytes(t, numberedRows(50)), 10)
	if !ok {
		t.Fatalf("refused: %d", rec.Code)
	}
	if !sheet.truncated || len(sheet.rows) != 11 {
		t.Errorf("rows = %d, truncated = %v, want the header plus 10", len(sheet.rows), sheet.truncated)
	}

	sheet, ok = readImportSheet(httptest.NewRecorder(), workbookBytes(t, numberedRows(11)), 10)
	if !ok || sheet.truncated || len(sheet.rows) != 11 {
		t.Errorf("a sheet at the cap: rows = %d, truncated = %v, ok = %v", len(sheet.rows), sheet.truncated, ok)
	}
}

// withSheetXML swaps in raw worksheet XML: excelize itself would allocate every
// row and column up to a far-away cell, which is the attack, not the test setup.
func withSheetXML(t *testing.T, data []byte, sheetData string) []byte {
	t.Helper()
	source, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	target := zip.NewWriter(&out)
	for _, member := range source.File {
		if member.Name != "xl/worksheets/sheet1.xml" {
			if err := target.Copy(member); err != nil {
				t.Fatal(err)
			}
			continue
		}
		sheet, err := target.Create(member.Name)
		if err != nil {
			t.Fatal(err)
		}
		xml := `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` + sheetData + `</sheetData></worksheet>`
		if _, err := sheet.Write([]byte(xml)); err != nil {
			t.Fatal(err)
		}
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestReadImportSheetBoundsSparseSheets(t *testing.T) {
	inline := func(ref string) string {
		return `<c r="` + ref + `" t="inlineStr"><is><t>x</t></is></c>`
	}
	data := withSheetXML(t, workbookBytes(t, numberedRows(1)),
		`<row r="1">`+inline("A1")+`</row><row r="2">`+inline("A2")+`</row>`+
			`<row r="3">`+inline("XFD3")+`</row><row r="1048576">`+inline("A1048576")+`</row>`)
	sheet, ok := readImportSheet(httptest.NewRecorder(), data, 10)
	if !ok {
		t.Fatal("refused")
	}
	if !sheet.truncated || len(sheet.rows) != 11 {
		t.Errorf("rows = %d, truncated = %v", len(sheet.rows), sheet.truncated)
	}
	for _, row := range sheet.rows {
		if len(row) > maxImportColumns {
			t.Fatalf("row kept %d columns", len(row))
		}
	}
}

func TestReadImportSheetRefusals(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, ok := readImportSheet(rec, []byte("not a workbook"), 10); ok || rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("garbage: ok = %v, status = %d", ok, rec.Code)
	}
	rec = httptest.NewRecorder()
	if _, ok := readImportSheet(rec, workbookBytes(t, numberedRows(1)), 10); ok || rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("header only: ok = %v, status = %d", ok, rec.Code)
	}
}
