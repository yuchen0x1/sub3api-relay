-- Recorded gateway request bodies for opt-in debugging and replay.
CREATE TABLE IF NOT EXISTS gateway_body_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    path VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    model VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS gateway_body_logs_request_id_key ON gateway_body_logs(request_id);
CREATE INDEX IF NOT EXISTS gateway_body_logs_api_key_id_idx ON gateway_body_logs(api_key_id);
CREATE INDEX IF NOT EXISTS gateway_body_logs_user_id_idx ON gateway_body_logs(user_id);
CREATE INDEX IF NOT EXISTS gateway_body_logs_created_at_idx ON gateway_body_logs(created_at);
