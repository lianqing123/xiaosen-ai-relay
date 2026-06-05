WITH mapping AS (
  SELECT '{
    "auto-kiro": "auto-kiro",
    "claude-haiku-4-5": "claude-haiku-4.5",
    "claude-haiku-4-5-20251001": "claude-haiku-4.5",
    "claude-haiku-4.5": "claude-haiku-4.5",
    "claude-opus-4-5": "claude-opus-4.5",
    "claude-opus-4-5-20251101": "claude-opus-4.5",
    "claude-opus-4.5": "claude-opus-4.5",
    "claude-opus-4-6": "claude-opus-4.6",
    "claude-opus-4.6": "claude-opus-4.6",
    "claude-opus-4-7": "claude-opus-4.7",
    "claude-opus-4-7-20260416": "claude-opus-4.7",
    "claude-opus-4.7": "claude-opus-4.7",
    "claude-sonnet-4": "claude-sonnet-4",
    "claude-sonnet-4-5": "claude-sonnet-4.5",
    "claude-sonnet-4-5-20250929": "claude-sonnet-4.5",
    "claude-sonnet-4.5": "claude-sonnet-4.5",
    "claude-sonnet-4-6": "claude-sonnet-4.6",
    "claude-sonnet-4.6": "claude-sonnet-4.6",
    "deepseek-3.2": "deepseek-3.2",
    "deepseek-3-2": "deepseek-3.2",
    "glm-5": "glm-5",
    "minimax-m2.1": "minimax-m2.1",
    "minimax-m2-1": "minimax-m2.1",
    "minimax-m2.5": "minimax-m2.5",
    "minimax-m2-5": "minimax-m2.5",
    "qwen3-coder-next": "qwen3-coder-next"
  }'::jsonb AS model_mapping
)
UPDATE accounts
SET credentials = COALESCE(credentials, '{}'::jsonb)
    || jsonb_build_object('model_mapping', mapping.model_mapping),
    updated_at = now()
FROM mapping
WHERE deleted_at IS NULL
  AND (id = 180 OR name = 'Kiro Gateway');

WITH mapping AS (
  SELECT '{
    "auto-kiro": "auto-kiro",
    "claude-haiku-4-5": "claude-haiku-4.5",
    "claude-haiku-4-5-20251001": "claude-haiku-4.5",
    "claude-haiku-4.5": "claude-haiku-4.5",
    "claude-opus-4-5": "claude-opus-4.5",
    "claude-opus-4-5-20251101": "claude-opus-4.5",
    "claude-opus-4.5": "claude-opus-4.5",
    "claude-opus-4-6": "claude-opus-4.6",
    "claude-opus-4.6": "claude-opus-4.6",
    "claude-opus-4-7": "claude-opus-4.7",
    "claude-opus-4-7-20260416": "claude-opus-4.7",
    "claude-opus-4.7": "claude-opus-4.7",
    "claude-sonnet-4": "claude-sonnet-4",
    "claude-sonnet-4-5": "claude-sonnet-4.5",
    "claude-sonnet-4-5-20250929": "claude-sonnet-4.5",
    "claude-sonnet-4.5": "claude-sonnet-4.5",
    "claude-sonnet-4-6": "claude-sonnet-4.6",
    "claude-sonnet-4.6": "claude-sonnet-4.6",
    "deepseek-3.2": "deepseek-3.2",
    "deepseek-3-2": "deepseek-3.2",
    "glm-5": "glm-5",
    "minimax-m2.1": "minimax-m2.1",
    "minimax-m2-1": "minimax-m2.1",
    "minimax-m2.5": "minimax-m2.5",
    "minimax-m2-5": "minimax-m2.5",
    "qwen3-coder-next": "qwen3-coder-next"
  }'::jsonb AS model_mapping
)
UPDATE channels
SET model_mapping = jsonb_build_object('anthropic', mapping.model_mapping),
    updated_at = now()
FROM mapping
WHERE id = 6
   OR name = 'Kiro 专属线路';

SELECT 'account' AS target, id, name, (
  SELECT count(*) FROM jsonb_object_keys(credentials->'model_mapping')
) AS models
FROM accounts
WHERE deleted_at IS NULL
  AND (id = 180 OR name = 'Kiro Gateway')
UNION ALL
SELECT 'channel' AS target, id, name, (
  SELECT count(*) FROM jsonb_object_keys(model_mapping->'anthropic')
) AS models
FROM channels
WHERE id = 6
   OR name = 'Kiro 专属线路'
ORDER BY target, id;
