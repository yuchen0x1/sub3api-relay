# Sub3API v0.1.182

## Security Hotfixes

- Prevented credential-bearing proxy subscription URLs from being persisted in administrator audit logs.
- Expanded subscription SSRF blocking to reject multicast, shared address space, and other non-public targets while preserving DNS-pinned dialing.
- Required a constant-time validated `X-Setup-Token` when remote setup is explicitly enabled with `SETUP_ALLOW_REMOTE=true`; tokens must be configured with `SETUP_REMOTE_TOKEN` and contain at least 32 characters.
- Replaced setup connection error details returned to browsers with generic messages while retaining safe operation context in server logs.
- Rejected line breaks in administrator passwords and safely quoted spaces, dollar signs, hashes, quotes, and backslashes so Compose preserves the original password exactly.
- Fixed the Docker deployment script to use only the official release repository as its supply-chain source.

## Docker Updates

- Added `SUB3API_IMAGE` to Compose environment configuration so deployments record the exact image tag they run.
- Release deployment bundles now pin `SUB3API_IMAGE` to the matching release version.
- The administrator version panel now updates or rolls back `SUB3API_IMAGE` before pulling and recreating containers.
- Removed the incomplete simple-release path from the stable Release workflow so every stable release includes binary archives, checksums, deployment assets, and multi-architecture images.

## Public Channel Status

- Kept `/monitor` as an anonymous public route when the administrator enables **Allow visitors to view channel status**.
- Replaced the authenticated dashboard shell with a standalone read-only public status page.
- Removed dashboard navigation, documentation/language controls, time-range selectors, manual refresh, auto-refresh controls, and card detail actions from the public page while retaining silent background status refresh.

## Public Homepage

- Removed the public sign-in and get-started buttons from the homepage.
- Removed links to API keys, usage, and other authenticated dashboard pages so anonymous visitors are not directed to the login form.
- Kept public service status, repository downloads, language selection, and configured community information available.

## Installation Documentation

- Updated English, Chinese, and Japanese guides to use an SSH tunnel for the loopback-only first-run setup wizard.
- Updated source-build requirements to Go 1.26.5, Node.js 24, and pnpm 11.20.0.
- Added the missing frontend build and `-tags embed` steps to the Chinese and Japanese source-build guides.
