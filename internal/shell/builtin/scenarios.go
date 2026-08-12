package builtin

// exampleTemplate is one canned scenario. Subject/From/To/Text/HTML are
// UNRENDERED — they contain {{ }} template-function calls verbatim, so the
// file "example" writes out is itself editable input for `send --json`
// (which templates from/to/subject/text/html per SPEC.md §7.5.5) or `send
// --template` (which treats the whole rendered .eml as the raw message).
// Also used by "randmsg --scenario <name>" (SPEC.md §7.6.2) as a
// subject/body seed, matched by a lowercased, space-stripped Title.
type exampleTemplate struct {
	Title   string
	From    string
	To      string
	Subject string
	Text    string
	HTML    string // may be empty
}

// exampleTemplates are the 10 canned scenarios shared by the "example" and
// "randmsg" builtins. Each exercises a handful of real template functions
// (SPEC.md §7.5.7) so a user gets a realistic, immediately-runnable starting
// point rather than a blank page, and can see functions in context before
// changing them.
var exampleTemplates = []exampleTemplate{
	{
		Title:   "Order confirmation",
		From:    "orders@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Your order is confirmed",
		// fOrder returns a dict; text/template can't dot-chain a field
		// straight off a function call (`fOrder.id` is invalid), so
		// dict fields are pulled out with `index (fOrder) "key"`.
		Text: "Hi {{ fName }},\n\nThanks for your order! Order ID: {{ index (fOrder) \"id\" }}\nTotal: {{ index (fOrder) \"total\" }} USD\nPlaced on: {{ rfc2822Date }}\n\n— {{ fCompany }}",
	},
	{
		Title:   "Password reset",
		From:    "security@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Reset your password",
		Text:    "We received a request to reset your password.\n\nReset code: {{ randString 8 \"ABCDEFGHJKLMNPQRSTUVWXYZ23456789\" }}\nThis code expires in 15 minutes. If you didn't request this, ignore this email.",
	},
	{
		Title:   "Welcome email",
		From:    "hello@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Welcome to {{ fCompany }}, {{ fFirstName }}!",
		Text:    "Hi {{ fFirstName }},\n\nWelcome aboard! Your account ({{ fUsername }}) is ready.\n\n{{ fSentence }}",
		HTML:    "<h1>Welcome, {{ fFirstName }}!</h1><p>Your account (<b>{{ fUsername }}</b>) is ready.</p><p>{{ fSentence }}</p>",
	},
	{
		Title:   "Invoice",
		From:    "billing@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Invoice from {{ fCompany }}",
		Text:    "Invoice {{ index (fInvoice) \"invoiceNumber\" }}\nBill to: {{ index (fInvoice) \"billTo\" }}\nTotal due: {{ index (fInvoice) \"total\" }} USD\nDue date: {{ index (fInvoice) \"dueDate\" }}",
	},
	{
		Title:   "Shipping notification",
		From:    "shipping@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Your package has shipped",
		Text:    "Good news, {{ fFirstName }} — your package is on its way!\n\nTracking number: {{ ksuid }}\nEstimated delivery: {{ randInt 2 7 }} business days\nShipping to: {{ fAddress }}, {{ fCity }}, {{ fState }} {{ fZip }}",
	},
	{
		Title:   "Appointment reminder",
		From:    "appointments@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Reminder: your appointment with {{ fCompany }}",
		Text:    "Hi {{ fFirstName }},\n\nThis is a reminder of your upcoming appointment.\nReference: {{ objectid }}\nPlease arrive 10 minutes early.",
	},
	{
		Title:   "Newsletter",
		From:    "newsletter@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "{{ fSubject }}",
		Text:    "{{ fParagraph 2 }}",
		HTML:    "{{ fHTMLBody 2 }}",
	},
	{
		Title:   "Security alert",
		From:    "noreply@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "New sign-in from {{ fCity }}, {{ fCountry }}",
		Text:    "We noticed a new sign-in to your account.\n\nIP address: {{ fIPv4 }}\nDevice: {{ fUserAgent }}\nTime: {{ rfc2822Date }}\n\nIf this wasn't you, reset your password immediately.",
	},
	{
		Title:   "Support ticket update",
		From:    "support@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "[Ticket {{ nanoid 8 }}] Update on your request",
		Text:    "Hi {{ fFirstName }},\n\nYour support ticket has been updated by {{ fName }}.\n\n{{ fParagraph 1 }}\n\n— {{ fCompany }} Support",
	},
	{
		Title:   "Subscription renewal",
		From:    "billing@{{ fDomain }}",
		To:      "{{ fEmail }}",
		Subject: "Your subscription renews soon",
		Text:    "Hi {{ fFirstName }},\n\nYour subscription will renew on {{ rfc2822Date }} for {{ index (fTransaction) \"amount\" }} USD.\nManage your subscription any time at {{ fURL }}.",
	},
}

// findScenario returns the exampleTemplate whose Title case-insensitively
// matches name (spaces and case ignored, so "invoice" matches "Invoice"),
// or false if no scenario matches.
func findScenario(name string) (exampleTemplate, bool) {
	norm := normalizeScenarioName(name)
	for _, ex := range exampleTemplates {
		if normalizeScenarioName(ex.Title) == norm {
			return ex, true
		}
	}
	return exampleTemplate{}, false
}

func normalizeScenarioName(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		case r == ' ' || r == '-' || r == '_':
			continue
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
