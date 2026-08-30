# Sub3API v0.1.194

This release fixes Hysteria2 proxy use for account upstream requests and adds
working latency sorting to IP management.

## Fixes

- Preserve Hysteria2 TLS query settings such as `insecure` and `sni` when
  building cached upstream clients.
- Route OpenAI WebSocket requests through the shared proxy transport, including
  Hysteria2, instead of treating every proxy as an HTTP proxy.
- Retire stateful Hysteria2 clients with their cached HTTP transports.
- Sort the IP management latency column across all filtered proxy pages in
  ascending or descending order, keeping unmeasured entries at the end.

Existing administrator accounts, passwords, proxy records, and account data
are preserved during upgrades.
