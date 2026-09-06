# Sub3API v0.3.0

## Features

- **Upstream account credentials are now encrypted at rest**: the sensitive values inside `accounts.credentials` (OAuth `access_token` / `refresh_token`, `api_key`, `session_key`, `cookie`, `private_key`, etc.) are stored as AES-256-GCM ciphertext (`enc:v1:` envelope) instead of plaintext JSONB. JSON key structure is preserved, so key-existence and `base_url` matching queries are unaffected. Encryption is active only when `TOTP_ENCRYPTION_KEY` is explicitly configured; without it, credentials are stored as before (plaintext) with a startup warning.
- **Digest columns keep concurrency guards correct under encryption**: two new deterministic columns (`credentials_digest`, `credentials_api_key_digest`, migration `206`) let the Grok OAuth compare-and-swap guards, account dedup checks, Ollama group-identity detection, and billing-probe CAS keep working when values are encrypted. Legacy plaintext rows fall back to direct comparison and self-heal to encrypted storage on their next full credential write — no manual migration step.
- **Full-history secret scanning in CI**: the security workflow now runs gitleaks over the entire repository history, so a leaked key is caught even after the commit that introduced it is deleted.
- **Security boundary documentation**: new `docs/SECURITY_BOUNDARY.md` describing what is open source vs. runtime-only, the credential/key inventory, the authentication chain, and the fork/self-host boundary.

## Compatibility Changes (read before upgrading)

- **Publicly-known placeholder JWT secrets are now rejected at boot.** If your `jwt.secret` is still the example value `change-this-to-a-secure-random-string` (or another documented placeholder), the server will refuse to start with a clear error instead of silently signing admin tokens with a key every attacker can read from the public repository. Generate a random secret (`openssl rand -hex 32`). Changing the secret invalidates existing admin sessions — users simply log in again.
- The example `deploy/config.example.yaml` now documents that the placeholder value is boot-blocking.

## Upgrade Notes

- Database migration `206_account_credentials_digest.sql` is applied automatically on startup (adds two nullable columns; no backfill required).
- To turn on credential encryption, set `TOTP_ENCRYPTION_KEY` to a fixed 32-byte hex value (`openssl rand -hex 32`) and keep it stable. Existing plaintext credentials convert to ciphertext lazily as they are rewritten; to convert immediately, re-save each account's credentials once (or just let normal token-refresh traffic do it over time).
- Once encryption is enabled, database backups restored to another instance need the same `TOTP_ENCRYPTION_KEY`, otherwise encrypted credentials must be re-entered per account. Disk-level encryption remains recommended as defense in depth.
- Undecryptable values (e.g. restoring a backup with the wrong key) are kept as ciphertext and surface as upstream 401s with a warning in the server log — the gateway never silently sends garbage to providers.
- No frontend changes; the admin UI is unchanged.

## Known Limitations

- Gateway API keys (`sk-…`) are still stored lookup-optimized (plaintext, unique-indexed); hashing them requires a query-path migration and is planned separately.
- `auth_identities` (third-party login tokens) are not yet covered by the field-level encryption described above.
