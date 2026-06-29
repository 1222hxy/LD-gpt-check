-- Redact local/private provider URLs. These usually come from local proxy tools
-- such as CC Switch and must not be treated as bridge base URLs.

UPDATE benchmark_submissions
SET codex_provider_base_url = '',
    codex_provider_host = '',
    codex_channel = 'local_private',
    codex_bridge_id = NULL
WHERE codex_channel != 'official'
  AND (
    codex_provider_host = 'localhost'
    OR codex_provider_host LIKE 'localhost:%'
    OR codex_provider_host LIKE '127.%'
    OR codex_provider_host LIKE '10.%'
    OR codex_provider_host LIKE '192.168.%'
    OR codex_provider_base_url LIKE 'https://localhost%'
    OR codex_provider_base_url LIKE 'https://127.%'
    OR codex_provider_base_url LIKE 'https://10.%'
    OR codex_provider_base_url LIKE 'https://192.168.%'
    OR codex_provider_base_url LIKE 'http://localhost%'
    OR codex_provider_base_url LIKE 'http://127.%'
    OR codex_provider_base_url LIKE 'http://10.%'
    OR codex_provider_base_url LIKE 'http://192.168.%'
    OR codex_provider_base_url GLOB 'https://172.1[6-9].*'
    OR codex_provider_base_url GLOB 'https://172.2[0-9].*'
    OR codex_provider_base_url GLOB 'https://172.3[0-1].*'
    OR codex_provider_base_url GLOB 'http://172.1[6-9].*'
    OR codex_provider_base_url GLOB 'http://172.2[0-9].*'
    OR codex_provider_base_url GLOB 'http://172.3[0-1].*'
  );

DELETE FROM bridge_suggestions
WHERE host = 'localhost'
   OR host LIKE 'localhost:%'
   OR host LIKE '127.%'
   OR host LIKE '10.%'
   OR host LIKE '192.168.%'
   OR base_url LIKE 'https://localhost%'
   OR base_url LIKE 'https://127.%'
   OR base_url LIKE 'https://10.%'
   OR base_url LIKE 'https://192.168.%'
   OR base_url LIKE 'http://localhost%'
   OR base_url LIKE 'http://127.%'
   OR base_url LIKE 'http://10.%'
   OR base_url LIKE 'http://192.168.%'
   OR base_url GLOB 'https://172.1[6-9].*'
   OR base_url GLOB 'https://172.2[0-9].*'
   OR base_url GLOB 'https://172.3[0-1].*'
   OR base_url GLOB 'http://172.1[6-9].*'
   OR base_url GLOB 'http://172.2[0-9].*'
   OR base_url GLOB 'http://172.3[0-1].*';
