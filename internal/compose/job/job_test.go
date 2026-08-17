package job

import (
	"context"
	"io"
	"net"
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSMTPServer accepts SMTP transactions in a loop (unlike a
// single-shot test helper) since a job's send loop dials a fresh
// connection per message.
func fakeSMTPServer(t *testing.T) (addr string, count func() int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var n int64
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
						atomic.AddInt64(&n, 1)
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
	return ln.Addr().String(), func() int { return int(atomic.LoadInt64(&n)) }
}

func TestManagerStartAndSnapshot(t *testing.T) {
	mgr := NewManager()
	j := mgr.Start("randmsg", func(ctx context.Context, progress func(sent, failed int)) error {
		progress(1, 0)
		return nil
	})

	deadline := time.After(2 * time.Second)
	for {
		if j.Done() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("job did not finish")
		case <-time.After(5 * time.Millisecond):
		}
	}

	snap := j.Snapshot()
	if snap.Status != StatusCompleted {
		t.Fatalf("status = %v, want completed", snap.Status)
	}
	if snap.Sent != 1 {
		t.Fatalf("sent = %d, want 1", snap.Sent)
	}

	got, ok := mgr.Get(j.ID)
	if !ok || got != j {
		t.Fatalf("Get(%q) = %v, %v", j.ID, got, ok)
	}
}

func TestManagerCancelStopsJob(t *testing.T) {
	mgr := NewManager()
	started := make(chan struct{})
	j := mgr.Start("intmsg", func(ctx context.Context, progress func(sent, failed int)) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})

	<-started
	if !mgr.Cancel(j.ID) {
		t.Fatal("Cancel returned false for a running job")
	}

	deadline := time.After(2 * time.Second)
	for {
		if j.Done() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("job did not stop after Cancel")
		case <-time.After(5 * time.Millisecond):
		}
	}

	if snap := j.Snapshot(); snap.Status != StatusCancelled {
		t.Fatalf("status = %v, want cancelled", snap.Status)
	}
}

func TestManagerCancelUnknownJob(t *testing.T) {
	mgr := NewManager()
	if mgr.Cancel("nonexistent") {
		t.Fatal("Cancel returned true for an unregistered job id")
	}
}

func TestManagerConcurrentJobs(t *testing.T) {
	mgr := NewManager()
	const n = 20
	jobs := make([]*Job, n)
	for i := 0; i < n; i++ {
		jobs[i] = mgr.Start("randmsg", func(ctx context.Context, progress func(sent, failed int)) error {
			progress(1, 0)
			return nil
		})
	}

	deadline := time.After(3 * time.Second)
	for _, j := range jobs {
		for !j.Done() {
			select {
			case <-deadline:
				t.Fatal("a job did not finish")
			case <-time.After(5 * time.Millisecond):
			}
		}
	}

	if got := len(mgr.List()); got != n {
		t.Fatalf("List() returned %d jobs, want %d", got, n)
	}
}

func TestManagerFailedRunMarksJobFailed(t *testing.T) {
	mgr := NewManager()
	j := mgr.Start("weirdmsg", func(ctx context.Context, progress func(sent, failed int)) error {
		return context.DeadlineExceeded
	})

	deadline := time.After(2 * time.Second)
	for !j.Done() {
		select {
		case <-deadline:
			t.Fatal("job did not finish")
		case <-time.After(5 * time.Millisecond):
		}
	}

	if snap := j.Snapshot(); snap.Status != StatusFailed || snap.Error == "" {
		t.Fatalf("snapshot = %+v, want status=failed with a non-empty error", snap)
	}
}
