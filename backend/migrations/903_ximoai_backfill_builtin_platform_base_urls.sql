UPDATE platforms
SET base_url = CASE slug
    WHEN 'anthropic' THEN 'https://api.anthropic.com'
    WHEN 'openai' THEN 'https://api.openai.com'
    WHEN 'gemini' THEN 'https://generativelanguage.googleapis.com'
    WHEN 'antigravity' THEN 'https://cloudcode-pa.googleapis.com'
    ELSE base_url
END,
updated_at = NOW()
WHERE builtin = TRUE
  AND slug IN ('anthropic', 'openai', 'gemini', 'antigravity')
  AND base_url = '';
