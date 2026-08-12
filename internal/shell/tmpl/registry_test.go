package tmpl

import (
	"sort"
	"testing"
)

// TestRegistry_UnderMVPBudget guards the M8.6 budget: a future PR that
// casually re-adds a dropped function (or grows a category) fails CI
// instead of silently blowing the <100-function MVP scope.
func TestRegistry_UnderMVPBudget(t *testing.T) {
	e, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	if n := len(e.Registry()); n >= 100 {
		t.Errorf("len(Registry()) = %d, want < 100", n)
	}
}

// TestRegistry_EveryEntryDocumented replaces docs_test.go's
// TestDocs_CoversEveryFuncMapEntry: every registered function has a real,
// non-placeholder description.
func TestRegistry_EveryEntryDocumented(t *testing.T) {
	e, err := New(1, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	var undocumented []string
	for _, d := range e.Registry() {
		if d.Description == "" {
			undocumented = append(undocumented, d.Name)
		}
	}
	sort.Strings(undocumented)
	if len(undocumented) > 0 {
		t.Errorf("%d Registry() entries have no description: %v", len(undocumented), undocumented)
	}
}

// TestRegistry_MatchesFuncMap replaces docs_test.go's
// TestDocs_NoStaleEntries: Registry() and FuncMap() must never drift — every
// Registry() name is callable, and FuncMap() has no extra names.
func TestRegistry_MatchesFuncMap(t *testing.T) {
	e, err := New(1, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	fm := e.FuncMap()
	reg := e.Registry()

	if len(fm) != len(reg) {
		t.Errorf("len(FuncMap()) = %d, len(Registry()) = %d, want equal (no duplicate/dropped names)", len(fm), len(reg))
	}
	for _, d := range reg {
		if _, ok := fm[d.Name]; !ok {
			t.Errorf("Registry() entry %q missing from FuncMap()", d.Name)
		}
	}
}

// TestRegistry_ExplicitlyCutNamesAbsent guards against the specific
// crypto/TLS/must*/deprecated/reflection functions M8.6 dropped silently
// creeping back in.
func TestRegistry_ExplicitlyCutNamesAbsent(t *testing.T) {
	e, err := New(1, true) // unsafe=true: only the *explicit* MVP cut list is checked, not env/expandenv/getHostByName
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer e.Close()

	fm := e.FuncMap()
	cut := []string{
		"sha1sum", "sha256sum", "sha512sum", "adler32sum", "bcrypt", "htpasswd", "derivePassword",
		"genPrivateKey", "buildCustomCert", "genCA", "genCAWithKey", "genSelfSignedCert",
		"genSelfSignedCertWithKey", "genSignedCert", "genSignedCertWithKey", "encryptAES", "decryptAES", "uuidv4",
		"mustAppend", "mustPush", "mustPrepend", "mustFirst", "mustRest", "mustInitial", "mustReverse",
		"mustUniq", "mustWithout", "mustHas", "mustCompact", "mustChunk", "mustSlice", "mustMerge",
		"mustMergeOverwrite", "mustDeepCopy", "mustToJson", "mustToPrettyJson", "mustToRawJson", "mustFromJson",
		"mustRegexMatch", "mustRegexFind", "mustRegexFindAll", "mustRegexReplaceAll",
		"mustRegexReplaceAllLiteral", "mustRegexSplit", "mustDateModify", "mustToDate",
		"trimall", "date_in_zone", "date_modify", "must_date_modify",
		"semver", "semverCompare", "urlParse", "urlJoin",
		"deepCopy", "deepEqual", "kindIs", "typeIs", "typeOf", "typeIsLike", "kindOf",
		"dig", "omit", "pluck", "toDecimal", "duration", "durationRound", "unixEpoch", "toDate",
		"htmlDate", "htmlDateInZone", "sortAlpha", "reverse", "rest", "initial", "uniq", "without",
		"compact", "merge", "mergeOverwrite", "seq", "until", "untilStep", "floor", "ceil", "mod",
		"float64", "coalesce", "fail", "readFileB64", "weighted",
	}
	var present []string
	for _, name := range cut {
		if _, ok := fm[name]; ok {
			present = append(present, name)
		}
	}
	if len(present) > 0 {
		t.Errorf("explicitly-cut functions still registered: %v", present)
	}
}
