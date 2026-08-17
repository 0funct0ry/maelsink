// Package job implements compose's in-memory job table (SPEC.md §7.7.4.3,
// M13.3): tracks one Job per running/finished randmsg|intmsg|weirdmsg|
// blast|deluge invocation started from the Jobs Panel. Jobs are lost on
// compose process restart by design (SPEC.md §7.7.7) — this package holds
// no persistence.
//
// The concurrency shape mirrors internal/shell.BackgroundJob
// (internal/shell/jobs.go): a mutex-guarded struct updated from the job's
// own goroutine and read concurrently by HTTP/WS handlers.
package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Status is a Job's lifecycle state.
type Status string

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
	StatusFailed    Status = "failed"
)

// Job tracks one job-kind invocation started from the Jobs Panel.
type Job struct {
	ID        string
	Kind      string
	StartedAt time.Time

	cancel context.CancelFunc

	mu         sync.Mutex
	status     Status
	sent       int
	failed     int
	err        error
	finishedAt time.Time
}

// Snapshot is a point-in-time, concurrency-safe view of a Job, used by both
// the HTTP list/get handlers and the WS progress stream.
type Snapshot struct {
	ID        string    `json:"jobId"`
	Kind      string    `json:"kind"`
	Status    Status    `json:"status"`
	Sent      int       `json:"sent"`
	Failed    int       `json:"failed"`
	StartedAt time.Time `json:"startedAt"`
	Elapsed   float64   `json:"elapsedSeconds"`
	Error     string    `json:"error,omitempty"`
}

// Snapshot returns a concurrency-safe copy of the job's current state.
func (j *Job) Snapshot() Snapshot {
	j.mu.Lock()
	defer j.mu.Unlock()

	end := time.Now()
	if !j.finishedAt.IsZero() {
		end = j.finishedAt
	}
	s := Snapshot{
		ID:        j.ID,
		Kind:      j.Kind,
		Status:    j.status,
		Sent:      j.sent,
		Failed:    j.failed,
		StartedAt: j.StartedAt,
		Elapsed:   end.Sub(j.StartedAt).Seconds(),
	}
	if j.err != nil {
		s.Error = j.err.Error()
	}
	return s
}

// Done reports whether the job has reached a terminal state.
func (j *Job) Done() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.status != StatusRunning
}

// updateProgress records the job's current sent/failed counts. Called from
// the job's own run goroutine.
func (j *Job) updateProgress(sent, failed int) {
	j.mu.Lock()
	j.sent, j.failed = sent, failed
	j.mu.Unlock()
}

// finish transitions the job to a terminal status exactly once. cancelled
// takes priority over err (a job stopped via Cancel is "cancelled" even if
// its run func also returned a context.Canceled error).
func (j *Job) finish(cancelled bool, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.status != StatusRunning {
		return
	}
	j.finishedAt = time.Now()
	switch {
	case cancelled:
		j.status = StatusCancelled
	case err != nil:
		j.status = StatusFailed
		j.err = err
	default:
		j.status = StatusCompleted
	}
}

func newJobID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
