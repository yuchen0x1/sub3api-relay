# Sub3API v0.1.181

## Highlights

- Added account-level Codex OAuth device fingerprint convergence with configurable strategies to reduce unstable upstream device and session identifiers.
- Added Hysteria2 outbound proxy support and secure HTTPS subscription parsing for bulk proxy imports.
- Improved the public homepage with live request activity, gateway availability, and optional public channel health visibility.
- Corrected the homepage version badge and changed the operations dashboard's default monitoring window to five minutes.
- Tightened the account management name column so long account names no longer consume excessive table width.

## Compatibility and Reliability

- Added nested Responses API usage parsing for `data.usage` and `data.response.usage` payloads while preserving the existing priority order.
- Preserved deterministic native Responses API `400` errors instead of converting them into retryable `502` responses.
- Excluded WebSocket v2 completion events from time-to-first-token measurements.
- Locked content-moderation fallback behavior to the semantics used by the synchronized upstream release.

## Release Experience

- Added sanitized in-app rendering for GitHub release notes and a dedicated changelog entry beside version controls.
- Added changelog links to the repository documentation and documented the release procedure.
- Required every release tag to include a matching, non-empty `RELEASE_NOTES_vX.Y.Z.md` file.
- Added strict release-time validation that backend and frontend versions match the pushed tag.
- Published a version-pinned, SHA-256-verified Docker deployment bundle with every release.

## Security and Deployment

- Restricted the first-run setup wizard to loopback access by default to prevent remote initialization takeover.
- Added DNS-pinned private-network blocking for setup database and Redis connection tests.
- Changed forwarded client IP trust to opt-in and hardened installer handling of generated credentials.
- Disabled in-container binary replacement for Docker deployments and now shows persistent Compose update commands instead.

## Upgrade

- Binary deployments can use the administrator version panel after the release assets are available.
- Docker deployments should pull the new image and recreate the container according to the deployment guide.
