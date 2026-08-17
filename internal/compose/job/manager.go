package job

import (
	"context"
	"sync"
	"time"
)

// RunFunc is a job kind's actual work: it runs until ctx is cancelled or it
// completes on its own (count/duration reached, or a synchronous kind like
// randmsg/weirdmsg simply returning), calling progress after every send
// attempt with the running sent/failed counts.
type RunFunc func(ctx context.Context, progress func(sent, failed int)) error

// Manager owns every Job for the life of the compose process (SPEC.md
// §7.7.7: no persistence, lost on restart).
type Manager struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

// NewManager constructs an empty Manager.
func NewManager() *Manager {
	return &Manager{jobs: make(map[string]*Job)}
}

// Start registers a new Job for kind and launches run in its own goroutine,
// returning immediately with the Job (already visible to Get/List/Cancel).
func (m *Manager) Start(kind string, run RunFunc) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{
		ID:        newJobID(),
		Kind:      kind,
		StartedAt: time.Now(),
		status:    StatusRunning,
		cancel:    cancel,
	}

	m.mu.Lock()
	m.jobs[j.ID] = j
	m.mu.Unlock()

	go func() {
		err := run(ctx, j.updateProgress)
		cancelled := ctx.Err() != nil
		j.finish(cancelled, err)
	}()

	return j
}

// Get returns the job registered under id, if any.
func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	return j, ok
}

// Cancel stops the job registered under id, if it is still running. Returns
// false if no such job is registered; a no-op (not an error) if the job has
// already reached a terminal state.
func (m *Manager) Cancel(id string) bool {
	j, ok := m.Get(id)
	if !ok {
		return false
	}
	j.cancel()
	return true
}

// List returns every registered job (running or finished), for the Jobs
// Panel's recent-jobs list to refresh from on load/reconnect.
func (m *Manager) List() []*Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out
}
