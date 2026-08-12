package tmpl

import (
	"os"
	"strings"
	"testing"
)

func TestFakePDF(t *testing.T) {
	e, err := New(11, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fPDF(2)
	if err != nil {
		t.Fatalf("fPDF: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Errorf("expected file to start with %%PDF-, got %q", string(data[:min(20, len(data))]))
	}

	tail := string(data)
	if len(tail) > 200 {
		tail = tail[len(tail)-200:]
	}
	if !strings.Contains(tail, "%%EOF") {
		t.Errorf("expected file to contain %%%%EOF near the end, tail was %q", tail)
	}
}
