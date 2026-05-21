UPDATE accounts AS a
SET credentials = jsonb_set(
    COALESCE(a.credentials, '{}'::jsonb),
    '{platform_protocol}',
    to_jsonb(p.protocol),
    true
)
FROM platforms AS p
WHERE a.platform = p.slug
  AND p.builtin = FALSE
  AND a.type = 'apikey'
  AND NOT (COALESCE(a.credentials, '{}'::jsonb) ? 'platform_protocol');
