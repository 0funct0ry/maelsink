package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
)

// apiClientFlags holds the values of the shared REST-client flags
// (SPEC.md §7.3: --api, --api-key, and --format where applicable). Each
// command that talks to the REST API registers its own instance via
// addAPIClientFlags, since these flags don't apply to serve/send/config.
type apiClientFlags struct {
	api    string
	format string
	apiKey string
}

// addAPIClientFlags registers --api and --api-key on cmd — every REST-client
// command needs these regardless of output shape.
func addAPIClientFlags(cmd *cobra.Command) *apiClientFlags {
	d := config.Defaults()
	f := &apiClientFlags{}
	cmd.Flags().StringVar(&f.api, "api", fmt.Sprintf("http://%s:%d", d.API.Host, d.API.Port), "maelsink REST API base URL")
	cmd.Flags().StringVar(&f.apiKey, "api-key", "", "REST API bearer key (if api.auth.enabled)")
	return f
}

// addFormatFlag registers --format on cmd. Only commands that actually
// render output in more than one shape (list, get) call this — delete,
// clear, and export always produce one fixed kind of output, so --format
// would be a silently-ignored no-op for them.
func addFormatFlag(cmd *cobra.Command, f *apiClientFlags) {
	cmd.Flags().StringVar(&f.format, "format", "table",
		"output format: table|json, or a Go template (e.g. '{{.ID}}: {{.Subject}}')")
}

func (f *apiClientFlags) client() *cliclient.Client {
	return cliclient.NewClient(f.api, f.apiKey)
}

// apiError formats err for stderr, keeping "server unreachable" visibly
// distinct from "API returned an error" (SPEC.md §7.3 DoD).
func apiError(err error) error {
	var httpErr *cliclient.HTTPError
	if errors.As(err, &httpErr) {
		return fmt.Errorf("error: %s", httpErr.Error())
	}
	return fmt.Errorf("error: %s", err.Error())
}
