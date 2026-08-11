package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

// hashTagColor deterministically maps a tag name to one of store.TagColors,
// bit-for-bit identical to web/src/lib/tagColor.ts's hashString/tagColor
// (an FNV-1a-ish hash, mod the palette length) so a tag backfilled from
// pre-M8.5 data keeps the same color it already appeared with client-side.
func hashTagColor(name string) string {
	h := uint32(2166136261)
	for i := 0; i < len(name); i++ {
		h ^= uint32(name[i])
		h *= 16777619
	}
	return store.TagColors[int(h)%len(store.TagColors)]
}

// backfillTagsTable inserts one tags row (with a hash-derived color) for
// every distinct tag already present across messages.tags JSON arrays that
// doesn't already have one — run once, after migrations, so upgrading an
// existing pre-M8.5 database gives every in-use tag a persisted, stable
// color instead of leaving the tags table empty.
func backfillTagsTable(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT json_each.value AS tag
		FROM messages, json_each(messages.tags)
		WHERE json_each.value NOT IN (SELECT name FROM tags)
	`)
	if err != nil {
		return fmt.Errorf("sqlite: querying tags to backfill: %w", err)
	}

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("sqlite: scanning tag to backfill: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	now := time.Now().Format(timeLayout)
	for _, name := range names {
		if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO tags (name, color, created_at) VALUES (?, ?, ?)`,
			name, hashTagColor(name), now); err != nil {
			return fmt.Errorf("sqlite: backfilling tag %q: %w", name, err)
		}
	}
	return nil
}
