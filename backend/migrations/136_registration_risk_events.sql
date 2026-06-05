CREATE TABLE IF NOT EXISTS registration_risk_events (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
  email TEXT NOT NULL,
  email_domain TEXT NOT NULL DEFAULT '',
  email_shape TEXT NOT NULL DEFAULT 'normal',
  client_ip INET NOT NULL,
  status TEXT NOT NULL DEFAULT 'success',
  reason TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_registration_risk_events_client_ip_created_at
  ON registration_risk_events (client_ip, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_registration_risk_events_shape_ip_created_at
  ON registration_risk_events (email_shape, client_ip, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_registration_risk_events_user_id
  ON registration_risk_events (user_id);
