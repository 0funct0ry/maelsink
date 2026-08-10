package cmd

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/api"
	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
)

// newTestAPIServer starts a real internal/api router (M3.0) over an
// in-memory store seeded with fixture messages, so cmd's REST-client
// commands (list/get/delete/clear/export) are tested against the actual
// API contract rather than a hand-rolled fake.
func newTestAPIServer(t *testing.T, seed ...*store.Message) (*httptest.Server, store.MessageStore) {
	t.Helper()
	st := store.NewMemoryStore()
	for _, m := range seed {
		if err := st.Save(t.Context(), m); err != nil {
			t.Fatalf("seed Save: %v", err)
		}
	}
	router := api.New(st, events.NewBus(), slog.New(slog.DiscardHandler), api.Config{})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, st
}

func fixtureMessage(id, from, to, subject string, receivedAt time.Time) *store.Message {
	return &store.Message{
		ID:         id,
		ReceivedAt: receivedAt,
		Size:       42,
		From:       []store.Address{{Address: from}},
		To:         []store.Address{{Address: to}},
		Subject:    subject,
		TextBody:   "body of " + subject,
		RawSource:  []byte("Subject: " + subject + "\r\n\r\nbody of " + subject + "\r\n"),
	}
}

// resetFlags restores every flag on cmd (and its subcommands, recursively)
// to its declared default, since cobra commands here are package-level
// singletons whose flag values (and Changed markers) otherwise leak between
// test cases run against the same rootCmd tree.
func resetFlags(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
	for _, c := range cmd.Commands() {
		resetFlags(c)
	}
}

// resetSendSliceFlags clears send's --to/--cc/--bcc/--attach StringArrayVar
// backing slices directly: pflag's stringArrayValue tracks its own private
// "changed" state independent of pflag.Flag.Changed, so Set(f.DefValue)
// alone appends "[]" instead of truly resetting between test cases.
func resetSendSliceFlags() {
	sendTo, sendCc, sendBcc, sendAttach = nil, nil, nil, nil
}

// execCommand runs rootCmd with args, returning captured stdout/stderr and
// the error from Execute.
func execCommand(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetFlags(rootCmd)
	resetSendSliceFlags()
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetIn(bytes.NewBufferString(stdin))
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetIn(nil)
	})
	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}
