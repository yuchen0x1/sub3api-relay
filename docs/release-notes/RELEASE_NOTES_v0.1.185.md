# Sub3API v0.1.185

## Fixes

- Fixed the public homepage status panel showing `--` when the visitor is not logged in.
- The homepage now fills service availability, connected model count, gateway latency, channel summary, and recent public status activity from channel-monitor summaries.
- Sensitive admin-only data remains private: request details, user usage records, API keys, costs, and operations dashboard metrics are still not exposed to anonymous visitors.

## Verification

- `pnpm typecheck`
- Frontend test suite
- `pnpm build`
