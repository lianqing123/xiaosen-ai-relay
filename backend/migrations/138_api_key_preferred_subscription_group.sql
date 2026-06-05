ALTER TABLE api_keys
  ADD COLUMN IF NOT EXISTS preferred_subscription_group_id BIGINT NULL;
