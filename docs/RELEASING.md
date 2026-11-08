# Local Release Build Preparation

The canonical source and update/download channel is GitHub:
`https://github.com/yuchen0x1/sub3api-relay`. The release workflow is strictly build/validation-only: it does not create a tag, publish a GitHub Release, upload release assets, or push a container image. Create the repository and publish only after the local preparation and verification below are complete.

## Local preparation

1. Keep the release version at `0.4.0` unless the user explicitly requests a version change; the source tag format is `vX.Y.Z`.
2. Keep `backend/cmd/server/VERSION`, `frontend/package.json`, release-note filename and asset names consistent.
3. Update README changelog links/highlights and ensure installation, cloning and download links use `yuchen0x1/sub3api-relay`.
4. Run relevant backend/frontend tests and inspect all existing worktree edits. Preserve unrelated changes.
5. Validate `.github/workflows/release.yml`. It stores generated assets only as workflow-run artifacts.

## Build artifact contract

The workflow artifact contains platform archives named `sub3api_<version>_<os>_<arch>.tar.gz` with `sub3api` (or `sub3api.exe`) at archive root, plus `checksums.txt` with SHA-256 entries. Binary updater and installer consume these exact names. It also contains `sub3api_<version>_web.tar.gz` and `sub3api_<version>_deploy.tar.gz`; the latter contains `docker-compose.local.yml`, `sub3api.env.example`, and `deploy-checksums.txt`, whose checksums cover those deploy files. The Docker preparation script extracts and verifies this bundle.

Build-time image validation is not image publication. No release or container image should be described as published unless a separate, explicitly authorized process does so.

## Update verification

- Binary deployments require a separately published GitHub release with the matching platform archive and `checksums.txt` before **Update Now** or installer upgrades can work.
- Docker deployments require a separately published image; recreate the container rather than replacing a binary inside it.
- Confirm the running version and health endpoint after restart.
