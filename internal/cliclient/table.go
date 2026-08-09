package cliclient

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
)

const maxSubjectLen = 40

// RenderTable writes msgs as an aligned table to w (SPEC.md §7.3: "genuinely
// readable in a terminal").
func RenderTable(w io.Writer, msgs []MessageSummary) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tFROM\tTO\tSUBJECT\tSIZE\tATTACHMENTS\tRECEIVED")
	for _, m := range msgs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			m.ID, m.From, formatRecipients(m.To), truncate(m.Subject, maxSubjectLen),
			m.SizeBytes, m.AttachmentCount, m.ReceivedAt)
	}
	tw.Flush()
}

// RenderTemplate executes tmplText (a text/template body, e.g.
// "{{.ID}}\t{{.From}}") once per message in msgs, docker-CLI-style
// (`docker ps --format '{{.ID}}: {{.Names}}'`), writing a trailing newline
// after each execution. Field names match MessageSummary's exported Go
// fields (.ID, .From, .To, .Cc, .Subject, .SizeBytes, .HasAttachments,
// .AttachmentCount, .ReceivedAt, .ParseWarning).
func RenderTemplate(w io.Writer, msgs []MessageSummary, tmplText string) error {
	tmpl, err := template.New("list").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("parsing --format template: %w", err)
	}
	for _, m := range msgs {
		if err := tmpl.Execute(w, m); err != nil {
			return fmt.Errorf("executing --format template: %w", err)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func formatRecipients(to []string) string {
	if len(to) == 0 {
		return ""
	}
	if len(to) == 1 {
		return to[0]
	}
	return fmt.Sprintf("%s, +%d more", to[0], len(to)-1)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

// RenderDetail writes a key/value view of a message detail to w.
func RenderDetail(w io.Writer, m *MessageDetail) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", m.ID)
	fmt.Fprintf(tw, "From:\t%s\n", m.From)
	fmt.Fprintf(tw, "To:\t%s\n", strings.Join(m.To, ", "))
	if len(m.Cc) > 0 {
		fmt.Fprintf(tw, "Cc:\t%s\n", strings.Join(m.Cc, ", "))
	}
	fmt.Fprintf(tw, "Subject:\t%s\n", m.Subject)
	fmt.Fprintf(tw, "Received:\t%s\n", m.ReceivedAt)
	fmt.Fprintf(tw, "Size:\t%d bytes\n", m.SizeBytes)
	fmt.Fprintf(tw, "Attachments:\t%d\n", m.AttachmentCount)
	if m.ParseWarning {
		fmt.Fprintf(tw, "Parse Warning:\tyes\n")
	}
	tw.Flush()

	if m.TextBody != "" {
		fmt.Fprintf(w, "\n--- Text Body ---\n%s\n", m.TextBody)
	}
	if m.HTMLBody != "" {
		fmt.Fprintf(w, "\n--- HTML Body ---\n%s\n", m.HTMLBody)
	}
}
