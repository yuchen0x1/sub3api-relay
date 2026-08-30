# Sub3API v0.1.196

This release makes the project's online update path Gitee-only and fixes
version detection when a tag exists before its Gitee Release is created.

## Update detection

- Project update checks no longer choose a source by IP or fall back to GitHub.
- Gitee tags are used as a detection fallback when the latest Gitee Release is
  not available yet, so a pushed semantic version is visible in the admin panel.
- Binary replacement still requires matching Gitee Release assets and checksums.
- Optional `UPDATE_GITEE_TOKEN` support improves API rate limits without sending
  a GitHub token to Gitee.
- Update warnings are shown in the version panel instead of being displayed as
  a false "up to date" result.

## Links and deployment

- Release, rollback, and source deployment links in the admin panel now point
  to the official Gitee repository.
- The update configuration examples now describe the Gitee update source.

## Upgrade notes

Existing administrator accounts, passwords, proxy records, and account data are
preserved. Configure a valid `UPDATE_GITEE_TOKEN` when Gitee API rate limiting
is encountered. A tag-only update can be detected by source and Docker builds,
but standalone binary replacement must wait for the Gitee Release assets.
