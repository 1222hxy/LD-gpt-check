ALTER TABLE benchmark_submissions ADD COLUMN upload_schema_version INTEGER;
ALTER TABLE benchmark_submissions ADD COLUMN codex_model_source TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN codex_model_provider TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN codex_provider_host TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN codex_sandbox TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN codex_ephemeral INTEGER;
ALTER TABLE benchmark_submissions ADD COLUMN codex_skip_git_repo_check INTEGER;
ALTER TABLE benchmark_submissions ADD COLUMN codex_disabled_features TEXT;
ALTER TABLE benchmark_submissions ADD COLUMN codex_invocation TEXT;

ALTER TABLE benchmark_attempts ADD COLUMN answer_preview_truncated INTEGER;
ALTER TABLE benchmark_attempts ADD COLUMN cached_input_tokens INTEGER;
ALTER TABLE benchmark_attempts ADD COLUMN total_tokens INTEGER;
ALTER TABLE benchmark_attempts ADD COLUMN codex_thread_id TEXT;
ALTER TABLE benchmark_attempts ADD COLUMN event_count INTEGER;
ALTER TABLE benchmark_attempts ADD COLUMN event_types TEXT;
ALTER TABLE benchmark_attempts ADD COLUMN tool_event_detected INTEGER;
ALTER TABLE benchmark_attempts ADD COLUMN answer_chars INTEGER;
