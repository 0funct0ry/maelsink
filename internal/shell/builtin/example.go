package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// exampleTemplate is one canned scenario. Subject/From/To/Text/HTML are
// UNRENDERED — they contain {{ }} template-function calls verbatim, so the
// file "example" writes out is itself editable input for `send --json`
// (which templates from/to/subject/text/html per SPEC.md §7.5.5) or `send
// --template` (which treats the whole rendered .eml as the raw message).
type exampleTemplate struct {
	Title   string
	From    string
	To      string
	Subject string
	Text    string
	HTML    string // may be empty
}

// exampleTemplates are the 10 canned scenarios for the "example" builtin.
// Each exercises a handful of real template functions (SPEC.md §7.5.7) so
// a user gets a realistic, immediately-runnable starting point rather than
// a blank page, and can see functions in context before changing them.
var exampleTemplates = []exampleTemplate{
	{
		Title:   "Order confirmation",
		From:    "orders@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Your order is confirmed",
		// fakeOrder returns a dict; text/template can't dot-chain a field
		// straight off a function call (`fakeOrder.id` is invalid), so
		// dict fields are pulled out with `index (fakeOrder) "key"`.
		Text: "Hi {{ fakeName }},\n\nThanks for your order! Order ID: {{ index (fakeOrder) \"id\" }}\nTotal: {{ index (fakeOrder) \"total\" }} USD\nPlaced on: {{ rfc2822Date }}\n\n— {{ fakeCompany }}",
	},
	{
		Title:   "Password reset",
		From:    "security@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Reset your password",
		Text:    "We received a request to reset your password.\n\nReset code: {{ randString 8 \"ABCDEFGHJKLMNPQRSTUVWXYZ23456789\" }}\nThis code expires in 15 minutes. If you didn't request this, ignore this email.",
	},
	{
		Title:   "Welcome email",
		From:    "hello@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Welcome to {{ fakeCompany }}, {{ fakeFirstName }}!",
		Text:    "Hi {{ fakeFirstName }},\n\nWelcome aboard! Your account ({{ fakeUsername }}) is ready.\n\n{{ fakeSentence }}",
		HTML:    "<h1>Welcome, {{ fakeFirstName }}!</h1><p>Your account (<b>{{ fakeUsername }}</b>) is ready.</p><p>{{ fakeSentence }}</p>",
	},
	{
		Title:   "Invoice",
		From:    "billing@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Invoice from {{ fakeCompany }}",
		Text:    "Invoice {{ index (fakeInvoice) \"invoiceNumber\" }}\nBill to: {{ index (fakeInvoice) \"billTo\" }}\nTotal due: {{ index (fakeInvoice) \"total\" }} USD\nDue date: {{ index (fakeInvoice) \"dueDate\" }}",
	},
	{
		Title:   "Shipping notification",
		From:    "shipping@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Your package has shipped",
		Text:    "Good news, {{ fakeFirstName }} — your package is on its way!\n\nTracking number: {{ ksuid }}\nEstimated delivery: {{ randInt 2 7 }} business days\nShipping to: {{ fakeAddress }}, {{ fakeCity }}, {{ fakeState }} {{ fakeZip }}",
	},
	{
		Title:   "Appointment reminder",
		From:    "appointments@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Reminder: your appointment with {{ fakeCompany }}",
		Text:    "Hi {{ fakeFirstName }},\n\nThis is a reminder of your upcoming appointment.\nReference: {{ objectid }}\nPlease arrive 10 minutes early.",
	},
	{
		Title:   "Newsletter",
		From:    "newsletter@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "{{ fakeSubject }}",
		Text:    "{{ fakeParagraph 2 }}",
		HTML:    "{{ fakeHTMLBody 2 }}",
	},
	{
		Title:   "Security alert",
		From:    "noreply@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "New sign-in from {{ fakeCity }}, {{ fakeCountry }}",
		Text:    "We noticed a new sign-in to your account.\n\nIP address: {{ fakeIPv4 }}\nDevice: {{ fakeUserAgent }}\nTime: {{ rfc2822Date }}\n\nIf this wasn't you, reset your password immediately.",
	},
	{
		Title:   "Support ticket update",
		From:    "support@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "[Ticket {{ nanoid 8 }}] Update on your request",
		Text:    "Hi {{ fakeFirstName }},\n\nYour support ticket has been updated by {{ fakeName }}.\n\n{{ fakeParagraph 1 }}\n\n— {{ fakeCompany }} Support",
	},
	{
		Title:   "Subscription renewal",
		From:    "billing@{{ fakeDomain }}",
		To:      "{{ fakeEmail }}",
		Subject: "Your subscription renews soon",
		Text:    "Hi {{ fakeFirstName }},\n\nYour subscription will renew on {{ rfc2822Date }} for {{ index (fakeTransaction) \"amount\" }} USD.\nManage your subscription any time at {{ fakeURL }}.",
	},
}

// Example implements the "example" builtin: generates a templated .eml or
// JSON message-spec file from one of 10 canned scenarios (picked randomly,
// or via --index), for a user to inspect, tweak with "edit -f <path>", and
// send with "send --template <path>" (eml) or "send --json <path>" (json).
// Every string field is left UNRENDERED — the point is to hand the user a
// realistic starting point that still contains live template functions
// they can see and change, not a one-off rendered sample.
type Example struct{}

func (Example) Name() string      { return "example" }
func (Example) Aliases() []string { return nil }
func (Example) Short() string     { return "Generate a templated example .eml/.json to edit and send" }

func (Example) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("example", pflag.ContinueOnError)
	fs.String("format", "eml", "output format: eml|json")
	fs.Int("index", 0, "pick a specific canned example (1-based; default: random)")
	fs.Bool("list", false, "list the canned examples instead of generating one")
	fs.StringP("out", "o", "", "output path (default: a generated path under the session's temp dir)")
	return fs
}

func (b Example) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	fs.SetOutput(s.Out)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if list, _ := fs.GetBool("list"); list {
		for i, ex := range exampleTemplates {
			fmt.Fprintf(s.Out, "%2d  %s\n", i+1, ex.Title)
		}
		return nil
	}

	format, _ := fs.GetString("format")
	if format != "eml" && format != "json" {
		return fmt.Errorf("example: --format must be eml or json, got %q", format)
	}

	idx, _ := fs.GetInt("index")
	if idx != 0 {
		if idx < 1 || idx > len(exampleTemplates) {
			return fmt.Errorf("example: --index out of range (1-%d)", len(exampleTemplates))
		}
		idx--
	} else {
		idx = s.Tmpl.Intn(len(exampleTemplates))
	}
	ex := exampleTemplates[idx]

	var content string
	ext := format
	switch format {
	case "json":
		spec := struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			Text    string   `json:"text,omitempty"`
			HTML    string   `json:"html,omitempty"`
		}{From: ex.From, To: []string{ex.To}, Subject: ex.Subject, Text: ex.Text, HTML: ex.HTML}
		raw, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return err
		}
		content = string(raw) + "\n"
	default: // eml
		contentType := "text/plain; charset=utf-8"
		body := ex.Text
		if ex.HTML != "" {
			contentType = "text/html; charset=utf-8"
			body = ex.HTML
		}
		content = fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\nContent-Type: %s\n\n%s\n",
			ex.From, ex.To, ex.Subject, contentType, body)
		ext = "eml"
	}

	out, _ := fs.GetString("out")
	if out == "" {
		out = filepath.Join(s.Tmpl.TempDir(), fmt.Sprintf("example-%d.%s", idx+1, ext))
	}
	if err := os.WriteFile(out, []byte(content), 0o600); err != nil {
		return err
	}

	// Both send --template and send --json derive the envelope from the
	// file's own (rendered) From/To content — --template by parsing the
	// rendered message's headers (same rule as --eml), --json from the
	// spec's own "from"/"to" fields — so neither needs extra flags.
	sendHint := "send --template " + out
	if format == "json" {
		sendHint = "send --json " + out
	}
	fmt.Fprintf(s.Out, "%s (%q) written to %s\n", format, ex.Title, out)
	fmt.Fprintf(s.Out, "edit:  edit -f %s\n", out)
	fmt.Fprintf(s.Out, "send:  %s\n", sendHint)
	return nil
}
