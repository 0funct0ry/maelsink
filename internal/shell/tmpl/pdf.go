package tmpl

import (
	"fmt"

	"github.com/go-pdf/fpdf"
)

// fakePDF creates a PDF with the given number of pages (default 1), each
// containing a paragraph of fake text, and returns its path under tempDir.
func (e *Engine) fakePDF(pages ...int) (string, error) {
	n := 1
	if len(pages) > 0 && pages[0] > 0 {
		n = pages[0]
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Arial", "", 12)

	for i := 0; i < n; i++ {
		pdf.AddPage()
		pdf.Cellf(0, 10, "Page %d", i+1)
		pdf.Ln(12)
		pdf.MultiCell(0, 8, e.faker.Paragraph(3, 5, 12, " "), "", "L", false)
	}

	path := e.tempFilePath(".pdf")
	if err := pdf.OutputFileAndClose(path); err != nil {
		return "", fmt.Errorf("tmpl: fakePDF: %w", err)
	}
	return path, nil
}
