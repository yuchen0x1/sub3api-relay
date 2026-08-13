# Sub3API v0.1.183

## Manus Gateway

- Added a Manus-specific Anthropic Messages bridge for `POST /v1/messages`, so Claude-style clients no longer fall through to the standard Claude gateway when a group is configured for Manus.
- Updated Manus chat task creation to send `message.content` as a typed content-block array, matching the Manus v2 task API shape.
- Kept Manus OpenAI Chat Completions compatibility intact while reusing the same asynchronous task creation and polling path for Anthropic Messages requests.
- Added Manus response-shape and task-payload tests to protect the "account connects but cannot use" path from regressing.

## Claude Fingerprint Stability

- Hardened Claude fingerprint reuse so malformed or sentinel user agents are rejected instead of being cached and reused.
- Added cache healing for poisoned fingerprints and upgraded the Claude CLI version used by the fingerprint profile.
- Added an administrator reset-fingerprint action in the account menu, including backend route, service support, frontend API wiring, localization, and tests.

## Tool Quota Compatibility

- Removed the dynamic-tool threshold so mimicable tools receive dynamic fake names consistently.
- This keeps gateway tool names aligned with the request shape that upstream quota checks expect.

## Verification

- `go test ./...`
- `pnpm typecheck`
- `pnpm build`
