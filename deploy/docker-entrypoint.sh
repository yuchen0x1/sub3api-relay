#!/bin/sh
set -e

# Fix data directory permissions when running as root.
# Docker named volumes / host bind-mounts may be owned by root,
# preventing the non-root sub3api user from writing files.
if [ "$(id -u)" = "0" ]; then
    mkdir -p /app/data
    # Use || true to avoid failure on read-only mounted files (e.g. config.yaml:ro)
    chown -R sub3api:sub3api /app/data 2>/dev/null || true
    # Re-invoke this script as sub3api so the flag-detection below
    # also runs under the correct user.
    exec su-exec sub3api "$0" "$@"
fi

# Compatibility: if the first arg looks like a flag (e.g. --help),
# prepend the default binary so it behaves the same as the old
# ENTRYPOINT ["/app/sub3api"] style.
if [ "${1#-}" != "$1" ]; then
    set -- /app/sub3api "$@"
fi

exec "$@"
