package tmpl

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`

// fakeDOCX hand-rolls a minimal, well-formed OOXML .docx containing the
// given number of paragraphs (default 3) of fake sentence text, and returns
// its path under tempDir. Uses only stdlib archive/zip + encoding/xml — no
// docx library.
func (e *Engine) fakeDOCX(paragraphs ...int) (string, error) {
	n := 3
	if len(paragraphs) > 0 && paragraphs[0] > 0 {
		n = paragraphs[0]
	}

	var body strings.Builder
	for i := 0; i < n; i++ {
		text := e.faker.Sentence(10)
		body.WriteString("<w:p><w:r><w:t xml:space=\"preserve\">")
		body.WriteString(escapeXML(text))
		body.WriteString("</w:t></w:r></w:p>")
	}
	body.WriteString(`<w:sectPr/>`)

	documentXML := fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+
			`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`+
			`<w:body>%s</w:body></w:document>`,
		body.String(),
	)

	// Sanity check: every XML member we're about to write must itself parse.
	for _, member := range []string{contentTypesXML, rootRelsXML, documentXML, documentRelsXML} {
		var v any
		if err := xml.Unmarshal([]byte(member), &v); err != nil {
			return "", fmt.Errorf("tmpl: fakeDOCX: generated invalid xml: %w", err)
		}
	}

	path := e.tempFilePath(".docx")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fakeDOCX: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	members := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  rootRelsXML,
		"word/document.xml":            documentXML,
		"word/_rels/document.xml.rels": documentRelsXML,
	}
	for name, content := range members {
		w, err := zw.Create(name)
		if err != nil {
			zw.Close()
			return "", fmt.Errorf("tmpl: fakeDOCX: %w", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			zw.Close()
			return "", fmt.Errorf("tmpl: fakeDOCX: %w", err)
		}
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("tmpl: fakeDOCX: %w", err)
	}

	return path, nil
}

func escapeXML(s string) string {
	var buf strings.Builder
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// docFuncMap merges the pdf/xlsx/docx generator functions into one FuncMap.
func (e *Engine) docFuncMap() map[string]any {
	return map[string]any{
		"fakePDF":  e.fakePDF,
		"fakeXLSX": e.fakeXLSX,
		"fakeDOCX": e.fakeDOCX,
	}
}
