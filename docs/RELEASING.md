# Release Build and Publication

The canonical source and update/download channel is GitHub:
`https://github.com/yuchen0x1/sub3api-relay`. A tag push starts the release workflow, which validates the version, builds the release assets, and publishes the matching GitHub Release with those assets. A manual `workflow_dispatch` run remains validation/build-only and does not publish.

## Local preparation

1. Keep the release version synchronized across all version files; the source tag format is `vX.Y.Z`.
2. Keep `backend/cmd/server/VERSION`, `frontend/package.json`, release-note filename and asset names consistent.
3. Update README changelog links/highlights and ensure installation, cloning and download links use `yuchen0x1/sub3api-relay`.
4. Run relevant backend/frontend tests and inspect all existing worktree edits. Preserve unrelated changes.
5. Validate `.github/workflows/release.yml`. Tag-triggered runs upload generated server assets and signed Tauri desktop updater assets to the GitHub Release and also retain server assets as workflow-run artifacts; manual runs only retain the server artifacts.

## Build artifact contract

The workflow artifact contains platform archives named `sub3api_<version>_<os>_<arch>.tar.gz` with `sub3api` (or `sub3api.exe`) at archive root, plus `checksums.txt` with SHA-256 entries. Binary updater and installer consume these exact names. It also contains `sub3api_<version>_web.tar.gz` and `sub3api_<version>_deploy.tar.gz`; the latter contains `docker-compose.local.yml`, `sub3api.env.example`, and `deploy-checksums.txt`, whose checksums cover those deploy files. The Docker preparation script extracts and verifies this bundle.

Build-time image validation is not image publication. The release workflow still does not push container images; verify the GitHub Release page and its asset list before describing a release as published.

## Update verification

- Binary deployments require the tag-triggered GitHub release with the matching platform archive and `checksums.txt` before **Update Now** or installer upgrades can work.
- Docker deployments require a separately published image; recreate the container rather than replacing a binary inside it.
- Confirm the running version and health endpoint after restart.
- Desktop releases additionally require the `TAURI_SIGNING_PRIVATE_KEY` and optional `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` Actions secrets. Confirm that `latest.json`, installer assets, and `.sig` files are present before calling the desktop update channel ready.
