-- +goose Up
CREATE TABLE IF NOT EXISTS mods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL UNIQUE,
    installed_files TEXT NOT NULL DEFAULT '[]',
    file_hash TEXT NOT NULL,
    version TEXT DEFAULT '',
    author TEXT DEFAULT '',
    installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_mods_name ON mods(name);
CREATE INDEX IF NOT EXISTS idx_mods_file_hash ON mods(file_hash);

-- +goose Down
DROP TABLE IF EXISTS mods;