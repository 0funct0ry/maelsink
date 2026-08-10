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
