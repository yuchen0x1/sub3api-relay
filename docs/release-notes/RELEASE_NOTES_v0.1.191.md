# Sub3API v0.1.191

This release syncs the Grok/xAI platform with the latest upstream capabilities, adds a new per-call search endpoint, introduces a fourth channel billing base, and fixes an availability bug where HTML 403 block pages could disable healthy accounts.

## New Features

- Grok 4.6: add `grok-4.6` / `grok-4.6-latest` to the model catalog with official pricing ($2 input / $0.50 cached input / $6 output per MTok). Requests with ≥200k total context are billed at the doubled long-context tier ($4 / $1 / $12 per MTok). Configurable reasoning effort is enabled for the new models. Existing `grok` / `grok-latest` aliases keep pointing to grok-4.5.
- Grok subscription tier from JWT: OAuth token refresh now decodes the tier claim (`subscription_tier`, falling back to `tier`) from the access/id token. A fresh tier overrides a stale stored subscription automatically; the stored credential is only used when the JWT carries no tier.
- Standalone `POST /x_search` endpoint (Grok groups only, available at `/v1/x_search` and `/x_search`): wraps the request into a Grok Responses turn with the native `x_search` tool and returns `{output, results}` JSON. It reuses the web_search audit, profit control, failover, and per-call billing pipeline. `model` is optional and defaults to the `grok` alias.
- New channel billing model source `upstream_response` ("bill by upstream response model"): when selected, pricing looks up the model the upstream actually declared in its response, catching silent upstream model swaps. Responses without an observed model keep the existing billing base. Available for both the OpenAI and Anthropic gateway billing paths, with admin UI and i18n (EN/CN) support.

## Bug Fixes

- HTML 403 block pages (WAF/CDN/IP risk-control pages) no longer mark accounts as errored. Any platform returning an HTML 403 body now triggers a short temporary unschedulable cooldown (5 minutes) instead of a permanent account disable, preventing mass account offline events. Structured API 403 responses keep their existing platform-specific handling.

## Upgrade Notes

- No database migrations are required.
- Accounts previously disabled by an "Access forbidden (403)" error that came from an HTML block page are not auto-recovered; re-enable them from the admin accounts page if needed.
- The `upstream_response` billing source is opt-in per channel; existing channels keep their current billing base.
- The Codex fingerprint convergence default for unconfigured accounts is unchanged in this release (implicit session-level convergence).

## Known Limitations

- Realtime audio-token surcharge billing is not implemented; `/v1/live` usage is metered with standard token counts.
