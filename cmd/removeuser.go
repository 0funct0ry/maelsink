package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/webauth"
)

var flagAuthRemoveUserWebAuthFile string

// removeuserCmd represents the removeuser command
var removeuserCmd = &cobra.Command{
	Use:   "removeuser <username>",
	Short: "Remove a Web UI Basic Auth user",
	Long: `Removes a user's entry from the --web-auth-file htpasswd-style credential
file. Works standalone against just the file — no running maelsink server
is required.

  maelsink auth removeuser bob --web-auth-file /data/webauth.htpasswd

  docker exec my-maelsink maelsink auth removeuser bob --web-auth-file /data/webauth.htpasswd

Errors if the username or the file itself doesn't exist.`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthRemoveUser,
}

func init() {
	authCmd.AddCommand(removeuserCmd)

	removeuserCmd.Flags().StringVarP(&flagAuthRemoveUserWebAuthFile, "web-auth-file", "L", defaultWebAuthFile, "path to the htpasswd-style basic-auth file")
}

func runAuthRemoveUser(cmd *cobra.Command, args []string) error {
	username := args[0]

	if err := webauth.Remove(flagAuthRemoveUserWebAuthFile, username); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "user %q removed from %s\n", username, flagAuthRemoveUserWebAuthFile)
	return nil
}
