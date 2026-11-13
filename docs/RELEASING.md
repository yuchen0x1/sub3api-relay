# Release Build and Publication

The canonical source and update/download channel is GitHub:
`https://github.com/yuchen0x1/sub3api-relay`. A tag push starts the release workflow, which validates the version, builds the release assets, and publishes the matching GitHub Release with those assets. A manual `workflow_dispatch` run remains validation/build-only and does not publish.

## Local preparation

1. Keep the release version synchronized across all version files; the source tag format is `vX.Y.Z`.
2. Keep `backend/cmd/server/VERSION`, `frontend/package.json`, release-note filename and asset names consistent.
3. Update `docs/releases/release-notes.md`, the per-version release note, and README highlights/links. Ensure installation, cloning and download links use `yuchen0x1/sub3api-relay`.
4. Run relevant backend/frontend tests and inspect all existing worktree edits. Preserve unrelated changes.
5. Validate `.github/workflows/release.yml`. The pinned Go toolchain and reachable-vulnerability gate must pass before publishing. Tag-triggered runs upload server archives, Web and deployment bundles, and checksums to the GitHub Release and retain them as workflow-run artifacts; manual runs only retain workflow artifacts. The optional Tauri shell is not part of the supported release channel.

## Build artifact contract

The workflow artifact contains seven platform archives named `sub3api_<version>_<os>_<arch>.tar.gz` with `sub3api` (or `sub3api.exe`) at archive root. Binary updater and installer consume these exact names. It also contains `sub3api_<version>_web.tar.gz` and `sub3api_<version>_deploy.tar.gz`; the latter contains `install.sh`, `docker-deploy.sh`, `docker-compose.local.yml`, `sub3api.env.example`, and `deploy-checksums.txt`, whose checksums cover those four deploy files. The Docker preparation script extracts and verifies this bundle. Starting with v0.4.4, `checksums.txt` covers all nine archives. These hashes provide integrity checks, not an independent publisher signature.

Release binaries use `-trimpath`, omit debug symbols, and embed the version, commit, build time, and `release` deployment type. This reduces incidental build-path exposure without hiding the public source code.

Build-time image validation is not image publication. The release workflow still does not push container images; verify the GitHub Release page and its asset list before describing a release as published.

## Update verification

- Binary deployments require the tag-triggered GitHub release with the matching platform archive and `checksums.txt` before **Update Now** or installer upgrades can work.
- Docker deployments require a separately published image; recreate the container rather than replacing a binary inside it.
- Confirm the running version and health endpoint after restart.
- Desktop installers, `latest.json`, and `.sig` files are not produced by this workflow. A future desktop update channel requires a separate signed release process and client installation verification.
