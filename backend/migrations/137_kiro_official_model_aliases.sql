-- Add official Claude model IDs to the Kiro Gateway account/channel mappings.
-- Kiro Gateway exposes compact model IDs like claude-haiku-4.5, while some
-- clients send Anthropic's full IDs such as claude-haiku-4-5-20251001.

WITH aliases AS (
  SELECT '{
    "claude-haiku-4-5-20251001": "claude-haiku-4.5",
    "claude-opus-4-5-20251101": "claude-opus-4.5",
    "claude-opus-4-7-20260416": "claude-opus-4.7",
    "claude-sonnet-4-5-20250929": "claude-sonnet-4.5"
  }'::jsonb AS model_mapping
),
updated_accounts AS (
  UPDATE accounts
  SET credentials = jsonb_set(
        COALESCE(credentials, '{}'::jsonb),
        '{model_mapping}',
        COALESCE(credentials->'model_mapping', '{}'::jsonb) || aliases.model_mapping,
        true
      ),
      updated_at = now()
  FROM aliases
  WHERE deleted_at IS NULL
    AND (id = 180 OR name = 'Kiro Gateway')
  RETURNING id
),
updated_channels AS (
  UPDATE channels AS c
  SET model_mapping = jsonb_set(
        COALESCE(c.model_mapping, '{}'::jsonb),
        '{anthropic}',
        COALESCE(c.model_mapping->'anthropic', '{}'::jsonb) || aliases.model_mapping,
        true
      ),
      updated_at = now()
  FROM aliases
  WHERE c.id = 6 OR c.name = 'Kiro 专属线路'
  RETURNING id
)
INSERT INTO scheduler_outbox (event_type, account_id, payload)
SELECT 'account_changed', id, '{"group_ids":[24,25]}'::jsonb
FROM updated_accounts;
