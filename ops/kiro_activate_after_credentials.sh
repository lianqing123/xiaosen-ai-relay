#!/usr/bin/env bash
set -euo pipefail

KIRO_HOME="${KIRO_HOME:-/opt/kiro-gateway}"
cd "$KIRO_HOME"

if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

KIRO_GROUP_NAME="${KIRO_GROUP_NAME:-Kiro 专属线路}"
KIRO_ACCOUNT_NAME="${KIRO_ACCOUNT_NAME:-Kiro Gateway}"
KIRO_CHANNEL_NAME="${KIRO_CHANNEL_NAME:-Kiro 专属线路}"
KIRO_GROUP_OLD_NAME="${KIRO_GROUP_OLD_NAME:-Kiro 测试线路}"
KIRO_ACCOUNT_OLD_NAME="${KIRO_ACCOUNT_OLD_NAME:-Kiro Gateway 内网测试}"
KIRO_ACCOUNT_CONCURRENCY_PER_ACCOUNT="${KIRO_ACCOUNT_CONCURRENCY_PER_ACCOUNT:-20}"
KIRO_ACCOUNT_CONCURRENCY_OVERRIDE="${KIRO_ACCOUNT_CONCURRENCY_OVERRIDE:-}"
KIRO_PUBLIC_DESCRIPTION="${KIRO_PUBLIC_DESCRIPTION:-本站以 codex 为主，claude 为不定时开放。}"
KIRO_GATEWAY_URL="${KIRO_GATEWAY_URL:-http://127.0.0.1:${SERVER_PORT:-8000}}"
KIRO_GATEWAY_API_KEY="${KIRO_GATEWAY_API_KEY:-${PROXY_API_KEY:-}}"

SUB2API_POSTGRES_CONTAINER="${SUB2API_POSTGRES_CONTAINER:-sub2api-postgres}"
SUB2API_REDIS_CONTAINER="${SUB2API_REDIS_CONTAINER:-sub2api-redis}"
SUB2API_DB_USER="${SUB2API_DB_USER:-sub2api}"
SUB2API_DB_NAME="${SUB2API_DB_NAME:-sub2api}"

if [ -z "$KIRO_GATEWAY_API_KEY" ]; then
  echo "kiro proxy api key is missing; set PROXY_API_KEY in $KIRO_HOME/.env"
  exit 1
fi

read -r KIRO_EFFECTIVE_ACCOUNT_COUNT KIRO_ACCOUNT_CONCURRENCY <<EOF
$(KIRO_HOME="$KIRO_HOME" KIRO_ACCOUNT_CONCURRENCY_PER_ACCOUNT="$KIRO_ACCOUNT_CONCURRENCY_PER_ACCOUNT" KIRO_ACCOUNT_CONCURRENCY_OVERRIDE="$KIRO_ACCOUNT_CONCURRENCY_OVERRIDE" python3 - <<'PY'
import json, os, pathlib, sys

def is_active(entry):
    if not isinstance(entry, dict):
        return False
    status = str(entry.get("status", "")).strip().lower()
    if entry.get("disabled") is True or entry.get("enabled") is False or status in {"disabled", "inactive", "error"}:
        return False
    return any(entry.get(k) for k in ("refreshToken", "refresh_token", "accessToken", "access_token", "path", "db_path"))

kiro_home = pathlib.Path(os.environ.get("KIRO_HOME", "/opt/kiro-gateway"))
path = kiro_home / "secrets" / "credentials.json"
if not path.exists():
    sys.exit('credentials.json not found')
data = json.loads(path.read_text(encoding='utf-8'))
if not isinstance(data, list) or not data:
    sys.exit('credentials.json is empty; import at least one Kiro/Amazon Q account first')
active_count = sum(1 for item in data if is_active(item))
if active_count <= 0:
    sys.exit('credentials.json has no active Kiro/Amazon Q account')
per_account = int(os.environ.get("KIRO_ACCOUNT_CONCURRENCY_PER_ACCOUNT", "20"))
override = os.environ.get("KIRO_ACCOUNT_CONCURRENCY_OVERRIDE", "").strip()
concurrency = int(override) if override else active_count * per_account
if concurrency <= 0:
    sys.exit('calculated Kiro account concurrency must be positive')
print(active_count, concurrency)
PY
)
EOF
echo "credentials-ok active_accounts=${KIRO_EFFECTIVE_ACCOUNT_COUNT} per_account_concurrency=${KIRO_ACCOUNT_CONCURRENCY_PER_ACCOUNT} account_concurrency=${KIRO_ACCOUNT_CONCURRENCY}"

start_kiro_gateway() {
  if docker compose version >/dev/null 2>&1; then
    docker compose up -d kiro-gateway
    return
  fi
  if docker ps -a --format '{{.Names}}' | grep -Fxq 'kiro-gateway'; then
    docker start kiro-gateway >/dev/null
    return
  fi
  echo 'docker compose is unavailable and kiro-gateway container does not exist'
  exit 1
}

show_kiro_gateway_logs() {
  if docker compose version >/dev/null 2>&1; then
    docker compose logs --tail=120 kiro-gateway || true
    return
  fi
  docker logs --tail=120 kiro-gateway || true
}

start_kiro_gateway

for i in $(seq 1 45); do
  if curl -fsS "$KIRO_GATEWAY_URL/health" >/dev/null; then
    echo 'kiro-health-ok'
    break
  fi
  if [ "$i" = "45" ]; then
    echo 'kiro-health-failed'
    show_kiro_gateway_logs
    exit 1
  fi
  sleep 2
done

read -r group_id account_id channel_id <<EOF
$(docker exec -i "$SUB2API_POSTGRES_CONTAINER" psql -U "$SUB2API_DB_USER" -d "$SUB2API_DB_NAME" -qAt \
  -v ON_ERROR_STOP=1 \
  -v group_name="$KIRO_GROUP_NAME" \
  -v account_name="$KIRO_ACCOUNT_NAME" \
  -v channel_name="$KIRO_CHANNEL_NAME" \
  -v old_group_name="$KIRO_GROUP_OLD_NAME" \
  -v old_account_name="$KIRO_ACCOUNT_OLD_NAME" \
  -v gateway_url="$KIRO_GATEWAY_URL" \
  -v proxy_api_key="$KIRO_GATEWAY_API_KEY" \
  -v account_concurrency="$KIRO_ACCOUNT_CONCURRENCY" \
  -v public_description="$KIRO_PUBLIC_DESCRIPTION" <<'SQL'
CREATE TEMP TABLE IF NOT EXISTS kiro_activation_result (
  group_id bigint NOT NULL,
  account_id bigint NOT NULL,
  channel_id bigint NOT NULL
) ON COMMIT PRESERVE ROWS;
TRUNCATE kiro_activation_result;

WITH defaults AS (
  SELECT '{
    "auto-kiro": "claude-sonnet-4.6",
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
INSERT INTO groups (
  name, description, rate_multiplier, is_exclusive, status, platform,
  subscription_type, default_validity_days, supported_model_scopes,
  sort_order, created_at, updated_at
)
SELECT
  :'group_name',
  :'public_description',
  1.0,
  false,
  'active',
  'anthropic',
  'standard',
  30,
  '["claude"]'::jsonb,
  80,
  now(),
  now()
WHERE NOT EXISTS (
  SELECT 1
  FROM groups
  WHERE deleted_at IS NULL
    AND name IN (:'group_name', :'old_group_name')
);

WITH target AS (
  SELECT id
  FROM groups
  WHERE deleted_at IS NULL
    AND name IN (:'group_name', :'old_group_name')
  ORDER BY CASE WHEN name = :'group_name' THEN 0 ELSE 1 END, id
  LIMIT 1
)
UPDATE groups
SET name = :'group_name',
    description = :'public_description',
    rate_multiplier = 1.0,
    is_exclusive = false,
    status = 'active',
    platform = 'anthropic',
    subscription_type = 'standard',
    default_validity_days = 30,
    supported_model_scopes = '["claude"]'::jsonb,
    sort_order = 80,
    allow_messages_dispatch = false,
    require_oauth_only = false,
    require_privacy_set = false,
    default_mapped_model = '',
    updated_at = now()
FROM target
WHERE groups.id = target.id;

WITH defaults AS (
  SELECT '{
    "auto-kiro": "claude-sonnet-4.6",
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
INSERT INTO accounts (
  name, platform, type, credentials, extra, concurrency, priority,
  status, schedulable, auto_pause_on_expired, rate_multiplier,
  created_at, updated_at
)
SELECT
  :'account_name',
  'anthropic',
  'apikey',
  jsonb_build_object(
    'api_key', :'proxy_api_key',
    'base_url', :'gateway_url',
    'model_mapping', defaults.model_mapping
  ),
  jsonb_build_object(
    'source', 'kiro-gateway',
    'deployment_path', '/opt/kiro-gateway',
    'hotfix_isolated', true,
    'needs_credentials', false,
    'web_search_emulation', 'disabled',
    'anthropic_passthrough', true
  ),
  :'account_concurrency'::int,
  70,
  'active',
  true,
  true,
  1.0,
  now(),
  now()
FROM defaults
WHERE NOT EXISTS (
  SELECT 1
  FROM accounts
  WHERE deleted_at IS NULL
    AND name IN (:'account_name', :'old_account_name')
);

WITH defaults AS (
  SELECT '{
    "auto-kiro": "claude-sonnet-4.6",
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
),
target AS (
  SELECT id
  FROM accounts
  WHERE deleted_at IS NULL
    AND name IN (:'account_name', :'old_account_name')
  ORDER BY CASE WHEN name = :'account_name' THEN 0 ELSE 1 END, id
  LIMIT 1
)
UPDATE accounts
SET name = :'account_name',
    platform = 'anthropic',
    type = 'apikey',
    credentials = COALESCE(accounts.credentials, '{}'::jsonb)
      || jsonb_build_object(
        'api_key', :'proxy_api_key',
        'base_url', :'gateway_url',
        'model_mapping', defaults.model_mapping
      ),
    extra = COALESCE(accounts.extra, '{}'::jsonb)
      || jsonb_build_object(
        'source', 'kiro-gateway',
        'deployment_path', '/opt/kiro-gateway',
        'hotfix_isolated', true,
        'needs_credentials', false,
        'web_search_emulation', 'disabled',
        'anthropic_passthrough', true
      ),
    concurrency = GREATEST(accounts.concurrency, :'account_concurrency'::int),
    priority = LEAST(accounts.priority, 70),
    status = 'active',
    schedulable = true,
    error_message = NULL,
    rate_limited_at = NULL,
    rate_limit_reset_at = NULL,
    overload_until = NULL,
    temp_unschedulable_until = NULL,
    temp_unschedulable_reason = '',
    updated_at = now()
FROM target, defaults
WHERE accounts.id = target.id;

INSERT INTO account_groups (account_id, group_id, priority, created_at)
SELECT a.id, g.id, 50, now()
FROM (
  SELECT id
  FROM accounts
  WHERE deleted_at IS NULL
    AND name = :'account_name'
  ORDER BY id
  LIMIT 1
) a
CROSS JOIN (
  SELECT id
  FROM groups
  WHERE deleted_at IS NULL
    AND name = :'group_name'
  ORDER BY id
  LIMIT 1
) g
ON CONFLICT (account_id, group_id) DO UPDATE
SET priority = EXCLUDED.priority;

WITH defaults AS (
  SELECT '{
    "auto-kiro": "claude-sonnet-4.6",
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
),
account_source AS (
  SELECT credentials
  FROM accounts
  WHERE deleted_at IS NULL
    AND name = :'account_name'
  ORDER BY id
  LIMIT 1
)
INSERT INTO channels (
  name, description, status, model_mapping, billing_model_source,
  restrict_models, features, features_config, apply_pricing_to_account_stats
)
SELECT
  :'channel_name',
  :'public_description',
  'active',
  jsonb_build_object('anthropic', defaults.model_mapping),
  'channel_mapped',
  false,
  '["Kiro Gateway","官方登录","独立线路"]',
  '{"source":"kiro-gateway"}'::jsonb,
  false
FROM account_source CROSS JOIN defaults
WHERE NOT EXISTS (
  SELECT 1
  FROM channels
  WHERE name = :'channel_name'
);

WITH defaults AS (
  SELECT '{
    "auto-kiro": "claude-sonnet-4.6",
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
),
account_source AS (
  SELECT credentials
  FROM accounts
  WHERE deleted_at IS NULL
    AND name = :'account_name'
  ORDER BY id
  LIMIT 1
)
UPDATE channels
SET name = :'channel_name',
    description = :'public_description',
    status = 'active',
    model_mapping = jsonb_build_object('anthropic', defaults.model_mapping),
    billing_model_source = 'channel_mapped',
    restrict_models = false,
    features = '["Kiro Gateway","官方登录","独立线路"]',
    features_config = '{"source":"kiro-gateway"}'::jsonb,
    apply_pricing_to_account_stats = false,
    updated_at = now()
FROM account_source CROSS JOIN defaults
WHERE channels.name = :'channel_name';

DELETE FROM channel_groups
WHERE group_id = (
  SELECT id FROM groups WHERE deleted_at IS NULL AND name = :'group_name' ORDER BY id LIMIT 1
)
AND channel_id <> (
  SELECT id FROM channels WHERE name = :'channel_name' ORDER BY id LIMIT 1
);

INSERT INTO channel_groups (channel_id, group_id, created_at)
SELECT c.id, g.id, now()
FROM (
  SELECT id FROM channels WHERE name = :'channel_name' ORDER BY id LIMIT 1
) c
CROSS JOIN (
  SELECT id FROM groups WHERE deleted_at IS NULL AND name = :'group_name' ORDER BY id LIMIT 1
) g
ON CONFLICT (group_id) DO UPDATE
SET channel_id = EXCLUDED.channel_id;

INSERT INTO account_groups (account_id, group_id, priority, created_at)
SELECT a.id, g.id, 50, now()
FROM accounts a
CROSS JOIN (
  SELECT id, platform
  FROM groups
  WHERE deleted_at IS NULL
    AND name = :'group_name'
  ORDER BY id
  LIMIT 1
) g
WHERE a.deleted_at IS NULL
  AND a.platform = g.platform
  AND (
    a.name ILIKE '%kiro%'
    OR COALESCE(a.extra->>'source','') = 'kiro-gateway'
    OR COALESCE(a.extra->>'deployment_path','') = '/opt/kiro-gateway'
    OR COALESCE(a.credentials->>'base_url','') = :'gateway_url'
  )
ON CONFLICT (account_id, group_id) DO UPDATE
SET priority = EXCLUDED.priority;

INSERT INTO settings (key, value, updated_at)
VALUES ('available_channels_enabled', 'true', now())
ON CONFLICT (key) DO UPDATE
SET value = 'true', updated_at = now();

INSERT INTO kiro_activation_result (group_id, account_id, channel_id)
SELECT g.id, a.id, c.id
FROM (
  SELECT id FROM groups WHERE deleted_at IS NULL AND name = :'group_name' ORDER BY id LIMIT 1
) g
CROSS JOIN (
  SELECT id FROM accounts WHERE deleted_at IS NULL AND name = :'account_name' ORDER BY id LIMIT 1
) a
CROSS JOIN (
  SELECT id FROM channels WHERE name = :'channel_name' ORDER BY id LIMIT 1
) c;

SELECT group_id || ' ' || account_id || ' ' || channel_id
FROM kiro_activation_result
LIMIT 1;
SQL
)
EOF

if [ -z "${group_id:-}" ] || [ -z "${account_id:-}" ] || [ -z "${channel_id:-}" ]; then
  echo 'failed to provision kiro group/account/channel in sub2api database'
  exit 1
fi

docker exec "$SUB2API_REDIS_CONTAINER" env -u REDISCLI_AUTH redis-cli DEL \
  "sched:acc:${account_id}" \
  "sched:meta:${account_id}" \
  "sched:ready:${group_id}:anthropic:mixed" \
  "sched:active:${group_id}:anthropic:mixed" >/dev/null || true

echo "sub2api-kiro-enabled group_id=${group_id} account_id=${account_id} channel_id=${channel_id} group='${KIRO_GROUP_NAME}' account='${KIRO_ACCOUNT_NAME}'"
