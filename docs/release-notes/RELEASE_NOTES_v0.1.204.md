# Sub3API v0.1.204

## Features

- **Automatic body log retention**: gateway body logs are now purged automatically by a background cleanup task (every 10 minutes, starting right after boot). Previously the `gateway_body_logs` table grew without bound while the feature was enabled — the delete helper existed but nothing called it. Retention is configurable via the new `gateway_body_log_retention_days` setting (default 7, 0 = keep forever), exposed as a number input in Settings → Gateway → Scheduling next to the body-logging toggle.

## Fixes

- **Settings round-trip**: `GET /api/v1/admin/settings` now returns `gateway_scheduling_selection_mode` and `gateway_body_log_enabled`. Both were missing from the read payload, so the account-selection dropdown and the body-logging toggle silently reset to their defaults every time the settings page was reloaded, even though the values were persisted correctly.
- **Platform-wide monitor pause no longer penalizes custom upstreams**: when a channel monitor's consecutive-failure linkage suspends scheduling for a provider, accounts with a custom `credentials.base_url` (independent upstreams such as relays) are now excluded — the pause only targets accounts served by the default official endpoint that the monitor actually probes.
- **Landing page**: the terminal mock no longer hardcodes a stale version badge (it is replaced with the live version at runtime; the static fallback is now neutral) and the curl example no longer points at the Vite dev-server port.
- Reduced log noise: an unconfigured body-log retention setting (existing databases after upgrade) falls back to the default silently instead of logging a warning every cache refresh.

## Upgrade Notes

- No database migrations. The retention setting defaults to 7 days on upgrade; existing body logs older than that are deleted on the first cleanup pass after startup. Set `gateway_body_log_retention_days` to 0 to keep everything.
- No configuration changes required.
