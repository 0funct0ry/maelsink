package tmpl

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"testing"
)

func TestFakeDOCX(t *testing.T) {
	e, err := New(5, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fDOCX(4)
	if err != nil {
		t.Fatalf("fDOCX: %v", err)
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer zr.Close()

	required := map[string]bool{
		"[Content_Types].xml":          false,
		"_rels/.rels":                  false,
		"word/document.xml":            false,
		"word/_rels/document.xml.rels": false,
	}

	for _, f := range zr.File {
		if _, ok := required[f.Name]; !ok {
			continue
		}
		required[f.Name] = true

		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}

		var v any
		if err := xml.Unmarshal(data, &v); err != nil {
			t.Errorf("member %s is not well-formed XML: %v", f.Name, err)
		}
	}

	for name, found := range required {
		if !found {
			t.Errorf("required docx member %q missing from archive", name)
		}
	}
}
