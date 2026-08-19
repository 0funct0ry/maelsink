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

// inMemoryDSN is the modernc.org/sqlite DSN for a private, transient
// database used when path is "" (SPEC.md §3: an unset --db/-d / storage.path
// means "no persistent database configured"). It exists only for the
// process lifetime and is lost on restart.
const inMemoryDSN = ":memory:"

// Open opens (creating if necessary) the SQLite database at path, enables
// WAL mode and foreign keys, and runs every pending migration. An empty path
// opens a transient in-memory database instead of a file.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	dsn := path
	if dsn == "" {
		dsn = inMemoryDSN
	} else if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("sqlite: creating db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// modernc.org/sqlite serializes writes (avoids SQLITE_BUSY churn), and a
	// single shared connection is also what makes the in-memory DSN work at
	// all: each *new* connection to ":memory:" would otherwise see its own
	// empty, unmigrated database instead of the same one.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	if dsn == inMemoryDSN {
		// WAL requires a real file on disk; it's meaningless (and rejected
		// by some SQLite builds) for an in-memory database.
		pragmas = pragmas[1:]
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("sqlite: %s: %w", pragma, err)
		}
	}

	if err := RunMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := backfillTagsTable(ctx, db); err != nil {
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

func tagsJSON(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseTagsJSON(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(s), &tags); err != nil {
		return nil, err
	}
	return tags, nil
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
	fromName := ""
	if len(msg.From) > 0 {
		fromAddr = msg.From[0].Address
		fromName = msg.From[0].Name
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
	tagsJSONStr, err := tagsJSON(msg.Tags)
	if err != nil {
		return fmt.Errorf("sqlite: encoding tags: %w", err)
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
		id, message_id, from_addr, from_name, to_addrs, cc_addrs, bcc_addrs, subject,
		text_body, html_body, raw_source, size_bytes, raw_size_bytes,
		has_attachments, parse_warning, parse_error, received_at, client_ip, client_helo, read, tags, session_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		msg.ID, headerValue(msg.Headers, "Message-Id"), fromAddr, nullableString(fromName), toJSON, ccJSON, bccJSON,
		msg.Subject, msg.TextBody, msg.HTMLBody, msg.RawSource, msg.Size, int64(len(msg.RawSource)),
		hasAttachments, msg.ParseWarning, msg.ParseError, msg.ReceivedAt.Format(timeLayout), "", "", msg.Read, tagsJSONStr,
		nullableString(msg.SessionID),
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

	now := time.Now().Format(timeLayout)
	for _, t := range msg.Tags {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO tags (name, color, created_at) VALUES (?, ?, ?)`,
			t, hashTagColor(t), now); err != nil {
			rollbackFiles()
			return fmt.Errorf("sqlite: registering tag %q: %w", t, err)
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
		id, from_addr, from_name, to_addrs, cc_addrs, bcc_addrs, subject, text_body, html_body,
		raw_source, size_bytes, parse_warning, parse_error, received_at, read, tags, session_id
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
		fromName                                                           sql.NullString
		rawSource                                                          []byte
		size                                                               int64
		parseWarning, read                                                 bool
		parseError, receivedAt, tagsJSONStr                                string
		sessionID                                                          sql.NullString
	)

	if err := row.Scan(&id, &fromAddr, &fromName, &toJSON, &ccJSON, &bccJSON, &subject, &textBody, &htmlBody,
		&rawSource, &size, &parseWarning, &parseError, &receivedAt, &read, &tagsJSONStr, &sessionID); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scanning message: %w", err)
	}

	msg, err := messageFromScannedFields(id, fromAddr, fromName.String, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody, rawSource, size, parseWarning, parseError, receivedAt)
	if err != nil {
		return nil, err
	}
	msg.Read = read
	msg.SessionID = sessionID.String
	tags, err := parseTagsJSON(tagsJSONStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: decoding tags: %w", err)
	}
	msg.Tags = tags
	return msg, nil
}

func messageFromScannedFields(id, fromAddr, fromName, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody string, rawSource []byte, size int64, parseWarning bool, parseError, receivedAt string) (*store.Message, error) {
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
		from = []store.Address{{Name: fromName, Address: fromAddr}}
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

// wrapListErr classifies a failure from List's count/select queries. When
// the query included a FTS5 MATCH clause (hasQuery), any driver failure is
// treated as a malformed search query (store.ErrInvalidQuery) rather than a
// generic storage error — the FTS5 MATCH clause is the only part of List's
// SQL built from unsanitized user input, so a query failure while it's
// present is overwhelmingly a syntax error in that input, not a real
// storage fault. The underlying driver error (which contains SQLite/FTS5
// internals like column/table names) is preserved via %w for logs, but
// callers must never surface err.Error() to end users for
// store.ErrInvalidQuery — only the sentinel's own generic message is safe
// to display.
func wrapListErr(err error, hasQuery bool, context string) error {
	if hasQuery {
		return fmt.Errorf("%w: sqlite: %s: %w", store.ErrInvalidQuery, context, err)
	}
	return fmt.Errorf("sqlite: %s: %w", context, err)
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
	if filter.Cc != "" {
		where = append(where, `m.cc_addrs LIKE ? ESCAPE '\'`)
		args = append(args, likeContains(filter.Cc))
	}
	if filter.Bcc != "" {
		where = append(where, `m.bcc_addrs LIKE ? ESCAPE '\'`)
		args = append(args, likeContains(filter.Bcc))
	}
	if !filter.Since.IsZero() {
		where = append(where, `m.received_at >= ?`)
		args = append(args, filter.Since.UTC().Format(timeLayout))
	}
	if !filter.Until.IsZero() {
		where = append(where, `m.received_at <= ?`)
		args = append(args, filter.Until.UTC().Format(timeLayout))
	}
	if tags := effectiveTags(filter); len(tags) > 0 {
		if filter.TagMode == "all" {
			for _, t := range tags {
				where = append(where, `EXISTS (SELECT 1 FROM json_each(m.tags) WHERE json_each.value = ?)`)
				args = append(args, t)
			}
		} else {
			placeholders := make([]string, len(tags))
			for i, t := range tags {
				placeholders[i] = "?"
				args = append(args, t)
			}
			where = append(where, `EXISTS (SELECT 1 FROM json_each(m.tags) WHERE json_each.value IN (`+strings.Join(placeholders, ",")+`))`)
		}
	}
	if filter.Read != nil {
		where = append(where, `m.read = ?`)
		args = append(args, *filter.Read)
	}
	if filter.HasAttachments != nil {
		if *filter.HasAttachments {
			where = append(where, `m.has_attachments = 1`)
		} else {
			where = append(where, `m.has_attachments = 0`)
		}
	}
	if filter.ParseWarning != nil {
		where = append(where, `m.parse_warning = ?`)
		args = append(args, *filter.ParseWarning)
	}

	fromClause := `FROM messages m ` + strings.Join(joins, " ")
	whereClause := ""
	if len(where) > 0 {
		whereClause = ` WHERE ` + strings.Join(where, " AND ")
	}

	var total int
	countQuery := `SELECT COUNT(*) ` + fromClause + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, wrapListErr(err, filter.Query != "", "counting messages")
	}

	order := `m.received_at DESC`
	if filter.Sort == store.SortReceivedAtAsc {
		order = `m.received_at ASC`
	}

	query := `SELECT
		m.id, m.from_addr, m.from_name, m.to_addrs, m.cc_addrs, m.bcc_addrs, m.subject, m.text_body, m.html_body,
		m.raw_source, m.size_bytes, m.parse_warning, m.parse_error, m.received_at, m.read, m.tags,
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
		return nil, 0, wrapListErr(err, filter.Query != "", "listing messages")
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

// effectiveTags returns filter.Tags, falling back to filter.Tag as sugar for
// a single-element slice when Tags is unset.
func effectiveTags(filter store.ListFilter) []string {
	if len(filter.Tags) > 0 {
		return filter.Tags
	}
	if filter.Tag != "" {
		return []string{filter.Tag}
	}
	return nil
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
		fromName                                                           sql.NullString
		rawSource                                                          []byte
		size                                                               int64
		parseWarning, read                                                 bool
		parseError, receivedAt, tagsJSONStr                                string
		attachmentCount                                                    int
	)

	if err := row.Scan(&id, &fromAddr, &fromName, &toJSON, &ccJSON, &bccJSON, &subject, &textBody, &htmlBody,
		&rawSource, &size, &parseWarning, &parseError, &receivedAt, &read, &tagsJSONStr, &attachmentCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, store.ErrNotFound
		}
		return nil, 0, fmt.Errorf("sqlite: scanning message: %w", err)
	}

	msg, err := messageFromScannedFields(id, fromAddr, fromName.String, toJSON, ccJSON, bccJSON, subject, textBody, htmlBody, rawSource, size, parseWarning, parseError, receivedAt)
	if err != nil {
		return nil, 0, err
	}
	msg.Read = read
	tags, err := parseTagsJSON(tagsJSONStr)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite: decoding tags: %w", err)
	}
	msg.Tags = tags
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// A session's message_id (M8.4) is a foreign key into messages; clear
	// it first so deleting the message doesn't violate that FK. The
	// session record + transcript are independently retained (they don't
	// cascade with the message they produced), so this just drops the
	// cross-link rather than the session itself.
	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET message_id = NULL WHERE message_id = ?`, full); err != nil {
		return fmt.Errorf("sqlite: clearing session message_id: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, full)
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

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}

	for _, p := range diskPaths {
		_ = os.Remove(p)
	}
	return nil
}

// MarkRead sets the read flag of the message with the given ID or
// unambiguous ID prefix, or returns store.ErrNotFound/store.ErrAmbiguousID.
func (s *Store) MarkRead(ctx context.Context, id string, read bool) error {
	full, err := s.resolveID(ctx, id)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE messages SET read = ? WHERE id = ?`, read, full); err != nil {
		return fmt.Errorf("sqlite: setting message read flag: %w", err)
	}
	return nil
}

// AddTag adds tag to the message's tag set. See store.MessageStore.AddTag.
func (s *Store) AddTag(ctx context.Context, id, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return store.ErrInvalidTag
	}

	// Resolve the ID and validate before opening a tx: the connection pool
	// is capped at 1 (see Open), so any s.db.* call made while a tx is open
	// would deadlock rather than fail fast with SQLITE_BUSY.
	full, err := s.resolveID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().Format(timeLayout)
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO tags (name, color, created_at) VALUES (?, ?, ?)`,
		tag, hashTagColor(tag), now); err != nil {
		return fmt.Errorf("sqlite: registering tag %q: %w", tag, err)
	}

	return s.mutateTags(ctx, full, func(tags []string) []string {
		for _, t := range tags {
			if t == tag {
				return tags
			}
		}
		return append(tags, tag)
	})
}

// RemoveTag removes tag from the message's tag set. See store.MessageStore.RemoveTag.
func (s *Store) RemoveTag(ctx context.Context, id, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return store.ErrInvalidTag
	}

	full, err := s.resolveID(ctx, id)
	if err != nil {
		return err
	}

	return s.mutateTags(ctx, full, func(tags []string) []string {
		next := make([]string, 0, len(tags))
		for _, t := range tags {
			if t != tag {
				next = append(next, t)
			}
		}
		return next
	})
}

// mutateTags applies mutate to the message's current tag set inside a
// transaction and persists the result. id must already be a resolved, full
// message ID.
func (s *Store) mutateTags(ctx context.Context, id string, mutate func([]string) []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	var tagsStr string
	if err := tx.QueryRowContext(ctx, `SELECT tags FROM messages WHERE id = ?`, id).Scan(&tagsStr); err != nil {
		if err == sql.ErrNoRows {
			return store.ErrNotFound
		}
		return fmt.Errorf("sqlite: reading tags: %w", err)
	}
	tags, err := parseTagsJSON(tagsStr)
	if err != nil {
		return fmt.Errorf("sqlite: parsing tags: %w", err)
	}

	next := mutate(tags)

	nextJSON, err := tagsJSON(next)
	if err != nil {
		return fmt.Errorf("sqlite: encoding tags: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE messages SET tags = ? WHERE id = ?`, nextJSON, id); err != nil {
		return fmt.Errorf("sqlite: updating tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: committing tag update: %w", err)
	}
	return nil
}

// Clear removes every stored message and any on-disk attachment files.
func (s *Store) Clear(ctx context.Context) error {
	diskPaths, err := s.attachmentDiskPaths(ctx, `1=1`)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// See Delete's identical comment: sessions.message_id (M8.4) is a
	// foreign key into messages, so it must be cleared before the messages
	// themselves are deleted. Sessions/transcripts are retained.
	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET message_id = NULL WHERE message_id IS NOT NULL`); err != nil {
		return fmt.Errorf("sqlite: clearing session message_ids: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM messages`); err != nil {
		return fmt.Errorf("sqlite: clearing messages: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
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

	err = s.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(CASE WHEN read = 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN has_attachments = 1 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN parse_warning = 1 THEN 1 ELSE 0 END), 0)
	FROM messages`).Scan(&stats.UnreadCount, &stats.AttachmentCount, &stats.ParseWarningCount)
	if err != nil {
		return store.Stats{}, fmt.Errorf("sqlite: querying stat counts: %w", err)
	}

	return stats, nil
}

// ListTagsWithStats returns every persisted tag with its usage stats. See
// store.MessageStore.ListTagsWithStats.
func (s *Store) ListTagsWithStats(ctx context.Context) ([]store.TagStats, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.name, t.color, COALESCE(m.cnt, 0), m.last_used
		FROM tags t
		LEFT JOIN (
			SELECT json_each.value AS tag, COUNT(*) AS cnt, MAX(messages.received_at) AS last_used
			FROM messages, json_each(messages.tags)
			GROUP BY json_each.value
		) m ON m.tag = t.name
		ORDER BY COALESCE(m.cnt, 0) DESC, m.last_used DESC, t.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: querying tags: %w", err)
	}
	defer rows.Close()

	out := []store.TagStats{}
	for rows.Next() {
		var (
			ts          store.TagStats
			lastUsedStr sql.NullString
		)
		if err := rows.Scan(&ts.Name, &ts.Color, &ts.Count, &lastUsedStr); err != nil {
			return nil, fmt.Errorf("sqlite: scanning tag: %w", err)
		}
		if lastUsedStr.Valid {
			t, err := time.Parse(timeLayout, lastUsedStr.String)
			if err != nil {
				return nil, fmt.Errorf("sqlite: parsing tag last_used: %w", err)
			}
			ts.LastUsed = &t
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}

// validateTagName trims and validates a tag name, returning store.ErrInvalidTag
// if empty/whitespace-only.
func validateTagName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", store.ErrInvalidTag
	}
	return name, nil
}

// validateTagColor validates color against store.TagColors.
func validateTagColor(color string) error {
	if !store.IsValidTagColor(color) {
		return store.ErrInvalidTag
	}
	return nil
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint")
}

// tagMessageIDs returns the IDs of every message currently carrying tag.
func (s *Store) tagMessageIDs(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, tag string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT DISTINCT messages.id FROM messages, json_each(messages.tags)
		WHERE json_each.value = ?
	`, tag)
	if err != nil {
		return nil, fmt.Errorf("sqlite: querying messages for tag: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sqlite: scanning message id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// mutateTagsTx applies mutate to id's current tag set within an already-open
// tx. Like mutateTags, but reuses a caller-supplied transaction instead of
// opening its own, for RenameTag/DeleteTag's per-message loops.
func mutateTagsTx(ctx context.Context, tx *sql.Tx, id string, mutate func([]string) []string) error {
	var tagsStr string
	if err := tx.QueryRowContext(ctx, `SELECT tags FROM messages WHERE id = ?`, id).Scan(&tagsStr); err != nil {
		return fmt.Errorf("sqlite: reading tags: %w", err)
	}
	tags, err := parseTagsJSON(tagsStr)
	if err != nil {
		return fmt.Errorf("sqlite: parsing tags: %w", err)
	}
	nextJSON, err := tagsJSON(mutate(tags))
	if err != nil {
		return fmt.Errorf("sqlite: encoding tags: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE messages SET tags = ? WHERE id = ?`, nextJSON, id); err != nil {
		return fmt.Errorf("sqlite: updating tags: %w", err)
	}
	return nil
}

// RenameTag renames oldName to newName, merging into newName's existing tag
// row if one already exists. See store.MessageStore.RenameTag.
func (s *Store) RenameTag(ctx context.Context, oldName, newName string) error {
	oldName, err := validateTagName(oldName)
	if err != nil {
		return err
	}
	newName, err = validateTagName(newName)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name = ?`, oldName).Scan(&exists); err != nil {
		return fmt.Errorf("sqlite: checking tag exists: %w", err)
	}
	if exists == 0 {
		return store.ErrTagNotFound
	}

	var merged int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name = ?`, newName).Scan(&merged); err != nil {
		return fmt.Errorf("sqlite: checking new tag name: %w", err)
	}

	ids, err := s.tagMessageIDs(ctx, tx, oldName)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := mutateTagsTx(ctx, tx, id, func(tags []string) []string {
			next := make([]string, 0, len(tags))
			hasNew := false
			for _, t := range tags {
				if t == oldName {
					continue
				}
				if t == newName {
					hasNew = true
				}
				next = append(next, t)
			}
			if !hasNew {
				next = append(next, newName)
			}
			return next
		}); err != nil {
			return err
		}
	}

	if merged > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE name = ?`, oldName); err != nil {
			return fmt.Errorf("sqlite: deleting merged tag: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE tags SET name = ? WHERE name = ?`, newName, oldName); err != nil {
			return fmt.Errorf("sqlite: renaming tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: committing tag rename: %w", err)
	}
	return nil
}

// RecolorTag updates name's persisted color. See store.MessageStore.RecolorTag.
func (s *Store) RecolorTag(ctx context.Context, name, color string) error {
	name, err := validateTagName(name)
	if err != nil {
		return err
	}
	if err := validateTagColor(color); err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx, `UPDATE tags SET color = ? WHERE name = ?`, color, name)
	if err != nil {
		return fmt.Errorf("sqlite: recoloring tag: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: checking rows affected: %w", err)
	}
	if n == 0 {
		return store.ErrTagNotFound
	}
	return nil
}

// CreateTag inserts a new tag with no messages attached. See
// store.MessageStore.CreateTag.
func (s *Store) CreateTag(ctx context.Context, name, color string) error {
	name, err := validateTagName(name)
	if err != nil {
		return err
	}
	if err := validateTagColor(color); err != nil {
		return err
	}

	now := time.Now().Format(timeLayout)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO tags (name, color, created_at) VALUES (?, ?, ?)`, name, color, now); err != nil {
		if isUniqueConstraintErr(err) {
			return store.ErrTagExists
		}
		return fmt.Errorf("sqlite: creating tag: %w", err)
	}
	return nil
}

// DeleteTag removes name from every message's tag set and deletes its tags
// row. See store.MessageStore.DeleteTag.
func (s *Store) DeleteTag(ctx context.Context, name string) error {
	name, err := validateTagName(name)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name = ?`, name).Scan(&exists); err != nil {
		return fmt.Errorf("sqlite: checking tag exists: %w", err)
	}
	if exists == 0 {
		return store.ErrTagNotFound
	}

	ids, err := s.tagMessageIDs(ctx, tx, name)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := mutateTagsTx(ctx, tx, id, func(tags []string) []string {
			next := make([]string, 0, len(tags))
			for _, t := range tags {
				if t != name {
					next = append(next, t)
				}
			}
			return next
		}); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE name = ?`, name); err != nil {
		return fmt.Errorf("sqlite: deleting tag: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: committing tag delete: %w", err)
	}
	return nil
}

// DeleteTagWithMessages deletes every message carrying name, then its tags
// row. See store.MessageStore.DeleteTagWithMessages.
func (s *Store) DeleteTagWithMessages(ctx context.Context, name string) error {
	name, err := validateTagName(name)
	if err != nil {
		return err
	}

	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE name = ?`, name).Scan(&exists); err != nil {
		return fmt.Errorf("sqlite: checking tag exists: %w", err)
	}
	if exists == 0 {
		return store.ErrTagNotFound
	}

	ids, err := s.tagMessageIDs(ctx, s.db, name)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil && err != store.ErrNotFound {
			return err
		}
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE name = ?`, name); err != nil {
		return fmt.Errorf("sqlite: deleting tag: %w", err)
	}
	return nil
}

// Ping verifies the underlying database connection is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// resolveSessionID resolves id to a full stored session ID, following the
// exact same exact-match-then-prefix-scan contract as resolveID (see its
// doc comment) against the sessions table.
func (s *Store) resolveSessionID(ctx context.Context, id string) (string, error) {
	if len(id) >= store.IDLength {
		var full string
		err := s.db.QueryRowContext(ctx, `SELECT id FROM sessions WHERE id = ?`, id).Scan(&full)
		if err == sql.ErrNoRows {
			return "", store.ErrNotFound
		}
		if err != nil {
			return "", fmt.Errorf("sqlite: resolving session id: %w", err)
		}
		return full, nil
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id FROM sessions WHERE id LIKE ? || '%' LIMIT 2`, id)
	if err != nil {
		return "", fmt.Errorf("sqlite: resolving session id prefix: %w", err)
	}
	defer rows.Close()

	var matches []string
	for rows.Next() {
		var full string
		if err := rows.Scan(&full); err != nil {
			return "", fmt.Errorf("sqlite: scanning session id prefix match: %w", err)
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

// CreateSession persists a finished session (header row + transcript rows)
// in a single transaction. See store.MessageStore.CreateSession.
func (s *Store) CreateSession(ctx context.Context, sess *store.Session) error {
	if sess.ID == "" {
		sess.ID = store.NewID()
	}

	var endedAt any
	if sess.EndedAt != nil {
		endedAt = sess.EndedAt.Format(timeLayout)
	}
	var messageID any
	if sess.MessageID != nil {
		messageID = *sess.MessageID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// handleConn calls CreateSession twice per connection: once up front
	// (a minimal row, so a message saved mid-session has a valid
	// session_id foreign key target) and once more with the finished
	// record. An UPSERT (rather than the message table's delete-then-
	// reinsert pattern) is required here because, by the second call, a
	// message may already reference this session's id — deleting the row
	// first would violate that FK.
	_, err = tx.ExecContext(ctx, `INSERT INTO sessions (
		id, client_ip, client_helo, started_at, ended_at, status, message_id
	) VALUES (?,?,?,?,?,?,?)
	ON CONFLICT(id) DO UPDATE SET
		client_ip = excluded.client_ip, client_helo = excluded.client_helo,
		started_at = excluded.started_at, ended_at = excluded.ended_at,
		status = excluded.status, message_id = excluded.message_id`,
		sess.ID, nullableString(sess.ClientIP), nullableString(sess.ClientHELO),
		sess.StartedAt.Format(timeLayout), endedAt, sess.Status, messageID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: upserting session: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM session_lines WHERE session_id = ?`, sess.ID); err != nil {
		return fmt.Errorf("sqlite: clearing previous session lines: %w", err)
	}

	for _, line := range sess.Transcript {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO session_lines (session_id, direction, line, position) VALUES (?,?,?,?)`,
			sess.ID, string(line.Direction), line.Line, line.Position,
		); err != nil {
			return fmt.Errorf("sqlite: inserting session line: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}
	return nil
}

// AppendSessionLine persists a single transcript line for an
// already-created session (M8.4a) — a single-row insert, cheap enough to
// call on every wire line so a still-open session's transcript is visible
// via GetSession, not just after CreateSession's final write at connection
// close. Requires the session's row to already exist (handleConn always
// creates it before the read loop starts, precisely for this).
func (s *Store) AppendSessionLine(ctx context.Context, sessionID string, line store.TranscriptLine) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO session_lines (session_id, direction, line, position) VALUES (?,?,?,?)`,
		sessionID, string(line.Direction), line.Line, line.Position,
	)
	if err != nil {
		return fmt.Errorf("sqlite: appending session line: %w", err)
	}
	return nil
}

// GetSession returns the session with the given ID or unambiguous ID
// prefix, including its ordered transcript, or
// store.ErrNotFound/store.ErrAmbiguousID.
func (s *Store) GetSession(ctx context.Context, id string) (*store.Session, error) {
	full, err := s.resolveSessionID(ctx, id)
	if err != nil {
		return nil, err
	}

	var (
		clientIP, clientHELO sql.NullString
		startedAt            string
		endedAt              sql.NullString
		status               string
		messageID            sql.NullString
	)
	row := s.db.QueryRowContext(ctx, `SELECT client_ip, client_helo, started_at, ended_at, status, message_id
		FROM sessions WHERE id = ?`, full)
	if err := row.Scan(&clientIP, &clientHELO, &startedAt, &endedAt, &status, &messageID); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scanning session: %w", err)
	}

	sess, err := sessionFromScannedFields(full, clientIP, clientHELO, startedAt, endedAt, status, messageID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT direction, line, position FROM session_lines
		WHERE session_id = ? ORDER BY position ASC`, full)
	if err != nil {
		return nil, fmt.Errorf("sqlite: loading session lines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var direction, line string
		var position int
		if err := rows.Scan(&direction, &line, &position); err != nil {
			return nil, fmt.Errorf("sqlite: scanning session line: %w", err)
		}
		sess.Transcript = append(sess.Transcript, store.TranscriptLine{
			Direction: direction[0],
			Line:      line,
			Position:  position,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sess, nil
}

func sessionFromScannedFields(id string, clientIP, clientHELO sql.NullString, startedAt string, endedAt sql.NullString, status string, messageID sql.NullString) (*store.Session, error) {
	st, err := time.Parse(timeLayout, startedAt)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parsing started_at: %w", err)
	}
	sess := &store.Session{
		ID:         id,
		ClientIP:   clientIP.String,
		ClientHELO: clientHELO.String,
		StartedAt:  st,
		Status:     status,
	}
	if endedAt.Valid {
		et, err := time.Parse(timeLayout, endedAt.String)
		if err != nil {
			return nil, fmt.Errorf("sqlite: parsing ended_at: %w", err)
		}
		sess.EndedAt = &et
	}
	if messageID.Valid {
		mid := messageID.String
		sess.MessageID = &mid
	}
	return sess, nil
}

// DeleteSession removes the session (and, via ON DELETE CASCADE, its
// transcript) with the given full ID or unambiguous ID prefix, or
// store.ErrNotFound/store.ErrAmbiguousID.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	full, err := s.resolveSessionID(ctx, id)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// messages.session_id is a foreign key into sessions; clear it first
	// so deleting the session doesn't violate that FK (mirrors Delete's
	// identical fix for the reverse direction). The message itself is
	// untouched — only the cross-link is dropped.
	if _, err := tx.ExecContext(ctx, `UPDATE messages SET session_id = NULL WHERE session_id = ?`, full); err != nil {
		return fmt.Errorf("sqlite: clearing message session_id: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, full)
	if err != nil {
		return fmt.Errorf("sqlite: deleting session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: deleting session: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}
	return nil
}

// ClearSessions removes every stored session (and, via ON DELETE CASCADE,
// every session's transcript). Every message's session_id cross-link is
// cleared first; the messages themselves are untouched.
func (s *Store) ClearSessions(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	if _, err := tx.ExecContext(ctx, `UPDATE messages SET session_id = NULL WHERE session_id IS NOT NULL`); err != nil {
		return fmt.Errorf("sqlite: clearing message session_ids: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return fmt.Errorf("sqlite: clearing sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}
	return nil
}

// ListSessions returns session summaries matching filter (newest-started
// first by default), paginated, plus the total count of matches ignoring
// pagination.
func (s *Store) ListSessions(ctx context.Context, filter store.SessionListFilter) ([]*store.SessionSummary, int, error) {
	var (
		where []string
		args  []any
	)

	if filter.Status != "" {
		where = append(where, `status = ?`)
		args = append(args, filter.Status)
	}
	if filter.ClientIP != "" {
		where = append(where, `client_ip = ?`)
		args = append(args, filter.ClientIP)
	}
	if !filter.Since.IsZero() {
		where = append(where, `started_at >= ?`)
		args = append(args, filter.Since.UTC().Format(timeLayout))
	}
	if !filter.Until.IsZero() {
		where = append(where, `started_at <= ?`)
		args = append(args, filter.Until.UTC().Format(timeLayout))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = ` WHERE ` + strings.Join(where, " AND ")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("sqlite: counting sessions: %w", err)
	}

	order := `started_at DESC`
	if filter.Sort == store.SortStartedAtAsc {
		order = `started_at ASC`
	}

	query := `SELECT id, client_ip, client_helo, started_at, ended_at, status, message_id
		FROM sessions` + whereClause + ` ORDER BY ` + order

	listArgs := append([]any{}, args...)
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		listArgs = append(listArgs, filter.Limit)
		if filter.Offset > 0 {
			query += ` OFFSET ?`
			listArgs = append(listArgs, filter.Offset)
		}
	} else if filter.Offset > 0 {
		query += ` LIMIT -1 OFFSET ?`
		listArgs = append(listArgs, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite: listing sessions: %w", err)
	}
	defer rows.Close()

	var out []*store.SessionSummary
	for rows.Next() {
		var (
			id, status           string
			clientIP, clientHELO sql.NullString
			startedAt            string
			endedAt, messageID   sql.NullString
		)
		if err := rows.Scan(&id, &clientIP, &clientHELO, &startedAt, &endedAt, &status, &messageID); err != nil {
			return nil, 0, fmt.Errorf("sqlite: scanning session: %w", err)
		}
		sess, err := sessionFromScannedFields(id, clientIP, clientHELO, startedAt, endedAt, status, messageID)
		if err != nil {
			return nil, 0, err
		}
		summary := store.NewSessionSummary(sess)
		out = append(out, &summary)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []*store.SessionSummary{}
	}

	return out, total, nil
}
