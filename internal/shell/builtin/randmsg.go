package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

// addRandContentFlags registers the shared content-shaping flags SPEC.md
// §7.6.5 lists across randmsg/intmsg/weirdmsg/blast/deluge, so every
// builtin's Flags() stays in lockstep with a single definition.
func addRandContentFlags(fs *pflag.FlagSet) {
	fs.StringP("to", "t", "", "To address (default: a fake email)")
	fs.StringP("from", "f", "", "From address (default: a fake email)")
	fs.StringArrayP("cc", "c", nil, "Cc address (repeatable)")
	fs.StringArrayP("bcc", "C", nil, "Bcc address (repeatable)")
	fs.StringP("subject", "s", "", "Subject (default: a fake subject, or the scenario's)")
	fs.StringP("body", "B", "random", "body source: text|html|both|random")
	fs.IntP("attachments", "a", 0, "number of generated attachments to include")
	fs.StringP("attachment-size", "A", "10KB", "size of each generated attachment")
	fs.StringArrayP("tags", "T", nil, "tag to attach to the message (repeatable)")
	fs.StringP("scenario", "S", "", "seed subject/body from a canned example scenario (see: example --list)")
}

// buildRandomSpec builds one cliclient.MessageSpec per SPEC.md §7.6.2's
// default table: every field defaults to a template-function call when its
// flag is omitted, so repeated calls (e.g. across a bulk send) each produce
// distinct fake content from the session's seeded Engine.
func buildRandomSpec(fs *pflag.FlagSet, s *shell.Session, data map[string]any) (cliclient.MessageSpec, error) {
	render := func(expr string) (string, error) { return s.Tmpl.Render(expr, data) }

	var spec cliclient.MessageSpec
	var err error

	to, _ := fs.GetString("to")
	if to == "" {
		to = "{{ fEmail }}"
	}
	if spec.To, err = renderAll(render, []string{to}); err != nil {
		return spec, err
	}

	from, _ := fs.GetString("from")
	if from == "" {
		from = "{{ fEmail }}"
	}
	if spec.From, err = render(from); err != nil {
		return spec, err
	}

	cc, _ := fs.GetStringArray("cc")
	if spec.Cc, err = renderAll(render, cc); err != nil {
		return spec, err
	}
	bcc, _ := fs.GetStringArray("bcc")
	if spec.Bcc, err = renderAll(render, bcc); err != nil {
		return spec, err
	}

	scenarioName, _ := fs.GetString("scenario")
	var scenario exampleTemplate
	haveScenario := false
	if scenarioName != "" {
		scenario, haveScenario = findScenario(scenarioName)
		if !haveScenario {
			return spec, fmt.Errorf("randmsg: unknown --scenario %q (see: example --list)", scenarioName)
		}
	}

	subject, _ := fs.GetString("subject")
	if subject == "" {
		if haveScenario {
			subject = scenario.Subject
		} else {
			subject = "{{ fSubject }}"
		}
	}
	if spec.Subject, err = render(subject); err != nil {
		return spec, err
	}

	if err := renderRandomBody(fs, s, data, haveScenario, scenario, &spec); err != nil {
		return spec, err
	}

	if n, _ := fs.GetInt("attachments"); n > 0 {
		size, _ := fs.GetString("attachment-size")
		specs, err := randomAttachments(s, data, n, size)
		if err != nil {
			return spec, err
		}
		spec.Attachments = specs
	}

	spec.Tags, _ = fs.GetStringArray("tags")

	return spec, nil
}

// renderRandomBody fills spec.Text/HTML per --body (text|html|both|random),
// preferring the scenario's own Text/HTML when --scenario is set (falling
// back to fake-generated bodies for whichever of Text/HTML the scenario
// left empty, matching --body's mode for that field).
func renderRandomBody(fs *pflag.FlagSet, s *shell.Session, data map[string]any, haveScenario bool, scenario exampleTemplate, spec *cliclient.MessageSpec) error {
	render := func(expr string) (string, error) { return s.Tmpl.Render(expr, data) }

	mode, _ := fs.GetString("body")
	switch mode {
	case "text", "html", "both", "random":
	default:
		return fmt.Errorf("randmsg: --body must be text, html, both, or random, got %q", mode)
	}
	if mode == "random" {
		switch s.Tmpl.Intn(3) {
		case 0:
			mode = "text"
		case 1:
			mode = "html"
		default:
			mode = "both"
		}
	}

	wantText := mode == "text" || mode == "both"
	wantHTML := mode == "html" || mode == "both"

	var err error
	if wantText {
		src := "{{ fTextBody }}"
		if haveScenario && scenario.Text != "" {
			src = scenario.Text
		}
		if spec.Text, err = render(src); err != nil {
			return err
		}
	}
	if wantHTML {
		src := "{{ fHTMLBody }}"
		if haveScenario && scenario.HTML != "" {
			src = scenario.HTML
		}
		if spec.HTML, err = render(src); err != nil {
			return err
		}
	}
	return nil
}

// randomAttachments generates n files of the given size via the template
// engine's fBinary function (through Render, since Engine's
// file-generating methods are private to the tmpl package) and returns them
// as AttachmentSpecs.
func randomAttachments(s *shell.Session, data map[string]any, n int, size string) ([]cliclient.AttachmentSpec, error) {
	specs := make([]cliclient.AttachmentSpec, 0, n)
	for i := 0; i < n; i++ {
		path, err := s.Tmpl.Render(fmt.Sprintf("{{ fBinary %q }}", size), data)
		if err != nil {
			return nil, fmt.Errorf("randmsg: generating attachment: %w", err)
		}
		specs = append(specs, cliclient.AttachmentSpec{Path: path})
	}
	return specs, nil
}

// runBulkRandom re-renders and sends count messages built by buildRandomSpec
// with bounded concurrency, mirroring send.go's bulk-send loop exactly. It
// backs both the "randmsg" and "deluge" builtins.
func runBulkRandom(ctx context.Context, s *shell.Session, fs *pflag.FlagSet, addr string, tlsOpts cliclient.TLSOptions, auth *cliclient.Auth, count, concurrency int) (sent int, errs []string, err error) {
	if count < 1 {
		count = 1
	}
	if concurrency < 1 {
		concurrency = 1
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			data := map[string]any{"index": idx, "count": count, "n": idx + 1}
			for k, v := range s.TemplateData() {
				data[k] = v
			}

			spec, buildErr := buildRandomSpec(fs, s, data)
			if buildErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("message %d: build: %v", idx+1, buildErr))
				mu.Unlock()
				return
			}
			raw, buildErr := spec.Build(time.Now())
			if buildErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("message %d: build: %v", idx+1, buildErr))
				mu.Unlock()
				return
			}
			from, to := spec.Envelope()
			if sendErr := cliclient.SendTLS(ctx, addr, tlsOpts, auth, from, to, raw); sendErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("message %d: send: %v", idx+1, sendErr))
				mu.Unlock()
				return
			}
			mu.Lock()
			sent++
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	if len(errs) > 0 {
		err = fmt.Errorf("%d of %d messages failed", len(errs), count)
	}
	return sent, errs, err
}

// RandMsg implements the "randmsg" builtin (SPEC.md §7.6.2): sends a
// randomly-generated message with zero required flags.
type RandMsg struct{}

func (RandMsg) Name() string      { return "randmsg" }
func (RandMsg) Aliases() []string { return nil }
func (RandMsg) Short() string     { return "Send a randomly-generated message" }

func (RandMsg) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("randmsg", pflag.ContinueOnError)
	addRandContentFlags(fs)
	fs.IntP("count", "n", 1, "number of messages to send")
	fs.IntP("concurrency", "j", 1, "max parallel SMTP connections")
	fs.BoolP("dry-run", "d", false, "render but do not send; print the result")
	fs.StringP("smtp-host", "H", "", "override the session's SMTP host for this invocation")
	fs.IntP("smtp-port", "P", 0, "override the session's SMTP port for this invocation")
	fs.StringP("auth-user", "U", "", "override SMTP AUTH username for this invocation")
	fs.StringP("auth-pass", "W", "", "override SMTP AUTH password for this invocation")
	fs.String("smtp-tls", "", "override transport security for this invocation: none|starttls|implicit")
	fs.Bool("smtp-tls-insecure-skip-verify", false, "accept a self-signed/dev SMTP TLS certificate without verification for this invocation")
	return fs
}

func (b RandMsg) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	count, _ := fs.GetInt("count")
	if count < 1 {
		count = 1
	}
	concurrency, _ := fs.GetInt("concurrency")

	addr, auth, tlsOpts, err := resolveSMTP(s, fs)
	if err != nil {
		return err
	}

	if dryRun, _ := fs.GetBool("dry-run"); dryRun {
		data := map[string]any{"index": 0, "count": count, "n": 1}
		for k, v := range s.TemplateData() {
			data[k] = v
		}
		spec, err := buildRandomSpec(fs, s, data)
		if err != nil {
			return err
		}
		raw, err := spec.Build(time.Now())
		if err != nil {
			return err
		}
		from, to := spec.Envelope()
		fmt.Fprintf(s.Out, "--- dry run (1 of %d shown): %s -> %v ---\n%s\n", count, from, to, string(raw))
		if count > 1 {
			fmt.Fprintf(s.Out, "(%d total messages would be sent)\n", count)
		}
		return nil
	}

	sent, errs, err := runBulkRandom(ctx, s, fs, addr, tlsOpts, auth, count, concurrency)
	fmt.Fprintf(s.Out, "%d/%d sent\n", sent, count)
	for _, e := range errs {
		fmt.Fprintln(s.Err, e)
	}
	return err
}
