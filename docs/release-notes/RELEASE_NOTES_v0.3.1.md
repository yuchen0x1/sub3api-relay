# Sub3API v0.3.1

This preparation release groups the current local changes since v0.3.0. The changes below describe gateway behavior and administrative settings; model availability and upstream capabilities remain controlled by each provider/account.

## Features

- **Expanded default model catalog and alias normalization**: add OpenAI-compatible GPT-5.6/GPT-6 family entries, including GPT-6.1 Sol/Terra and GPT-6 Luna; add Claude 5-series and Gemini 3.1/4 fallback model entries; and expand xAI Grok 4.7/4.6/4.5, Composer, Imagine and 4.20 model IDs. Known OpenAI Codex model aliases normalize to their canonical IDs. These defaults improve listing and routing compatibility; they do not guarantee that an upstream account or provider has access to a model.
- **Configurable API Key creation safeguards**: admins can set a per-user maximum number of active API Keys (default 200) and a successful-creation limit per user per hour (default 60). Set either limit to `0` to disable that limit. The hourly counter relies on the configured API Key cache; failures while checking counts are logged and do not block creation. Existing independent validation/attempt throttling is unchanged.
- **Minimum output token setting for protocol conversion**: add `gateway.min_output_tokens` (default 128; minimum accepted value 16). On supported conversions between Claude Messages, OpenAI Chat Completions, and Responses, a client-supplied output-token limit below this floor is raised to the configured minimum. If the client omitted the limit, this setting does not inject one. The setting is loaded at process startup; changing it requires a restart. It does not constrain requests that stay on a protocol path without the covered conversion.
- **OpenAI image multipart parsing hardening**: support parsing multipart image requests and reject a multipart field that exceeds the per-part upload limit rather than silently accepting truncated bytes. This does not increase the overall HTTP request-body limit or guarantee compatibility with every upstream's image/edit semantics.
- **Cloudflare 1010 handling**: recognize Cloudflare bot-signature blocks in plain text and marked structured JSON responses and avoid treating them as invalid account credentials or disabling the account. This prevents account-level penalties for an edge rejection; it cannot bypass Cloudflare or resolve the network/client-signature cause.

## Upgrade Notes

- Keep `gateway.min_output_tokens` at 16 or higher. Values below 16 cause configuration validation to fail; the default is 128.
- The API Key creation defaults impose limits on users with more than 200 active keys or more than 60 successful creations in an hour. Administrators can tune either limit (or set it to `0` for no limit) in system settings.
- For binaries and frontend packages, the version is `0.3.1`. The CI workflow builds and validates assets only as workflow artifacts; a maintainer must publish the matching GitHub Release and attach the generated assets before automatic updates or installer upgrades can work.

## Known Limitations

- Default model catalogs are compatibility hints, not provider entitlement checks; unsupported models can still be rejected by upstreams.
- API Key active-count checks fail open when the database count cannot be read, and hourly throttling is effective only when the API Key cache is configured and available.
- `gateway.min_output_tokens` is process-startup configuration and only affects the listed protocol conversion paths. Clients requesting an output limit below the configured floor may receive more output tokens than they asked for, up to that floor.
- Multipart field limits are per part and do not replace request-size limits. Cloudflare 1010 handling prevents account penalties but does not retry around or bypass the edge block.
