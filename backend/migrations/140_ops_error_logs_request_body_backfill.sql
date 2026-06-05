-- Backfill request body columns for existing ops_error_logs tables created before
-- 033_ops_monitoring_vnext included retry request-body storage.

ALTER TABLE ops_error_logs
  ADD COLUMN IF NOT EXISTS request_body JSONB;

ALTER TABLE ops_error_logs
  ADD COLUMN IF NOT EXISTS request_body_truncated BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE ops_error_logs
  ADD COLUMN IF NOT EXISTS request_body_bytes INT;

COMMENT ON COLUMN ops_error_logs.request_body IS 'Sanitized request body captured for failed requests and retry.';
COMMENT ON COLUMN ops_error_logs.request_body_truncated IS 'Whether request_body was truncated before storage.';
COMMENT ON COLUMN ops_error_logs.request_body_bytes IS 'Original sanitized request body byte size.';
