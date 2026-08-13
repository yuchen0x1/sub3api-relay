# Sub3API v0.1.187

## Card code admin entry

This release fixes the missing admin entry for the card-code generation page.

### Fixed
- Restored the admin route for card-code management at `/admin/redeem`.
- Added aliases `/admin/cards` and `/admin/card-codes` for easier direct access.
- Added the `Card Codes` / `卡密管理` item to the administrator sidebar.
- Updated Chinese admin labels from “兑换码” to “卡密” on the management page so the generation flow is easier to find.

### Verification
- pnpm typecheck
- pnpm vitest run src/components/layout/__tests__/AppSidebar.spec.ts src/router/__tests__/guards.spec.ts src/composables/__tests__/useRoutePrefetch.spec.ts
- pnpm build
