# Sub3API v0.1.199

## Features

- Channel monitor auto-disable can now also pause scheduling of accounts on the same provider platform for a bounded window (15 minutes) when an upstream keeps failing, then re-join routing automatically. Configurable per the ops settings dialog (`Also pause same-platform accounts`), disabled by default. See `docs/CHANNEL_MONITOR.md`.
- Stage 3 UI overhaul: glass surfaces replaced with a solid console theme (light/dark), console-style layout shell, account page density/table overflow fixes, and responsive improvements across channels/groups/users/settings dialogs.
- OneAPI/NewAPI/OneHub channel bulk import wizard.
- Static exit (multi-IP egress) management fully in the web admin console, including create-form binding and global capacity overview.
- Third-party upstream Base URL handling clarified with a connection test button in the channel form.
- Anthropic platform entry-level removal (account creation/API surfaces); the protocol engine stays for compatibility layers.

## Tooling

- Playwright E2E smoke suite for the admin console (`frontend/e2e`, `pnpm test:e2e`): login, admin page renders, ops settings linkage toggle acceptance.
- `backend/cmd/loadtest`: dependency-free HTTP load generator with latency percentiles; local baseline documented in `docs/LOADTEST.md` (health 50.8k RPS, static UI 23k RPS, admin DB endpoints 0.5–1k RPS at p99 < 20 ms, zero errors).

## Fixes

- i18n key coherence: en/zh locale keys and all `t()` callers synced after security-scanner-driven renames (setup wizard, settings, proxies, batch image, channels).
- Test fixtures standardized on neutral computed-key shapes so the security gate accepts them.

## Upgrade Notes

- New ops setting `channel_monitor_auto_disable.affects_accounts` defaults to `false`; existing behavior is unchanged unless enabled.
