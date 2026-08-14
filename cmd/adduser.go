package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/0funct0ry/maelsink/internal/webauth"
)

var (
	flagAuthAddUserWebAuthFile   string
	flagAuthAddUserPassword      string
	flagAuthAddUserPasswordStdin bool
)

// defaultWebAuthFile is the default --web-auth-file path used by the auth
// subcommands when the flag is omitted: a maelsink.htpasswd file in the
// current directory, matching the standard htpasswd file extension.
const defaultWebAuthFile = "maelsink.htpasswd"

// adduserCmd represents the adduser command
var adduserCmd = &cobra.Command{
	Use:   "adduser [username]",
	Short: "Add or update a Web UI Basic Auth user",
	Long: `Adds a new user to the --web-auth-file htpasswd-style credential file, or
updates an existing user's password in place. Works standalone against just
the file — no running maelsink server is required.

Fully interactive (prompts for both username and password, password masked,
no shell history/ps exposure), writing to ./maelsink.htpasswd by default:

  maelsink auth adduser

Interactive password only (username given as a positional arg):

  maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd

Non-interactive, for Docker/CI (preferred: --password-stdin):

  echo "$PASSWORD" | maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin

  docker exec my-maelsink maelsink auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin <<< "$PASSWORD"

  docker run --rm -v $(pwd)/webauth.htpasswd:/data/webauth.htpasswd maelsink \
    auth adduser bob --web-auth-file /data/webauth.htpasswd --password-stdin <<< "$PASSWORD"

--password takes the value directly as a flag; it is provided for
convenience but is visible in shell history and "ps" output on most
systems, so --password-stdin is the recommended non-interactive path.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAuthAddUser,
}

func init() {
	authCmd.AddCommand(adduserCmd)

	adduserCmd.Flags().StringVarP(&flagAuthAddUserWebAuthFile, "web-auth-file", "L", defaultWebAuthFile, "path to the htpasswd-style basic-auth file")
	adduserCmd.Flags().StringVar(&flagAuthAddUserPassword, "password", "", "password value (visible in shell history/ps — prefer --password-stdin)")
	adduserCmd.Flags().BoolVar(&flagAuthAddUserPasswordStdin, "password-stdin", false, "read the password from stdin (recommended for scripted/Docker use)")
}

func runAuthAddUser(cmd *cobra.Command, args []string) error {
	username, err := resolveUsername(cmd, args)
	if err != nil {
		return err
	}

	password, err := resolveAddUserPassword(cmd)
	if err != nil {
		return err
	}
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}

	if err := webauth.Upsert(flagAuthAddUserWebAuthFile, username, password); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "user %q added to %s\n", username, flagAuthAddUserWebAuthFile)
	return nil
}

// resolveUsername returns the username from the positional arg if given,
// otherwise prompts for it interactively (plain text — usernames aren't
// secret, unlike passwords).
func resolveUsername(cmd *cobra.Command, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	fmt.Fprint(cmd.OutOrStdout(), "Username: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("reading username: %w", err)
	}
	username := strings.TrimSpace(line)
	if username == "" {
		return "", fmt.Errorf("username must not be empty")
	}
	return username, nil
}

// resolveAddUserPassword resolves the password from --password,
// --password-stdin, or an interactive masked double-prompt, in that order
// of precedence.
func resolveAddUserPassword(cmd *cobra.Command) (string, error) {
	if flagAuthAddUserPassword != "" {
		return flagAuthAddUserPassword, nil
	}
	if flagAuthAddUserPasswordStdin {
		// If stdin is an actual terminal (the user typed --password-stdin but
		// didn't pipe anything in), read it masked via term.ReadPassword so the
		// password never lands in the terminal's visible output/scrollback —
		// a plain bufio read would otherwise echo it verbatim as the user
		// types. When stdin is genuinely piped/redirected (the documented,
		// intended use), there's no terminal to mask and term.ReadPassword
		// would fail, so fall back to a plain line read.
		if isatty.IsTerminal(os.Stdin.Fd()) {
			line, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(cmd.OutOrStdout())
			if err != nil {
				return "", fmt.Errorf("reading password from stdin: %w", err)
			}
			return string(line), nil
		}

		reader := bufio.NewReader(cmd.InOrStdin())
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("reading password from stdin: %w", err)
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	fmt.Fprint(cmd.OutOrStdout(), "Password: ")
	first, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.OutOrStdout())
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), "Confirm password: ")
	second, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.OutOrStdout())
	if err != nil {
		return "", fmt.Errorf("reading password confirmation: %w", err)
	}

	if string(first) != string(second) {
		return "", fmt.Errorf("passwords do not match")
	}
	return string(first), nil
}
