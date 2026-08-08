// Package version holds build-time provenance information injected via
// -ldflags -X (see the Makefile's LDFLAGS). Never compute these at runtime.
package version

import "runtime"

var (
	Version   = "0.0.0"
	Commit    = "none"
	BuildDate = "unknown"
)

// Info is the JSON-serializable shape returned by `maelsink version --json`
// and the future /api/v1/version endpoint.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Go      string `json:"go"`
}

func Get() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Go:      runtime.Version(),
	}
}
