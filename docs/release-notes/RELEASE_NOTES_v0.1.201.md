# Sub3API v0.1.201

## Fixes

- Upstream 401/403 rejection reasons are now surfaced to admins instead of being reduced to a bare status code. The account "test connection" error now reads e.g. `authentication failed: upstream rejected the API key (HTTP 403: {"error":{"message":"Invalid token ..."}})`, and "sync upstream models" toasts include the upstream body for 401/403. Response bodies are condensed (control characters stripped, capped at 300 chars). Other status codes (5xx etc.) remain status-only to avoid echoing potentially sensitive upstream internals.

## Diagnosis Notes

- The earlier `host is not allowed` failures are gone since v0.1.200 (upstream URL allowlist removed). If an upstream still rejects with 403 while the same endpoint returns 401 "Invalid token" for bad keys from other networks, the rejection is upstream-side: egress IP ban / region block (common for CN relays behind Tencent EdgeOne) or a token-level restriction (IP whitelist, disabled token, group permission). The now-visible upstream body states the exact reason.
