package webauth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpsertAddsNewUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")

	if err := Upsert(path, "alice", "s3cret"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if !Verify(path, "alice", "s3cret") {
		t.Fatal("Verify: expected correct password to succeed")
	}
}

func TestUpsertUpdatesExistingUserNoDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")

	if err := Upsert(path, "alice", "first"); err != nil {
		t.Fatalf("Upsert (1st): %v", err)
	}
	if err := Upsert(path, "alice", "second"); err != nil {
		t.Fatalf("Upsert (2nd): %v", err)
	}

	entries, err := parseFile(path)
	if err != nil {
		t.Fatalf("parseFile: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if Verify(path, "alice", "first") {
		t.Fatal("old password should no longer verify")
	}
	if !Verify(path, "alice", "second") {
		t.Fatal("new password should verify")
	}
}

func TestRemoveDeletesUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")

	if err := Upsert(path, "alice", "pw1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := Upsert(path, "bob", "pw2"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := Remove(path, "alice"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if Verify(path, "alice", "pw1") {
		t.Fatal("removed user should not verify")
	}
	if !Verify(path, "bob", "pw2") {
		t.Fatal("remaining user should still verify")
	}
}

func TestRemoveMissingUserErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")
	if err := Upsert(path, "alice", "pw1"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := Remove(path, "nobody"); err == nil {
		t.Fatal("expected error removing nonexistent user")
	}
}

func TestRemoveMissingFileErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.htpasswd")
	if err := Remove(path, "alice"); err == nil {
		t.Fatal("expected error removing from nonexistent file")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")
	if err := Upsert(path, "alice", "correct"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if Verify(path, "alice", "wrong") {
		t.Fatal("wrong password should not verify")
	}
}

func TestVerifyUnknownUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")
	if err := Upsert(path, "alice", "correct"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if Verify(path, "nobody", "anything") {
		t.Fatal("unknown user should not verify")
	}
}

func TestValidateFileRejectsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.htpasswd")
	if err := ValidateFile(path); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidateFileRejectsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.htpasswd")
	if err := os.WriteFile(path, []byte("# just a comment\n\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateFile(path); err == nil {
		t.Fatal("expected error for empty/no-valid-lines file")
	}
}

func TestValidateFileAcceptsValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webauth.htpasswd")
	if err := Upsert(path, "alice", "pw"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := ValidateFile(path); err != nil {
		t.Fatalf("ValidateFile: unexpected error: %v", err)
	}
}
