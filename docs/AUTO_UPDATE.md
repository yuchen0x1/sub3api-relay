# Automatic updates

Sub3API has three update paths. They share the GitHub Release channel but use the mechanism supported by each deployment type.

## Deployment matrix

| Deployment | Detection | Install path | Verification |
| --- | --- | --- | --- |
| Tauri desktop (`desktop/`) | Official Tauri updater on startup and **Check now** | Signed installer, then relaunch | Tauri updater signature and HTTPS endpoint |
| Standalone release binary | Admin version panel or `deploy/update-binary.sh` | Download archive, verify, backup, replace, health check | SHA-256 from `checksums.txt` |
| Docker / source build | Admin version panel | Operator updates the pinned image or source checkout | Release metadata and deployment health checks |

The browser/PWA frontend uses the existing `vite-plugin-pwa` service worker. It checks for a new web bundle when the page loads and periodically requests a service-worker update.

## Desktop updater

The desktop shell uses `@tauri-apps/plugin-updater` and `tauri-plugin-updater`. The endpoint is:

`https://github.com/yuchen0x1/sub3api-relay/releases/latest/download/latest.json`

Release builds must include the signed platform installer, its `.sig` file, and the generated `latest.json`. The public verification key is committed in `desktop/src-tauri/tauri.conf.json`; the private key must never be committed.

The desktop UI reports real download events from the updater plugin. It does not show a progress bar until the plugin provides progress events. Windows uses the configured passive installer mode; macOS and Linux relaunch after installation.

## GitHub configuration

Add these repository Actions secrets before pushing a release tag:

- `TAURI_SIGNING_PRIVATE_KEY`: the full contents of the local Tauri signing key generated with `pnpm tauri signer generate`.
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`: the key password. Leave it empty only when the key was deliberately generated without a password.

The release workflow already uses the minimum `contents: write` permission and the built-in `GITHUB_TOKEN`. It validates that backend, frontend, desktop, and release-note versions all match the `vMAJOR.MINOR.PATCH` tag. Manual workflow runs remain validation/build-only for the server artifacts; signed desktop publication is tag-only.

## Publishing a release

1. Update `backend/cmd/server/VERSION`, `frontend/package.json`, `desktop/package.json`, `desktop/src-tauri/Cargo.toml`, and `desktop/src-tauri/tauri.conf.json` to the same semantic version.
2. Add `docs/release-notes/RELEASE_NOTES_vX.Y.Z.md`.
3. Run the local frontend and desktop checks.
4. Push the commit and tag, for example `v0.4.2`.
5. Confirm the GitHub Release contains the server archives, checksums, desktop installers, `.sig` files, and `latest.json` for all published targets.

## Failure handling

- Desktop signature or manifest verification failure prevents installation; the running app remains available.
- Binary updates verify SHA-256 before stopping the service and restore the previous binary when startup or health checks fail.
- Docker deployments do not replace files inside a running container. Pin the previous image tag until the upgraded container passes its health checks, then remove the old image according to the operator's retention policy.
- A missing GitHub Release, unavailable network, or incomplete asset set is reported as an update failure and does not disable the current service.

## Local verification

```bash
pnpm --dir frontend typecheck
pnpm --dir frontend exec vitest run
pnpm --dir desktop build
pnpm --dir desktop test:run
```

Native Tauri packaging additionally requires the Rust toolchain and platform prerequisites. A real signed release requires the GitHub Secrets above and a tag-triggered workflow run.

## Known limitations

- The desktop shell performs a startup check and exposes **Check now** plus configurable startup-only, six-hour, and 24-hour polling. It does not yet expose a beta channel selector.
- The Tauri package signature protects desktop installers. Windows Authenticode signing still requires a separate code-signing certificate and CI secret.
- Standalone backend binaries currently verify the published SHA-256 checksum. They do not have an additional binary signature layer.
- The online desktop channel remains unavailable until a signed GitHub Release publishes `latest.json` and the platform installers.
