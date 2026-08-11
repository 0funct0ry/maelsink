CREATE TABLE tags (
  name TEXT PRIMARY KEY,
  color TEXT NOT NULL CHECK(color IN ('indigo', 'emerald', 'amber', 'rose', 'cyan', 'fuchsia', 'lime', 'orange')),
  created_at TEXT NOT NULL
);
