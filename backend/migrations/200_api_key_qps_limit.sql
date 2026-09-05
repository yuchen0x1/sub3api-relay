-- Per-key request rate limiting: maximum requests per second (0 = unlimited).
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS qps_limit INT NOT NULL DEFAULT 0;
