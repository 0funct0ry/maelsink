package builtin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// newTestSession builds a Session pointed at a fake API server (mux is nil
// for local-only builtins that never touch the network).
func newTestSession(t *testing.T, mux *http.ServeMux) (*shell.Session, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var client *cliclient.Client
	if mux != nil {
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		client = cliclient.NewClient(srv.URL, "")
	}
	engine, err := tmpl.New(42, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	t.Cleanup(func() { engine.Close() })

	var out, errBuf bytes.Buffer
	s := shell.NewSession(config.Shell{TemplateEnabled: true, ShEnabled: true}, client, "", nil, engine, &out, &errBuf, nil)
	s.Interactive = false
	return s, &out, &errBuf
}
