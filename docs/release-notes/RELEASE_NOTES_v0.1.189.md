# Sub3API v0.1.189

## Promo card redemption and API key delivery

This release completes the homepage card-code flow for distributing prepaid API keys.

### Added
- Added a simplified promo-card generator that sets credit, per-key concurrency, card expiry, and quantity in one form.
- Added an immediate success dialog with the full API base URL and one-time API key display after redemption.
- Added balance lookup by either card code or API key.
- Added admin columns for promo-card concurrency, used credit, remaining credit, linked API key preview, and key status.

### Fixed
- Fixed lowercase legacy card codes returning `invalid card code` on the public homepage.
- Fixed card redemption so a card is not consumed when API key creation fails.
- Fixed API key balance lookup to read current database usage instead of stale authentication cache data.
- Fixed balance lookup selecting an old API key when a new card code is entered.
- Fixed clipboard feedback when the browser clipboard API is unavailable.

### Compatibility
- Existing unused balance cards remain redeemable and receive a concurrency limit of 1 when converted to an API key.
- Credit-exhausted API keys become unusable but remain recorded for balance queries and administrator auditing.
- No database migration is required.

### Verification
- `go test ./internal/service ./internal/repository ./internal/handler/admin -run "PublicCard|PromoConcurrency|Redeem" -count=1`
- `go test -tags=integration ./internal/repository -run "TestRedeemCodeRepoSuite/TestGetByCode_(IgnoresCaseForLegacyCodes|PreloadsPromoAPIKeyUsage)" -count=1`
- `pnpm typecheck`
- `pnpm build`
