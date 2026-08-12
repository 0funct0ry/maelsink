package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

// attachPathDelim mirrors internal/shell/tmpl/funcs_binary.go's unexported
// attachDelim: paths produced by the `attach` template func are joined with
// "::" so send's --attach flag can chain multiple generated files from one
// template expression.
const attachPathDelim = "::"

// Send implements the "send" builtin (SPEC.md §7.5.4/§7.5.5) — the largest
// builtin, supporting every body-source mode, bulk sending with
// per-message template re-rendering, and dry-run.
type Send struct{}

func (Send) Name() string      { return "send" }
func (Send) Aliases() []string { return nil }
func (Send) Short() string     { return "Compose and send a message over SMTP" }

func (Send) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("send", pflag.ContinueOnError)
	fs.String("from", "", "From address")
	fs.StringArray("to", nil, "To address (repeatable)")
	fs.StringArray("cc", nil, "Cc address (repeatable)")
	fs.StringArray("bcc", nil, "Bcc address (repeatable)")
	fs.String("subject", "", "Subject")
	fs.String("text", "", "plain-text body")
	fs.String("html", "", "HTML body")
	fs.StringArray("attach", nil, "attach a file (repeatable; `::`-joined paths from the attach template func are split)")
	fs.String("dir", "", "attach every regular file in this directory")
	fs.Bool("recursive", false, "with --dir, walk subdirectories too")
	fs.String("eml", "", "send this file verbatim as RFC 5322 (not templated)")
	fs.String("template", "", "render this file's full RFC 5322 message via the template engine")
	fs.String("body-file", "", "render this file as the message body; headers come from flags")
	fs.String("json", "", "read a cliclient.MessageSpec-shaped JSON file, templating its string fields")
	fs.Int("count", 1, "number of messages to send")
	fs.Int("concurrency", 1, "max parallel SMTP connections")
	fs.Bool("dry-run", false, "render but do not send; print the result(s)")
	fs.String("smtp-host", "", "override the session's SMTP host for this invocation")
	fs.Int("smtp-port", 0, "override the session's SMTP port for this invocation")
	fs.String("auth-user", "", "override SMTP AUTH username for this invocation")
	fs.String("auth-pass", "", "override SMTP AUTH password for this invocation")
	return fs
}

// sendMode identifies which primary body source send.go is using.
type sendMode int

const (
	modeFlags sendMode = iota
	modeEML
	modeTemplate
	modeBodyFile
	modeJSON
)

func (b Send) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	eml, _ := fs.GetString("eml")
	tmplFile, _ := fs.GetString("template")
	bodyFile, _ := fs.GetString("body-file")
	jsonFile, _ := fs.GetString("json")

	mode, primary, err := resolveSendMode(eml, tmplFile, bodyFile, jsonFile)
	if err != nil {
		return err
	}
	_ = primary

	count, _ := fs.GetInt("count")
	if count < 1 {
		count = 1
	}
	concurrency, _ := fs.GetInt("concurrency")
	if concurrency < 1 {
		concurrency = 1
	}
	dryRun, _ := fs.GetBool("dry-run")

	addr, auth, err := resolveSMTP(s, fs)
	if err != nil {
		return err
	}

	attachPaths, err := collectAttachments(fs)
	if err != nil {
		return err
	}

	// --eml is sent verbatim, never templated, and never looped per §7.5.5.
	if mode == modeEML {
		raw, from, to, err := buildEMLMessage(fs, eml)
		if err != nil {
			return err
		}
		if dryRun {
			fmt.Fprintf(s.Out, "--- dry run: %s -> %v ---\n%s\n", from, to, string(raw))
			return nil
		}
		if err := cliclient.Send(ctx, addr, auth, from, to, raw); err != nil {
			return fmt.Errorf("send: %w", err)
		}
		fmt.Fprintln(s.Out, "sent 1 message")
		return nil
	}

	var tmplContent string
	if mode == modeTemplate {
		data, err := os.ReadFile(tmplFile)
		if err != nil {
			return err
		}
		tmplContent = string(data)
	}

	type built struct {
		raw  []byte
		from string
		to   []string
	}

	buildOne := func(idx int) (built, error) {
		data := map[string]any{"index": idx, "count": count, "n": idx + 1}
		for k, v := range s.TemplateData() {
			data[k] = v
		}

		switch mode {
		case modeTemplate:
			rendered, err := s.Tmpl.Render(tmplContent, data)
			if err != nil {
				return built{}, err
			}
			// The rendered content IS a complete RFC 5322 message
			// (SPEC.md §7.5.5) — derive the envelope from ITS OWN
			// From/To headers, exactly like --eml does, with --from/--to
			// flags (if given) overriding. Re-derived per message in bulk
			// sends, so a template whose From/To vary per render (e.g.
			// via {{ fEmail }}) produces a correctly-matching envelope
			// for each one, not a single envelope reused for all of them.
			from, to, err := envelopeFromMessage(fs, []byte(rendered), "--template")
			if err != nil {
				return built{}, err
			}
			return built{raw: []byte(rendered), from: from, to: to}, nil

		case modeBodyFile:
			rawContent, err := os.ReadFile(bodyFile)
			if err != nil {
				return built{}, err
			}
			body, err := s.Tmpl.Render(string(rawContent), data)
			if err != nil {
				return built{}, err
			}
			spec, err := headerSpecFromFlags(fs, s, data)
			if err != nil {
				return built{}, err
			}
			isHTML := strings.HasSuffix(strings.ToLower(bodyFile), ".html") || strings.HasSuffix(strings.ToLower(bodyFile), ".htm")
			if hv, _ := fs.GetString("html"); hv != "" {
				isHTML = true
			}
			if tv, _ := fs.GetString("text"); tv != "" {
				isHTML = false
			}
			if isHTML {
				spec.HTML = body
			} else {
				spec.Text = body
			}
			spec.Attachments = attachmentSpecsFor(attachPaths, fs)
			raw, err := spec.Build(time.Now())
			if err != nil {
				return built{}, err
			}
			from, to := spec.Envelope()
			return built{raw: raw, from: from, to: to}, nil

		case modeJSON:
			spec, err := jsonSpec(jsonFile, s, data)
			if err != nil {
				return built{}, err
			}
			spec.Attachments = append(spec.Attachments, attachmentSpecsFor(attachPaths, fs)...)
			raw, err := spec.Build(time.Now())
			if err != nil {
				return built{}, err
			}
			from, to := spec.Envelope()
			return built{raw: raw, from: from, to: to}, nil

		default: // modeFlags
			spec, err := headerSpecFromFlags(fs, s, data)
			if err != nil {
				return built{}, err
			}
			spec.Attachments = attachmentSpecsFor(attachPaths, fs)
			raw, err := spec.Build(time.Now())
			if err != nil {
				return built{}, err
			}
			from, to := spec.Envelope()
			return built{raw: raw, from: from, to: to}, nil
		}
	}

	if dryRun {
		b0, err := buildOne(0)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "--- dry run (1 of %d shown): %s -> %v ---\n%s\n", count, b0.from, b0.to, string(b0.raw))
		if len(attachPaths) > 0 {
			fmt.Fprintf(s.Out, "attachments: %s\n", strings.Join(attachPaths, ", "))
		}
		if count > 1 {
			fmt.Fprintf(s.Out, "(%d total messages would be sent)\n", count)
		}
		return nil
	}

	// Bulk send with bounded concurrency; each message gets its own fresh
	// render (buildOne re-renders the template per call, since the
	// Engine's seeded rand naturally advances across renders).
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var errs []string
	sent := 0

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			bmsg, err := buildOne(idx)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("message %d: build: %v", idx+1, err))
				mu.Unlock()
				return
			}
			if err := cliclient.Send(ctx, addr, auth, bmsg.from, bmsg.to, bmsg.raw); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("message %d: send: %v", idx+1, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			sent++
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	fmt.Fprintf(s.Out, "%d/%d sent\n", sent, count)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(s.Err, e)
		}
		return fmt.Errorf("send: %d of %d messages failed", len(errs), count)
	}
	return nil
}

func resolveSendMode(eml, tmplFile, bodyFile, jsonFile string) (sendMode, string, error) {
	present := map[string]string{}
	if eml != "" {
		present["--eml"] = eml
	}
	if tmplFile != "" {
		present["--template"] = tmplFile
	}
	if bodyFile != "" {
		present["--body-file"] = bodyFile
	}
	if jsonFile != "" {
		present["--json"] = jsonFile
	}
	if len(present) > 1 {
		names := make([]string, 0, len(present))
		for k := range present {
			names = append(names, k)
		}
		return 0, "", fmt.Errorf("send: only one primary body source is allowed, got %s", strings.Join(names, " and "))
	}
	switch {
	case eml != "":
		return modeEML, eml, nil
	case tmplFile != "":
		return modeTemplate, tmplFile, nil
	case bodyFile != "":
		return modeBodyFile, bodyFile, nil
	case jsonFile != "":
		return modeJSON, jsonFile, nil
	default:
		return modeFlags, "", nil
	}
}

func resolveSMTP(s *shell.Session, fs *pflag.FlagSet) (string, *cliclient.Auth, error) {
	addr := s.SMTPAddr
	host, port := "", ""
	if h, _ := fs.GetString("smtp-host"); h != "" {
		host = h
	}
	if p, _ := fs.GetInt("smtp-port"); p != 0 {
		port = fmt.Sprintf("%d", p)
	}
	if host != "" || port != "" {
		curHost, curPort := splitAddr(addr)
		if host == "" {
			host = curHost
		}
		if port == "" {
			port = curPort
		}
		addr = host + ":" + port
	}

	auth := s.SMTPAuth
	user, _ := fs.GetString("auth-user")
	pass, _ := fs.GetString("auth-pass")
	if user != "" || pass != "" {
		a := &cliclient.Auth{}
		if auth != nil {
			*a = *auth
		}
		if user != "" {
			a.Username = user
		}
		if pass != "" {
			a.Password = pass
		}
		auth = a
	}
	return addr, auth, nil
}

func splitAddr(addr string) (host, port string) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, ""
	}
	return addr[:idx], addr[idx+1:]
}

// collectAttachments resolves --attach (splitting any `::`-joined values
// from the attach template func) plus --dir/--recursive into one flat list
// of file paths.
func collectAttachments(fs *pflag.FlagSet) ([]string, error) {
	var paths []string
	attachFlags, _ := fs.GetStringArray("attach")
	for _, a := range attachFlags {
		paths = append(paths, strings.Split(a, attachPathDelim)...)
	}

	dir, _ := fs.GetString("dir")
	if dir != "" {
		recursive, _ := fs.GetBool("recursive")
		entries, err := walkDir(dir, recursive)
		if err != nil {
			return nil, err
		}
		paths = append(paths, entries...)
	}
	return paths, nil
}

func walkDir(dir string, recursive bool) ([]string, error) {
	var out []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		if e.IsDir() {
			if recursive {
				sub, err := walkDir(full, recursive)
				if err != nil {
					return nil, err
				}
				out = append(out, sub...)
			}
			continue
		}
		out = append(out, full)
	}
	return out, nil
}

func attachmentSpecsFor(paths []string, fs *pflag.FlagSet) []cliclient.AttachmentSpec {
	specs := make([]cliclient.AttachmentSpec, 0, len(paths))
	for _, p := range paths {
		specs = append(specs, cliclient.AttachmentSpec{Path: p})
	}
	return specs
}

// headerSpecFromFlags builds a MessageSpec from --from/--to/--cc/--bcc/
// --subject/--text/--html, templating each string field through s.Tmpl for
// consistency with --json's per-field templating (SPEC.md §7.5.5).
func headerSpecFromFlags(fs *pflag.FlagSet, s *shell.Session, data map[string]any) (cliclient.MessageSpec, error) {
	render := func(v string) (string, error) { return s.Tmpl.Render(v, data) }

	from, _ := fs.GetString("from")
	subject, _ := fs.GetString("subject")
	text, _ := fs.GetString("text")
	html, _ := fs.GetString("html")
	to, _ := fs.GetStringArray("to")
	cc, _ := fs.GetStringArray("cc")
	bcc, _ := fs.GetStringArray("bcc")

	var spec cliclient.MessageSpec
	var err error
	if spec.From, err = render(from); err != nil {
		return spec, err
	}
	if spec.Subject, err = render(subject); err != nil {
		return spec, err
	}
	if spec.Text, err = render(text); err != nil {
		return spec, err
	}
	if spec.HTML, err = render(html); err != nil {
		return spec, err
	}
	if spec.To, err = renderAll(render, to); err != nil {
		return spec, err
	}
	if spec.Cc, err = renderAll(render, cc); err != nil {
		return spec, err
	}
	if spec.Bcc, err = renderAll(render, bcc); err != nil {
		return spec, err
	}
	return spec, nil
}

func renderAll(render func(string) (string, error), in []string) ([]string, error) {
	out := make([]string, len(in))
	for i, v := range in {
		r, err := render(v)
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

// jsonSpec reads path as a cliclient.MessageSpec JSON document and renders
// every string field (Subject/Text/HTML/From, and each To/Cc/Bcc entry)
// through s.Tmpl before returning (SPEC.md §7.5.5's "String fields inside
// it are templated").
func jsonSpec(path string, s *shell.Session, data map[string]any) (cliclient.MessageSpec, error) {
	var spec cliclient.MessageSpec
	raw, err := os.ReadFile(path)
	if err != nil {
		return spec, err
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return spec, fmt.Errorf("send --json: parsing %s: %w", path, err)
	}

	render := func(v string) (string, error) { return s.Tmpl.Render(v, data) }
	if spec.From, err = render(spec.From); err != nil {
		return spec, err
	}
	if spec.Subject, err = render(spec.Subject); err != nil {
		return spec, err
	}
	if spec.Text, err = render(spec.Text); err != nil {
		return spec, err
	}
	if spec.HTML, err = render(spec.HTML); err != nil {
		return spec, err
	}
	if spec.To, err = renderAll(render, spec.To); err != nil {
		return spec, err
	}
	if spec.Cc, err = renderAll(render, spec.Cc); err != nil {
		return spec, err
	}
	if spec.Bcc, err = renderAll(render, spec.Bcc); err != nil {
		return spec, err
	}
	return spec, nil
}

// buildEMLMessage reads path verbatim (never templated) and derives the
// SMTP envelope from its From/To headers via envelopeFromMessage, unless
// --from/--to override.
func buildEMLMessage(fs *pflag.FlagSet, path string) (raw []byte, from string, to []string, err error) {
	raw, err = os.ReadFile(path)
	if err != nil {
		return nil, "", nil, err
	}
	from, to, err = envelopeFromMessage(fs, raw, "--eml")
	if err != nil {
		return nil, "", nil, err
	}
	return raw, from, to, nil
}

// envelopeFromMessage parses raw as an RFC 5322 message and derives the
// SMTP envelope from its From/To headers, with --from/--to flags — when
// given — overriding whatever the headers say. modeLabel is used only in
// error messages (e.g. "--eml" or "--template") to say which mode's rules
// produced them.
//
// Used by both --eml (raw, never-templated content) and --template
// (rendered content — SPEC.md §7.5.5 describes both as producing "a
// complete RFC 5322 message", so both derive their envelope the same way;
// templating and header-parsing are orthogonal, nothing about the content
// having been rendered from a template changes how its headers are read).
func envelopeFromMessage(fs *pflag.FlagSet, raw []byte, modeLabel string) (from string, to []string, err error) {
	msg, perr := mail.ReadMessage(strings.NewReader(string(raw)))
	if perr == nil {
		if addrs, aerr := msg.Header.AddressList("From"); aerr == nil && len(addrs) > 0 {
			from = addrs[0].Address
		}
		if addrs, aerr := msg.Header.AddressList("To"); aerr == nil {
			for _, a := range addrs {
				to = append(to, a.Address)
			}
		}
	}

	if fromFlag, _ := fs.GetString("from"); fromFlag != "" {
		from = fromFlag
	}
	if toFlags, _ := fs.GetStringArray("to"); len(toFlags) > 0 {
		to = toFlags
	}
	if from == "" {
		return "", nil, fmt.Errorf("send %s: could not determine From (no From header and no --from override)", modeLabel)
	}
	if len(to) == 0 {
		return "", nil, fmt.Errorf("send %s: could not determine To (no To header and no --to override)", modeLabel)
	}
	return from, to, nil
}
