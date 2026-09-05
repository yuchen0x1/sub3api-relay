# Sub3API v0.1.192

This maintenance release prepares the current build for deployment on the new server.

## Build and Deployment

- Upgrade the Docker Go builder image to `golang:1.26.6-alpine`.
- Include the frontend workspace metadata and npm configuration in the Docker build context so pnpm workspace installs remain reproducible.
- Keep local agent, audit, and dependency artifacts out of Docker build contexts.

## Upgrade Notes

- No database migrations are required.
- Existing PostgreSQL, Redis, application data, and fixed secrets can be reused during deployment.
