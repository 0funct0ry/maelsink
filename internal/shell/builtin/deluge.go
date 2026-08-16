package builtin

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Deluge implements the "deluge" builtin (SPEC.md §7.6.4): fires N
// randmsg-style messages with no interval or jitter, bounded only by
// --concurrency — a raw throughput ceiling check, as distinct from
// intmsg's realistic timing model.
type Deluge struct{}

func (Deluge) Name() string      { return "deluge" }
func (Deluge) Aliases() []string { return nil }
func (Deluge) Short() string     { return "Fire N random messages at maximum throughput" }

func (Deluge) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("deluge", pflag.ContinueOnError)
	addRandContentFlags(fs)
	fs.IntP("count", "n", 100, "number of messages to send")
	fs.IntP("concurrency", "j", 10, "max parallel SMTP connections")
	fs.StringP("smtp-host", "H", "", "override the session's SMTP host for this invocation")
	fs.IntP("smtp-port", "P", 0, "override the session's SMTP port for this invocation")
	fs.StringP("auth-user", "U", "", "override SMTP AUTH username for this invocation")
	fs.StringP("auth-pass", "W", "", "override SMTP AUTH password for this invocation")
	fs.String("smtp-tls", "", "override transport security for this invocation: none|starttls|implicit")
	fs.Bool("smtp-tls-insecure-skip-verify", false, "accept a self-signed/dev SMTP TLS certificate without verification for this invocation")
	return fs
}

func (b Deluge) Run(ctx context.Context, s *shell.Session, args []string) error {
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

	sent, errs, err := runBulkRandom(ctx, s, fs, addr, tlsOpts, auth, count, concurrency)
	fmt.Fprintf(s.Out, "%d/%d sent\n", sent, count)
	for _, e := range errs {
		fmt.Fprintln(s.Err, e)
	}
	return err
}
