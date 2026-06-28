ALTER TABLE users ADD COLUMN login TEXT;
ALTER TABLE users ADD COLUMN name TEXT;
ALTER TABLE users ADD COLUMN email TEXT;
ALTER TABLE users ADD COLUMN avatar_template TEXT;
ALTER TABLE users ADD COLUMN active INTEGER;
ALTER TABLE users ADD COLUMN trust_level INTEGER;
ALTER TABLE users ADD COLUMN silenced INTEGER;
ALTER TABLE users ADD COLUMN linuxdo_profile_json TEXT;

ALTER TABLE benchmark_submissions ADD COLUMN started_at TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN finished_at TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN duration_seconds REAL;
ALTER TABLE benchmark_submissions ADD COLUMN question_suite TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN client_timezone TEXT;

ALTER TABLE benchmark_attempts ADD COLUMN answer_hash TEXT;
ALTER TABLE benchmark_attempts ADD COLUMN error_code TEXT;
ALTER TABLE benchmark_attempts ADD COLUMN started_at TEXT;
ALTER TABLE benchmark_attempts ADD COLUMN finished_at TEXT;
ALTER TABLE benchmark_attempts ADD COLUMN timeout_seconds REAL;
