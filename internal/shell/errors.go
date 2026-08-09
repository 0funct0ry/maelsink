package shell

import (
	"errors"
	"fmt"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

// FormatClientError formats an error returned from an internal/cliclient
// call, keeping "server unreachable" (a transport-level failure) visibly
// distinct from "the API returned an error" (an *cliclient.HTTPError) — the
// same distinction cmd/apiclient_flags.go's apiError() makes for the
// non-shell CLI commands. internal/shell/builtin replicates this here since
// it must not import cmd/.
func FormatClientError(err error) string {
	if err == nil {
		return ""
	}
	var httpErr *cliclient.HTTPError
	if errors.As(err, &httpErr) {
		return fmt.Sprintf("error: %s", httpErr.Error())
	}
	return fmt.Sprintf("error: %s", err.Error())
}

// AmbiguousIDCode is the internal/api error code returned when an <id>
// prefix matches more than one message (SPEC.md §7.5.4 "ID arguments").
const AmbiguousIDCode = "ambiguous_id"

// IsAmbiguousID reports whether err is an *cliclient.HTTPError carrying the
// ambiguous_id code, so builtins can render a readable "supply more
// characters" message instead of the raw API error text.
func IsAmbiguousID(err error) bool {
	var httpErr *cliclient.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code == AmbiguousIDCode
	}
	return false
}
