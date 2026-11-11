# Dependency security remediation

Date: 2026-11-12

## Resolved in this change

- **minimatch globparts denial of service**: `frontend/pnpm-workspace.yaml` now forces the `glob` child dependencies to `minimatch` `^11.0.5` and upgrades vulnerable `glob` ranges to `^14.0.2`. The regenerated lockfile resolves `minimatch@11.0.6` and `glob@14.0.4`; it no longer contains the vulnerable `minimatch@10.2.6` or `minimatch@9.0.9` paths.
- **deepmerge prototype pollution**: `frontend/patches/deepmerge@4.3.1.patch` rejects `__proto__`, `constructor`, and `prototype` keys and writes copied keys with `Object.defineProperty`. The patch is declared in `frontend/pnpm-workspace.yaml` and recorded in `frontend/pnpm-lock.yaml`.

## Remaining upstream item

- **Rust `glib::VariantStrIter` undefined behavior** remains tracked. `desktop/src-tauri/Cargo.lock` currently resolves Tauri 2.11.5 through GTK 0.18.2 and glib 0.18.5. The compatible dependency family cannot be upgraded by editing the lockfile alone; it requires a Tauri/GTK release that supports glib 0.20 or newer. Keep this item open until that upstream transition is available, then regenerate the lockfile with Cargo and run the desktop build.
- `deepmerge` is still version 4.3.1 because no upstream fixed release is available in the configured registry. The local patch is the active mitigation; Dependabot may continue to report the advisory until the package publishes a fixed version.

## Validation

```text
pnpm --dir frontend install --frozen-lockfile --registry=https://registry.npmjs.org
pnpm --dir frontend audit --json --registry=https://registry.npmjs.org
deepmerge prototype-pollution regression check: passed
```

The official npm audit now reports only the `deepmerge@4.3.1` advisory, which is covered by the committed local patch above.

