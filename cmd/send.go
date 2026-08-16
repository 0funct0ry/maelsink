package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
)

var (
	sendTo                    []string
	sendCc                    []string
	sendBcc                   []string
	sendFrom                  string
	sendSubject               string
	sendText                  string
	sendHTML                  string
	sendAttach                []string
	sendRaw                   bool
	sendFile                  string
	sendSMTPHost              string
	sendSMTPPort              int
	sendAuthUser              string
	sendAuthPass              string
	sendSMTPTLS               string
	sendTLSInsecureSkipVerify bool
)

// sendCmd is a sendmail-equivalent SMTP client (SPEC.md §7.2).
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Compose and send a test message to a maelsink instance",
	Long:  `A sendmail-equivalent SMTP client for scripting/CI: send via flags, a raw RFC 5322 message on stdin (--raw), or a JSON message spec (--file).`,
	RunE:  runSend,
}

func init() {
	rootCmd.AddCommand(sendCmd)

	d := config.Defaults()
	sendCmd.Flags().StringArrayVar(&sendTo, "to", nil, "recipient address (repeatable)")
	sendCmd.Flags().StringArrayVar(&sendCc, "cc", nil, "cc address (repeatable)")
	sendCmd.Flags().StringArrayVar(&sendBcc, "bcc", nil, "bcc address (repeatable)")
	sendCmd.Flags().StringVar(&sendFrom, "from", "", "from address")
	sendCmd.Flags().StringVar(&sendSubject, "subject", "", "message subject")
	sendCmd.Flags().StringVar(&sendText, "text", "", "plain text body")
	sendCmd.Flags().StringVar(&sendHTML, "html", "", "HTML body")
	sendCmd.Flags().StringArrayVar(&sendAttach, "attach", nil, "path to a file to attach (repeatable)")
	sendCmd.Flags().BoolVar(&sendRaw, "raw", false, "read a full RFC 5322 message from stdin and send it verbatim")
	sendCmd.Flags().StringVar(&sendFile, "file", "", "path to a JSON message spec to send")
	sendCmd.Flags().StringVar(&sendSMTPHost, "smtp-host", d.SMTP.Host, "SMTP server host")
	sendCmd.Flags().IntVar(&sendSMTPPort, "smtp-port", d.SMTP.Port, "SMTP server port")
	sendCmd.Flags().StringVar(&sendAuthUser, "auth-user", "", "SMTP AUTH username")
	sendCmd.Flags().StringVar(&sendAuthPass, "auth-pass", "", "SMTP AUTH password")
	sendCmd.Flags().StringVar(&sendSMTPTLS, "smtp-tls", "none", "transport security: none|starttls|implicit")
	sendCmd.Flags().BoolVar(&sendTLSInsecureSkipVerify, "smtp-tls-insecure-skip-verify", false, "accept a self-signed/dev SMTP TLS certificate without verification (local/CI use only)")
}

func runSend(cmd *cobra.Command, args []string) error {
	addr := fmt.Sprintf("%s:%d", sendSMTPHost, sendSMTPPort)

	tlsMode, err := cliclient.ParseTLSMode(sendSMTPTLS)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	tlsOpts := cliclient.TLSOptions{Mode: tlsMode, InsecureSkipVerify: sendTLSInsecureSkipVerify}

	var auth *cliclient.Auth
	if sendAuthUser != "" {
		auth = &cliclient.Auth{Username: sendAuthUser, Password: sendAuthPass}
	}

	var (
		from string
		to   []string
		raw  []byte
	)

	switch {
	case sendRaw:
		raw, err = io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("send: reading stdin: %w", err)
		}
		from, to, err = envelopeFromRaw(raw)
		if err != nil {
			return fmt.Errorf("send: %w", err)
		}

	case sendFile != "":
		spec, ferr := loadMessageSpecFile(sendFile)
		if ferr != nil {
			return fmt.Errorf("send: %w", ferr)
		}
		from, to = spec.Envelope()
		raw, err = spec.Build(time.Now())
		if err != nil {
			return fmt.Errorf("send: %w", err)
		}

	default:
		spec := cliclient.MessageSpec{
			From: sendFrom, To: sendTo, Cc: sendCc, Bcc: sendBcc,
			Subject: sendSubject, Text: sendText, HTML: sendHTML,
		}
		for _, path := range sendAttach {
			spec.Attachments = append(spec.Attachments, cliclient.AttachmentSpec{Path: path})
		}
		from, to = spec.Envelope()
		raw, err = spec.Build(time.Now())
		if err != nil {
			return fmt.Errorf("send: %w", err)
		}
	}

	if err := cliclient.SendTLS(cmd.Context(), addr, tlsOpts, auth, from, to, raw); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "message sent")
	return nil
}

func loadMessageSpecFile(path string) (cliclient.MessageSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cliclient.MessageSpec{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var spec cliclient.MessageSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return cliclient.MessageSpec{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return spec, nil
}

// envelopeFromRaw extracts MAIL FROM / RCPT TO for --raw mode by parsing the
// message's own From/To/Cc headers, since raw mode has no explicit --to.
func envelopeFromRaw(raw []byte) (from string, to []string, err error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return "", nil, fmt.Errorf("parsing raw message: %w", err)
	}

	if fromAddrs, err := msg.Header.AddressList("From"); err == nil && len(fromAddrs) > 0 {
		from = fromAddrs[0].Address
	}
	for _, field := range []string{"To", "Cc", "Bcc"} {
		if addrs, err := msg.Header.AddressList(field); err == nil {
			for _, a := range addrs {
				to = append(to, a.Address)
			}
		}
	}
	if from == "" {
		return "", nil, fmt.Errorf("raw message has no From header")
	}
	if len(to) == 0 {
		return "", nil, fmt.Errorf("raw message has no To/Cc/Bcc headers")
	}
	return from, to, nil
}
