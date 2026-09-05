# Release Process

Every published version must include a reviewed release-notes file and a
matching Gitee Release. GitHub is not used as this project's publishing source.

## Prepare

1. Choose a semantic version tag in the exact form `vX.Y.Z`.
2. Create `docs/release-notes/RELEASE_NOTES_vX.Y.Z.md`.
3. Summarize user-visible changes, compatibility fixes, upgrade notes, and known limitations.
4. Update the README changelog links when the highlighted release changes.
5. Run the relevant backend and frontend tests before tagging.

## Publish

1. Commit the release notes and all release changes.
2. Create and push the `vX.Y.Z` tag to the Gitee remote.
3. Create the matching Gitee Release and attach the platform archives,
   `checksums.txt`, frontend bundle, and deployment bundle.
4. Verify the Gitee Release body, checksums, container tags, and frontend bundle.

## Verify Updates

- For binary deployments, open the administrator version panel, refresh version information, and test **Update Now** only after Gitee release assets are available.
- For Docker deployments, pull the published image and recreate the container; do not rely on replacing the binary inside a running container.
- Confirm the running version and health endpoint after the service restarts.
