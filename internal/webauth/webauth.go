// Package webauth implements Apache-htpasswd-compatible Basic Auth
// credential storage for the Web UI's login wall (SPEC.md §5.4, M8.8).
// Files are a flat list of "username:bcrypt-hash" lines — no RBAC, no
// sessions server-side, interoperable with the standard htpasswd CLI.
package webauth

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// dummyHash is compared against on an unknown username so a failed lookup
// takes a comparable amount of time to a wrong-password match, avoiding
// trivial username enumeration via response timing.
var dummyHash = []byte("$2a$10$CwTycUXWue0Thq9StjUM0uJ8u5Jh8g3wG0v0OwjO3z5eq5g5tE5PW")

// entry is one parsed "username:bcrypt-hash" line.
type entry struct {
	username string
	hash     string
}

// parseFile reads path and returns its entries, ignoring blank and
// '#'-prefixed lines. A missing file returns an empty slice, not an error,
// so Upsert can create a file from scratch.
func parseFile(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		user, hash, ok := strings.Cut(line, ":")
		if !ok || user == "" || hash == "" {
			continue
		}
		entries = append(entries, entry{username: user, hash: hash})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// writeFile atomically replaces path's contents with entries: write to a
// temp file in the same directory, then rename into place, so concurrent
// readers (a running `serve`) never observe a partially-written file.
func writeFile(path string, entries []entry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".webauth-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once renamed

	w := bufio.NewWriter(tmp)
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s:%s\n", e.username, e.hash); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// Verify reports whether username/password matches an entry in the file at
// path. It runs a bcrypt comparison even on an unknown username (against a
// fixed dummy hash) so the response time doesn't leak which usernames
// exist.
func Verify(path, username, password string) bool {
	entries, err := parseFile(path)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if e.username == username {
			return bcrypt.CompareHashAndPassword([]byte(e.hash), []byte(password)) == nil
		}
	}

	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	return false
}

// Upsert adds username to the file at path, or updates its password hash
// in place if the username already exists. Creates the file (and parent
// directory) if it doesn't exist yet.
func Upsert(path, username, password string) error {
	if username == "" {
		return fmt.Errorf("webauth: username must not be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("webauth: hashing password: %w", err)
	}

	entries, err := parseFile(path)
	if err != nil {
		return fmt.Errorf("webauth: reading %q: %w", path, err)
	}

	replaced := false
	for i, e := range entries {
		if e.username == username {
			entries[i].hash = string(hash)
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry{username: username, hash: string(hash)})
	}

	if err := writeFile(path, entries); err != nil {
		return fmt.Errorf("webauth: writing %q: %w", path, err)
	}
	return nil
}

// Remove deletes username's entry from the file at path. Returns an error
// if the file doesn't exist or contains no matching username.
func Remove(path, username string) error {
	entries, err := parseFile(path)
	if err != nil {
		return fmt.Errorf("webauth: reading %q: %w", path, err)
	}
	if entries == nil {
		return fmt.Errorf("webauth: %q does not exist", path)
	}

	out := entries[:0]
	found := false
	for _, e := range entries {
		if e.username == username {
			found = true
			continue
		}
		out = append(out, e)
	}
	if !found {
		return fmt.Errorf("webauth: username %q not found in %q", username, path)
	}

	if err := writeFile(path, out); err != nil {
		return fmt.Errorf("webauth: writing %q: %w", path, err)
	}
	return nil
}

// ValidateFile fails fast if path is missing, unreadable, empty, or
// contains no valid username:bcrypt-hash lines — used at `serve` startup
// so a misconfigured --web-auth-file never silently disables auth.
func ValidateFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("web-auth-file %q: %w", path, err)
	}
	entries, err := parseFile(path)
	if err != nil {
		return fmt.Errorf("web-auth-file %q: %w", path, err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("web-auth-file %q: contains no valid username:bcrypt-hash lines", path)
	}
	return nil
}
