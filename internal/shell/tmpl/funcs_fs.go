package tmpl

import (
	"os"
	"path/filepath"
)

// filesDocs documents straightforward filesystem helper template functions.
func (e *Engine) filesDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "readFile", Category: CategoryFiles, Args: "path", Returns: "string",
			Description: "Returns the file's contents as a string.", Fn: readFile},
		{Name: "glob", Category: CategoryFiles, Args: "pattern", Returns: "[]string",
			Description: "Returns matching file paths.", Fn: glob},
		{Name: "basename", Category: CategoryFiles, Args: "path", Returns: "string",
			Description: "Returns the final path element.", Fn: filepath.Base},
		{Name: "dirname", Category: CategoryFiles, Args: "path", Returns: "string",
			Description: "Returns all but the final path element.", Fn: filepath.Dir},
		{Name: "ext", Category: CategoryFiles, Args: "path", Returns: "string",
			Description: "Returns the file extension, including the leading dot.", Fn: filepath.Ext},
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

// glob returns filesystem paths matching pattern.
func glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
