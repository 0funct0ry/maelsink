CREATE TABLE IF NOT EXISTS messages (
    id              TEXT PRIMARY KEY,
    message_id      TEXT,
    from_addr       TEXT,
    to_addrs        TEXT NOT NULL DEFAULT '[]',
    cc_addrs        TEXT NOT NULL DEFAULT '[]',
    bcc_addrs       TEXT NOT NULL DEFAULT '[]',
    subject         TEXT,
    text_body       TEXT,
    html_body       TEXT,
    raw_source      BLOB,
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    raw_size_bytes  INTEGER NOT NULL DEFAULT 0,
    has_attachments INTEGER NOT NULL DEFAULT 0,
    parse_warning   INTEGER NOT NULL DEFAULT 0,
    parse_error     TEXT,
    received_at     TEXT NOT NULL,
    client_ip       TEXT,
    client_helo     TEXT
);

CREATE INDEX IF NOT EXISTS idx_messages_received_at ON messages(received_at);

CREATE TABLE IF NOT EXISTS headers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    value      TEXT NOT NULL,
    position   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_headers_message_id ON headers(message_id);

CREATE TABLE IF NOT EXISTS attachments (
    id           TEXT PRIMARY KEY,
    message_id   TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename     TEXT,
    content_type TEXT,
    size_bytes   INTEGER NOT NULL DEFAULT 0,
    content_id   TEXT,
    is_inline    INTEGER NOT NULL DEFAULT 0,
    data         BLOB,
    disk_path    TEXT
);

CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    id UNINDEXED, subject, from_addr, to_addrs, text_body
);

CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(id, subject, from_addr, to_addrs, text_body)
    VALUES (new.id, new.subject, new.from_addr, new.to_addrs, new.text_body);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
    DELETE FROM messages_fts WHERE id = old.id;
END;
