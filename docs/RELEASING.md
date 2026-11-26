# Release Build and Publication

The canonical source and update/download channel is GitHub:
`https://github.com/yuchen0x1/sub3api-relay`. A tag push starts the release workflow, which validates the version, builds the release assets, and publishes the matching GitHub Release with those assets. A manual `workflow_dispatch` run remains validation/build-only and does not publish.

## Local preparation

1. Keep the release version synchronized across all version files; the source tag format is `vX.Y.Z`.
2. Keep `backend/cmd/server/VERSION`, `frontend/package.json`, release-note filename and asset names consistent.
3. Update `docs/releases/release-notes.md`, the per-version release note, and README highlights/links. Ensure installation, cloning and download links use `yuchen0x1/sub3api-relay`.
4. Run relevant backend/frontend tests and inspect all existing worktree edits. Preserve unrelated changes.
5. Validate `.github/workflows/release.yml`. The pinned Go toolchain and reachable-vulnerability gate must pass before publishing. Tag-triggered runs upload server archives, Web and deployment bundles, and checksums to the GitHub Release and retain them as workflow-run artifacts; manual runs only retain workflow artifacts. The optional Tauri shell is not part of the supported release channel.
6. Scan the worktree, staged diff and full Git history for API keys, tokens and private keys before pushing. Use redacted scanner output and never include live deployment configuration, local caches, authentication state or scanner reports in release assets.
7. For v0.5.7, document migration 207 and the database-restore requirement for downgrades. Do not describe binary-only rollback or mixed-version operation as supported after the credential digest backfill.

## README synchronization through protected branches

`sync-release-readme.yml` uses the pinned, API-only GitHub script to rebuild `automation/sync-release-readme` from the current default branch and create or reuse a PR into that branch. Only `README.md` is managed by this branch, so each run starts from current `main` instead of stacking commits on a stale branch; this prevents the synchronization PR from remaining permanently conflicted after later release commits. Release text is escaped data and is never executed.

Repository Actions settings must allow GitHub Actions to create pull requests (`can_approve_pull_request_reviews: true`); the default workflow permission can remain read-only. This workflow requests only job-scoped `contents: write` and `pull-requests: write`. A permissions failure must be resolved by a maintainer, not by bypassing protection.

All seven required checks remain mandatory: `shell`, `secret-scan`, `test`, `backend-security`, `golangci-lint`, `frontend`, and `dependency-review`. Events created with `GITHUB_TOKEN` may not start these workflows. If checks are absent, a maintainer must close and reopen the PR using their own account to trigger the normal `pull_request` checks, and wait for them before merging. The synchronization workflow neither merges nor bypasses checks and does not use `pull_request_target`.

## Build artifact contract

The workflow artifact contains seven platform archives named `sub3api_<version>_<os>_<arch>.tar.gz` with `sub3api` (or `sub3api.exe`) at archive root. Binary updater and installer consume these exact names. It also contains `sub3api_<version>_web.tar.gz` and `sub3api_<version>_deploy.tar.gz`; the latter contains `install.sh`, `docker-deploy.sh`, `docker-compose.local.yml`, `sub3api.env.example`, and `deploy-checksums.txt`, whose checksums cover those four deploy files. The Docker preparation script extracts and verifies this bundle. Starting with v0.4.4, `checksums.txt` covers all nine archives. These hashes provide integrity checks, not an independent publisher signature.

Release binaries use `-trimpath`, omit debug symbols, and embed the version, commit, build time, and `release` deployment type. This reduces incidental build-path exposure without hiding the public source code.

The workflow builds Linux AMD64 and ARM64 runtime images on matching runners and probes the setup server. Publication depends on CI, security, archive smoke checks and both image jobs. The publishing job attaches GitHub provenance to all release files and publishes the tested images as a versioned GHCR manifest. Check package visibility with an anonymous pull on first publication.

## Operator verification

```bash
gh attestation verify "$archive" --repo yuchen0x1/sub3api-relay --signer-workflow yuchen0x1/sub3api-relay/.github/workflows/release.yml --source-ref "refs/tags/$tag"
```

Run `sha256sum --check checksums.txt` with all archives present. A disposable Linux systemd job starts the previous published binary, exercises failure recovery within the supported database boundary, installs the candidate, checks its version and health, and repeats discovery. The v0.5.7 boundary additionally requires that failed cross-migration updates stay stopped until database recovery, rather than restarting an incompatible old binary. Production data recovery requires a separate test with the operator’s backups.

The optional Tauri scaffolding has no signed release channel. See [maintenance coverage](maintenance/README.md).
