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

- Binary updates verify SHA-256 before replacing the service binary and restore the previous binary when startup or health checks fail.
- Docker deployments keep the previous image available until the upgraded container passes health checks.
- Missing releases, unavailable networks, or incomplete assets are reported as update failures and do not disable the current service.

## Known limitations

- From v0.5.0, set `SUB3API_REQUIRE_ATTESTATION=1` when invoking `deploy/update-binary.sh` to also require GitHub CLI provenance verification for the archive, repository, workflow and tag. Install a current `gh` first. Failure stops the update before service interruption. Earlier releases have no such attestation. The admin-panel binary updater uses SHA-256.
- Web/PWA sessions check when opened and hourly while visible and online. A waiting worker displays a save-and-refresh notice. “Later” dismisses it for the current document. Closing all tabs permits the browser to activate the new worker. The v0.4.5 client follows its original refresh policy on first transition.
- For Compose use `SUB3API_IMAGE=ghcr.io/yuchen0x1/sub3api-relay:0.5.2`, run `docker compose pull sub3api` and `docker compose up -d sub3api`. Check `/health` and the reported version. Keep the previous image digest and a database backup for rollback.
- The optional Tauri shell is not an official desktop distribution.
