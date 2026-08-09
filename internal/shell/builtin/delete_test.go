package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDelete_Specific(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/msg_1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	s, out, _ := newTestSession(t, mux)

	if err := (Delete{}).Run(context.Background(), s, []string{"msg_1"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "deleted msg_1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestDelete_AllNonInteractiveRequiresYes(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.Interactive = false
	err := (Delete{}).Run(context.Background(), s, []string{"--all"})
	if err == nil {
		t.Fatal("expected hard error without --yes in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("err = %v", err)
	}
}

func TestDelete_AllWithYes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("confirm") != "true" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "confirmation_required"}})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	s, out, _ := newTestSession(t, mux)
	s.Interactive = false

	if err := (Delete{}).Run(context.Background(), s, []string{"--all", "--yes"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "all messages deleted") {
		t.Errorf("output = %q", out.String())
	}
}

func TestClear_Interactive_Confirms(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"total_messages": 5})
	})
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	s, out, _ := newTestSession(t, mux)
	s.Interactive = true
	s.In = strings.NewReader("y\n")

	if err := (Clear{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "This will delete 5 messages") {
		t.Errorf("output = %q", out.String())
	}
}
