package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell"
)

var weirdKinds = []string{"bounce", "malformed", "huge", "unicode", "spoof", "thread", "invite"}

// WeirdMsg implements the "weirdmsg" builtin (SPEC.md §7.6.4): generates one
// message of a deliberately awkward shape, for exercising MIME-parser edge
// cases and size limits end-to-end.
type WeirdMsg struct{}

func (WeirdMsg) Name() string      { return "weirdmsg" }
func (WeirdMsg) Aliases() []string { return nil }
func (WeirdMsg) Short() string     { return "Send one message of an awkward, edge-case shape" }

func (WeirdMsg) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("weirdmsg", pflag.ContinueOnError)
	fs.StringP("kind", "k", "random", "bounce|malformed|huge|unicode|spoof|thread|invite|random")
	fs.StringP("size", "s", "10MB", "body size for --kind huge")
	fs.IntP("depth", "d", 5, "chain length for --kind thread")
	fs.StringP("to", "t", "", "To address (default: a fake email)")
	fs.StringP("from", "f", "", "From address (default: a fake email)")
	fs.StringP("smtp-host", "H", "", "override the session's SMTP host for this invocation")
	fs.IntP("smtp-port", "P", 0, "override the session's SMTP port for this invocation")
	fs.StringP("auth-user", "U", "", "override SMTP AUTH username for this invocation")
	fs.StringP("auth-pass", "W", "", "override SMTP AUTH password for this invocation")
	return fs
}

func (b WeirdMsg) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}

	kind, _ := fs.GetString("kind")
	if kind == "random" {
		kind = weirdKinds[s.Tmpl.Intn(len(weirdKinds))]
	} else {
		valid := false
		for _, k := range weirdKinds {
			if k == kind {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("weirdmsg: --kind must be one of %s, or random, got %q", strings.Join(weirdKinds, ", "), kind)
		}
	}

	addr, auth, err := resolveSMTP(s, fs)
	if err != nil {
		return err
	}

	data := map[string]any{"count": 1, "n": 1, "index": 0}
	for k, v := range s.TemplateData() {
		data[k] = v
	}

	to, _ := fs.GetString("to")
	if to == "" {
		if to, err = s.Tmpl.Render("{{ fakeEmail }}", data); err != nil {
			return err
		}
	}
	from, _ := fs.GetString("from")
	if from == "" {
		if from, err = s.Tmpl.Render("{{ fakeEmail }}", data); err != nil {
			return err
		}
	}

	if kind == "thread" {
		depth, _ := fs.GetInt("depth")
		return sendWeirdThread(ctx, s, addr, auth, from, to, depth)
	}

	envFrom := from
	raw, err := buildWeird(fs, s, data, kind, from, to)
	if err != nil {
		return err
	}
	if kind == "spoof" {
		// Envelope MAIL FROM deliberately differs from the message's own
		// From: header baked into raw by buildWeird — that mismatch is the
		// whole point of --kind spoof (SPEC.md §7.6.4).
		if envFrom, err = s.Tmpl.Render("{{ fakeEmail }}", data); err != nil {
			return err
		}
	}

	if err := cliclient.Send(ctx, addr, auth, envFrom, []string{to}, raw); err != nil {
		return fmt.Errorf("weirdmsg: %w", err)
	}
	fmt.Fprintf(s.Out, "sent 1 %s message\n", kind)
	return nil
}

func buildWeird(fs *pflag.FlagSet, s *shell.Session, data map[string]any, kind, from, to string) ([]byte, error) {
	switch kind {
	case "bounce":
		return weirdBounce(s, data, from, to)
	case "malformed":
		return weirdMalformed(s, data, from, to)
	case "huge":
		size, _ := fs.GetString("size")
		return weirdHuge(s, data, from, to, size)
	case "unicode":
		return weirdUnicode(s, data, from, to)
	case "spoof":
		return weirdSpoof(s, data, from, to)
	case "invite":
		return weirdInvite(s, data, from, to)
	default:
		return nil, fmt.Errorf("weirdmsg: unhandled kind %q", kind)
	}
}

func weirdBounce(s *shell.Session, data map[string]any, from, to string) ([]byte, error) {
	subject, err := s.Tmpl.Render("Undelivered Mail Returned to Sender", data)
	if err != nil {
		return nil, err
	}
	diag, err := s.Tmpl.Render("{{ fakeDomain }}", data)
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

// weirdMalformed builds a deliberately non-conformant multipart message
// (mismatched boundary, a bogus Content-Transfer-Encoding) that is still a
// SMTP-transaction-valid blob of bytes — never broken enough to fail DATA
// itself, per SPEC.md §7.6.4.
func weirdMalformed(s *shell.Session, data map[string]any, from, to string) ([]byte, error) {
	text, err := s.Tmpl.Render("{{ fakeParagraph }}", data)
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

func weirdHuge(s *shell.Session, data map[string]any, from, to, size string) ([]byte, error) {
	bodyPath, err := s.Tmpl.Render(fmt.Sprintf("{{ fakeBinary %q }}", size), data)
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

func weirdUnicode(s *shell.Session, data map[string]any, from, to string) ([]byte, error) {
	subject, err := s.Tmpl.Render("测试 テスト בדיקה اختبار 🎉", data)
	if err != nil {
		return nil, err
	}
	text, err := s.Tmpl.Render("こんにちは {{ fakeName }} — 你好 — مرحبا — שלום 🎉", data)
	if err != nil {
		return nil, err
	}
	spec := cliclient.MessageSpec{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Text:    text,
	}
	return spec.Build(time.Now())
}

func weirdSpoof(s *shell.Session, data map[string]any, from, to string) ([]byte, error) {
	spec := cliclient.MessageSpec{
		From:    from,
		To:      []string{to},
		Subject: "spoof test message",
		Text:    "This message's From: header does not match its envelope MAIL FROM.",
	}
	return spec.Build(time.Now())
}

func weirdInvite(s *shell.Session, data map[string]any, from, to string) ([]byte, error) {
	subject, err := s.Tmpl.Render("Meeting invite: {{ fakeSentence }}", data)
	if err != nil {
		return nil, err
	}
	uid, err := s.Tmpl.Render("{{ uuid }}", data)
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

// sendWeirdThread sends depth messages sharing a Message-ID/References
// lineage and a common subject ("Re: ..." on replies), sent in order.
func sendWeirdThread(ctx context.Context, s *shell.Session, addr string, auth *cliclient.Auth, from, to string, depth int) error {
	if depth < 1 {
		depth = 1
	}
	data := map[string]any{"count": 1, "n": 1, "index": 0}
	for k, v := range s.TemplateData() {
		data[k] = v
	}
	subject, err := s.Tmpl.Render("{{ fakeSubject }}", data)
	if err != nil {
		return err
	}

	var references []string
	for i := 0; i < depth; i++ {
		msgID, err := s.Tmpl.Render("<{{ uuid }}@maelsink.local>", data)
		if err != nil {
			return err
		}
		subj := subject
		if i > 0 {
			subj = "Re: " + subject
		}
		text, err := s.Tmpl.Render("{{ fakeParagraph }}", data)
		if err != nil {
			return err
		}

		var headerLines strings.Builder
		fmt.Fprintf(&headerLines, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: %s\r\n", from, to, subj, msgID)
		if len(references) > 0 {
			fmt.Fprintf(&headerLines, "In-Reply-To: %s\r\nReferences: %s\r\n", references[len(references)-1], strings.Join(references, " "))
		}
		headerLines.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
		headerLines.WriteString(text)
		headerLines.WriteString("\r\n")

		if err := cliclient.Send(ctx, addr, auth, from, []string{to}, []byte(headerLines.String())); err != nil {
			return fmt.Errorf("weirdmsg thread message %d: %w", i+1, err)
		}
		references = append(references, msgID)
	}
	fmt.Fprintf(s.Out, "sent %d thread messages\n", depth)
	return nil
}
