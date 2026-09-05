# Sub3API v0.1.188

## Card code API recovery

This hotfix restores the administrator API routes used by the card-code management page.

### Fixed
- Restored card-code list, statistics, export, generation, batch update, deletion, and expiry endpoints under `/api/v1/admin/redeem-codes`.
- Fixed the card-code management page showing `Failed to load card code list` after upgrading to v0.1.187.
- Fixed card-code generation actions returning HTTP 404.
- Added a route registration regression test so the admin page and backend API cannot silently drift apart again.

### Verification
- `go test ./internal/server/routes -count=1`
- `go test ./internal/handler/admin -count=1`
