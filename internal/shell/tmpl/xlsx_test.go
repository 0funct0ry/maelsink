package tmpl

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestFakeXLSX(t *testing.T) {
	e, err := New(9, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakeXLSX(7, 3)
	if err != nil {
		t.Fatalf("fakeXLSX: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("excelize.OpenFile: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}

	// Header row + 7 data rows = 8 rows.
	if len(rows) != 8 {
		t.Errorf("expected 8 rows (1 header + 7 data), got %d", len(rows))
	}
	for i, row := range rows {
		if len(row) != 3 {
			t.Errorf("row %d: expected 3 cols, got %d (%v)", i, len(row), row)
		}
	}
}
