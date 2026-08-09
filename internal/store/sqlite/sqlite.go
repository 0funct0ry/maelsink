// Package sqlite implements store.MessageStore against a CGO-free SQLite
// database (modernc.org/sqlite), per SPEC.md §6. It is the durable backend
// that replaces internal/store.MemoryStore starting at M2.0.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/0funct0ry/maelsink/internal/store"
)

const timeLayout = time.RFC3339Nano

// Open opens (creating if necessary) the SQLite database at path, enables
// WAL mode and foreign keys, and runs every pending migration.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("sqlite: creating db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	db.SetMaxOpenConns(1) // modernc.org/sqlite serializes writes; avoid SQLITE_BUSY churn.

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("sqlite: %s: %w", pragma, err)
		}
	}

	if err := RunMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// Store is a SQLite-backed store.MessageStore.
type Store struct {
	db           *sql.DB
	attachOnDisk bool
	diskPath     string
}

// New returns a Store using db (already migrated, per Open). When
// attachOnDisk is true, attachment and inline-image bytes are written to
// files under diskPath instead of a BLOB column.
func New(db *sql.DB, attachOnDisk bool, diskPath string) *Store {
	return &Store{db: db, attachOnDisk: attachOnDisk, diskPath: diskPath}
}

// Close closes the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

func addrsJSON(addrs []store.Address) (string, error) {
	if addrs == nil {
		addrs = []store.Address{}
	}
	b, err := json.Marshal(addrs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseAddrsJSON(s string) ([]store.Address, error) {
	if s == "" {
		return nil, nil
	}
	var addrs []store.Address
	if err := json.Unmarshal([]byte(s), &addrs); err != nil {
		return nil, err
	}
	return addrs, nil
}

// attachmentFilePath returns the on-disk path for the given attachment ID.
func (s *Store) attachmentFilePath(id string) string {
	return filepath.Join(s.diskPath, id)
}

// Save persists msg (message row + header rows + attachment/inline-image
// rows) in a single transaction, generating msg.ID if empty. Attachment
// bytes are written to disk (when configured) before the transaction
// commits; any file already written is best-effort removed if a later step
// in Save fails.
func (s *Store) Save(ctx context.Context, msg *store.Message) error {
	if msg.ID == "" {
		msg.ID = store.NewID()
	}

	fromAddr := ""
	if len(msg.From) > 0 {
		fromAddr = msg.From[0].Address
	} else {
		fromAddr = msg.EnvelopeFrom
	}

	toJSON, err := addrsJSON(msg.To)
	if err != nil {
		return fmt.Errorf("sqlite: encoding to_addrs: %w", err)
	}
	ccJSON, err := addrsJSON(msg.Cc)
	if err != nil {
		return fmt.Errorf("sqlite: encoding cc_addrs: %w", err)
	}
	bccJSON, err := addrsJSON(msg.Bcc)
	if err != nil {
		return fmt.Errorf("sqlite: encoding bcc_addrs: %w", err)
	}

	hasAttachments := len(msg.Attachments) > 0 || len(msg.InlineImages) > 0

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// Replace any existing row for this ID (Save overwrites, per
	// MessageStore's documented semantics) — delete first so cascades clean
	// up stale headers/attachments before re-inserting.
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, msg.ID); err != nil {
		return fmt.Errorf("sqlite: clearing previous row: %w", err)
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO messages (
		id, message_id, from_addr, to_addrs, cc_addrs, bcc_addrs, subject,
		text_body, html_body, raw_source, size_bytes, raw_size_bytes,
		has_attachments, parse_warning, parse_error, received_at, client_ip, client_helo, read
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		msg.ID, headerValue(msg.Headers, "Message-Id"), fromAddr, toJSON, ccJSON, bccJSON,
		msg.Subject, msg.TextBody, msg.HTMLBody, msg.RawSource, msg.Size, int64(len(msg.RawSource)),
		hasAttachments, msg.ParseWarning, msg.ParseError, msg.ReceivedAt.Format(timeLayout), "", "", msg.Read,
	)
	if err != nil {
		return fmt.Errorf("sqlite: inserting message: %w", err)
	}

	for i, h := range msg.Headers {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO headers (message_id, name, value, position) VALUES (?,?,?,?)`,
			msg.ID, h.Name, h.Value, i,
		); err != nil {
			return fmt.Errorf("sqlite: inserting header: %w", err)
		}
	}

	var writtenFiles []string
	rollbackFiles := func() {
		for _, p := range writtenFiles {
			_ = os.Remove(p)
		}
	}

	for i := range msg.Attachments {
		a := &msg.Attachments[i]
		if a.ID == "" {
			a.ID = store.NewID()
		}
		if err := s.insertAttachment(ctx, tx, msg.ID, a.ID, a.Filename, a.ContentType, a.Size, "", false, a.Data, &writtenFiles); err != nil {
			rollbackFiles()
			return err
		}
	}
	for i := range msg.InlineImages {
		img := &msg.InlineImages[i]
		if img.ID == "" {
			img.ID = store.NewID()
		}
		if err := s.insertAttachment(ctx, tx, msg.ID, img.ID, img.Filename, img.ContentType, img.Size, img.ContentID, true, img.Data, &writtenFiles); err != nil {
			rollbackFiles()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		rollbackFiles()
		return fmt.Errorf("sqlite: commit: %w", err)
	}

	return nil
}

func (s *Store) insertAttachment(ctx context.Context, tx *sql.Tx, messageID, id, filename, contentType string, size int64, contentID string, isInline bool, data []byte, writtenFiles *[]string) error {
	var diskPath string
	var blob []byte

	if s.attachOnDisk {
		if err := os.MkdirAll(s.diskPath, 0o755); err != nil {
			return fmt.Errorf("sqlite: creating attachment disk path: %w", err)
		}
		p := s.attachmentFilePath(id)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			return fmt.Errorf("sqlite: writing attachment to disk: %w", err)
		}
		*writtenFiles = append(*writtenFiles, p)
		diskPath = p
	} else {
		blob = data
	}

	_, err := tx.ExecContext(ctx, `INSERT INTO attachments (
		id, message_id, filename, content_type, size_bytes, content_id, is_inline, data, disk_path
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		id, messageID, filename, contentType, size, contentID, isInline, blob, nullableString(diskPath),
	)
	if err != nil {
		return fmt.Errorf("sqlite: inserting attachment: %w", err)
	}
	return nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func headerValue(headers []store.Header, name string) string {
	for _, h := range headers {
		if h.Name == name {
			return h.Value
		}
	}
	return ""
}

// resolveID resolves id to a full stored message ID. Strings of
// store.IDLength or longer are looked up with an exact, index-seek `WHERE
// id = ?` (the common case, and the fast path — no prefix scan). Shorter
// strings are resolved as a prefix via `WHERE id LIKE ? || '%'`: since `id`
// is a TEXT PRIMARY KEY, SQLite's query planner can still use that index for
// a LIKE pattern with no leading wildcard, so this stays an index range scan
// rather than a full table scan even with a large message count. Message
// IDs are lowercase hex (store.NewID), so the prefix can never itself
// contain a LIKE metacharacter (`%`/`_`) and needs no escaping. Zero matches
// is store.ErrNotFound; more than one is store.ErrAmbiguousID.
func (s *Store) resolveID(ctx context.Context, id string) (string, error) {
	if len(id) >= store.IDLength {
		var full string
		err := s.db.QueryRowContext(ctx, `SELECT id FROM messages WHERE id = ?`, id).Scan(&full)
		if err == sql.ErrNoRows {
			return "", store.ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("sqlite: resolving id: %w", err)
		}
		return full, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id FROM messages WHERE id LIKE ? || '%' LIMIT 2`, id)
	if err != nil {
		return "", fmt.Errorf("sqlite: resolving id prefix: %w", err)
	}
	defer rows.Close()

	var matches []string
	for rows.Next() {
		var full string
		if err := rows.Scan(&full); err != nil {
			return "", fmt.Errorf("sqlite: scanning id prefix match: %w", err)
		}
		matches = append(matches, full)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	switch len(matches) {
	case 0:
		return "", store.ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return "", store.ErrAmbiguousID
	}
}

// Get returns the message with the given ID or unambiguous ID prefix (see
// store.IDLength), or store.ErrNotFound/store.ErrAmbiguousID.
func (s *Store) Get(ctx context.Context, id string) (*store.Message, error) {
	full, err := s.resolveID(ctx, id)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(ctx, `SELECT
		id, from_addr, to_addrs, cc_addrs, bcc_addrs, subject, text_body, html_body,
		raw_source, size_bytes, parse_warning, parse_error, received_at, read
	FROM messages WHERE id = ?`, full)

	msg, err := scanMessage(row)
	if err != nil {
		return nil, err
	}

	if err := s.loadHeaders(ctx, msg); err != nil {
		return nil, err
	}
	if err := s.loadAttachments(ctx, msg); err != nil {
		return nil, err
	}
	msg.AttachmentCount = len(msg.Attachments) + len(msg.InlineImages)

	return msg, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanMessage(row scannable) (*store.Message, error) {
	var (
		id, fromAddr, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody string
		rawSource                                                          []byte
		size                                                               int64
		parseWarning, read                                                 bool
		parseError, receivedAt                                             string
	)

	if err := row.Scan(&id, &fromAddr, &toJSON, &ccJSON, &bccJSON, &subject, &textBody, &htmlBody,
		&rawSource, &size, &parseWarning, &parseError, &receivedAt, &read); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scanning message: %w", err)
	}

	msg, err := messageFromScannedFields(id, fromAddr, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody, rawSource, size, parseWarning, parseError, receivedAt)
	if err != nil {
		return nil, err
	}
	msg.Read = read
	return msg, nil
}

func messageFromScannedFields(id, fromAddr, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody string, rawSource []byte, size int64, parseWarning bool, parseError, receivedAt string) (*store.Message, error) {
	to, err := parseAddrsJSON(toJSON)
	if err != nil {
		return nil, fmt.Errorf("sqlite: decoding to_addrs: %w", err)
	}
	cc, err := parseAddrsJSON(ccJSON)
	if err != nil {
		return nil, fmt.Errorf("sqlite: decoding cc_addrs: %w", err)
	}
	bcc, err := parseAddrsJSON(bccJSON)
	if err != nil {
		return nil, fmt.Errorf("sqlite: decoding bcc_addrs: %w", err)
	}

	rt, err := time.Parse(timeLayout, receivedAt)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parsing received_at: %w", err)
	}

	var from []store.Address
	if fromAddr != "" {
		from = []store.Address{{Address: fromAddr}}
	}

	return &store.Message{
		ID:           id,
		ReceivedAt:   rt,
		Size:         size,
		From:         from,
		To:           to,
		Cc:           cc,
		Bcc:          bcc,
		Subject:      subject,
		TextBody:     textBody,
		HTMLBody:     htmlBody,
		RawSource:    rawSource,
		ParseWarning: parseWarning,
		ParseError:   parseError,
	}, nil
}

func (s *Store) loadHeaders(ctx context.Context, msg *store.Message) error {
	rows, err := s.db.QueryContext(ctx, `SELECT name, value FROM headers WHERE message_id = ? ORDER BY position ASC`, msg.ID)
	if err != nil {
		return fmt.Errorf("sqlite: querying headers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h store.Header
		if err := rows.Scan(&h.Name, &h.Value); err != nil {
			return fmt.Errorf("sqlite: scanning header: %w", err)
		}
		msg.Headers = append(msg.Headers, h)
	}
	return rows.Err()
}

func (s *Store) loadAttachments(ctx context.Context, msg *store.Message) error {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, filename, content_type, size_bytes, content_id, is_inline, data, disk_path
	FROM attachments WHERE message_id = ?`, msg.ID)
	if err != nil {
		return fmt.Errorf("sqlite: querying attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, filename, contentType, contentID string
			size                                 int64
			isInline                             bool
			data                                 []byte
			diskPath                             sql.NullString
		)
		if err := rows.Scan(&id, &filename, &contentType, &size, &contentID, &isInline, &data, &diskPath); err != nil {
			return fmt.Errorf("sqlite: scanning attachment: %w", err)
		}

		if diskPath.Valid && diskPath.String != "" {
			b, err := os.ReadFile(diskPath.String)
			if err != nil {
				return fmt.Errorf("sqlite: reading attachment from disk: %w", err)
			}
			data = b
		}

		if isInline {
			msg.InlineImages = append(msg.InlineImages, store.InlineImage{
				ID:          id,
				ContentID:   contentID,
				Filename:    filename,
				ContentType: contentType,
				Size:        size,
				Data:        data,
			})
		} else {
			msg.Attachments = append(msg.Attachments, store.Attachment{
				ID:          id,
				Filename:    filename,
				ContentType: contentType,
				Size:        size,
				Data:        data,
			})
		}
	}
	return rows.Err()
}

// List returns messages matching filter (newest-first by default, or
// oldest-first when filter.Sort == store.SortReceivedAtAsc), paginated, plus
// the total count of matches ignoring pagination. Bodies/headers/attachments
// are not loaded for list rows (only AttachmentCount), matching the summary
// shape callers need for a listing view.
func (s *Store) List(ctx context.Context, filter store.ListFilter) ([]*store.Message, int, error) {
	var (
		joins []string
		where []string
		args  []any
	)

	if filter.Query != "" {
		joins = append(joins, `JOIN messages_fts f ON f.id = m.id`)
		where = append(where, `messages_fts MATCH ?`)
		args = append(args, filter.Query)
	}
	if filter.From != "" {
		where = append(where, `m.from_addr LIKE ? ESCAPE '\'`)
		args = append(args, likeContains(filter.From))
	}
	if filter.To != "" {
		where = append(where, `m.to_addrs LIKE ? ESCAPE '\'`)
		args = append(args, likeContains(filter.To))
	}
	if filter.Subject != "" {
		where = append(where, `m.subject LIKE ? ESCAPE '\'`)
		args = append(args, likeContains(filter.Subject))
	}
	if !filter.Since.IsZero() {
		where = append(where, `m.received_at >= ?`)
		args = append(args, filter.Since.UTC().Format(timeLayout))
	}
	if !filter.Until.IsZero() {
		where = append(where, `m.received_at <= ?`)
		args = append(args, filter.Until.UTC().Format(timeLayout))
	}

	fromClause := `FROM messages m ` + strings.Join(joins, " ")
	whereClause := ""
	if len(where) > 0 {
		whereClause = ` WHERE ` + strings.Join(where, " AND ")
	}

	var total int
	countQuery := `SELECT COUNT(*) ` + fromClause + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("sqlite: counting messages: %w", err)
	}

	order := `m.received_at DESC`
	if filter.Sort == store.SortReceivedAtAsc {
		order = `m.received_at ASC`
	}

	query := `SELECT
		m.id, m.from_addr, m.to_addrs, m.cc_addrs, m.bcc_addrs, m.subject, m.text_body, m.html_body,
		m.raw_source, m.size_bytes, m.parse_warning, m.parse_error, m.received_at, m.read,
		(SELECT COUNT(*) FROM attachments a WHERE a.message_id = m.id) AS attachment_count
	` + fromClause + whereClause + ` ORDER BY ` + order

	listArgs := append([]any{}, args...)
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		listArgs = append(listArgs, filter.Limit)
		if filter.Offset > 0 {
			query += ` OFFSET ?`
			listArgs = append(listArgs, filter.Offset)
		}
	} else if filter.Offset > 0 {
		// SQLite requires a LIMIT before OFFSET; -1 means "no limit".
		query += ` LIMIT -1 OFFSET ?`
		listArgs = append(listArgs, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite: listing messages: %w", err)
	}
	defer rows.Close()

	var out []*store.Message
	for rows.Next() {
		msg, count, err := scanMessageWithCount(rows)
		if err != nil {
			return nil, 0, err
		}
		msg.AttachmentCount = count
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []*store.Message{}
	}

	return out, total, nil
}

// likeContains escapes SQLite LIKE metacharacters in s and wraps it for a
// case-sensitive-by-default (but SQLite's LIKE is ASCII case-insensitive)
// substring match.
func likeContains(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

func scanMessageWithCount(row scannable) (*store.Message, int, error) {
	var (
		id, fromAddr, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody string
		rawSource                                                          []byte
		size                                                               int64
		parseWarning, read                                                 bool
		parseError, receivedAt                                             string
		attachmentCount                                                    int
	)

	if err := row.Scan(&id, &fromAddr, &toJSON, &ccJSON, &bccJSON, &subject, &textBody, &htmlBody,
		&rawSource, &size, &parseWarning, &parseError, &receivedAt, &read, &attachmentCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, store.ErrNotFound
		}
		return nil, 0, fmt.Errorf("sqlite: scanning message: %w", err)
	}

	msg, err := messageFromScannedFields(id, fromAddr, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody, rawSource, size, parseWarning, parseError, receivedAt)
	if err != nil {
		return nil, 0, err
	}
	msg.Read = read
	return msg, attachmentCount, nil
}

// Delete removes the message with the given ID or unambiguous ID prefix
// (and its on-disk attachment files, if any), or returns
// store.ErrNotFound/store.ErrAmbiguousID. This is the single delete path
// also used by the retention sweeper (which always passes a full ID, so it
// never hits the prefix path).
func (s *Store) Delete(ctx context.Context, id string) error {
	full, err := s.resolveID(ctx, id)
	if err != nil {
		return err
	}

	diskPaths, err := s.attachmentDiskPaths(ctx, `message_id = ?`, full)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, full)
	if err != nil {
		return fmt.Errorf("sqlite: deleting message: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: checking rows affected: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}

	for _, p := range diskPaths {
		_ = os.Remove(p)
	}
	return nil
}

// MarkRead marks the message with the given ID or unambiguous ID prefix as
// read, or returns store.ErrNotFound/store.ErrAmbiguousID.
func (s *Store) MarkRead(ctx context.Context, id string) error {
	full, err := s.resolveID(ctx, id)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE messages SET read = 1 WHERE id = ?`, full); err != nil {
		return fmt.Errorf("sqlite: marking message read: %w", err)
	}
	return nil
}

// Clear removes every stored message and any on-disk attachment files.
func (s *Store) Clear(ctx context.Context) error {
	diskPaths, err := s.attachmentDiskPaths(ctx, `1=1`)
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM messages`); err != nil {
		return fmt.Errorf("sqlite: clearing messages: %w", err)
	}

	for _, p := range diskPaths {
		_ = os.Remove(p)
	}
	return nil
}

func (s *Store) attachmentDiskPaths(ctx context.Context, where string, args ...any) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT disk_path FROM attachments WHERE disk_path IS NOT NULL AND `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: querying attachment disk paths: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("sqlite: scanning disk path: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Stats returns a snapshot summary of the store's current contents.
func (s *Store) Stats(ctx context.Context) (store.Stats, error) {
	var (
		total     int
		totalSize sql.NullInt64
		oldestStr sql.NullString
		newestStr sql.NullString
	)
	err := s.db.QueryRowContext(ctx, `SELECT
		COUNT(*), COALESCE(SUM(size_bytes), 0), MIN(received_at), MAX(received_at)
	FROM messages`).Scan(&total, &totalSize, &oldestStr, &newestStr)
	if err != nil {
		return store.Stats{}, fmt.Errorf("sqlite: querying stats: %w", err)
	}

	stats := store.Stats{TotalMessages: total, TotalSizeBytes: totalSize.Int64}
	if oldestStr.Valid {
		t, err := time.Parse(timeLayout, oldestStr.String)
		if err != nil {
			return store.Stats{}, fmt.Errorf("sqlite: parsing oldest received_at: %w", err)
		}
		stats.OldestReceivedAt = &t
	}
	if newestStr.Valid {
		t, err := time.Parse(timeLayout, newestStr.String)
		if err != nil {
			return store.Stats{}, fmt.Errorf("sqlite: parsing newest received_at: %w", err)
		}
		stats.NewestReceivedAt = &t
	}
	return stats, nil
}

// Ping verifies the underlying database connection is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
