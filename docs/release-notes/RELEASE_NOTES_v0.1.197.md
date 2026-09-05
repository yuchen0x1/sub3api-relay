# Sub3API v0.1.197

## Features

- Added a React 19 + Vite + Tauri 2 Cockpit control-plane scaffold pinned to `jlcodes99/cockpit-tools` v1.3.34 provenance.
- Added account-owned static proxy exits with a fixed capacity of 10 concurrent requests per exit.
- Added atomic Redis aggregate and per-exit capacity accounting, including live request slots.
- Added static proxy exit administration APIs and account capacity summaries.
- Added aggregate/per-exit capacity display such as `29/100` and `10/10` in the account panel.

## Fixes

- Hardened upstream model synchronization against empty upstream responses and structured API errors.
- Clarified the admin usage log IP column as the client IP and kept it separate from upstream proxy exit IPs.

## Compatibility

- Existing accounts without static proxy exits retain their legacy account-level concurrency and proxy behavior.
- Static proxy routing is enabled on the generic gateway forwarding path; dedicated OpenAI, Gemini, Antigravity, Manus, Codex, and WebSocket paths retain legacy routing until their allocation adapters are enabled.
