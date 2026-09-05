-- Versioned snapshots of the whole settings table, taken before each admin save.
CREATE TABLE IF NOT EXISTS setting_revisions (
    id BIGSERIAL PRIMARY KEY,
    snapshot TEXT NOT NULL,
    changed_by BIGINT NOT NULL DEFAULT 0,
    note VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
