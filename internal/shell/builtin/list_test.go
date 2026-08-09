package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestList_Basic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "hello" {
			t.Errorf("expected q=hello, got %q", r.URL.Query().Get("q"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{{"id": "msg_1", "from": "a@b.com", "subject": "hi"}},
			"total":    1, "limit": 50, "offset": 0,
		})
	})
	s, out, _ := newTestSession(t, mux)

	if err := (List{}).Run(context.Background(), s, []string{"hello"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "msg_1") {
		t.Errorf("output = %q", out.String())
	}
	if v, _ := s.GetVar("last_id"); v != "msg_1" {
		t.Errorf("last_id = %q", v)
	}
}

func TestList_LimitClamp(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	err := (List{}).Run(context.Background(), s, []string{"--limit", "501"})
	if err == nil {
		t.Fatal("expected error for --limit over max")
	}
}

func TestList_IDsOnly(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"messages": []map[string]any{{"id": "msg_1"}, {"id": "msg_2"}},
			"total":    2,
		})
	})
	s, out, _ := newTestSession(t, mux)
	if err := (List{}).Run(context.Background(), s, []string{"--ids"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.String() != "msg_1\nmsg_2\n" {
		t.Errorf("output = %q", out.String())
	}
}
