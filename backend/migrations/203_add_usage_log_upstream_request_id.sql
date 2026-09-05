ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS upstream_request_id VARCHAR(200);
