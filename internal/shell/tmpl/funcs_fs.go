package tmpl

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"text/template"
)

// fsFuncMap returns straightforward filesystem helper template functions.
func (e *Engine) fsFuncMap() template.FuncMap {
	return template.FuncMap{
		"readFile":    readFile,
		"readFileB64": readFileB64,
		"glob":        glob,
		"basename":    filepath.Base,
		"dirname":     filepath.Dir,
		"ext":         filepath.Ext,
	}
}

// readFile returns the contents of path as a string.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// readFileB64 returns the base64-encoded contents of path.
func readFileB64(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// glob returns filesystem paths matching pattern.
func glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
