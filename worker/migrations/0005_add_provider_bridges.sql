ALTER TABLE benchmark_submissions ADD COLUMN codex_provider_base_url TEXT NOT NULL DEFAULT '';
ALTER TABLE benchmark_submissions ADD COLUMN codex_channel TEXT NOT NULL DEFAULT 'unknown_bridge';
ALTER TABLE benchmark_submissions ADD COLUMN codex_bridge_id TEXT;

CREATE TABLE IF NOT EXISTS bridges (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bridge_base_urls (
  id TEXT PRIMARY KEY,
  bridge_id TEXT NOT NULL,
  base_url TEXT NOT NULL UNIQUE,
  host TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(bridge_id) REFERENCES bridges(id)
);

CREATE INDEX IF NOT EXISTS idx_bridge_base_urls_host ON bridge_base_urls(host, is_active);
CREATE INDEX IF NOT EXISTS idx_benchmark_submissions_channel ON benchmark_submissions(codex_channel, created_at);
