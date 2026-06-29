UPDATE benchmark_submissions
SET codex_channel = 'domestic_official',
    codex_bridge_id = NULL
WHERE lower(COALESCE(codex_provider_host, '')) = 'api.deepseek.com'
   OR codex_provider_base_url IN ('https://api.deepseek.com', 'https://api.deepseek.com/v1');

UPDATE bridge_suggestions
SET status = 'rejected',
    updated_at = datetime('now')
WHERE lower(COALESCE(host, '')) = 'api.deepseek.com'
  AND status = 'pending';
