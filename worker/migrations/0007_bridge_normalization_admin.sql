ALTER TABLE bridges ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE bridges ADD COLUMN status TEXT NOT NULL DEFAULT 'confirmed';
ALTER TABLE bridges ADD COLUMN primary_base_url_id TEXT;
ALTER TABLE bridges ADD COLUMN root_domain TEXT NOT NULL DEFAULT '';
ALTER TABLE bridges ADD COLUMN canonical_url TEXT NOT NULL DEFAULT '';
ALTER TABLE bridges ADD COLUMN og_image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE bridges ADD COLUMN usage_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bridges ADD COLUMN user_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bridges ADD COLUMN confidence INTEGER NOT NULL DEFAULT 100;
ALTER TABLE bridges ADD COLUMN last_seen_at TEXT;
ALTER TABLE bridges ADD COLUMN merged_into_bridge_id TEXT;
ALTER TABLE bridges ADD COLUMN merge_suppressed INTEGER NOT NULL DEFAULT 0;

ALTER TABLE bridge_base_urls ADD COLUMN root_domain TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_base_urls ADD COLUMN path TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_base_urls ADD COLUMN is_primary INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bridge_base_urls ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bridge_base_urls ADD COLUMN usage_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bridge_base_urls ADD COLUMN last_seen_at TEXT;
ALTER TABLE bridge_base_urls ADD COLUMN confidence INTEGER NOT NULL DEFAULT 100;
ALTER TABLE bridge_base_urls ADD COLUMN source TEXT NOT NULL DEFAULT 'admin';

ALTER TABLE bridge_suggestions ADD COLUMN root_domain TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN homepage_url TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN canonical_url TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN page_description TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN og_image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN detected_name TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN confidence INTEGER NOT NULL DEFAULT 0;
ALTER TABLE bridge_suggestions ADD COLUMN candidate_bridge_id TEXT;
ALTER TABLE bridge_suggestions ADD COLUMN candidate_bridge_name TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN candidate_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE bridge_suggestions ADD COLUMN user_count INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS bridge_merge_ignores (
  id TEXT PRIMARY KEY,
  bridge_a_id TEXT NOT NULL,
  bridge_b_id TEXT NOT NULL,
  created_by TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bridge_admin_events (
  id TEXT PRIMARY KEY,
  admin_user_id TEXT,
  action TEXT NOT NULL,
  bridge_id TEXT,
  target_bridge_id TEXT,
  suggestion_id TEXT,
  base_url TEXT,
  details_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bridges_status ON bridges(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_bridges_root_domain ON bridges(root_domain, status);
CREATE INDEX IF NOT EXISTS idx_bridge_base_urls_root_domain ON bridge_base_urls(root_domain, is_active);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_root_domain ON bridge_suggestions(root_domain, status);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_candidate ON bridge_suggestions(candidate_bridge_id, status);
