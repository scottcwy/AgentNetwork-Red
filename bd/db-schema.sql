CREATE TABLE IF NOT EXISTS xhs_seed_accounts (
    seed_id TEXT PRIMARY KEY,
    original_profile_url TEXT NOT NULL,
    normalized_profile_url TEXT NOT NULL UNIQUE,
    cover_photo TEXT NOT NULL,
    source_event TEXT NOT NULL,
    captured_at TEXT NOT NULL,
    collector TEXT NOT NULL DEFAULT 'Zeena',
    display_name TEXT,
    team_name TEXT,
    team_scope TEXT,
    note TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS xhs_crawl_snapshots (
    snapshot_id TEXT PRIMARY KEY,
    seed_id TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    crawler TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    parse_status TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    FOREIGN KEY (seed_id) REFERENCES xhs_seed_accounts(seed_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_xhs_seed_accounts_captured_at
    ON xhs_seed_accounts (captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_xhs_crawl_snapshots_seed_fetched_at
    ON xhs_crawl_snapshots (seed_id, fetched_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_xhs_crawl_snapshots_source_hash
    ON xhs_crawl_snapshots (source_hash);
