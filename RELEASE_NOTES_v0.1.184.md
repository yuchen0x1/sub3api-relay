# Sub3API v0.1.184

## Security Hotfixes

- Added server-side brute-force protection for `POST /api/v1/auth/login`.
- Repeated failed password attempts now temporarily lock both the client IP and the targeted login identifier.
- Locked login attempts return HTTP `429`, `AUTH_LOGIN_TEMPORARILY_LOCKED`, and a `Retry-After` header so clients and logs clearly show the block.
- Reduced the public login route's fallback request limit from 20/minute to 8/minute.
- Successful password login clears the related failure counters for that IP and login identifier.

## Why It Matters

- This protects the administrator panel from scripts repeatedly testing passwords.
- The protection runs in-process as a fallback, so single-node deployments stay protected even if Redis is unavailable.
- Login identifiers are hashed before being stored in the failure tracker.

## Verification

- `go test ./...`
- `pnpm typecheck`
- `pnpm build`
