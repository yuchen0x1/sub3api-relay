# Sub3API v0.1.186

## Public card redemption and API key balance query

This release adds a public card-code redemption flow for promotion and trial distribution.

### Added
- Public card redemption endpoint: redeem a one-time balance card into a generated API key.
- Public API key balance query endpoint: check status, quota, used amount, remaining amount, expiry, and last-used time.
- Homepage card panel: visitors can redeem a card code, copy the generated SK, and query remaining SK balance without logging in.
- Safe separation between card codes and API keys: card codes are one-time vouchers, while generated SK values remain the actual credentials.

### Security
- Public card endpoints are registered under the existing public IP rate limiter.
- Public query responses expose only key status and quota information, not user email, admin data, or account details.
- Generated promotional users are isolated from the admin login flow.
- Exhausted keys are blocked by the existing API key quota-exhaustion flow while preserving auditability.

### Notes
- Only positive balance redeem codes can be redeemed through the public homepage.
- Full SK values are shown only once immediately after redemption. Users should save them immediately.
- Existing admin redeem-code generation remains the source of card codes.

### Verification
- go test ./internal/service -run '^$' -count=1
- go test ./internal/handler ./internal/server/routes -run '^$' -count=1
- go test ./cmd/server -run '^$' -count=1
- pnpm typecheck
- pnpm build
