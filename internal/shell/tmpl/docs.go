package tmpl

// FuncDoc documents one template function for the "functions" builtin
// (internal/shell/builtin).
type FuncDoc struct {
	Signature   string
	Description string
}

// Docs returns documentation for every function in Engine.FuncMap(): both
// maelsink's own additions (below) and every sprig utility function
// (sprigDocs, in docs_sprig.go) — so "functions"/"functions <name>" never
// falls back to a generic "help unavailable" message. Keys match
// FuncMap()'s names exactly; where maelsink overrides a sprig function of
// the same name (randInt, randBytes, pick, shuffle, ext — see
// docs_sprig.go's own comment), the entry here documents maelsink's
// actual, overriding behavior, and sprigDocs() omits that key entirely to
// avoid describing code that never runs.
func Docs() map[string]FuncDoc {
	d := maelsinkDocs()
	for name, doc := range sprigDocs() {
		d[name] = doc
	}
	return d
}

// maelsinkDocs documents every function maelsink itself contributes
// (identifiers, fakers, binary generators, mime/fs helpers, ANSI colors).
func maelsinkDocs() map[string]FuncDoc {
	return map[string]FuncDoc{
		// Identifiers
		"uuid":      {"uuid", "Random UUIDv4, seeded from the session's PRNG."},
		"uuidv7":    {"uuidv7", "Time-ordered UUIDv7. NOT reproducible under a fixed --seed (see engine_test.go): its rand_a field derives from a package-level wall-clock counter, not the seeded entropy source."},
		"ulid":      {"ulid", "Lexicographically-sortable ULID. NOT reproducible under a fixed --seed: its 48-bit timestamp component is real wall-clock time by design (a ULID's whole purpose is sorting by real creation time)."},
		"nanoid":    {"nanoid [size]", "URL-safe random ID, default length 21."},
		"objectid":  {"objectid", "24-hex-char MongoDB-style ObjectID (4-byte timestamp + 5 random + 3-byte counter)."},
		"ksuid":     {"ksuid", "27-char base62 K-Sortable ID (4-byte timestamp + 16 random bytes)."},
		"messageID": {"messageID [domain]", "RFC 5322 Message-Id value, e.g. <hex@domain>. domain defaults to \"maelsink.local\"."},

		// Pattern / primitive generators
		"regex":      {"regex pattern", "A random string matching the given RE2 pattern (gofakeit)."},
		"randInt":    {"randInt min max", "Random int in [min,max]."},
		"randFloat":  {"randFloat min max [decimals]", "Random float64 in [min,max], optionally rounded to `decimals` places."},
		"randBool":   {"randBool", "Random true/false."},
		"randString": {"randString n [charset]", "Random string of length n from charset (default alphanumeric)."},
		"randBytes":  {"randBytes n", "n random bytes."},
		"pick":       {"pick a b c ...", "Returns one of its arguments at random."},
		"shuffle":    {"shuffle list", "Returns list with its elements in random order."},
		"weighted":   {"weighted dict", "Picks a key from dict at random, weighted by its numeric value."},

		// Person / contact / internet fakes
		"fakeName":      {"fakeName", "Random full name."},
		"fakeFirstName": {"fakeFirstName", "Random first name."},
		"fakeLastName":  {"fakeLastName", "Random last name."},
		"fakeEmail":     {"fakeEmail [domain]", "Random email address, optionally on the given domain."},
		"fakeUsername":  {"fakeUsername", "Random username."},
		"fakePhone":     {"fakePhone", "Random phone number."},
		"fakeAddress":   {"fakeAddress", "Random street address."},
		"fakeStreet":    {"fakeStreet", "Random street name."},
		"fakeCity":      {"fakeCity", "Random city name."},
		"fakeState":     {"fakeState", "Random US state."},
		"fakeZip":       {"fakeZip", "Random ZIP/postal code."},
		"fakeCountry":   {"fakeCountry", "Random country name."},
		"fakeDomain":    {"fakeDomain", "Random domain name."},
		"fakeURL":       {"fakeURL", "Random URL."},
		"fakeIPv4":      {"fakeIPv4", "Random IPv4 address."},
		"fakeIPv6":      {"fakeIPv6", "Random IPv6 address."},
		"fakeMAC":       {"fakeMAC", "Random MAC address."},
		"fakeUserAgent": {"fakeUserAgent", "Random browser User-Agent string."},
		"fakeCompany":   {"fakeCompany", "Random company name."},
		"fakeJobTitle":  {"fakeJobTitle", "Random job title."},

		// Text
		"fakeWord":      {"fakeWord", "A single random word."},
		"fakeSentence":  {"fakeSentence", "A random ~10-word sentence."},
		"fakeParagraph": {"fakeParagraph [n]", "n random paragraphs (default 1), separated by blank lines."},
		"fakeSubject":   {"fakeSubject", "A random ~6-word email-subject-like sentence."},
		"fakeHTMLBody":  {"fakeHTMLBody [paragraphs]", "Random HTML body: `paragraphs` <p> blocks (default 1)."},
		"fakeTextBody":  {"fakeTextBody [paragraphs]", "Random plain-text body: `paragraphs` blocks (default 1)."},

		// Domain packs (each returns a dict)
		"fakeCreditCard":  {"fakeCreditCard [type]", "dict{number,type,cvv,exp} — Luhn-valid test card."},
		"fakeTransaction": {"fakeTransaction", "dict{id,amount,currency,status,timestamp,merchant}."},
		"fakeProduct":     {"fakeProduct", "dict{sku,name,category,price,currency,qty}."},
		"fakeOrder":       {"fakeOrder [items]", "dict{id,items,total,currency,created} — `items` fakeProduct line items (default 1-3)."},
		"fakeInvoice":     {"fakeInvoice [items]", "dict{invoiceNumber,order,subtotal,tax,total,currency,issued,dueDate,billTo}."},

		// Binary / attachment generators (each returns a filesystem path)
		"fakePDF":    {"fakePDF [pages]", "Generates an N-page PDF (default 1) and returns its path."},
		"fakeXLSX":   {"fakeXLSX [rows] [cols]", "Generates a workbook (default 10x5) and returns its path."},
		"fakeDOCX":   {"fakeDOCX [paragraphs]", "Generates a minimal .docx (default 1 paragraph) and returns its path."},
		"fakeCSV":    {"fakeCSV [rows] [cols]", "Generates a CSV file and returns its path."},
		"fakePNG":    {"fakePNG [w] [h]", "Generates a PNG image and returns its path."},
		"fakeJPEG":   {"fakeJPEG [w] [h]", "Generates a JPEG image and returns its path."},
		"fakeGIF":    {"fakeGIF [w] [h]", "Generates a GIF image and returns its path."},
		"fakeZIP":    {"fakeZIP [files...]", "Bundles the given paths (or one generated file) into a .zip and returns its path."},
		"fakeBinary": {"fakeBinary size", "Writes `size` (e.g. \"2MB\", \"512KB\", or a plain byte count) of pseudo-random bytes and returns the path."},
		"fileOf":     {"fileOf path", "Validates path exists and returns it unchanged (passthrough for --attach chains)."},
		"attach":     {"attach path...", "Joins multiple paths with \"::\" for `send --attach`'s chaining convention."},

		// MIME / date helpers
		"quotedPrintable": {"quotedPrintable s", "Quoted-printable encodes s (RFC 2045)."},
		"mimeWord":        {"mimeWord s", "RFC 2047 encoded-word (UTF-8 Q-encoding) for non-ASCII header values."},
		"rfc2822Date":     {"rfc2822Date [time]", "Formats the given time (default now) as RFC 1123Z, for email Date headers."},

		// Filesystem helpers
		"readFile":    {"readFile path", "Returns the file's contents as a string."},
		"readFileB64": {"readFileB64 path", "Returns the file's contents, base64-encoded."},
		"glob":        {"glob pattern", "Returns matching file paths."},
		"basename":    {"basename path", "Returns the final path element."},
		"dirname":     {"dirname path", "Returns all but the final path element."},
		"ext":         {"ext path", "Returns the file extension, including the leading dot."},

		// ANSI color/style helpers (SPEC.md §7.5.10 prompt colors)
		"ansi":        {"ansi code text", "Wraps text in the given SGR escape code(s) (e.g. \"1;32\"), resetting after."},
		"ansiReset":   {"ansiReset", "The bare ANSI reset sequence."},
		"ansiBold":    {"ansiBold text", "Wraps text in bold, resetting after."},
		"ansiDim":     {"ansiDim text", "Wraps text in dim, resetting after."},
		"ansiRed":     {"ansiRed text", "Wraps text in red, resetting after."},
		"ansiGreen":   {"ansiGreen text", "Wraps text in green, resetting after."},
		"ansiYellow":  {"ansiYellow text", "Wraps text in yellow, resetting after."},
		"ansiBlue":    {"ansiBlue text", "Wraps text in blue, resetting after."},
		"ansiMagenta": {"ansiMagenta text", "Wraps text in magenta, resetting after."},
		"ansiCyan":    {"ansiCyan text", "Wraps text in cyan, resetting after."},
		"ansiWhite":   {"ansiWhite text", "Wraps text in white, resetting after."},
	}
}
