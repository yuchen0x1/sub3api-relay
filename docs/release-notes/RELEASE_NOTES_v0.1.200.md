# Sub3API v0.1.200

## Removals

- **Upstream URL allowlist removed entirely.** The `security.url_allowlist` configuration block (`enabled`, `upstream_hosts`, `pricing_hosts`, `crs_hosts`, `allow_private_hosts`, `allow_insecure_http`) and all enforcement were deleted. Any valid `http`/`https` URL is now accepted as an upstream Base URL: official APIs, third-party relays, and self-hosted gateways — including `http://localhost`, LAN IPs, and plaintext HTTP — with no configuration required. The account "test connection" probe no longer rejects custom domains such as `token.sensenova.cn`.
- Outbound requests no longer perform DNS resolved-IP (SSRF) re-checks; the per-request DNS validation transport in the shared HTTP client pool was removed as well.

## Upgrade Notes

- Existing `config.yaml` files or environments that still define `SECURITY_URL_ALLOWLIST_*` variables are unaffected: unknown keys are ignored silently and can be cleaned up at leisure.
- Operator-owned hardening that is unrelated to the removed allowlist is preserved: Grok OAuth trusted-host validation, the Gemini download-host guard, proxy subscription fetch protections (HTTPS-only + private-IP dialer guard), and API Key `ip_whitelist` / account model whitelists.
- Because upstreams may now be plaintext HTTP by design, prefer HTTPS upstreams in production to avoid credential interception and MITM exposure.

## Fixes

- Pre-existing unit-tag test gaps addressed: `stubAccountRepo` now implements `SuspendAccountsByPlatform`; xai runtime sanity policy text updated to match the open upstream policy.
