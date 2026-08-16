package builtin

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// IntMsg implements the "intmsg" builtin (SPEC.md §7.6.3): sends
// randmsg-style messages at randomized real-time intervals to simulate live
// traffic for load-testing the SMTP ingest path and realtime fan-out.
//
// --background detaches the run into its own goroutine and returns control
// to the prompt immediately, printing a session-local job ID; --stop <id>
// cancels a (possibly already-finished) backgrounded run and prints its
// final summary. A backgrounded run never writes to s.Out/s.Err while
// active — nothing in the shell's readline layer supports printing safely
// above an active prompt — it only accumulates its summary for retrieval
// via --stop.
type IntMsg struct{}

func (IntMsg) Name() string      { return "intmsg" }
func (IntMsg) Aliases() []string { return nil }
func (IntMsg) Short() string     { return "Send random messages at randomized intervals" }

func (IntMsg) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("intmsg", pflag.ContinueOnError)
	addRandContentFlags(fs)
	fs.DurationP("interval", "i", time.Second, "mean inter-message gap (or the quiet-period gap for --profile bursty)")
	fs.Float64P("rate", "r", 0, "mean messages per second (alternative to --interval)")
	fs.StringP("jitter", "j", "0", "jitter around --interval: a duration (e.g. 200ms) or a percentage (e.g. 20%)")
	fs.StringP("profile", "p", "steady", "interval distribution: steady|poisson|bursty")
	fs.IntP("burst-size", "k", 5, "messages per burst, for --profile bursty")
	fs.DurationP("burst-interval", "b", 100*time.Millisecond, "spacing between messages within a burst, for --profile bursty")
	fs.IntP("count", "n", 0, "stop after this many messages (0: unbounded)")
	fs.DurationP("duration", "d", 0, "stop after this long (0: unbounded)")
	fs.DurationP("stats-interval", "I", 5*time.Second, "how often to print a running summary line")
	fs.BoolP("quiet", "q", false, "suppress per-message send confirmations")
	fs.BoolP("until-error", "e", false, "stop on the first SMTP failure instead of logging and continuing")
	fs.BoolP("background", "g", false, "run detached from the prompt; prints a job id, usable with --stop")
	fs.StringP("stop", "X", "", "stop a --background run by job id (from its startup message) and print its summary")
	fs.BoolP("list", "l", false, "list background intmsg runs and their live status")
	fs.StringP("smtp-host", "H", "", "override the session's SMTP host for this invocation")
	fs.IntP("smtp-port", "P", 0, "override the session's SMTP port for this invocation")
	fs.StringP("auth-user", "U", "", "override SMTP AUTH username for this invocation")
	fs.StringP("auth-pass", "W", "", "override SMTP AUTH password for this invocation")
	fs.String("smtp-tls", "", "override transport security for this invocation: none|starttls|implicit")
	fs.Bool("smtp-tls-insecure-skip-verify", false, "accept a self-signed/dev SMTP TLS certificate without verification for this invocation")
	return fs
}

func (b IntMsg) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	if stopID, _ := fs.GetString("stop"); stopID != "" {
		return stopIntMsgJob(ctx, s, stopID)
	}
	if list, _ := fs.GetBool("list"); list {
		return listIntMsgJobs(s)
	}

	interval, _ := fs.GetDuration("interval")
	rate, _ := fs.GetFloat64("rate")
	if fs.Changed("interval") && fs.Changed("rate") {
		return fmt.Errorf("intmsg: only one of --interval or --rate may be set")
	}
	if rate > 0 {
		interval = time.Duration(float64(time.Second) / rate)
	}
	if interval <= 0 {
		return fmt.Errorf("intmsg: --interval/--rate must resolve to a positive duration")
	}

	jitterStr, _ := fs.GetString("jitter")
	jitter, err := parseJitter(jitterStr, interval)
	if err != nil {
		return err
	}

	profile, _ := fs.GetString("profile")
	switch profile {
	case "steady", "poisson", "bursty":
	default:
		return fmt.Errorf("intmsg: --profile must be steady, poisson, or bursty, got %q", profile)
	}

	addr, auth, tlsOpts, err := resolveSMTP(s, fs)
	if err != nil {
		return err
	}

	cfg := intmsgRunConfig{
		fs:            fs,
		addr:          addr,
		auth:          auth,
		tlsOpts:       tlsOpts,
		sched:         &intervalScheduler{profile: profile, mean: interval, jitter: jitter, tmpl: s.Tmpl},
		profile:       profile,
		burstSize:     mustGetInt(fs, "burst-size"),
		burstInterval: mustGetDuration(fs, "burst-interval"),
		count:         mustGetInt(fs, "count"),
		duration:      mustGetDuration(fs, "duration"),
		statsInterval: mustGetDuration(fs, "stats-interval"),
		quiet:         mustGetBool(fs, "quiet"),
		untilError:    mustGetBool(fs, "until-error"),
	}

	if background, _ := fs.GetBool("background"); background {
		job := s.NewJob()
		jobCtx, cancel := context.WithCancel(ctx)
		job.Cancel = cancel
		go func() {
			summary := runIntMsgLoop(jobCtx, s, cfg, nil, false, job.UpdateProgress)
			job.Finish(summary)
		}()
		fmt.Fprintf(s.Out, "intmsg started in background (id=%s); stop with: intmsg --stop %s\n", job.ID, job.ID)
		return nil
	}

	// Overrides the shell's normal Ctrl-C behavior ("discard line, do not
	// exit" — SPEC.md §7.5.9) for the duration of this run: the first
	// Ctrl-C here stops the run and prints a summary; readline's own
	// interrupt handling resumes once Run returns control to the prompt.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	summary := runIntMsgLoop(ctx, s, cfg, sigCh, true, nil)
	fmt.Fprintln(s.Out, summary)
	return nil
}

// listIntMsgJobs prints every registered background intmsg run — running or
// already finished but not yet --stop'd — with its live sent/failed counts.
func listIntMsgJobs(s *shell.Session) error {
	jobs := s.Jobs()
	if len(jobs) == 0 {
		fmt.Fprintln(s.Out, "no background intmsg runs")
		return nil
	}
	fmt.Fprintf(s.Out, "%-4s  %-8s  %6s  %6s  %10s  %8s\n", "ID", "STATUS", "SENT", "FAILED", "ELAPSED", "RATE")
	for _, job := range jobs {
		sent, failed, done := job.Snapshot()
		status := "running"
		if done {
			status = "finished"
		}
		elapsed := time.Since(job.StartedAt)
		rate := 0.0
		if elapsed > 0 {
			rate = float64(sent) / elapsed.Seconds()
		}
		fmt.Fprintf(s.Out, "%-4s  %-8s  %6d  %6d  %10s  %6.1f/s\n",
			job.ID, status, sent, failed, elapsed.Round(time.Millisecond), rate)
	}
	return nil
}

// stopIntMsgJob cancels the background job registered under id and prints
// its final summary — whether it was still running (Cancel triggers the
// loop's next select to exit) or had already finished on its own (Wait
// returns immediately, since Finish already closed the done channel).
func stopIntMsgJob(ctx context.Context, s *shell.Session, id string) error {
	job, ok := s.Job(id)
	if !ok {
		return fmt.Errorf("intmsg: no background run with id %q", id)
	}
	job.Cancel()

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	summary, finished := job.Wait(waitCtx)
	s.RemoveJob(id)
	if !finished {
		return fmt.Errorf("intmsg: run %q did not stop within 5s", id)
	}
	fmt.Fprintln(s.Out, summary)
	return nil
}

// intmsgRunConfig holds everything runIntMsgLoop needs, parsed once by
// Run() and shared between the foreground and --background code paths.
type intmsgRunConfig struct {
	fs            *pflag.FlagSet
	addr          string
	auth          *cliclient.Auth
	tlsOpts       cliclient.TLSOptions
	sched         *intervalScheduler
	profile       string
	burstSize     int
	burstInterval time.Duration
	count         int
	duration      time.Duration
	statsInterval time.Duration
	quiet         bool
	untilError    bool
}

// runIntMsgLoop runs the interval-scheduled send loop until count/duration
// is reached, sigCh fires (nil when backgrounded — a background run is only
// stoppable via "intmsg --stop", not Ctrl-C, since Ctrl-C at the prompt
// refers to whatever else is being typed), or ctx is cancelled (by --stop's
// job.Cancel(), for a background run). It returns the final summary line;
// when live is true (foreground only) it also prints progress/live-stats
// directly to s.Out as it goes — a backgrounded run (live=false) never
// touches s.Out/s.Err, since nothing here can safely print above an active
// readline prompt. progress, when non-nil, is called after every send
// attempt with the running sent/failed counts — used by a backgrounded run
// so "intmsg --list" can report live status without needing to stop it
// first.
func runIntMsgLoop(ctx context.Context, s *shell.Session, cfg intmsgRunConfig, sigCh <-chan os.Signal, live bool, progress func(sent, failed int)) string {
	sent, failed := 0, 0
	start := time.Now()
	deadline := time.Time{}
	if cfg.duration > 0 {
		deadline = start.Add(cfg.duration)
	}

	var statsTickerC <-chan time.Time
	if live {
		statsTicker := time.NewTicker(cfg.statsInterval)
		defer statsTicker.Stop()
		statsTickerC = statsTicker.C
	}

	summaryLine := func() string {
		elapsed := time.Since(start)
		rate := 0.0
		if elapsed > 0 {
			rate = float64(sent) / elapsed.Seconds()
		}
		return fmt.Sprintf("sent=%d failed=%d elapsed=%s rate=%.1f/s", sent, failed, elapsed.Round(time.Millisecond), rate)
	}

	sendOne := func() bool {
		data := map[string]any{"count": cfg.count, "n": sent + 1, "index": sent}
		for k, v := range s.TemplateData() {
			data[k] = v
		}
		spec, err := buildRandomSpec(cfg.fs, s, data)
		if err == nil {
			var raw []byte
			raw, err = spec.Build(time.Now())
			if err == nil {
				from, to := spec.Envelope()
				err = cliclient.SendTLS(ctx, cfg.addr, cfg.tlsOpts, cfg.auth, from, to, raw)
			}
		}
		if err != nil {
			failed++
			if live {
				fmt.Fprintf(s.Err, "intmsg: message %d: %v\n", sent+failed, err)
			}
			if progress != nil {
				progress(sent, failed)
			}
			return false
		}
		sent++
		if live && !cfg.quiet {
			fmt.Fprintf(s.Out, "sent message %d\n", sent)
		}
		if progress != nil {
			progress(sent, failed)
		}
		return true
	}

	stop := func() bool {
		if cfg.count > 0 && sent >= cfg.count {
			return true
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return true
		}
		return false
	}

runLoop:
	for !stop() {
		select {
		case <-ctx.Done():
			break runLoop
		case <-sigChOrNever(sigCh):
			break runLoop
		case <-statsTickerCOrNever(statsTickerC):
			if live {
				fmt.Fprintln(s.Out, summaryLine())
			}
			continue
		default:
		}

		if cfg.profile == "bursty" {
			for i := 0; i < cfg.burstSize && !stop(); i++ {
				if !sendOne() && cfg.untilError {
					break runLoop
				}
				select {
				case <-ctx.Done():
					break runLoop
				case <-sigChOrNever(sigCh):
					break runLoop
				case <-time.After(cfg.burstInterval):
				}
			}
		} else if !sendOne() && cfg.untilError {
			break runLoop
		}

		if stop() {
			break
		}

		wait := cfg.sched.next()
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			break runLoop
		case <-sigChOrNever(sigCh):
			timer.Stop()
			break runLoop
		case <-timer.C:
		}
	}

	return summaryLine()
}

// sigChOrNever returns ch if non-nil, or a channel that never fires — used
// so runIntMsgLoop's select statements work identically whether or not a
// Ctrl-C override channel is in play (background runs pass nil).
func sigChOrNever(ch <-chan os.Signal) <-chan os.Signal {
	if ch != nil {
		return ch
	}
	return neverSignal
}

// statsTickerCOrNever is the <-chan time.Time analogue of sigChOrNever, for
// the live-stats ticker (nil when backgrounded, since a background run
// never prints).
func statsTickerCOrNever(ch <-chan time.Time) <-chan time.Time {
	if ch != nil {
		return ch
	}
	return neverTime
}

var (
	neverSignal = make(chan os.Signal)
	neverTime   = make(chan time.Time)
)

func mustGetInt(fs *pflag.FlagSet, name string) int {
	v, _ := fs.GetInt(name)
	return v
}

func mustGetDuration(fs *pflag.FlagSet, name string) time.Duration {
	v, _ := fs.GetDuration(name)
	return v
}

func mustGetBool(fs *pflag.FlagSet, name string) bool {
	v, _ := fs.GetBool(name)
	return v
}

// intervalScheduler draws the next inter-message gap per SPEC.md §7.6.3's
// three profiles, from the session's seeded tmpl.Engine so --seed
// reproduces both message content and timing.
type intervalScheduler struct {
	profile string
	mean    time.Duration
	jitter  time.Duration
	tmpl    *tmpl.Engine
}

func (sc *intervalScheduler) next() time.Duration {
	switch sc.profile {
	case "poisson":
		d := time.Duration(sc.tmpl.ExpFloat64() * float64(sc.mean))
		if d < 0 {
			d = 0
		}
		return d
	default: // steady, and bursty's quiet-period gap
		if sc.jitter <= 0 {
			return sc.mean
		}
		delta := time.Duration(sc.tmpl.Float64()*2*float64(sc.jitter)) - sc.jitter
		d := sc.mean + delta
		if d < 0 {
			d = 0
		}
		return d
	}
}

// parseJitter accepts either a bare duration string (e.g. "200ms") or a
// percentage string (e.g. "20%"), the latter resolved against mean per
// SPEC.md §7.6.3.
func parseJitter(s string, mean time.Duration) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}
	if strings.HasSuffix(s, "%") {
		pctStr := strings.TrimSuffix(s, "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0, fmt.Errorf("intmsg: invalid --jitter percentage %q: %w", s, err)
		}
		return time.Duration(float64(mean) * pct / 100), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("intmsg: invalid --jitter %q: %w", s, err)
	}
	return d, nil
}
