CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  provider_user_id TEXT NOT NULL,
  username TEXT,
  login TEXT,
  name TEXT,
  email TEXT,
  avatar_url TEXT,
  avatar_template TEXT,
  active INTEGER,
  trust_level INTEGER,
  silenced INTEGER,
  linuxdo_profile_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(provider, provider_user_id)
);

CREATE TABLE IF NOT EXISTS device_sessions (
  id TEXT PRIMARY KEY,
  device_code_hash TEXT NOT NULL UNIQUE,
  user_code_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL,
  user_id TEXT,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  approved_at TEXT,
  last_polled_at TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS access_tokens (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  device_name TEXT,
  created_at TEXT NOT NULL,
  last_used_at TEXT,
  revoked_at TEXT,
  expires_at TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS benchmark_submissions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  upload_id TEXT NOT NULL,
  upload_schema_version INTEGER,
  client_version TEXT,
  model TEXT,
  reasoning_effort TEXT,
  question_count INTEGER,
  attempt_count INTEGER,
  correct_count INTEGER,
  accuracy REAL,
  avg_input_tokens REAL,
  avg_output_tokens REAL,
  avg_reason_tokens REAL,
  avg_time_seconds REAL,
  avg_tps REAL,
  is_anonymous INTEGER NOT NULL DEFAULT 0,
  started_at TEXT,
  finished_at TEXT,
  duration_seconds REAL,
  question_suite TEXT,
  client_timezone TEXT,
  os TEXT,
  arch TEXT,
  codex_version TEXT,
  codex_model_source TEXT,
  codex_model_provider TEXT,
  codex_provider_host TEXT,
  codex_provider_base_url TEXT NOT NULL DEFAULT '',
  codex_channel TEXT NOT NULL DEFAULT 'unknown_bridge',
  codex_bridge_id TEXT,
  codex_sandbox TEXT,
  codex_ephemeral INTEGER,
  codex_skip_git_repo_check INTEGER,
  codex_disabled_features TEXT,
  codex_invocation TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(user_id, upload_id),
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(codex_bridge_id) REFERENCES bridges(id)
);

CREATE TABLE IF NOT EXISTS bridges (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  icon_url TEXT NOT NULL DEFAULT '',
  homepage_url TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'confirmed',
  primary_base_url_id TEXT,
  root_domain TEXT NOT NULL DEFAULT '',
  canonical_url TEXT NOT NULL DEFAULT '',
  og_image_url TEXT NOT NULL DEFAULT '',
  usage_count INTEGER NOT NULL DEFAULT 0,
  user_count INTEGER NOT NULL DEFAULT 0,
  confidence INTEGER NOT NULL DEFAULT 100,
  last_seen_at TEXT,
  merged_into_bridge_id TEXT,
  merge_suppressed INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bridge_base_urls (
  id TEXT PRIMARY KEY,
  bridge_id TEXT NOT NULL,
  base_url TEXT NOT NULL UNIQUE,
  host TEXT NOT NULL,
  root_domain TEXT NOT NULL DEFAULT '',
  path TEXT NOT NULL DEFAULT '',
  is_primary INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  usage_count INTEGER NOT NULL DEFAULT 0,
  last_seen_at TEXT,
  confidence INTEGER NOT NULL DEFAULT 100,
  source TEXT NOT NULL DEFAULT 'admin',
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(bridge_id) REFERENCES bridges(id)
);

CREATE TABLE IF NOT EXISTS bridge_suggestions (
  id TEXT PRIMARY KEY,
  user_id TEXT,
  base_url TEXT NOT NULL UNIQUE,
  host TEXT NOT NULL,
  source TEXT NOT NULL,
  submitted_name TEXT,
  page_title TEXT,
  page_description TEXT NOT NULL DEFAULT '',
  icon_url TEXT,
  homepage_url TEXT NOT NULL DEFAULT '',
  canonical_url TEXT NOT NULL DEFAULT '',
  og_image_url TEXT NOT NULL DEFAULT '',
  detected_name TEXT NOT NULL DEFAULT '',
  root_domain TEXT NOT NULL DEFAULT '',
  confidence INTEGER NOT NULL DEFAULT 0,
  candidate_bridge_id TEXT,
  candidate_bridge_name TEXT NOT NULL DEFAULT '',
  candidate_reason TEXT NOT NULL DEFAULT '',
  user_count INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'pending',
  occurrence_count INTEGER NOT NULL DEFAULT 1,
  bridge_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(bridge_id) REFERENCES bridges(id)
);

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

CREATE TABLE IF NOT EXISTS benchmark_question_results (
  id TEXT PRIMARY KEY,
  submission_id TEXT NOT NULL,
  question_id TEXT NOT NULL,
  question_version TEXT NOT NULL,
  question_title TEXT,
  grader_type TEXT,
  expected_answer TEXT,
  prompt_hash TEXT,
  test_count INTEGER,
  correct_count INTEGER,
  accuracy REAL,
  avg_input_tokens REAL,
  avg_output_tokens REAL,
  avg_reason_tokens REAL,
  avg_time_seconds REAL,
  avg_tps REAL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(submission_id) REFERENCES benchmark_submissions(id)
);

CREATE TABLE IF NOT EXISTS benchmark_attempts (
  id TEXT PRIMARY KEY,
  submission_id TEXT NOT NULL,
  question_id TEXT NOT NULL,
  question_version TEXT NOT NULL,
  case_index INTEGER,
  status TEXT,
  is_correct INTEGER,
  expected_answer TEXT,
  extracted_answer TEXT,
  failure_reason TEXT,
  answer_preview TEXT,
  answer_preview_truncated INTEGER,
  answer_hash TEXT,
  input_tokens INTEGER,
  cached_input_tokens INTEGER,
  output_tokens INTEGER,
  reasoning_tokens INTEGER,
  total_tokens INTEGER,
  time_seconds REAL,
  tps REAL,
  codex_thread_id TEXT,
  event_count INTEGER,
  event_types TEXT,
  tool_event_detected INTEGER,
  answer_chars INTEGER,
  error_code TEXT,
  started_at TEXT,
  finished_at TEXT,
  timeout_seconds REAL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(submission_id) REFERENCES benchmark_submissions(id)
);

CREATE TABLE IF NOT EXISTS oauth_states (
  id TEXT PRIMARY KEY,
  state_hash TEXT NOT NULL UNIQUE,
  redirect_path TEXT,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  used_at TEXT
);

CREATE TABLE IF NOT EXISTS web_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  session_hash TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  last_used_at TEXT,
  revoked_at TEXT,
  expires_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS rate_limits (
  key TEXT PRIMARY KEY,
  window_start TEXT NOT NULL,
  count INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS question_banks (
  id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  questions_json TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  created_by TEXT,
  updated_by TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(created_by) REFERENCES users(id),
  FOREIGN KEY(updated_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_benchmark_submissions_user_created ON benchmark_submissions(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_benchmark_question_results_submission ON benchmark_question_results(submission_id);
CREATE INDEX IF NOT EXISTS idx_benchmark_attempts_submission ON benchmark_attempts(submission_id);
CREATE INDEX IF NOT EXISTS idx_benchmark_attempts_question ON benchmark_attempts(question_id, question_version);
CREATE INDEX IF NOT EXISTS idx_access_tokens_hash ON access_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_web_sessions_hash ON web_sessions(session_hash);
CREATE INDEX IF NOT EXISTS idx_rate_limits_window ON rate_limits(window_start);
CREATE INDEX IF NOT EXISTS idx_question_banks_active ON question_banks(is_active, updated_at);
CREATE INDEX IF NOT EXISTS idx_bridge_base_urls_host ON bridge_base_urls(host, is_active);
CREATE INDEX IF NOT EXISTS idx_benchmark_submissions_channel ON benchmark_submissions(codex_channel, created_at);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_status ON bridge_suggestions(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_host ON bridge_suggestions(host, status);
CREATE INDEX IF NOT EXISTS idx_bridges_status ON bridges(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_bridges_root_domain ON bridges(root_domain, status);
CREATE INDEX IF NOT EXISTS idx_bridge_base_urls_root_domain ON bridge_base_urls(root_domain, is_active);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_root_domain ON bridge_suggestions(root_domain, status);
CREATE INDEX IF NOT EXISTS idx_bridge_suggestions_candidate ON bridge_suggestions(candidate_bridge_id, status);
