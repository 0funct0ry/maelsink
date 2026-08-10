package shell

import (
	"context"
	"sync"
	"time"
)

// BackgroundJob tracks one builtin run that continues executing in its own
// goroutine after Run() has returned control to the interactive prompt
// (e.g. "intmsg --background"). A builtin's own Run method for its "stop"
// mode looks the job up via Session.Job, calls Cancel, then Wait()s for the
// run's own goroutine to observe the cancellation and report back — nothing
// prints asynchronously to the session's Out/Err while a job is running,
// since nothing in the shell's readline layer supports that safely; a
// backgrounded run stores its result for later retrieval instead.
type BackgroundJob struct {
	// ID is the session-local identifier a user passes back to a builtin's
	// stop flag (e.g. "intmsg --stop 3").
	ID string

	// Cancel stops the job's run loop. Set by the builtin that created the
	// job, once it has derived a cancellable context for the goroutine.
	Cancel context.CancelFunc

	// StartedAt is set once, at creation, so --list can report elapsed time
	// for a still-running job (a finished job's elapsed time is baked into
	// its stored summary instead).
	StartedAt time.Time

	doneCh  chan struct{}
	mu      sync.Mutex
	summary string
	done    bool
	sent    int
	failed  int
}

// UpdateProgress records the run's current sent/failed counts, for a
// "--list" builtin flag to report on a still-running job without needing to
// stop it first. Safe to call from the job's own goroutine while --list
// reads it concurrently from Snapshot.
func (j *BackgroundJob) UpdateProgress(sent, failed int) {
	j.mu.Lock()
	j.sent, j.failed = sent, failed
	j.mu.Unlock()
}

// Snapshot returns the job's current sent/failed counts and whether it has
// finished, for a "--list" builtin flag to render without disturbing a
// still-running job.
func (j *BackgroundJob) Snapshot() (sent, failed int, done bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.sent, j.failed, j.done
}

// Finish records the job's final summary and signals any waiter. Called
// exactly once, by the job's own goroutine, whether it stopped because of
// Cancel, or because it reached its own count/duration limit on its own.
func (j *BackgroundJob) Finish(summary string) {
	j.mu.Lock()
	j.summary = summary
	j.done = true
	j.mu.Unlock()
	close(j.doneCh)
}

// Wait blocks until the job finishes (returning its summary and true) or
// ctx is done first (returning false) — used by a builtin's "stop" mode
// after calling Cancel, to retrieve the final summary once the run loop
// actually observes the cancellation and exits.
func (j *BackgroundJob) Wait(ctx context.Context) (summary string, finished bool) {
	select {
	case <-j.doneCh:
		j.mu.Lock()
		defer j.mu.Unlock()
		return j.summary, true
	case <-ctx.Done():
		return "", false
	}
}
