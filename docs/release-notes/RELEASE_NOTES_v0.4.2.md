# Sub3API v0.4.2

This release publishes the supported Web management console, server binaries, Docker deployment assets, and release checksums.

## Improvements

- Improved GitHub release update checks and public release page fallback.
- Improved Docker local setup and deployment health handling.
- Improved proxy handling, scheduling caches, administrative settings, and embedded Web assets.
- Preserved backend binary checksum verification and rollback flow.

## Upgrade Notes

- The release tag is `v0.4.2`; backend and frontend package versions report `0.4.2`.
- The repository contains an optional Tauri shell, but no Windows, Linux, or macOS desktop product is published by this release.
- Release consumers should use the server archives, Web bundle, deployment bundle, and `checksums.txt`.
