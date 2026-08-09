package builtin

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestExport_Single(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/msg_1/export", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Subject: hi\r\n\r\nbody\r\n"))
	})
	s, out, _ := newTestSession(t, mux)

	dir := t.TempDir()
	path := filepath.Join(dir, "msg_1.eml")
	if err := (Export{}).Run(context.Background(), s, []string{"msg_1", "-o", path}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "Subject: hi\r\n\r\nbody\r\n" {
		t.Errorf("data = %q", data)
	}
	_ = out
}

func TestAttachment_List(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/msg_1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"msg_1","attachments":[{"id":"att_1","filename":"logo.png","content_type":"image/png","size_bytes":10}]}`))
	})
	s, out, _ := newTestSession(t, mux)

	if err := (Attachment{}).Run(context.Background(), s, []string{"msg_1"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !contains(out.String(), "logo.png") {
		t.Errorf("output = %q", out.String())
	}
}

func TestAttachment_DownloadByIndex(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/msg_1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"msg_1","attachments":[{"id":"att_1","filename":"logo.png"}]}`))
	})
	mux.HandleFunc("/api/v1/messages/msg_1/attachments/att_1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="logo.png"`)
		w.Write([]byte("pngdata"))
	})
	s, out, _ := newTestSession(t, mux)

	dir := t.TempDir()
	outPath := filepath.Join(dir, "logo.png")
	if err := (Attachment{}).Run(context.Background(), s, []string{"msg_1", "1", "-o", outPath}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "pngdata" {
		t.Errorf("data = %q", data)
	}
	_ = out
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
