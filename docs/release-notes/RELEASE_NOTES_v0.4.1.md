# Sub3API v0.4.1

This patch release improves update reliability and local Docker operation while carrying the reliability, proxy, scheduling and desktop configuration changes prepared after v0.4.0.

## Fixes

- **GitHub update checks under API limits**: when the unauthenticated GitHub Releases API returns `403` or `429`, the updater follows the public release page and reads its expanded asset links. Version checks and release downloads can continue without exposing a token to the frontend.
- **Docker host-loopback setup**: local Compose deployments now recognize Docker Desktop's private bridge source when the published host port is explicitly bound to a loopback address. Public bindings remain protected by the remote setup guard.
- **Local Vertex defaults**: local development keeps Vertex batch output disabled until a managed GCS bucket is configured, preventing a normal-mode restart loop after setup completes.

## Improvements

- Improved VMess/proxy protocol handling and scheduler concurrency cache behavior.
- Improved desktop static-exit controls, default API endpoint configuration and administrative settings behavior.
- Kept build, dependency, security and release metadata checks aligned with the v0.4.1 source and frontend versions.

## Upgrade Notes

- The release tag is `v0.4.1`; backend `VERSION` and frontend `package.json` both report `0.4.1`.
- Docker deployments using Vertex batch output must set `BATCH_IMAGE_VERTEX_MANAGED_GCS_BUCKET` and related Vertex settings before enabling `BATCH_IMAGE_VERTEX_ENABLED=true`.
- `SETUP_ALLOW_REMOTE` remains disabled by default. Remote setup still requires explicit opt-in and a valid setup token.
