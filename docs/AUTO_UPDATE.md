# Automatic updates

Sub3API currently publishes a Web management console, standalone server binaries, and Docker deployment assets. It does not currently publish a Windows, Linux, or macOS desktop application.

## Supported update paths

| Deployment | Detection | Install path | Verification |
| --- | --- | --- | --- |
| Standalone release binary | Admin version panel or `deploy/update-binary.sh` | Download archive, verify, backup, replace, health check | SHA-256 from `checksums.txt` |
| Docker / source build | Operator or deployment tooling | Update the pinned image or source checkout | Release metadata and deployment health checks |
| Web management console | Browser/service worker | Refresh the embedded Web bundle after server update | Build and deployment health checks |

## Optional Tauri scaffold

The repository contains an optional Tauri shell under `desktop/`. It is not currently a supported product release and is not part of the formal GitHub Release pipeline. Therefore the current release does not require Tauri signing secrets, desktop installers, `.sig` files, or `latest.json`.

If a future product decision introduces an official desktop edition, its updater must be reintroduced as a separate release track with signed installers, platform metadata, `latest.json`, and a real old-client upgrade test.

## GitHub configuration

The current release workflow uses the built-in `GITHUB_TOKEN` with `contents: write`. It validates the backend version, frontend version, release tag, and release notes, then publishes server, Web, deployment, and checksum assets. Tauri signing secrets are not required for this workflow.

## Publishing a release

1. Update `backend/cmd/server/VERSION` and `frontend/package.json` to the same semantic version.
2. Add `docs/release-notes/RELEASE_NOTES_vX.Y.Z.md`.
3. Run the project checks appropriate to the changed code.
4. Push the commit and tag, for example `v0.4.4`.
5. Confirm the Release contains the server archives, Web bundle, deployment bundle, and `checksums.txt`.

## Failure handling

- Binary updates verify SHA-256 before replacing the service binary. Automatic binary rollback is allowed only within the same supported database-compatibility boundary; crossing the v0.5.7 migration requires a database backup and manual recovery after a failed startup or health check.
- Docker deployments keep the previous image available until the upgraded container passes health checks.
- Missing releases, unavailable networks, or incomplete assets are reported as update failures and do not disable the current service.

## v0.5.7 Migration

- The stable release discovery URL, `vX.Y.Z` tags, archive names and `checksums.txt` contract are unchanged. Existing supported server clients keep their update-notification path; Windows standalone deployments must stop the process and replace the executable manually.
- Back up the database, configuration and encryption keys before upgrading. Stop all pre-v0.5.7 instances before starting the new version. Use the current deployment script or perform a stopped-service upgrade; older updaters cannot enforce the new migration safeguards. Migration 207 adds a digest-version column and the application backfills versioned HMAC credential fingerprints in bounded CAS-protected batches.
- Do not run older and newer server versions against the same database. A rollback to v0.5.6 or earlier requires restoring the pre-upgrade database and configuration as well as the binary. A binary-only health-check rollback does not reverse this data migration.
- The current `deploy/update-binary.sh` requires `SUB3API_DATABASE_BACKUP_CONFIRMED=1` for an upgrade from before v0.5.7 across this boundary. Set it only after verifying the backup and stopping every old instance. If the candidate fails, keep the service stopped and restore the database, configuration and binary together before restarting. The admin updater rejects downgrades below the compatible boundary and cannot safely infer the version of an unlabelled backup.
- Same-origin Web login moves refresh tokens to an HttpOnly Cookie and access tokens to memory. Reload the updated Web application after the server upgrade. Separately hosted frontends retain the bearer API contract but keep new tokens only in page memory, so reloading requires another login. Serve the UI and API under one origin to enable persistent Cookie mode. Legacy localStorage tokens are read only to migrate existing sessions and are no longer renewed there.

## Known limitations

- From v0.5.0, set `SUB3API_REQUIRE_ATTESTATION=1` when invoking `deploy/update-binary.sh` to also require GitHub CLI provenance verification for the archive, repository, workflow and tag. Install a current `gh` first. Failure stops the update before service interruption. Earlier releases have no such attestation. The admin-panel binary updater uses SHA-256.
- Web/PWA sessions check when opened and hourly while visible and online. A waiting worker displays a save-and-refresh notice. “Later” dismisses it for the current document. Closing all tabs permits the browser to activate the new worker. The v0.4.5 client follows its original refresh policy on first transition.
- For Compose use `SUB3API_IMAGE=ghcr.io/yuchen0x1/sub3api-relay:0.5.2`, run `docker compose pull sub3api` and `docker compose up -d sub3api`. Check `/health` and the reported version. Keep the previous image digest and a database backup for rollback.
- The optional Tauri shell is not an official desktop distribution.
