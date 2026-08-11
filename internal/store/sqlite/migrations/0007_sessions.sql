-- SMTP session records + protocol transcript (SPEC.md §4/§6, M8.4): one
-- row per accepted connection, independent of whether it produced a stored
-- message, plus its ordered raw transcript lines.

CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT PRIMARY KEY,
    client_ip    TEXT,
    client_helo  TEXT,
    started_at   TEXT NOT NULL,
    ended_at     TEXT,
    status       TEXT NOT NULL,
    message_id   TEXT REFERENCES messages(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);

CREATE TABLE IF NOT EXISTS session_lines (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    direction  TEXT NOT NULL,
    line       TEXT NOT NULL,
    position   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_lines_session_id ON session_lines(session_id);
