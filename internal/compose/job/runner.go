package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/msgspec"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// Target carries the SMTP connection info a job kind sends to — a minimal
// projection of internal/compose.TargetConfig (job must not import
// internal/compose, which imports job).
type Target struct {
	SMTPAddr string
	SMTPUser string
	SMTPPass string
}

func (t Target) auth() *cliclient.Auth {
	if t.SMTPUser == "" && t.SMTPPass == "" {
		return nil
	}
	return &cliclient.Auth{Username: t.SMTPUser, Password: t.SMTPPass}
}

// ContentParams mirrors SPEC.md §7.6.5's shared content flags, as accepted
// in a job's JSON request body.
type ContentParams struct {
	To             string   `json:"to,omitempty"`
	From           string   `json:"from,omitempty"`
	Cc             []string `json:"cc,omitempty"`
	Bcc            []string `json:"bcc,omitempty"`
	Subject        string   `json:"subject,omitempty"`
	Body           string   `json:"body,omitempty"` // text|html|both|random
	Attachments    int      `json:"attachments,omitempty"`
	AttachmentSize string   `json:"attachmentSize,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Scenario       string   `json:"scenario,omitempty"`
}

func (p ContentParams) toMsgspec() msgspec.ContentParams {
	return msgspec.ContentParams{
		To: p.To, From: p.From, Cc: p.Cc, Bcc: p.Bcc, Subject: p.Subject,
		Body: p.Body, Attachments: p.Attachments, AttachmentSize: p.AttachmentSize,
		Tags: p.Tags, Scenario: p.Scenario,
	}
}

func newEngine() (*tmpl.Engine, error) { return tmpl.New(0, false) }

// --- randmsg (SPEC.md §7.6.2) --------------------------------------------

// RandMsgParams is randmsg's job body: send Count messages with up to
// Concurrency parallel SMTP connections. Completes synchronously.
type RandMsgParams struct {
	ContentParams
	Count       int `json:"count,omitempty"`
	Concurrency int `json:"concurrency,omitempty"`
}

// RunRandMsg sends p.Count randomly-generated messages against target.
func RunRandMsg(target Target, p RandMsgParams) RunFunc {
	return func(ctx context.Context, progress func(sent, failed int)) error {
		count := p.Count
		if count < 1 {
			count = 1
		}
		concurrency := p.Concurrency
		if concurrency < 1 {
			concurrency = 1
		}

		engine, err := newEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)
		var mu sync.Mutex
		sent, failed := 0, 0
		var firstErr error

		for i := 0; i < count; i++ {
			if ctx.Err() != nil {
				break
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int) {
				defer wg.Done()
				defer func() { <-sem }()

				data := map[string]any{"index": idx, "count": count, "n": idx + 1}
				sendErr := sendOneRandom(ctx, engine, target, p.ContentParams, data)

				mu.Lock()
				if sendErr != nil {
					failed++
					if firstErr == nil {
						firstErr = sendErr
					}
				} else {
					sent++
				}
				progress(sent, failed)
				mu.Unlock()
			}(i)
		}
		wg.Wait()

		if failed > 0 {
			return fmt.Errorf("randmsg: %d of %d messages failed: %w", failed, count, firstErr)
		}
		return nil
	}
}

func sendOneRandom(ctx context.Context, engine *tmpl.Engine, target Target, p ContentParams, data map[string]any) error {
	spec, err := msgspec.BuildRandomSpec(engine, data, p.toMsgspec())
	if err != nil {
		return err
	}
	raw, err := spec.Build(time.Now())
	if err != nil {
		return err
	}
	from, to := spec.Envelope()
	return cliclient.SendTLS(ctx, target.SMTPAddr, cliclient.TLSOptions{}, target.auth(), from, to, raw)
}

// --- intmsg (SPEC.md §7.6.3) ---------------------------------------------

// IntMsgParams is intmsg's job body: send randmsg-style messages at
// randomized real-time intervals until Count or Duration is reached.
type IntMsgParams struct {
	ContentParams
	IntervalMS      int64   `json:"intervalMs,omitempty"`
	Rate            float64 `json:"rate,omitempty"`
	Jitter          string  `json:"jitter,omitempty"`
	Profile         string  `json:"profile,omitempty"` // steady|poisson|bursty
	BurstSize       int     `json:"burstSize,omitempty"`
	BurstIntervalMS int64   `json:"burstIntervalMs,omitempty"`
	Count           int     `json:"count,omitempty"`
	DurationMS      int64   `json:"durationMs,omitempty"`
	UntilError      bool    `json:"untilError,omitempty"`
}

// RunIntMsg runs intmsg's interval-scheduled send loop until p.Count/
// p.DurationMS is reached or ctx is cancelled (job Cancel).
func RunIntMsg(target Target, p IntMsgParams) RunFunc {
	return func(ctx context.Context, progress func(sent, failed int)) error {
		interval := time.Duration(p.IntervalMS) * time.Millisecond
		if p.Rate > 0 {
			interval = time.Duration(float64(time.Second) / p.Rate)
		}
		if interval <= 0 {
			interval = time.Second
		}

		jitter, err := msgspec.ParseJitter(p.Jitter, interval)
		if err != nil {
			return fmt.Errorf("intmsg: %w", err)
		}

		profile := p.Profile
		if profile == "" {
			profile = "steady"
		}
		switch profile {
		case "steady", "poisson", "bursty":
		default:
			return fmt.Errorf("intmsg: profile must be steady, poisson, or bursty, got %q", profile)
		}

		burstSize := p.BurstSize
		if burstSize < 1 {
			burstSize = 5
		}
		burstInterval := time.Duration(p.BurstIntervalMS) * time.Millisecond
		if burstInterval <= 0 {
			burstInterval = 100 * time.Millisecond
		}
		duration := time.Duration(p.DurationMS) * time.Millisecond

		engine, err := newEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		sched := &msgspec.IntervalScheduler{Profile: profile, Mean: interval, Jitter: jitter, Tmpl: engine}

		sent, failed := 0, 0
		start := time.Now()
		deadline := time.Time{}
		if duration > 0 {
			deadline = start.Add(duration)
		}

		stop := func() bool {
			if p.Count > 0 && sent >= p.Count {
				return true
			}
			if !deadline.IsZero() && time.Now().After(deadline) {
				return true
			}
			return false
		}

		sendOne := func() bool {
			data := map[string]any{"count": p.Count, "n": sent + 1, "index": sent}
			if err := sendOneRandom(ctx, engine, target, p.ContentParams, data); err != nil {
				failed++
				progress(sent, failed)
				return false
			}
			sent++
			progress(sent, failed)
			return true
		}

	runLoop:
		for !stop() {
			select {
			case <-ctx.Done():
				break runLoop
			default:
			}

			if profile == "bursty" {
				for i := 0; i < burstSize && !stop(); i++ {
					if !sendOne() && p.UntilError {
						break runLoop
					}
					select {
					case <-ctx.Done():
						break runLoop
					case <-time.After(burstInterval):
					}
				}
			} else if !sendOne() && p.UntilError {
				break runLoop
			}

			if stop() {
				break
			}

			timer := time.NewTimer(sched.Next())
			select {
			case <-ctx.Done():
				timer.Stop()
				break runLoop
			case <-timer.C:
			}
		}

		return nil
	}
}

// --- weirdmsg (SPEC.md §7.6.4) -------------------------------------------

var weirdKinds = []string{"bounce", "malformed", "huge", "unicode", "spoof", "thread", "invite"}

// WeirdMsgParams is weirdmsg's job body: send one message of an awkward,
// edge-case shape. Completes synchronously.
type WeirdMsgParams struct {
	Kind  string `json:"kind,omitempty"` // bounce|malformed|huge|unicode|spoof|thread|invite|random
	Size  string `json:"size,omitempty"`
	Depth int    `json:"depth,omitempty"`
	To    string `json:"to,omitempty"`
	From  string `json:"from,omitempty"`
}

// RunWeirdMsg builds and sends one weird-shaped message per p.Kind.
func RunWeirdMsg(target Target, p WeirdMsgParams) RunFunc {
	return func(ctx context.Context, progress func(sent, failed int)) error {
		engine, err := newEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		kind := p.Kind
		if kind == "" || kind == "random" {
			kind = weirdKinds[engine.Intn(len(weirdKinds))]
		} else {
			valid := false
			for _, k := range weirdKinds {
				if k == kind {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("weirdmsg: kind must be one of %v, or random, got %q", weirdKinds, kind)
			}
		}

		data := map[string]any{"count": 1, "n": 1, "index": 0}
		to := p.To
		if to == "" {
			if to, err = engine.Render("{{ fEmail }}", data); err != nil {
				return err
			}
		}
		from := p.From
		if from == "" {
			if from, err = engine.Render("{{ fEmail }}", data); err != nil {
				return err
			}
		}

		if kind == "thread" {
			depth := p.Depth
			if depth < 1 {
				depth = 5
			}
			if err := sendWeirdThread(ctx, engine, target, from, to, depth); err != nil {
				progress(0, 1)
				return err
			}
			progress(depth, 0)
			return nil
		}

		envFrom := from
		raw, err := buildWeird(engine, data, kind, from, to, p.Size)
		if err != nil {
			progress(0, 1)
			return err
		}
		if kind == "spoof" {
			if envFrom, err = engine.Render("{{ fEmail }}", data); err != nil {
				return err
			}
		}

		if err := cliclient.SendTLS(ctx, target.SMTPAddr, cliclient.TLSOptions{}, target.auth(), envFrom, []string{to}, raw); err != nil {
			progress(0, 1)
			return fmt.Errorf("weirdmsg: %w", err)
		}
		progress(1, 0)
		return nil
	}
}

func buildWeird(engine *tmpl.Engine, data map[string]any, kind, from, to, size string) ([]byte, error) {
	switch kind {
	case "bounce":
		return weirdBounce(engine, data, to)
	case "malformed":
		return weirdMalformed(engine, data, from, to)
	case "huge":
		if size == "" {
			size = "10MB"
		}
		return weirdHuge(engine, data, from, to, size)
	case "unicode":
		return weirdUnicode(engine, data, from, to)
	case "spoof":
		return weirdSpoof(from, to)
	case "invite":
		return weirdInvite(engine, data, from, to)
	default:
		return nil, fmt.Errorf("weirdmsg: unhandled kind %q", kind)
	}
}

func weirdBounce(engine *tmpl.Engine, data map[string]any, to string) ([]byte, error) {
	subject, err := engine.Render("Undelivered Mail Returned to Sender", data)
	if err != nil {
		return nil, err
	}
	diag, err := engine.Render("{{ fDomain }}", data)
	if err != nil {
		return nil, err
	}
	boundary := "bounce-boundary"
	now := time.Now().UTC().Format(time.RFC1123Z)
	body := fmt.Sprintf(
		"From: mailer-daemon@%s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/report; report-type=delivery-status; boundary=%q\r\n\r\n"+
			"--%s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nThe following message could not be delivered:\r\n%s\r\n\r\n"+
			"--%s\r\nContent-Type: message/delivery-status\r\n\r\nReporting-MTA: dns; %s\r\nAction: failed\r\nStatus: 5.1.1\r\nDiagnostic-Code: smtp; 550 5.1.1 User unknown\r\n\r\n"+
			"--%s--\r\n",
		diag, to, subject, now, boundary,
		boundary, subject,
		boundary, diag,
		boundary,
	)
	return []byte(body), nil
}

func weirdMalformed(engine *tmpl.Engine, data map[string]any, from, to string) ([]byte, error) {
	text, err := engine.Render("{{ fParagraph }}", data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC1123Z)
	body := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: malformed test message\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=\"declared-boundary\"\r\n\r\n"+
			"--wrong-boundary\r\nContent-Type: text/plain\r\nContent-Transfer-Encoding: bogus-7bit\r\n\r\n%s\r\n"+
			"--declared-boundary--\r\n",
		from, to, now, text,
	)
	return []byte(body), nil
}

func weirdHuge(engine *tmpl.Engine, data map[string]any, from, to, size string) ([]byte, error) {
	bodyPath, err := engine.Render(fmt.Sprintf("{{ fBinary %q }}", size), data)
	if err != nil {
		return nil, err
	}
	spec := cliclient.MessageSpec{
		From:    from,
		To:      []string{to},
		Subject: "huge test message",
		Text:    fmt.Sprintf("(generated body at %s)", bodyPath),
		Attachments: []cliclient.AttachmentSpec{
			{Path: bodyPath},
		},
	}
	return spec.Build(time.Now())
}

func weirdUnicode(engine *tmpl.Engine, data map[string]any, from, to string) ([]byte, error) {
	subject, err := engine.Render("测试 テスト בדיקה اختبار 🎉", data)
	if err != nil {
		return nil, err
	}
	text, err := engine.Render("こんにちは {{ fName }} — 你好 — مرحبا — שלום 🎉", data)
	if err != nil {
		return nil, err
	}
	spec := cliclient.MessageSpec{From: from, To: []string{to}, Subject: subject, Text: text}
	return spec.Build(time.Now())
}

func weirdSpoof(from, to string) ([]byte, error) {
	spec := cliclient.MessageSpec{
		From:    from,
		To:      []string{to},
		Subject: "spoof test message",
		Text:    "This message's From: header does not match its envelope MAIL FROM.",
	}
	return spec.Build(time.Now())
}

func weirdInvite(engine *tmpl.Engine, data map[string]any, from, to string) ([]byte, error) {
	subject, err := engine.Render("Meeting invite: {{ fSentence }}", data)
	if err != nil {
		return nil, err
	}
	uid, err := engine.Render("{{ uuid }}", data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ics := fmt.Sprintf(
		"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nMETHOD:REQUEST\r\nBEGIN:VEVENT\r\nUID:%s\r\nDTSTAMP:%s\r\nDTSTART:%s\r\nDTEND:%s\r\nSUMMARY:%s\r\nORGANIZER:mailto:%s\r\nATTENDEE:mailto:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		uid, now.Format("20060102T150405Z"), now.Format("20060102T150405Z"), now.Add(time.Hour).Format("20060102T150405Z"), subject, from, to,
	)
	boundary := "invite-boundary"
	body := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nDate: %s\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n"+
			"--%s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nYou have been invited to a meeting: %s\r\n\r\n"+
			"--%s\r\nContent-Type: text/calendar; method=REQUEST; charset=utf-8\r\n\r\n%s\r\n"+
			"--%s--\r\n",
		from, to, subject, now.Format(time.RFC1123Z), boundary,
		boundary, subject,
		boundary, ics,
		boundary,
	)
	return []byte(body), nil
}

func sendWeirdThread(ctx context.Context, engine *tmpl.Engine, target Target, from, to string, depth int) error {
	data := map[string]any{"count": 1, "n": 1, "index": 0}
	subject, err := engine.Render("{{ fSubject }}", data)
	if err != nil {
		return err
	}

	var references []string
	for i := 0; i < depth; i++ {
		msgID, err := engine.Render("<{{ uuid }}@maelsink.local>", data)
		if err != nil {
			return err
		}
		subj := subject
		if i > 0 {
			subj = "Re: " + subject
		}
		text, err := engine.Render("{{ fParagraph }}", data)
		if err != nil {
			return err
		}

		var b []byte
		header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: %s\r\n", from, to, subj, msgID)
		if len(references) > 0 {
			header += fmt.Sprintf("In-Reply-To: %s\r\nReferences: %s\r\n", references[len(references)-1], joinSpace(references))
		}
		header += "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n"
		b = append(b, []byte(header+text+"\r\n")...)

		if err := cliclient.SendTLS(ctx, target.SMTPAddr, cliclient.TLSOptions{}, target.auth(), from, []string{to}, b); err != nil {
			return fmt.Errorf("weirdmsg thread message %d: %w", i+1, err)
		}
		references = append(references, msgID)
	}
	return nil
}

func joinSpace(in []string) string {
	out := ""
	for i, v := range in {
		if i > 0 {
			out += " "
		}
		out += v
	}
	return out
}

// --- blast (SPEC.md §7.6.4) ----------------------------------------------

// BlastParams is blast's job body: send one message to Recipients generated
// recipients, distributed across To/Cc/Bcc per Split. Completes
// synchronously.
type BlastParams struct {
	ContentParams
	Recipients int    `json:"recipients,omitempty"`
	Split      string `json:"split,omitempty"` // to|cc|bcc|mixed
}

// RunBlast sends one rendered message to p.Recipients generated recipients.
func RunBlast(target Target, p BlastParams) RunFunc {
	return func(ctx context.Context, progress func(sent, failed int)) error {
		n := p.Recipients
		if n < 1 {
			n = 10
		}
		split := p.Split
		if split == "" {
			split = "to"
		}
		switch split {
		case "to", "cc", "bcc", "mixed":
		default:
			return fmt.Errorf("blast: split must be to, cc, bcc, or mixed, got %q", split)
		}

		engine, err := newEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		data := map[string]any{"count": 1, "n": 1, "index": 0}
		spec, err := msgspec.BuildRandomSpec(engine, data, p.ContentParams.toMsgspec())
		if err != nil {
			progress(0, 1)
			return err
		}

		spec.To, spec.Cc, spec.Bcc = nil, nil, nil
		for i := 0; i < n; i++ {
			rcpt, err := engine.Render("{{ fEmail }}", data)
			if err != nil {
				progress(0, 1)
				return err
			}
			bucket := split
			if split == "mixed" {
				switch i % 3 {
				case 0:
					bucket = "to"
				case 1:
					bucket = "cc"
				default:
					bucket = "bcc"
				}
			}
			switch bucket {
			case "to":
				spec.To = append(spec.To, rcpt)
			case "cc":
				spec.Cc = append(spec.Cc, rcpt)
			case "bcc":
				spec.Bcc = append(spec.Bcc, rcpt)
			}
		}
		if len(spec.To) == 0 {
			spec.To = []string{spec.From}
		}

		raw, err := spec.Build(time.Now())
		if err != nil {
			progress(0, 1)
			return err
		}
		from, to := spec.Envelope()

		if err := cliclient.SendTLS(ctx, target.SMTPAddr, cliclient.TLSOptions{}, target.auth(), from, to, raw); err != nil {
			progress(0, 1)
			return fmt.Errorf("blast: %w", err)
		}
		progress(1, 0)
		return nil
	}
}

// --- deluge (SPEC.md §7.6.4) ----------------------------------------------

// DelugeParams is deluge's job body: fire Count messages with no interval
// or jitter, bounded only by Concurrency.
type DelugeParams struct {
	ContentParams
	Count       int `json:"count,omitempty"`
	Concurrency int `json:"concurrency,omitempty"`
}

// RunDeluge fires p.Count messages at maximum throughput.
func RunDeluge(target Target, p DelugeParams) RunFunc {
	return RunRandMsg(target, RandMsgParams{
		ContentParams: p.ContentParams,
		Count:         orDefault(p.Count, 100),
		Concurrency:   orDefault(p.Concurrency, 10),
	})
}

func orDefault(v, def int) int {
	if v < 1 {
		return def
	}
	return v
}
