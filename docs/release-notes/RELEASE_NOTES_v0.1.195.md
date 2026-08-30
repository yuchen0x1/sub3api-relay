# Sub3API v0.1.195

This release expands account data imports and hardens account and proxy
handling based on the Cockpit Tools account workflow.

## Account imports

- Accept Codex, ChatGPT Session, CLIProxyAPI, Claude, Gemini/ADC, Vertex
  Service Account, Bedrock, Grok, OpenAI Compatible, JSONL, TXT, token, and
  proxy URL data formats where their required fields are present.
- Normalize common wrapped, nested, and line-oriented account exports before
  validation, while returning an item-level error for unsupported records.
- Detect duplicate account identities during a single import and against
  existing accounts, without storing raw credentials in the identity index.
- Redact sensitive credential fields in import responses and related account
  metadata.

## Security and proxy scheduling

- Stop retaining complete account credentials in scheduler cache metadata.
- Enable the URL allowlist by default and reject private hosts and insecure HTTP
  unless explicitly configured by an administrator.
- Require explicit trusted-proxy configuration before trusting forwarded client
  IP headers.
- Retire Hysteria2 runtime support and migrate existing active Hysteria2 proxy
  records to an inactive state.
- Prevent expired proxy fallback from selecting inactive nodes.

## Upgrade notes

Existing administrator accounts, passwords, proxy records, and account data are
preserved. Review imported records that report validation errors and explicitly
configure trusted proxies or allowlist exceptions required by your deployment.

## Known limitations

Imports still require credentials that identify a supported account type; an
arbitrary proprietary export cannot be imported without mapping its fields to a
supported format. Existing repository-wide tests unrelated to this release may
still require external upstream billing, captcha, or scheduler fixtures.
