-- Reconcile bridge aliases and cached usage after bridge normalization fixes.

UPDATE benchmark_submissions
SET
  codex_bridge_id = (
    SELECT bridge_base_urls.bridge_id
    FROM bridge_base_urls
    JOIN bridges ON bridges.id = bridge_base_urls.bridge_id
    WHERE bridge_base_urls.base_url = benchmark_submissions.codex_provider_base_url
      AND bridge_base_urls.is_active = 1
      AND bridges.is_active = 1
      AND bridges.status != 'merged'
      AND bridges.merged_into_bridge_id IS NULL
    LIMIT 1
  ),
  codex_channel = 'bridge'
WHERE codex_channel != 'official'
  AND codex_provider_base_url IS NOT NULL
  AND codex_provider_base_url != ''
  AND EXISTS (
    SELECT 1
    FROM bridge_base_urls
    JOIN bridges ON bridges.id = bridge_base_urls.bridge_id
    WHERE bridge_base_urls.base_url = benchmark_submissions.codex_provider_base_url
      AND bridge_base_urls.is_active = 1
      AND bridges.is_active = 1
      AND bridges.status != 'merged'
      AND bridges.merged_into_bridge_id IS NULL
  );

DELETE FROM bridge_base_urls
WHERE NOT EXISTS (
    SELECT 1 FROM benchmark_submissions
    WHERE benchmark_submissions.codex_provider_base_url = bridge_base_urls.base_url
  )
  AND EXISTS (
    SELECT 1
    FROM bridges
    WHERE bridges.id = bridge_base_urls.bridge_id
      AND bridges.homepage_url != ''
      AND rtrim(bridges.homepage_url, '/') = bridge_base_urls.base_url
  );

UPDATE bridge_base_urls
SET
  usage_count = (
    SELECT COUNT(*)
    FROM benchmark_submissions
    WHERE benchmark_submissions.codex_provider_base_url = bridge_base_urls.base_url
  ),
  last_seen_at = (
    SELECT MAX(created_at)
    FROM benchmark_submissions
    WHERE benchmark_submissions.codex_provider_base_url = bridge_base_urls.base_url
  );

UPDATE bridges
SET
  usage_count = (
    SELECT COUNT(*)
    FROM benchmark_submissions
    WHERE benchmark_submissions.codex_bridge_id = bridges.id
  ),
  user_count = (
    SELECT COUNT(DISTINCT user_id)
    FROM benchmark_submissions
    WHERE benchmark_submissions.codex_bridge_id = bridges.id
  ),
  last_seen_at = (
    SELECT MAX(created_at)
    FROM benchmark_submissions
    WHERE benchmark_submissions.codex_bridge_id = bridges.id
  )
WHERE status != 'merged'
  AND merged_into_bridge_id IS NULL;

UPDATE bridges
SET usage_count = 0,
    user_count = 0,
    last_seen_at = NULL,
    is_active = 0
WHERE status = 'merged'
   OR merged_into_bridge_id IS NOT NULL;
