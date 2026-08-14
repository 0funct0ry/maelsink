package cmd

import (
	"github.com/spf13/cobra"
)

// authCmd groups credential-management subcommands for the Web UI's
// htpasswd-style Basic Auth login wall (--web-auth-file, M8.8, SPEC.md
// §5.4). It manages only that file — it has no effect on the REST API's
// separate api.auth bearer-key gate.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Web UI Basic Auth credentials",
	Long: `Manage the htpasswd-style credential file used by the Web UI's
Basic Auth login wall (--web-auth-file). This is independent of the REST
API's bearer-key auth (--api-auth-enabled/--api-auth-api-key) and has no
effect on it.`,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
