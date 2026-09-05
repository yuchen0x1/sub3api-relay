-- Account-owned static proxy assignments.
-- The assignment points at an existing proxy and has a fixed capacity of ten.

CREATE TABLE IF NOT EXISTS account_static_proxy_exits (
    id                  BIGSERIAL PRIMARY KEY,
    account_id          BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    proxy_id            BIGINT NOT NULL REFERENCES proxies(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,
    position            INT NOT NULL DEFAULT 0 CHECK (position >= 0),
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    capacity            INT NOT NULL DEFAULT 10 CHECK (capacity = 10),
    health_status       VARCHAR(20) NOT NULL DEFAULT 'unknown',
    failure_count       INT NOT NULL DEFAULT 0 CHECK (failure_count >= 0),
    last_failure_at     TIMESTAMPTZ,
    last_failure_reason TEXT,
    last_checked_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS account_static_proxy_exits_account_id_idx
    ON account_static_proxy_exits(account_id);
CREATE INDEX IF NOT EXISTS account_static_proxy_exits_proxy_id_idx
    ON account_static_proxy_exits(proxy_id);
CREATE INDEX IF NOT EXISTS account_static_proxy_exits_account_enabled_idx
    ON account_static_proxy_exits(account_id, enabled);
CREATE INDEX IF NOT EXISTS account_static_proxy_exits_account_position_idx
    ON account_static_proxy_exits(account_id, position);
CREATE INDEX IF NOT EXISTS account_static_proxy_exits_health_status_idx
    ON account_static_proxy_exits(health_status);
CREATE INDEX IF NOT EXISTS account_static_proxy_exits_deleted_at_idx
    ON account_static_proxy_exits(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS account_static_proxy_exits_account_proxy_active_idx
    ON account_static_proxy_exits(account_id, proxy_id)
    WHERE deleted_at IS NULL;
