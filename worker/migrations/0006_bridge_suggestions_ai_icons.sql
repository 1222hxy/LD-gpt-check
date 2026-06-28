ALTER TABLE bridges ADD COLUMN icon_url TEXT NOT NULL DEFAULT '';
ALTER TABLE bridges ADD COLUMN homepage_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS bridge_suggestions (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  base_url TEXT NOT NULL UNIQUE,
  host TEXT NOT NULL,
  source TEXT NOT NULL,
  submitted_name TEXT,
  page_title TEXT,
  icon_url TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  occurrence_count INTEGER NOT NULL DEFAULT 1,
  bridge_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(bridge_id) REFERENCES bridges(id)
);

CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_status ON bridge_suggestions(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_host ON bridge_suggestions(host, status);
