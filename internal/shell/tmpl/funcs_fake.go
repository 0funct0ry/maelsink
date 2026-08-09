package tmpl

import (
	"fmt"
	"strings"
	"text/template"
)

// fakeFuncMap returns gofakeit-backed template functions, all driven by the
// Engine's seeded Faker instance.
func (e *Engine) fakeFuncMap() template.FuncMap {
	return template.FuncMap{
		"fakeName":      e.faker.Name,
		"fakeFirstName": e.faker.FirstName,
		"fakeLastName":  e.faker.LastName,
		"fakeEmail":     e.fakeEmail,
		"fakeUsername":  e.faker.Username,
		"fakePhone":     e.faker.Phone,
		"fakeAddress":   func() string { return e.faker.Address().Address },
		"fakeStreet":    e.faker.Street,
		"fakeCity":      e.faker.City,
		"fakeState":     e.faker.State,
		"fakeZip":       e.faker.Zip,
		"fakeCountry":   e.faker.Country,
		"fakeDomain":    e.faker.DomainName,
		"fakeURL":       e.faker.URL,
		"fakeIPv4":      e.faker.IPv4Address,
		"fakeIPv6":      e.faker.IPv6Address,
		"fakeMAC":       e.faker.MacAddress,
		"fakeUserAgent": e.faker.UserAgent,
		"fakeCompany":   e.faker.Company,
		"fakeJobTitle":  e.faker.JobTitle,
		"fakeWord":      e.faker.Word,
		"fakeSentence":  func() string { return e.faker.Sentence(10) },
		"fakeParagraph": e.fakeParagraph,
		"fakeSubject":   func() string { return e.faker.Sentence(6) },
		"fakeHTMLBody":  e.fakeHTMLBody,
		"fakeTextBody":  e.fakeTextBody,
		"regex":         e.faker.Regex,
	}
}

// fakeEmail generates a fake email, optionally overriding the domain when
// domain is provided.
func (e *Engine) fakeEmail(domain ...string) string {
	email := e.faker.Email()
	if len(domain) == 0 || domain[0] == "" {
		return email
	}
	parts := strings.SplitN(email, "@", 2)
	return parts[0] + "@" + domain[0]
}

// fakeParagraph generates n paragraphs of fake text (default 1), separated
// by blank lines.
func (e *Engine) fakeParagraph(n ...int) string {
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

// fakeHTMLBody generates n paragraphs (default 3) of fake text wrapped in
// <p> tags.
func (e *Engine) fakeHTMLBody(n ...int) string {
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

// fakeTextBody generates n paragraphs (default 3) of fake plain text.
func (e *Engine) fakeTextBody(n ...int) string {
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
