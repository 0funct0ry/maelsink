package cmd

import (
	"github.com/spf13/cobra"
)

// sendCmd is a sendmail-equivalent SMTP client (SPEC.md §7.2).
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Compose and send a test message to a maelsink instance",
	Long:  `A sendmail-equivalent SMTP client for scripting/CI: send via flags, a raw RFC 5322 message on stdin (--raw), or a JSON message spec (--file).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(cmd, "M4.0")
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
