-- Restore subscriptions that were incorrectly split by multi-group subscription
-- fulfillment. The fixed fulfillment path only renews the primary
-- subscription_group_id; included groups remain a plan/routing relation and
-- must not become separate user subscriptions.
--
-- This repair is intentionally narrow and idempotent:
--   1. It only targets completed subscription orders whose snapshot contains
--      multiple subscription_group_ids.
--   2. It only touches non-primary group subscriptions whose notes contain the
--      exact "payment order <id>" line from that order.
--   3. If the split subscription was created only by those bad fulfillments, it
--      is soft-deleted. If it existed before, only the wrongly-added days are
--      subtracted and the bad order note lines are removed.

CREATE TEMP TABLE tmp_restore_split_subscription_raw ON COMMIT DROP AS
SELECT DISTINCT
  po.id AS order_id,
  po.user_id,
  po.subscription_group_id AS primary_group_id,
  split_group.group_id AS split_group_id,
  po.subscription_days::integer AS subscription_days,
  COALESCE(po.completed_at, po.updated_at, po.paid_at, po.created_at) AS fulfilled_at,
  us.id AS split_subscription_id,
  ps.id AS primary_subscription_id
FROM payment_orders po
CROSS JOIN LATERAL (
  SELECT value::bigint AS group_id
  FROM jsonb_array_elements_text(COALESCE(po.subscription_group_ids, '[]'::jsonb)) AS t(value)
) split_group
JOIN user_subscriptions us
  ON us.user_id = po.user_id
 AND us.group_id = split_group.group_id
 AND us.deleted_at IS NULL
JOIN user_subscriptions ps
  ON ps.user_id = po.user_id
 AND ps.group_id = po.subscription_group_id
 AND ps.deleted_at IS NULL
WHERE po.order_type = 'subscription'
  AND po.status = 'COMPLETED'
  AND po.subscription_group_id IS NOT NULL
  AND po.subscription_days IS NOT NULL
  AND po.subscription_days > 0
  AND jsonb_typeof(COALESCE(po.subscription_group_ids, '[]'::jsonb)) = 'array'
  AND jsonb_array_length(COALESCE(po.subscription_group_ids, '[]'::jsonb)) > 1
  AND split_group.group_id <> po.subscription_group_id
  AND EXISTS (
    SELECT 1
    FROM regexp_split_to_table(COALESCE(us.notes, ''), E'\n') AS note_line(line)
    WHERE btrim(note_line.line) = 'payment order ' || po.id::text
  );

CREATE TEMP TABLE tmp_restore_split_subscription_repair ON COMMIT DROP AS
WITH aggregated AS (
  SELECT
    split_subscription_id,
    primary_subscription_id,
    MIN(user_id) AS user_id,
    MIN(split_group_id) AS split_group_id,
    SUM(subscription_days)::integer AS days_to_remove,
    MIN(fulfilled_at) AS first_fulfilled_at,
    ARRAY_AGG(DISTINCT order_id ORDER BY order_id) AS order_ids,
    ARRAY_AGG(DISTINCT 'payment order ' || order_id::text ORDER BY 'payment order ' || order_id::text) AS target_notes
  FROM tmp_restore_split_subscription_raw
  GROUP BY split_subscription_id, primary_subscription_id
)
SELECT
  a.*,
  us.expires_at AS split_expires_at,
  us.status AS split_status,
  us.notes AS split_notes,
  NOT EXISTS (
    SELECT 1
    FROM regexp_split_to_table(COALESCE(us.notes, ''), E'\n') AS note_line(line)
    WHERE btrim(note_line.line) <> ''
      AND NOT (btrim(note_line.line) = ANY(a.target_notes))
  ) AS created_only_by_split_orders
FROM aggregated a
JOIN user_subscriptions us ON us.id = a.split_subscription_id
WHERE a.days_to_remove > 0;

-- Usage that landed on the incorrectly-created/extended included subscription
-- after the bad fulfillment belongs to the primary package subscription.
UPDATE usage_logs ul
SET subscription_id = r.primary_subscription_id
FROM tmp_restore_split_subscription_repair r
WHERE ul.subscription_id = r.split_subscription_id
  AND ul.created_at >= r.first_fulfilled_at
  AND r.primary_subscription_id IS NOT NULL;

-- Included subscriptions that contain only the bad payment-order notes were
-- created by the split fulfillment itself. Soft-delete them so the unique
-- active (user_id, group_id) constraint remains reusable.
UPDATE user_subscriptions us
SET
  deleted_at = COALESCE(us.deleted_at, NOW()),
  status = CASE WHEN us.status = 'active' THEN 'expired' ELSE us.status END,
  updated_at = NOW(),
  notes = concat_ws(
    E'\n',
    NULLIF(us.notes, ''),
    '[repair] restored split subscription fulfillment; moved usage to primary subscription and soft-deleted included-group subscription'
  )
FROM tmp_restore_split_subscription_repair r
WHERE us.id = r.split_subscription_id
  AND r.created_only_by_split_orders = TRUE
  AND us.deleted_at IS NULL;

-- If the included subscription existed before the bad fulfillment, undo only
-- the extra days added by those orders and remove the bad order-note lines.
WITH kept_notes AS (
  SELECT
    r.split_subscription_id,
    string_agg(note_line.line, E'\n' ORDER BY note_line.ordinality) AS notes
  FROM tmp_restore_split_subscription_repair r
  CROSS JOIN LATERAL regexp_split_to_table(COALESCE(r.split_notes, ''), E'\n')
    WITH ORDINALITY AS note_line(line, ordinality)
  WHERE btrim(note_line.line) <> ''
    AND NOT (btrim(note_line.line) = ANY(r.target_notes))
  GROUP BY r.split_subscription_id
)
UPDATE user_subscriptions us
SET
  expires_at = us.expires_at - make_interval(days => r.days_to_remove),
  status = CASE
    WHEN us.expires_at - make_interval(days => r.days_to_remove) <= NOW() THEN 'expired'
    ELSE us.status
  END,
  updated_at = NOW(),
  notes = concat_ws(
    E'\n',
    NULLIF(kn.notes, ''),
    '[repair] restored split subscription fulfillment; removed ' || r.days_to_remove::text || ' wrongly-added day(s) from included-group subscription'
  )
FROM tmp_restore_split_subscription_repair r
LEFT JOIN kept_notes kn ON kn.split_subscription_id = r.split_subscription_id
WHERE us.id = r.split_subscription_id
  AND r.created_only_by_split_orders = FALSE
  AND us.deleted_at IS NULL;
