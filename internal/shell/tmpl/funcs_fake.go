package tmpl

import (
	"fmt"
	"strings"
)

// fakeDocs documents gofakeit-backed template functions, all driven by the
// Engine's seeded Faker instance.
func (e *Engine) fakeDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "regex", Category: CategoryGenerate, Args: "pattern", Returns: "string",
			Description: "A string matching the given RE2 pattern. Accepts a bare, unquoted pattern in template source (e.g. {{ regex [a-z]{2,4} }}) as well as the normal quoted form.", Fn: e.faker.Regex},
		{Name: "fName", Category: CategoryGenerate, Returns: "string",
			Description: "Random full name.", Fn: e.faker.Name},
		{Name: "fFirstName", Category: CategoryGenerate, Returns: "string",
			Description: "Random first name.", Fn: e.faker.FirstName},
		{Name: "fLastName", Category: CategoryGenerate, Returns: "string",
			Description: "Random last name.", Fn: e.faker.LastName},
		{Name: "fUsername", Category: CategoryGenerate, Returns: "string",
			Description: "Random username.", Fn: e.faker.Username},
		{Name: "fPhone", Category: CategoryGenerate, Returns: "string",
			Description: "Random phone number.", Fn: e.faker.Phone},
		{Name: "fAddress", Category: CategoryGenerate, Returns: "string",
			Description: "Random street address.", Fn: func() string { return e.faker.Address().Address }},
		{Name: "fStreet", Category: CategoryGenerate, Returns: "string",
			Description: "Random street name.", Fn: e.faker.Street},
		{Name: "fCity", Category: CategoryGenerate, Returns: "string",
			Description: "Random city name.", Fn: e.faker.City},
		{Name: "fState", Category: CategoryGenerate, Returns: "string",
			Description: "Random US state.", Fn: e.faker.State},
		{Name: "fZip", Category: CategoryGenerate, Returns: "string",
			Description: "Random ZIP/postal code.", Fn: e.faker.Zip},
		{Name: "fCountry", Category: CategoryGenerate, Returns: "string",
			Description: "Random country name.", Fn: e.faker.Country},
		{Name: "fDomain", Category: CategoryGenerate, Returns: "string",
			Description: "Random domain name.", Fn: e.faker.DomainName},
		{Name: "fURL", Category: CategoryGenerate, Returns: "string",
			Description: "Random URL.", Fn: e.faker.URL},
		{Name: "fIPv4", Category: CategoryGenerate, Returns: "string",
			Description: "Random IPv4 address.", Fn: e.faker.IPv4Address},
		{Name: "fIPv6", Category: CategoryGenerate, Returns: "string",
			Description: "Random IPv6 address.", Fn: e.faker.IPv6Address},
		{Name: "fMAC", Category: CategoryGenerate, Returns: "string",
			Description: "Random MAC address.", Fn: e.faker.MacAddress},
		{Name: "fUserAgent", Category: CategoryGenerate, Returns: "string",
			Description: "Random browser User-Agent string.", Fn: e.faker.UserAgent},
		{Name: "fCompany", Category: CategoryGenerate, Returns: "string",
			Description: "Random company name.", Fn: e.faker.Company},
		{Name: "fJobTitle", Category: CategoryGenerate, Returns: "string",
			Description: "Random job title.", Fn: e.faker.JobTitle},
		{Name: "fWord", Category: CategoryGenerate, Returns: "string",
			Description: "A single random word.", Fn: e.faker.Word},
		{Name: "fSentence", Category: CategoryGenerate, Returns: "string",
			Description: "A random ~10-word sentence.", Fn: func() string { return e.faker.Sentence(10) }},
		{Name: "fParagraph", Category: CategoryGenerate, Args: "[n]", Returns: "string",
			Description: "n random paragraphs (default 1), separated by blank lines.", Fn: e.fParagraph},
		{Name: "fSubject", Category: CategoryGenerate, Returns: "string",
			Description: "A random ~6-word email-subject-like sentence.", Fn: func() string { return e.faker.Sentence(6) }},
		{Name: "fHTMLBody", Category: CategoryGenerate, Args: "[paragraphs]", Returns: "string",
			Description: "Random HTML body: `paragraphs` <p> blocks (default 3).", Fn: e.fHTMLBody},
		{Name: "fTextBody", Category: CategoryGenerate, Args: "[paragraphs]", Returns: "string",
			Description: "Random plain-text body: `paragraphs` blocks (default 3).", Fn: e.fTextBody},
	}
}

// fEmail generates a fake email, optionally overriding the domain when
// domain is provided.
func (e *Engine) fEmail(domain ...string) string {
	email := e.faker.Email()
	if len(domain) == 0 || domain[0] == "" {
		return email
	}
	parts := strings.SplitN(email, "@", 2)
	return parts[0] + "@" + domain[0]
}

// fParagraph generates n paragraphs of fake text (default 1), separated
// by blank lines.
func (e *Engine) fParagraph(n ...int) string {
	count := 1
	if len(n) > 0 && n[0] > 0 {
		count = n[0]
	}
	paras := make([]string, count)
	for i := 0; i < count; i++ {
		paras[i] = e.faker.Paragraph(3, 5, 12, " ")
	}
	return strings.Join(paras, "\n\n")
}

// fHTMLBody generates n paragraphs (default 3) of fake text wrapped in
// <p> tags.
func (e *Engine) fHTMLBody(n ...int) string {
	count := 3
	if len(n) > 0 && n[0] > 0 {
		count = n[0]
	}
	var sb strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&sb, "<p>%s</p>\n", e.faker.Paragraph(3, 5, 12, " "))
	}
	return sb.String()
}

// fTextBody generates n paragraphs (default 3) of fake plain text.
func (e *Engine) fTextBody(n ...int) string {
	count := 3
	if len(n) > 0 && n[0] > 0 {
		count = n[0]
	}
	paras := make([]string, count)
	for i := 0; i < count; i++ {
		paras[i] = e.faker.Paragraph(3, 5, 12, " ")
	}
	return strings.Join(paras, "\n\n")
}
