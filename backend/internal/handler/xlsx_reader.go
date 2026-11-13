package handler

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/haydary1986/turath-research/backend/internal/model"
)

const (
	// The upload limit bounds the compressed size only: a zip of zeros unpacks a
	// thousandfold. These bound what unpacking may cost; sheets past the XML limit
	// spill to a temp file instead of memory.
	maxUnzippedBytes    = 128 << 20
	maxUnzippedXMLBytes = 32 << 20
	// A cell far to the right makes excelize pad the row with blanks up to it.
	maxImportColumns = 64
)

var errSheetEmpty = errors.New("workbook has no sheets")

type importSheet struct {
	rows      [][]string // the header row, then at most maxDataRows lines
	truncated bool       // the sheet held more lines than were read
}

func openImportBook(data []byte) (*excelize.File, error) {
	return excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit: maxUnzippedBytes, UnzipXMLSizeLimit: maxUnzippedXMLBytes,
	})
}

func isUnzipLimitError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unzip size exceeds")
}

// readFirstSheet streams rows and stops at the cap rather than loading the whole sheet.
func readFirstSheet(book *excelize.File, maxDataRows int) (importSheet, error) {
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return importSheet{}, errSheetEmpty
	}
	iterator, err := book.Rows(sheets[0])
	if err != nil {
		return importSheet{}, err
	}
	defer iterator.Close() //nolint:errcheck // removes a temp file at worst

	sheet := importSheet{rows: [][]string{}}
	for iterator.Next() {
		if len(sheet.rows) > maxDataRows {
			sheet.truncated = true
			return sheet, nil
		}
		cells, err := iterator.Columns()
		if err != nil {
			return importSheet{}, err
		}
		if len(cells) > maxImportColumns {
			cells = cells[:maxImportColumns]
		}
		sheet.rows = append(sheet.rows, cells)
	}
	return sheet, iterator.Error()
}

// readImportSheet opens an upload and answers the client itself on any failure.
func readImportSheet(w http.ResponseWriter, data []byte, maxDataRows int) (importSheet, bool) {
	book, err := openImportBook(data)
	if isUnzipLimitError(err) {
		model.RespondError(w, http.StatusRequestEntityTooLarge, "file_too_large")
		return importSheet{}, false
	}
	if err != nil {
		model.RespondError(w, http.StatusUnsupportedMediaType, "unsupported_file_type")
		return importSheet{}, false
	}
	defer book.Close() //nolint:errcheck // removes temp files at worst

	sheet, err := readFirstSheet(book, maxDataRows)
	if err != nil || len(sheet.rows) < 2 {
		model.RespondError(w, http.StatusUnprocessableEntity, "empty_sheet")
		return importSheet{}, false
	}
	return sheet, true
}
