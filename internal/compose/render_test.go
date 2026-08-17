package compose

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/compose/job"
)

func postRequest(t *testing.T, engine http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestRenderHandler(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

	t.Run("eml with var substitution", func(t *testing.T) {
		rec := postRequest(t, engine, "/compose-api/v1/render", renderRequest{
			Format:   "eml",
			Template: "From: {{ .from }}\r\nTo: to@example.com\r\nSubject: hi\r\n\r\nbody",
			Vars:     map[string]string{"from": "sender@example.com"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		rendered, _ := body["rendered"].(string)
		if !strings.Contains(rendered, "sender@example.com") {
			t.Fatalf("rendered = %q, want it to contain the substituted var", rendered)
		}
	})

	t.Run("json spec with var substitution", func(t *testing.T) {
		rec := postRequest(t, engine, "/compose-api/v1/render", renderRequest{
			Format:   "json",
			Template: `{"from":"a@example.com","to":["b@example.com"],"subject":"{{ .subj }}","text":"hello"}`,
			Vars:     map[string]string{"subj": "Rendered Subject"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		rendered, _ := body["rendered"].(string)
		if !strings.Contains(rendered, "Rendered Subject") {
			t.Fatalf("rendered = %q, want it to contain the substituted subject", rendered)
		}
	})

	t.Run("malformed template surfaces position info", func(t *testing.T) {
		rec := postRequest(t, engine, "/compose-api/v1/render", renderRequest{
			Format:   "eml",
			Template: "From: a@example.com\r\nTo: b@example.com\r\nSubject: hi\r\n\r\n{{ .foo",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		errObj, _ := body["error"].(map[string]any)
		if errObj["code"] != "render_failed" {
			t.Fatalf("error.code = %v, want render_failed", errObj["code"])
		}
		if _, ok := errObj["line"]; !ok {
			t.Fatalf("error = %+v, want a line position, not a bare render-failed message", errObj)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		rec := postRequest(t, engine, "/compose-api/v1/render", renderRequest{Format: "yaml", Template: "x"})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("json attachment path is templated by a generating function", func(t *testing.T) {
		rec := postRequest(t, engine, "/compose-api/v1/render", renderRequest{
			Format:   "json",
			Template: `{"from":"a@example.com","to":["b@example.com"],"subject":"report","html":"<p>see attached</p>","attachments":[{"path":"{{ fCSV }}","filename":"report.csv"}]}`,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		rendered, _ := body["rendered"].(string)
		if strings.Contains(rendered, "{{") {
			t.Fatalf("rendered = %q, want the attachment path templated, not left literal", rendered)
		}
		if !strings.Contains(rendered, ".csv") {
			t.Fatalf("rendered = %q, want the generated attachment path to end in .csv", rendered)
		}
	})

	t.Run("eml attachments are resolved and returned as metadata, body left untouched", func(t *testing.T) {
		rec := postRequest(t, engine, "/compose-api/v1/render", renderRequest{
			Format:   "eml",
			Template: "From: a@example.com\r\nTo: b@example.com\r\nSubject: report\r\n\r\nsee attached",
			Attachments: []cliclient.AttachmentSpec{
				{Path: "{{ fCSV }}", Filename: "report.csv"},
			},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		rendered, _ := body["rendered"].(string)
		if strings.Contains(rendered, "multipart") {
			t.Fatalf("rendered = %q, want the plain single-part body (preview stays readable), not a multipart envelope", rendered)
		}
		attachments, _ := body["attachments"].([]any)
		if len(attachments) != 1 {
			t.Fatalf("attachments = %+v, want exactly 1 resolved attachment", body["attachments"])
		}
		att, _ := attachments[0].(map[string]any)
		path, _ := att["path"].(string)
		if strings.Contains(path, "{{") || !strings.HasSuffix(path, ".csv") {
			t.Fatalf("attachment path = %q, want it templated to a real .csv path", path)
		}
	})
}

func TestFunctionsHandler(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

	rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/functions")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	fns, _ := body["functions"].([]any)
	if len(fns) == 0 {
		t.Fatalf("functions list is empty, want the registered template functions")
	}
	first, _ := fns[0].(map[string]any)
	if _, hasFn := first["Fn"]; hasFn {
		t.Fatalf("functions[0] = %+v, want the Fn field dropped", first)
	}
	if first["name"] == "" || first["name"] == nil {
		t.Fatalf("functions[0] = %+v, want a non-empty name", first)
	}
}
