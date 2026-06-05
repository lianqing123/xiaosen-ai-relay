ALTER TABLE subscription_plans
  ADD COLUMN IF NOT EXISTS included_group_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE payment_orders
  ADD COLUMN IF NOT EXISTS subscription_group_ids JSONB NOT NULL DEFAULT '[]'::jsonb;
