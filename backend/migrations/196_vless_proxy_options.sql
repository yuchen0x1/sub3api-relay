-- Store transport-specific proxy options (for example VLESS WebSocket SNI,
-- Host header and path) without overloading username/password columns.
ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS options JSONB NOT NULL DEFAULT '{}'::jsonb;
