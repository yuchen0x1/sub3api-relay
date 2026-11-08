# Sub3API v0.4.0

This release focuses on reachability and personalization: a new in-app notification center delivers balance alerts inside the console, and PWA support lets the panel be installed to a desktop or home screen. The UI gains six accent themes and a complete Japanese locale, while operations charts can now be exported as PNG.

## Features

- **In-app notification center**: a header bell with an unread badge, a `/notifications` page with per-type filtering, mark-read / mark-all-read and cursor-based pagination. Backed by a new `user_notifications` table (migration `207`) and a dedicated user API; distinct from the broadcast announcements system, which targets groups or all users.
- **Low-balance alerts in the notification center**: when a balance crosses the configured low threshold, the gateway now writes an in-app notification (current balance, threshold, recharge URL metadata) in addition to the existing email dispatch. Existing email recipients and toggle settings are unchanged.
- **PWA installability**: a web app manifest (standalone display, 192/512 and maskable icons) with an auto-updating service worker; the embedded frontend server serves `manifest.webmanifest` with `application/manifest+json` and no-cache headers for the manifest and service worker so a stale worker cannot delay a new release. An install button appears in the sidebar and on the login page when the browser supports installation.
- **Six accent themes**: cyan, blue, violet, emerald, amber and rose, selectable from the new appearance menu, applied instantly through runtime CSS variables and persisted locally. Chart palettes intentionally stay fixed so multi-series charts remain distinguishable.
- **Japanese locale**: a complete `ja` language pack in the language switcher, auto-selected when the browser language is Japanese, with a key-parity spec that keeps `ja` aligned with `zh` and `en`.
- **Operational chart PNG export**: a download button on throughput, error trend, error distribution, latency histogram and switch-rate charts.

## Improvements

- **Appearance menu**: the sidebar's single theme toggle becomes an appearance menu with light/dark/system plus accent color; the home and key-lookup headers use the same inline menu. Explicitly choosing light now writes the `light` root class and is no longer overridden by the system dark preference.
- **Wide-table scrolling**: shared scroll container behind admin tables with a dismissible "swipe left/right" hint and edge-fade shadows on small screens.
- **Actionable toasts**: toasts can carry action buttons; enabling or disabling a static exit shows a success toast with an undo affordance that reverts the toggle.
- **Dashboard skeleton screens**: the user dashboard shows layout-matching skeletons while loading instead of a single spinner.
- **README**: documents the local development default admin credentials and warns that production deployments must set independent strong credentials.

## Fixes

- **Chart export transparency**: Chart.js canvases are transparent by default, so exported PNGs looked broken in most viewers; export now repaints the theme surface color behind the canvas before encoding.
- **PWA asset serving**: the embedded server registers the correct MIME type for `manifest.webmanifest` and serves manifest/Service Worker with `no-cache`.

## Upgrade Notes

- For binaries and frontend packages, the version is `0.4.0`. The CI workflow builds and validates assets only as workflow artifacts; a maintainer must publish the matching GitHub Release and attach the generated assets before automatic updates or installer upgrades can work.
- Migration `207_user_notifications.sql` adds the `user_notifications` table through the standard migration flow.
- Notification types `account_quota`, `payment_received` and `key_rotated` are reserved type conventions in the schema and frontend routing; only `balance_low` notifications are produced by the backend in this release.

## Known Limitations

- In-app notifications currently cover low-balance alerts only; other reserved types will populate as the corresponding features land.
- The service worker updates automatically when a new version is detected on page load; already-installed clients pick up a freshly published release on their next visit rather than pushed mid-session.
- Accent themes do not recolor chart palettes, by design.
