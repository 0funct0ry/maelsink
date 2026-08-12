package shell

import (
	"context"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// Session holds all per-session mutable shell state: connection info,
// variables/aliases/abbreviations, the template engine, and I/O streams.
// It is shared across every command evaluated in the session.
type Session struct {
	Client   *cliclient.Client
	SMTPAddr string
	SMTPAuth *cliclient.Auth

	Vars    map[string]string
	Aliases map[string]string
	Abbrs   map[string]string

	CommandPrefix string
	Cfg           config.Shell

	Tmpl *tmpl.Engine

	Out io.Writer
	Err io.Writer
	In  io.Reader

	Interactive bool
	ExitOnError bool
	LastStatus  int

	// Registry is the builtin command table this session dispatches
	// against. It is set post-construction by internal/shell.Run (and by
	// cmd/shell.go), since the Registry itself is built from builtins that
	// may need a *Session to construct — breaking that chicken-and-egg
	// cycle. Builtins that need to recursively evaluate lines (e.g.
	// "source", "set --from-command") read it directly. May be nil.
	Registry *Registry

	// History is the session's command history, set post-construction the
	// same way as Registry. Builtins that inspect history (e.g. the
	// "history" builtin) read it directly. May be nil.
	History *History

	// PendingBuffer, when non-empty, is text a builtin wants LOADED into
	// the next interactive prompt's line buffer rather than executed
	// immediately (SPEC.md §7.5.9: the edit builtin and Ctrl-X Ctrl-E
	// share one contract — replace the buffer, don't submit it). The
	// interactive REPL loop in shell.go consumes and clears this after
	// every evaluated line; it has no effect in -e/--script/piped modes,
	// which have no line buffer to seed.
	PendingBuffer string

	// jobsMu guards jobs/nextJobID — the only Session state touched by a
	// goroutine other than the single-threaded eval loop (background
	// builtin runs, e.g. "intmsg --background"). Every other Session field
	// is written unsynchronized on the assumption only the eval loop
	// touches it; this is the sole exception.
	jobsMu    sync.Mutex
	jobs      map[string]*BackgroundJob
	jobOrder  []string
	nextJobID int
}

// SetPendingBuffer stages text to be loaded into the next interactive
// prompt's line buffer (see the PendingBuffer field doc).
func (s *Session) SetPendingBuffer(text string) {
	s.PendingBuffer = text
}

// SetRegistry assigns the session's builtin registry, used by builtins that
// need to dispatch other commands (e.g. "source", "set --from-command").
func (s *Session) SetRegistry(reg *Registry) {
	s.Registry = reg
}

// SetHistory assigns the session's history, used by builtins that inspect
// or manage history (e.g. the "history" builtin).
func (s *Session) SetHistory(h *History) {
	s.History = h
}

// NewSession constructs a Session ready for use. Vars/Aliases/Abbrs start
// empty (non-nil).
func NewSession(cfg config.Shell, client *cliclient.Client, smtpAddr string, smtpAuth *cliclient.Auth, tmplEngine *tmpl.Engine, out, errW io.Writer, in io.Reader) *Session {
	s := &Session{
		Client:        client,
		SMTPAddr:      smtpAddr,
		SMTPAuth:      smtpAuth,
		Vars:          make(map[string]string),
		Aliases:       make(map[string]string),
		Abbrs:         make(map[string]string),
		CommandPrefix: cfg.CommandPrefix,
		Cfg:           cfg,
		Tmpl:          tmplEngine,
		Out:           out,
		Err:           errW,
		In:            in,
	}
	// $connected must always be a real string, never an absent map key:
	// text/template returns an invalid reflect.Value for a missing map
	// key, which prints fine bare ({{ .connected }}) but errors ("invalid
	// value; expected string") the moment it's passed into any function
	// expecting a string — e.g. a user prompt like {{ cyan }}{{ .connected
	// }}. It's "" (empty) when false and "true" when true — NOT the
	// literal string "false" — because Go template truthiness treats any
	// NON-EMPTY string as true, including the string "false" itself; the
	// default prompt's {{ if not .connected }} (SPEC.md §7.5.10) only
	// works correctly against this empty/"true" convention.
	// RefreshConnected updates this to the real state once the session can
	// reach (or fails to reach) the API.
	s.SetVar("connected", "")
	return s
}

// Close releases session resources (the template engine's temp dir).
func (s *Session) Close() error {
	if s.Tmpl != nil {
		return s.Tmpl.Close()
	}
	return nil
}

// SetVar sets a session variable.
func (s *Session) SetVar(k, v string) {
	if s.Vars == nil {
		s.Vars = make(map[string]string)
	}
	s.Vars[k] = v
}

// GetVar returns a session variable's value and whether it was set.
func (s *Session) GetVar(k string) (string, bool) {
	v, ok := s.Vars[k]
	return v, ok
}

// TemplateData merges user-set Vars with reserved vars (connected, last_id,
// status) into a single map suitable as template render data.
func (s *Session) TemplateData() map[string]string {
	data := make(map[string]string, len(s.Vars))
	for k, v := range s.Vars {
		data[k] = v
	}
	return data
}

// RefreshConnected probes the API server's health endpoint with a short
// timeout and sets Vars["connected"] to "true" (reachable) or "" (not —
// see NewSession's doc comment for why empty, not the string "false", is
// the false value). It never blocks long and never panics or propagates
// an error.
func (s *Session) RefreshConnected(ctx context.Context) {
	if s.Client == nil {
		s.SetVar("connected", "")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, err := s.Client.Health(ctx)
	if err != nil {
		s.SetVar("connected", "")
		return
	}
	s.SetVar("connected", "true")
}

// NewJob registers a new BackgroundJob under a fresh session-local ID (a
// small incrementing counter, e.g. "1", "2" — easy to type back into a
// "--stop <id>" flag) and returns it. Used by builtins that support running
// detached from the interactive prompt (e.g. "intmsg --background").
func (s *Session) NewJob() *BackgroundJob {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	if s.jobs == nil {
		s.jobs = make(map[string]*BackgroundJob)
	}
	s.nextJobID++
	job := &BackgroundJob{ID: strconv.Itoa(s.nextJobID), StartedAt: time.Now(), doneCh: make(chan struct{})}
	s.jobs[job.ID] = job
	s.jobOrder = append(s.jobOrder, job.ID)
	return job
}

// Job returns the background job registered under id, if any.
func (s *Session) Job(id string) (*BackgroundJob, bool) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	job, ok := s.jobs[id]
	return job, ok
}

// RemoveJob unregisters a background job (whether finished or still
// running) once its owner no longer needs to query it.
func (s *Session) RemoveJob(id string) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	delete(s.jobs, id)
	for i, existing := range s.jobOrder {
		if existing == id {
			s.jobOrder = append(s.jobOrder[:i], s.jobOrder[i+1:]...)
			break
		}
	}
}

// Jobs returns every currently-registered background job, oldest first —
// used by a "--list" builtin flag to enumerate running/finished background
// runs (e.g. "intmsg --list").
func (s *Session) Jobs() []*BackgroundJob {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	out := make([]*BackgroundJob, 0, len(s.jobOrder))
	for _, id := range s.jobOrder {
		if job, ok := s.jobs[id]; ok {
			out = append(out, job)
		}
	}
	return out
}
