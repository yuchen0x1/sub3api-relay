-- Versioned HMAC digests prevent offline guessing of low-entropy account
-- credentials while preserving CAS compatibility for legacy SHA-256 rows.
-- The application backfills existing rows in bounded, CAS-protected batches
-- after the persistent JWT secret is loaded; NULL marks legacy rows.
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS credentials_digest_version SMALLINT;

COMMENT ON COLUMN accounts.credentials_digest_version IS
    'Credential digest format version: NULL/1=legacy SHA-256, 2=domain-separated HMAC-SHA-256';

COMMENT ON COLUMN accounts.credentials_digest IS
    'HMAC-SHA-256 hex of canonical plaintext credentials when version=2; legacy rows use SHA-256';

COMMENT ON COLUMN accounts.credentials_api_key_digest IS
    'HMAC-SHA-256 hex of canonical credentials.api_key when version=2; legacy rows use SHA-256';
