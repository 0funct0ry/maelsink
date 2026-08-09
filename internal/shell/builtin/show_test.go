package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestShow_Basic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/msg_1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "from": "a@b.com", "subject": "hi", "text_body": "hello world",
		})
	})
	s, out, _ := newTestSession(t, mux)

	if err := (Show{}).Run(context.Background(), s, []string{"msg_1", "--part", "text"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "hello world") {
		t.Errorf("output = %q", out.String())
	}
}

func TestShow_AmbiguousID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/msg", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "ambiguous_id", "message": "matches more than one"}})
	})
	s, _, errBuf := newTestSession(t, mux)

	err := (Show{}).Run(context.Background(), s, []string{"msg"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(errBuf.String(), "supply more characters") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}
