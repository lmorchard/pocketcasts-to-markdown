-- Initial database schema for pocketcasts-to-markdown
-- Version 1

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO schema_migrations (version) VALUES (1);

-- Episodes archive. Rows are upserted from /user/history and /user/starred.
-- We never delete on un-star or history rollover; this DB is a growing archive.
CREATE TABLE IF NOT EXISTS episodes (
    uuid           TEXT PRIMARY KEY,
    podcast_uuid   TEXT,
    podcast_title  TEXT,
    title          TEXT,
    url            TEXT,
    published      TEXT,    -- ISO8601 from the API
    played_up_to   INTEGER, -- seconds
    duration       INTEGER, -- seconds, if API returns it
    is_starred     INTEGER NOT NULL DEFAULT 0,
    in_history     INTEGER NOT NULL DEFAULT 0,
    first_seen_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_episodes_published ON episodes(published);
CREATE INDEX IF NOT EXISTS idx_episodes_in_history ON episodes(in_history);
CREATE INDEX IF NOT EXISTS idx_episodes_is_starred ON episodes(is_starred);

-- Key/value store for the cached auth token, last-run timestamps, etc.
CREATE TABLE IF NOT EXISTS kv (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
