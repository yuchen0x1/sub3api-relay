# Sub3API v0.1.203

## Features

- **Per-account traffic weight**: every account accepts a weight (1–10000, default 1, inherited by spark shadows and copies). A new `weighted` scheduling selection mode picks candidates by weighted random draw in the layered path and shuffles within a priority tier by weight. Configure it in Settings → Scheduling (selection-mode dropdown) or via the `gateway_scheduling_selection_mode` setting; account create/edit/bulk-edit forms and the account list (column hidden by default, sortable) expose the weight.
- **Per-API-key QPS rate limiting**: `api_keys.qps_limit` (0 = unlimited) enforced by a Redis fixed-window counter with fail-open semantics in both auth middleware variants, after the IP ACL. Over-limit requests get `429 API_KEY_QPS_LIMITED` and an ingress reject reason `qps_limited`. The user keys view gains a QPS input; create/update key APIs accept the field.
- **Telegram, WeCom, and Discord alert channels**: three new push providers alongside email/bark/webhook, each with its own severity gate and hourly rate limit. Telegram needs a bot token + chat id, WeCom a `qyapi.weixin.qq.com` bot webhook, Discord an `https://` webhook. The ops email notification card gains three channel editor blocks.
- **Gateway body log recording and replay** (opt-in, off by default): requests bodies are stored in a new `gateway_body_logs` table keyed by unique request id; over-limit bodies keep original 413 semantics. Admin endpoints `/admin/gateway-body-logs` (list/get/replay); the usage table's model cells get a body button opening a view/replay dialog. Replay re-posts the stored body over loopback with the original key credential.
- **Setting revisions with rollback**: every settings save records a full KV snapshot into a new `setting_revisions` table before applying; rollback re-applies a snapshot wholesale. Admin endpoints `/admin/setting-revisions` + `/:id/rollback`; the settings page gains a history/rollback dialog and a body-logging toggle in scheduling settings.

## Fixes

- Subscription fetching now presents a clash-verge User-Agent (with a v2rayN retry on 403/404) because airport panels gate requests by UA and previously answered the custom UA with 404 before parsing started.
- Clash YAML configs (`proxies:` list) are accepted as subscription bodies: each node is reconstructed into a standard share-link URI and validated by the existing URI path, so cipher/alterId/transport/flow restrictions and per-node skip reasons are identical. Unsupported node types (hysteria2, tuic, …) are skipped with explicit reasons instead of failing the whole subscription.

## Upgrade Notes

- Database migrations 199–202 run automatically on startup (account weight, key QPS limit, gateway body logs, setting revisions).
- Body logging is disabled by default; enable it in Settings → Scheduling. Recorded bodies include full request payloads — keep retention in mind on busy gateways.
- No configuration changes are required; existing accounts, keys, proxies, and subscriptions are unaffected.
