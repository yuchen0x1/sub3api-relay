# Sub3API v0.1.202

## Features

- Proxy subscriptions now import Shadowsocks, Trojan, and VMess nodes in addition to VLESS (WS/TLS). The subscription importer parses `ss://` (SIP002, AEAD ciphers: aes-128-gcm, aes-256-gcm, chacha20-ietf-poly1305), `trojan://` (TLS + password), and `vmess://` (AEAD, alterId 0, ws/tcp) URIs from plain or base64 subscription bodies.
- New tunnel dialers implemented natively (no external proxy runtime): Shadowsocks SIP004 AEAD with per-direction salt/subkey, Trojan over TLS, and the VMess AEAD request header with SHAKE-128 chunk masking for both TCP and WebSocket transports.
- Nodes using transports an HTTP dialer cannot carry (VLESS Reality/XTLS flow, hysteria2, Shadowsocks 2022, plugins) are now skipped with a per-reason count shown in the parse result instead of failing the whole subscription with a generic error.
- VLESS URI parsing tolerates unknown query parameters (serviceName, quicSecurity, headerType, sid, pbk, ...) emitted by real-world subscription generators; only dialer-unsupported values (security/transport/encryption/flow/headerType) are rejected.
- New protocols are available across the proxy stack: create/edit forms, batch import, subscription import, data import/export validation, proxy URL serialization, connection-pool cache keys, and the req-client dial bridge.

## Notes

- VMess requires alterId 0 (AEAD handshake); legacy non-AEAD credentials are rejected at parse time.
- VMess/Trojan nodes that advertise ws+tls are dialed over WebSocket accordingly; VMess cipher follows the node's `scy` (aes-128-gcm default, chacha20-ietf-poly1305 supported).

## Upgrade Notes

- No configuration changes. Existing proxies and subscriptions are unaffected; re-parsing a previously failed subscription now imports the supported subset and reports what was skipped and why.
