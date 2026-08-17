package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/msgspec"
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
// default table from fs's shared content flags (addRandContentFlags), via
// internal/msgspec.BuildRandomSpec — the Session-independent implementation
// shared with compose's job kinds (M13.3).
func buildRandomSpec(fs *pflag.FlagSet, s *shell.Session, data map[string]any) (cliclient.MessageSpec, error) {
	to, _ := fs.GetString("to")
	from, _ := fs.GetString("from")
	cc, _ := fs.GetStringArray("cc")
	bcc, _ := fs.GetStringArray("bcc")
	subject, _ := fs.GetString("subject")
	body, _ := fs.GetString("body")
	attachments, _ := fs.GetInt("attachments")
	attachmentSize, _ := fs.GetString("attachment-size")
	tags, _ := fs.GetStringArray("tags")
	scenario, _ := fs.GetString("scenario")

	spec, err := msgspec.BuildRandomSpec(s.Tmpl, data, msgspec.ContentParams{
		To:             to,
		From:           from,
		Cc:             cc,
		Bcc:            bcc,
		Subject:        subject,
		Body:           body,
		Attachments:    attachments,
		AttachmentSize: attachmentSize,
		Tags:           tags,
		Scenario:       scenario,
	})
	if err != nil {
		return spec, fmt.Errorf("randmsg: %w", err)
	}
	return spec, nil
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
