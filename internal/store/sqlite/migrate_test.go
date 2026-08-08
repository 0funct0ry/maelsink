package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRunMigrations_Idempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var before int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&before); err != nil {
		t.Fatalf("counting migrations: %v", err)
	}
	if before == 0 {
		t.Fatalf("expected at least one migration recorded, got 0")
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("second RunMigrations: %v", err)
	}

	var after int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&after); err != nil {
		t.Fatalf("counting migrations: %v", err)
	}
	if after != before {
		t.Fatalf("expected migration count unchanged, got %d -> %d", before, after)
	}
}
