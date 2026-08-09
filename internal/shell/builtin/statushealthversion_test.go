package builtin

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestStats_Basic(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"total_messages":3,"total_size_bytes":1024}`))
	})
	s, out, _ := newTestSession(t, mux)
	if err := (Stats{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "3") {
		t.Errorf("output = %q", out.String())
	}
}

func TestHealth_DegradedReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"degraded","db":"error","smtp":"listening"}`))
	})
	s, out, _ := newTestSession(t, mux)
	err := (Health{}).Run(context.Background(), s, nil)
	if err == nil {
		t.Fatal("expected non-nil error for degraded status")
	}
	if !strings.Contains(out.String(), "degraded") {
		t.Errorf("output = %q", out.String())
	}
}

func TestHealth_OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","db":"ok","smtp":"listening"}`))
	})
	s, _, _ := newTestSession(t, mux)
	if err := (Health{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestVersion_Local(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Version{}).Run(context.Background(), s, []string{"--local"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "Version:") {
		t.Errorf("output = %q", out.String())
	}
}

func TestVersion_Remote(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"version":"1.2.3","commit":"abc","go":"go1.26"}`))
	})
	s, out, _ := newTestSession(t, mux)
	if err := (Version{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "1.2.3") {
		t.Errorf("output = %q", out.String())
	}
}
