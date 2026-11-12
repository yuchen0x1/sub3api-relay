# Sub3API v0.4.2

This release publishes the Tauri desktop updater integration and the matching signed desktop release workflow.

## Improvements

- Added the official Tauri updater with HTTPS GitHub Release metadata, public-key verification, real download progress, install and relaunch handling.
- Added Windows, Linux and macOS desktop release jobs with updater metadata and signature uploads.
- Added configurable desktop update checks and documented release-secret requirements.
- Preserved the existing backend binary checksum verification and rollback flow.

## Upgrade Notes

- The release tag is `v0.4.2`; backend, frontend and desktop package versions all report `0.4.2`.
- Desktop updater publication requires the `TAURI_SIGNING_PRIVATE_KEY` and `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` GitHub Actions secrets.
- Confirm the Release contains platform installers, `.sig` files and `latest.json` before treating the desktop update channel as ready.