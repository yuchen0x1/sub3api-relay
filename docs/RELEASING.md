# Release Process

Every published version must include a reviewed release-notes file. The GitHub Actions workflow rejects malformed tags and releases without notes.

## Prepare

1. Choose a semantic version tag in the exact form `vX.Y.Z`.
2. Create `RELEASE_NOTES_vX.Y.Z.md` at the repository root.
3. Summarize user-visible changes, compatibility fixes, upgrade notes, and known limitations.
4. Update the README changelog links when the highlighted release changes.
5. Run the relevant backend and frontend tests before tagging.

## Publish

1. Commit the release notes and all release changes.
2. Create and push the `vX.Y.Z` tag, or run the Release workflow manually with that existing tag.
3. Confirm that the workflow loads the matching release-notes file and publishes all expected assets.
4. Verify the GitHub Release body, checksums, container tags, and frontend bundle.

## Verify Updates

- For binary deployments, open the administrator version panel, refresh version information, and test **Update Now** only after release assets are available.
- For Docker deployments, pull the published image and recreate the container; do not rely on replacing the binary inside a running container.
- Confirm the running version and health endpoint after the service restarts.
