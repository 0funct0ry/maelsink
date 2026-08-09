package tmpl

import (
	"archive/zip"
	"encoding/csv"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
)

func TestFakePNG(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakePNG(32, 24)
	if err != nil {
		t.Fatalf("fakePNG: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 32 || b.Dy() != 24 {
		t.Errorf("expected 32x24, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestFakeJPEG(t *testing.T) {
	e, err := New(2, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakeJPEG(40, 20)
	if err != nil {
		t.Fatalf("fakeJPEG: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatalf("jpeg.Decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 40 || b.Dy() != 20 {
		t.Errorf("expected 40x20, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestFakeGIF(t *testing.T) {
	e, err := New(3, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakeGIF(16, 16)
	if err != nil {
		t.Fatalf("fakeGIF: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	img, err := gif.Decode(f)
	if err != nil {
		t.Fatalf("gif.Decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 16 || b.Dy() != 16 {
		t.Errorf("expected 16x16, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestFakeCSV(t *testing.T) {
	e, err := New(4, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakeCSV(6, 3)
	if err != nil {
		t.Fatalf("fakeCSV: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll: %v", err)
	}
	// Header + 6 data rows.
	if len(rows) != 7 {
		t.Errorf("expected 7 rows (1 header + 6 data), got %d", len(rows))
	}
	for i, row := range rows {
		if len(row) != 3 {
			t.Errorf("row %d: expected 3 cols, got %d", i, len(row))
		}
	}
}

func TestFakeZIP(t *testing.T) {
	e, err := New(5, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakeZIP()
	if err != nil {
		t.Fatalf("fakeZIP: %v", err)
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer zr.Close()

	if len(zr.File) == 0 {
		t.Error("expected zip to be non-empty")
	}
}

func TestFakeBinary(t *testing.T) {
	e, err := New(6, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	path, err := e.fakeBinary("2KB")
	if err != nil {
		t.Fatalf("fakeBinary: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() != 2*1024 {
		t.Errorf("expected size 2048, got %d", info.Size())
	}
}
