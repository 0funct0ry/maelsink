package compose

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/0funct0ry/maelsink/internal/compose/job"
)

// jobFakeSMTPServer accepts SMTP transactions in a loop — a job's send loop
// dials a fresh connection per message, unlike send_test.go's
// single-transaction fakeSMTPServer.
func jobFakeSMTPServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				tp := textproto.NewConn(conn)
				tp.PrintfLine("220 fake.local ESMTP")
				for {
					line, err := tp.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						tp.PrintfLine("250 fake.local")
					case strings.HasPrefix(line, "MAIL FROM"):
						tp.PrintfLine("250 OK")
					case strings.HasPrefix(line, "RCPT TO"):
						tp.PrintfLine("250 OK")
					case line == "DATA":
						tp.PrintfLine("354 go ahead")
						_, _ = io.ReadAll(tp.DotReader())
						tp.PrintfLine("250 OK")
					case line == "QUIT":
						tp.PrintfLine("221 bye")
						return
					}
				}
			}()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

func newJobsTestServer(t *testing.T, smtpAddr string) *httptest.Server {
	t.Helper()
	client, err := NewTargetClient(TargetConfig{})
	if err != nil {
		t.Fatalf("NewTargetClient: %v", err)
	}
	engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, job.NewManager(), Config{})
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return srv
}

func TestStartJobUnknownKind(t *testing.T) {
	srv := newJobsTestServer(t, "")
	resp, err := http.Post(srv.URL+"/compose-api/v1/jobs/nonsense", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestStartRandMsgJobCompletesAndAppearsInList(t *testing.T) {
	smtpAddr := jobFakeSMTPServer(t)
	srv := newJobsTestServer(t, smtpAddr)

	body := bytes.NewBufferString(`{"count":2,"concurrency":1}`)
	resp, err := http.Post(srv.URL+"/compose-api/v1/jobs/randmsg", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var started struct {
		JobID string `json:"jobId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	if started.JobID == "" {
		t.Fatal("expected a non-empty jobId")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		listResp, err := http.Get(srv.URL + "/compose-api/v1/jobs")
		if err != nil {
			t.Fatal(err)
		}
		var list struct {
			Jobs []job.Snapshot `json:"jobs"`
		}
		if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
			t.Fatal(err)
		}
		listResp.Body.Close()
		if len(list.Jobs) == 1 && list.Jobs[0].Status == job.StatusCompleted {
			if list.Jobs[0].Sent != 2 {
				t.Fatalf("sent = %d, want 2", list.Jobs[0].Sent)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("randmsg job did not complete within 2s")
}

func TestJobStreamAndCancel(t *testing.T) {
	smtpAddr := jobFakeSMTPServer(t)
	srv := newJobsTestServer(t, smtpAddr)

	body := bytes.NewBufferString(`{"intervalMs":20}`)
	resp, err := http.Post(srv.URL+"/compose-api/v1/jobs/intmsg", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	var started struct {
		JobID string `json:"jobId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/compose-api/v1/jobs/" + started.JobID + "/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	var firstTick job.Snapshot
	if err := conn.ReadJSON(&firstTick); err != nil {
		t.Fatalf("reading first tick: %v", err)
	}
	if firstTick.Status != job.StatusRunning {
		t.Fatalf("first tick status = %v, want running", firstTick.Status)
	}

	cancelResp, err := http.Post(srv.URL+"/compose-api/v1/jobs/"+started.JobID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200", cancelResp.StatusCode)
	}

	// Drain ticks until the stream reports a terminal state and closes.
	var last job.Snapshot
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var tick job.Snapshot
		if err := conn.ReadJSON(&tick); err != nil {
			break
		}
		last = tick
	}
	if last.Status != job.StatusCancelled {
		t.Fatalf("final tick status = %v, want cancelled", last.Status)
	}
}

func TestCancelUnknownJob(t *testing.T) {
	srv := newJobsTestServer(t, "")
	resp, err := http.Post(srv.URL+"/compose-api/v1/jobs/nonexistent/cancel", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestJobStreamUnknownJob(t *testing.T) {
	srv := newJobsTestServer(t, "")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/compose-api/v1/jobs/nonexistent/stream"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail for an unknown job id")
	}
	if resp != nil && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
