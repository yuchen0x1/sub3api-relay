# Sub3API v0.1.193

This release adds a predictable first-install administrator credential.

## Initial Administrator

- New installations default to `admin / admin` when `ADMIN_EMAIL` and `ADMIN_PASSWORD` are not set.
- Docker Compose, Docker deployment, Apple container deployment, AUTO_SETUP, and source setup defaults are aligned.
- Existing administrator accounts and passwords are never overwritten during upgrades.
- Change the password immediately after the first login, especially before exposing the service publicly.
