package tmpl

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// attachDelim is the delimiter joined paths from attach() are split on by
// the shell's send builtin. Using "::" rather than NUL keeps the joined
// string printable/loggable while still being unlikely to collide with a
// real filesystem path.
const attachDelim = "::"

// binaryDocs documents template functions that generate binary/structured
// files on disk under the Engine's temp dir and return their paths.
func (e *Engine) binaryDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "fPNG", Category: CategoryGenerate, Args: "[w] [h]", Returns: "string",
			Description: "Generates a PNG image (default 64x64) and returns its path.", Fn: e.fPNG},
		{Name: "fJPEG", Category: CategoryGenerate, Args: "[w] [h]", Returns: "string",
			Description: "Generates a JPEG image (default 64x64) and returns its path.", Fn: e.fJPEG},
		{Name: "fGIF", Category: CategoryGenerate, Args: "[w] [h]", Returns: "string",
			Description: "Generates a GIF image (default 64x64) and returns its path.", Fn: e.fGIF},
		{Name: "fCSV", Category: CategoryGenerate, Args: "[rows] [cols]", Returns: "string",
			Description: "Generates a CSV file (default 10x5) and returns its path.", Fn: e.fCSV},
		{Name: "fZIP", Category: CategoryGenerate, Args: "[files...]", Returns: "string",
			Description: "Bundles the given paths (or one generated file) into a .zip and returns its path.", Fn: e.fZIP},
		{Name: "fBinary", Category: CategoryGenerate, Args: "size", Returns: "string",
			Description: `Writes size (e.g. "2MB", "512KB", or a plain byte count) of pseudo-random bytes and returns the path.`, Fn: e.fBinary},
	}
}

func dims(wh []int) (w, h int) {
	w, h = 64, 64
	if len(wh) > 0 && wh[0] > 0 {
		w = wh[0]
	}
	if len(wh) > 1 && wh[1] > 0 {
		h = wh[1]
	}
	return
}

func (e *Engine) fakeImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: byte(e.rnd.Intn(256)),
				G: byte(e.rnd.Intn(256)),
				B: byte(e.rnd.Intn(256)),
				A: 255,
			})
		}
	}
	return img
}

// fPNG writes a w x h (default 64x64) PNG of pseudo-random pixels to
// tempDir and returns its path.
func (e *Engine) fPNG(wh ...int) (string, error) {
	w, h := dims(wh)
	path := e.tempFilePath(".png")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fPNG: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, e.fakeImage(w, h)); err != nil {
		return "", fmt.Errorf("tmpl: fPNG: %w", err)
	}
	return path, nil
}

// fJPEG writes a w x h (default 64x64) JPEG of pseudo-random pixels to
// tempDir and returns its path.
func (e *Engine) fJPEG(wh ...int) (string, error) {
	w, h := dims(wh)
	path := e.tempFilePath(".jpg")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fJPEG: %w", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, e.fakeImage(w, h), &jpeg.Options{Quality: 80}); err != nil {
		return "", fmt.Errorf("tmpl: fJPEG: %w", err)
	}
	return path, nil
}

// fGIF writes a w x h (default 64x64) GIF of pseudo-random pixels to
// tempDir and returns its path.
func (e *Engine) fGIF(wh ...int) (string, error) {
	w, h := dims(wh)
	path := e.tempFilePath(".gif")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fGIF: %w", err)
	}
	defer f.Close()

	rgba := e.fakeImage(w, h)
	palettedImg := image.NewPaletted(rgba.Bounds(), palette256())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			palettedImg.Set(x, y, rgba.At(x, y))
		}
	}

	if err := gif.Encode(f, palettedImg, nil); err != nil {
		return "", fmt.Errorf("tmpl: fGIF: %w", err)
	}
	return path, nil
}

func palette256() color.Palette {
	pal := make(color.Palette, 256)
	for i := range pal {
		v := byte(i)
		pal[i] = color.RGBA{R: v, G: v, B: v, A: 255}
	}
	return pal
}

// fCSV writes a CSV of rows x cols (default 10x5) of fake cell values to
// tempDir and returns its path.
func (e *Engine) fCSV(rowsCols ...int) (string, error) {
	rows, cols := 10, 5
	if len(rowsCols) > 0 && rowsCols[0] > 0 {
		rows = rowsCols[0]
	}
	if len(rowsCols) > 1 && rowsCols[1] > 0 {
		cols = rowsCols[1]
	}

	path := e.tempFilePath(".csv")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fCSV: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := make([]string, cols)
	for c := 0; c < cols; c++ {
		header[c] = fmt.Sprintf("col_%d", c+1)
	}
	if err := w.Write(header); err != nil {
		return "", fmt.Errorf("tmpl: fCSV: %w", err)
	}
	for r := 0; r < rows; r++ {
		row := make([]string, cols)
		for c := 0; c < cols; c++ {
			row[c] = e.randString(8)
		}
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("tmpl: fCSV: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("tmpl: fCSV: %w", err)
	}
	return path, nil
}

// fZIP bundles the given existing file paths (or, if none given, one
// small generated text file) into a .zip in tempDir and returns its path.
func (e *Engine) fZIP(files ...string) (string, error) {
	if len(files) == 0 {
		txtPath := e.tempFilePath(".txt")
		if err := os.WriteFile(txtPath, []byte(e.faker.Paragraph(2, 4, 8, " ")), 0o600); err != nil {
			return "", fmt.Errorf("tmpl: fZIP: %w", err)
		}
		files = []string{txtPath}
	}

	zipPath := e.tempFilePath(".zip")
	zf, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fZIP: %w", err)
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	for _, path := range files {
		if err := addFileToZip(zw, path); err != nil {
			zw.Close()
			return "", fmt.Errorf("tmpl: fZIP: %w", err)
		}
	}
	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("tmpl: fZIP: %w", err)
	}
	return zipPath, nil
}

func addFileToZip(zw *zip.Writer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	w, err := zw.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

var binSizeRe = regexp.MustCompile(`(?i)^\s*([0-9]+(?:\.[0-9]+)?)\s*(B|KB|MB|GB)?\s*$`)

// parseSize parses a size string like "2MB", "512KB", or a plain byte
// count.
func parseSize(size string) (int64, error) {
	m := binSizeRe.FindStringSubmatch(size)
	if m == nil {
		return 0, fmt.Errorf("invalid size %q", size)
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	mult := 1.0
	switch strings.ToUpper(m[2]) {
	case "", "B":
		mult = 1
	case "KB":
		mult = 1024
	case "MB":
		mult = 1024 * 1024
	case "GB":
		mult = 1024 * 1024 * 1024
	}
	return int64(val * mult), nil
}

// fBinary writes size (e.g. "2MB", "512KB", or a plain byte count) of
// pseudo-random bytes to tempDir and returns its path.
func (e *Engine) fBinary(size string) (string, error) {
	n, err := parseSize(size)
	if err != nil {
		return "", fmt.Errorf("tmpl: fBinary: %w", err)
	}
	path := e.tempFilePath(".bin")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("tmpl: fBinary: %w", err)
	}
	defer f.Close()

	const chunkSize = 64 * 1024
	buf := make([]byte, chunkSize)
	var written int64
	for written < n {
		toWrite := chunkSize
		if remaining := n - written; remaining < int64(chunkSize) {
			toWrite = int(remaining)
		}
		for i := 0; i < toWrite; i++ {
			buf[i] = byte(e.rnd.Intn(256))
		}
		nw, err := f.Write(buf[:toWrite])
		if err != nil {
			return "", fmt.Errorf("tmpl: fBinary: %w", err)
		}
		written += int64(nw)
	}
	return path, nil
}

// fileOf validates that path exists and returns it unchanged (identity
// passthrough, useful for chaining into attach()).
func (e *Engine) fileOf(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("tmpl: fileOf: %w", err)
	}
	return path, nil
}

// attach joins multiple file paths with attachDelim ("::") for later
// splitting by the shell's send builtin.
func (e *Engine) attach(paths ...string) (string, error) {
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("tmpl: attach: %w", err)
		}
	}
	return strings.Join(paths, attachDelim), nil
}
