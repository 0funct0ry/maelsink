package tmpl

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// fakeXLSX creates a single-sheet workbook of rows x cols (default 10x5) of
// fake cell values and returns its path under tempDir.
func (e *Engine) fakeXLSX(rowsCols ...int) (string, error) {
	rows, cols := 10, 5
	if len(rowsCols) > 0 && rowsCols[0] > 0 {
		rows = rowsCols[0]
	}
	if len(rowsCols) > 1 && rowsCols[1] > 0 {
		cols = rowsCols[1]
	}

	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Sheet1"

	for c := 0; c < cols; c++ {
		cell, err := excelize.CoordinatesToCellName(c+1, 1)
		if err != nil {
			return "", fmt.Errorf("tmpl: fakeXLSX: %w", err)
		}
		if err := f.SetCellValue(sheet, cell, fmt.Sprintf("col_%d", c+1)); err != nil {
			return "", fmt.Errorf("tmpl: fakeXLSX: %w", err)
		}
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return "", fmt.Errorf("tmpl: fakeXLSX: %w", err)
			}
			if err := f.SetCellValue(sheet, cell, e.randString(8)); err != nil {
				return "", fmt.Errorf("tmpl: fakeXLSX: %w", err)
			}
		}
	}

	path := e.tempFilePath(".xlsx")
	if err := f.SaveAs(path); err != nil {
		return "", fmt.Errorf("tmpl: fakeXLSX: %w", err)
	}
	return path, nil
}
