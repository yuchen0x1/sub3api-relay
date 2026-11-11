# Sub3API Cockpit Desktop

This directory is a focused, clean-room desktop control-plane scaffold for Sub3API. It implements the first feature slice, an Accounts view, using React 19, Vite, TypeScript, and Tauri 2.

## Scope

- Feature-first React structure under `src/features/accounts`.
- Typed `fetch` adapter for the existing `GET /api/v1/admin/accounts` REST endpoint.
- Normalization for the standard Sub3API `{ code, data }` response envelope and common account-list shapes.
- Aggregate and per-static-IP capacity display when the API supplies those fields.
- Explicit loading, authentication/error, retry, and empty states.
- Volatile admin session state. Tauri stores the selected API key or bearer JWT in managed Rust state for the process lifetime. Vite/browser mode uses module memory only. No token is written to `localStorage`, `sessionStorage`, a file, or a web cookie.
- Unit tests for capacity calculation and response normalization.

## Run

From this directory:

```bash
pnpm install
pnpm dev
pnpm test:run
pnpm build
```

The desktop shell defaults to the local development backend at `http://127.0.0.1:18081`. Set `VITE_SUB3API_BASE_URL` when the backend is hosted elsewhere:

```bash
VITE_SUB3API_BASE_URL=https://sub3api.example.com pnpm dev
```

For the native shell, install the Tauri 2 system prerequisites and run `pnpm tauri dev`. The native build runs the same frontend build through `src-tauri/tauri.conf.json`.

## Lock strategy

`packageManager` pins pnpm 11.20.0, matching the repository existing frontend toolchain. The checked-in `pnpm-lock.yaml` is the reproducible dependency snapshot for this scaffold. Use `pnpm install --frozen-lockfile` in CI and release builds, and regenerate the lockfile only with an intentional dependency update.

## Upstream provenance

The approved architectural reference is [`jlcodes99/cockpit-tools`](https://github.com/jlcodes99/cockpit-tools), tag `v1.3.34`, pinned to commit `a528c23548ec370ba1cb6871c30353f2dd86c7a0`. This scaffold records that provenance in `package.json` metadata but does not import, copy, or vendor the upstream source tree or copyrighted assets. The adapter and UI here are an independent integration boundary for the existing Sub3API API.
