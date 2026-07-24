UPDATE platforms
SET
    capabilities = capabilities - ARRAY['videos', 'audio', 'realtime'],
    updated_at = NOW()
WHERE builtin = FALSE
  AND protocol = 'openai_compatible'
  AND capabilities ?| ARRAY['videos', 'audio', 'realtime'];
