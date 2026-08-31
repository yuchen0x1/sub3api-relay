# Sub3API v0.1.198

## Fixes

- Restored account group selection in create and edit account dialogs, including simple mode.
- Improved OpenAI-compatible upstream model synchronization error handling and validation diagnostics.
- Preserved secure upstream host allowlisting and SSRF protections for custom OpenAI-compatible gateways.

## Notes

Custom upstream domains must be explicitly added to `security.url_allowlist.upstream_hosts`; disabling the allowlist is not supported as a workaround.
