-- Backlink from a message to the SMTP session that produced it (M8.4), for
-- the Message Detail -> Session Detail cross-link. Nullable: messages saved
-- outside a tracked SMTP session (e.g. tests) have no session.

ALTER TABLE messages ADD COLUMN session_id TEXT REFERENCES sessions(id);
