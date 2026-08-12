package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

// Blast implements the "blast" builtin (SPEC.md §7.6.4): sends one rendered
// message to N generated recipients in a single SMTP transaction,
// distributed across To/Cc/Bcc per --split.
type Blast struct{}

func (Blast) Name() string      { return "blast" }
func (Blast) Aliases() []string { return nil }
func (Blast) Short() string     { return "Send one message to many generated recipients" }

func (Blast) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("blast", pflag.ContinueOnError)
	addRandContentFlags(fs)
	fs.IntP("recipients", "r", 10, "number of generated recipients")
	fs.StringP("split", "x", "to", "recipient distribution: to|cc|bcc|mixed")
	fs.BoolP("dry-run", "d", false, "render but do not send; print the result")
	fs.StringP("smtp-host", "H", "", "override the session's SMTP host for this invocation")
	fs.IntP("smtp-port", "P", 0, "override the session's SMTP port for this invocation")
	fs.StringP("auth-user", "U", "", "override SMTP AUTH username for this invocation")
	fs.StringP("auth-pass", "W", "", "override SMTP AUTH password for this invocation")
	return fs
}

func (b Blast) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	n, _ := fs.GetInt("recipients")
	if n < 1 {
		n = 1
	}
	split, _ := fs.GetString("split")
	switch split {
	case "to", "cc", "bcc", "mixed":
	default:
		return fmt.Errorf("blast: --split must be to, cc, bcc, or mixed, got %q", split)
	}

	addr, auth, err := resolveSMTP(s, fs)
	if err != nil {
		return err
	}

	data := map[string]any{"count": 1, "n": 1, "index": 0}
	for k, v := range s.TemplateData() {
		data[k] = v
	}

	spec, err := buildRandomSpec(fs, s, data)
	if err != nil {
		return err
	}

	// Recipients generated here override buildRandomSpec's own default
	// single --to, since blast's whole point is the recipient list shape.
	spec.To, spec.Cc, spec.Bcc = nil, nil, nil
	for i := 0; i < n; i++ {
		rcpt, err := s.Tmpl.Render("{{ fEmail }}", data)
		if err != nil {
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
		// SMTP/RFC 5322 needs at least one visible recipient; --split
		// cc/bcc alone would leave the To: header empty.
		spec.To = []string{spec.From}
	}

	raw, err := spec.Build(time.Now())
	if err != nil {
		return err
	}
	from, to := spec.Envelope()

	if dryRun, _ := fs.GetBool("dry-run"); dryRun {
		fmt.Fprintf(s.Out, "--- dry run: %s -> %v ---\n%s\n", from, to, string(raw))
		return nil
	}

	if err := cliclient.Send(ctx, addr, auth, from, to, raw); err != nil {
		return fmt.Errorf("blast: %w", err)
	}
	fmt.Fprintf(s.Out, "sent 1 message to %d recipients (to=%d cc=%d bcc=%d)\n", n, len(spec.To), len(spec.Cc), len(spec.Bcc))
	return nil
}
