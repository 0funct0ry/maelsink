package compose

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/compose/job"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func newTestClient(t *testing.T, target *httptest.Server) *cliclient.Client {
	t.Helper()
	client, err := NewTargetClient(TargetConfig{APIAddr: target.URL})
	if err != nil {
		t.Fatalf("NewTargetClient: %v", err)
	}
	return client
}

func doRequest(t *testing.T, engine http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestListMessagesHandler(t *testing.T) {
	cases := []struct {
		name       string
		targetFunc http.HandlerFunc
		closeEarly bool
		wantStatus int
		wantCode   string
	}{
		{
			name: "success",
			targetFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(cliclient.ListResponse{
					Messages: []cliclient.MessageSummary{{ID: "abc", Subject: "hi"}},
					Total:    1,
				})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "target 500",
			targetFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"boom"}}`))
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := httptest.NewServer(tc.targetFunc)
			defer target.Close()

			client := newTestClient(t, target)
			engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

			rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/messages")
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantCode != "" {
				var body map[string]any
				_ = json.Unmarshal(rec.Body.Bytes(), &body)
				errObj, _ := body["error"].(map[string]any)
				if errObj["code"] != tc.wantCode {
					t.Fatalf("error.code = %v, want %v", errObj["code"], tc.wantCode)
				}
			}
		})
	}

	t.Run("transport error (target unreachable)", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		target.Close() // closed before use: connection refused

		client := newTestClient(t, target)
		engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

		rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/messages")
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		errObj, _ := body["error"].(map[string]any)
		if errObj["code"] != "target_unreachable" {
			t.Fatalf("error.code = %v, want target_unreachable", errObj["code"])
		}
	})
}

func TestGetMessageHandler(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/messages/notfound" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"no such message"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cliclient.MessageDetail{MessageSummary: cliclient.MessageSummary{ID: "abc"}})
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

	rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/messages/abc")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	rec = doRequest(t, engine, http.MethodGet, "/compose-api/v1/messages/notfound")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDeleteMessageHandler(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/messages/notfound" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"no such message"}}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

	rec := doRequest(t, engine, http.MethodDelete, "/compose-api/v1/messages/abc")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}

	rec = doRequest(t, engine, http.MethodDelete, "/compose-api/v1/messages/notfound")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestClearMessagesHandler(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

	rec := doRequest(t, engine, http.MethodDelete, "/compose-api/v1/messages")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}

	target.Close()
	rec = doRequest(t, engine, http.MethodDelete, "/compose-api/v1/messages")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 on transport error", rec.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	t.Run("target reachable and healthy", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		defer target.Close()

		client := newTestClient(t, target)
		engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

		rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/health")
		if rec.Code != http.StatusOK {
			t.Fatalf("compose's own health endpoint status = %d, want 200", rec.Code)
		}
		var body healthResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if !body.TargetReachable || body.Status != "green" {
			t.Fatalf("body = %+v, want reachable/green", body)
		}
	})

	t.Run("target reachable but erroring", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer target.Close()

		client := newTestClient(t, target)
		engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

		rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/health")
		if rec.Code != http.StatusOK {
			t.Fatalf("compose's own health endpoint status = %d, want 200", rec.Code)
		}
		var body healthResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if !body.TargetReachable || body.Status != "yellow" {
			t.Fatalf("body = %+v, want reachable/yellow", body)
		}
	})

	t.Run("target unreachable", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		target.Close()

		client := newTestClient(t, target)
		engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

		rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/health")
		if rec.Code != http.StatusOK {
			t.Fatalf("compose's own health endpoint status = %d, want 200 even when target is down", rec.Code)
		}
		var body healthResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.TargetReachable || body.Status != "red" {
			t.Fatalf("body = %+v, want unreachable/red", body)
		}
	})
}

// TestStartsEvenWhenTargetDown covers the DoD requirement that compose must
// start successfully even when the target address is unreachable — no
// upfront health check gates New/ListenAndServe.
func TestStartsEvenWhenTargetDown(t *testing.T) {
	client, err := NewTargetClient(TargetConfig{APIAddr: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("NewTargetClient: %v", err)
	}
	engine := New(client, testLogger(), TargetConfig{}, job.NewManager(), Config{})

	srv := httptest.NewServer(engine)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/compose-api/v1/health")
	if err != nil {
		t.Fatalf("GET /compose-api/v1/health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
