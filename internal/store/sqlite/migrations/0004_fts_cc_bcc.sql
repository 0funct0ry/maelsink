-- Rebuild messages_fts to also index cc_addrs/bcc_addrs (SPEC.md §6, M8.1).
-- FTS5 virtual tables can't ALTER-add columns, so this drops and recreates
-- the table, then backfills it from the existing messages table so
-- pre-migration rows become searchable by cc/bcc without being re-sent.

DROP TRIGGER IF EXISTS messages_ai;
DROP TRIGGER IF EXISTS messages_ad;
DROP TABLE IF EXISTS messages_fts;

CREATE VIRTUAL TABLE messages_fts USING fts5(
    id UNINDEXED, subject, from_addr, to_addrs, cc_addrs, bcc_addrs, text_body
);

CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(id, subject, from_addr, to_addrs, cc_addrs, bcc_addrs, text_body)
    VALUES (new.id, new.subject, new.from_addr, new.to_addrs, new.cc_addrs, new.bcc_addrs, new.text_body);
END;

CREATE TRIGGER messages_ad AFTER DELETE ON messages BEGIN
    DELETE FROM messages_fts WHERE id = old.id;
END;

INSERT INTO messages_fts(id, subject, from_addr, to_addrs, cc_addrs, bcc_addrs, text_body)
SELECT id, subject, from_addr, to_addrs, cc_addrs, bcc_addrs, text_body FROM messages;
