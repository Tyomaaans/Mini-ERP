# Deployment and operations

## Local development

Use `docker-compose.yml` from `backend/`. It starts PostgreSQL, Redis, a one-shot migration job and the Air development server. Demo seed is an explicit tools-profile command. Local service ports bind to 127.0.0.1 and remain configurable through `.env`.

## Production container

`Dockerfile` builds a static Go binary, then runs it under a non-root Alpine account. Migrations and OpenAPI are embedded. The production Compose file does not mount source code, exposes only the application on loopback, uses a read-only application filesystem, drops Linux capabilities and keeps PostgreSQL/Redis private to its network.

Create a private production environment file from `.env.example`. Set random hex passwords for `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `APP_DATABASE_PASSWORD`, and a private `JWT_SECRET_KEY`. Set `APP_DATABASE_USER=erp_app`; `POSTGRES_USER` remains the separate migration owner. Set the exact frontend origins in `ALLOWED_ORIGINS`.

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build
```

On first database initialization, `deploy/postgres-init.sh` creates the restricted runtime role. Migration grants DML access to application tables, revokes access to migration metadata and revokes UPDATE/DELETE on stock movements and audit logs. The HTTP application connects with the runtime account; the migration service connects with the owner. Set `ADMIN_EMAIL` and `ADMIN_PASSWORD` in the private production environment file, then initialize the first administrator:

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml run --rm bootstrap
```

Bootstrap creates the access definitions and one administrator, with an audit entry. It refuses to run if an active administrator already exists. Create departments' operational users and custom roles through the API afterward. Production mode refuses demo seeding. The runtime app does not receive the bootstrap password.

The initialization script runs only for a new PostgreSQL volume. For a preexisting database, provision the runtime login with your database administrator before migration. Database volumes must be retained across releases. Use separate Compose project names when running development and production on the same machine.

Terminate HTTPS at a reverse proxy in front of the loopback application port. Configure timeouts greater than the application's bounded request/lock timeouts. The server does not trust forwarded client IP headers by default. Change database TLS/connection settings when using an external managed PostgreSQL service.

## Releases and rollback

1. Back up PostgreSQL and retain the prior image tag.
2. Build the new image and run the migration job against the target database.
3. Start the new application image and check `/api/v1/health`.
4. Inspect request logs, error counts and latency after the release.

The migration runner serializes concurrent jobs with a session advisory lock. Each SQL file and its checksum/version record commit atomically. Applied migration files must not be edited. Add another numbered up/down pair for changes. `migrate down` reverts only the latest migration and can drop data when reverting an initial schema; an application rollback to the previous image is preferable when the schema remains compatible.

## Health and observability

- `GET /api/v1/health/live`: process liveness, 200 with `status=ok`.
- `GET /api/v1/health`: PostgreSQL/Redis readiness; returns 503 when a dependency is unavailable.
- `GET /api/v1/metrics`: authenticated Prometheus request counters by status class and duration histogram; requires `system.metrics.read`.
- JSON request logs include UUID request ID, method, route pattern, status and elapsed milliseconds. Request bodies, credentials and tokens are not logged.

The process handles SIGINT/SIGTERM and allows ten seconds for HTTP shutdown. Compose allows fifteen seconds before killing the application. Requests have a twenty-second context deadline and lock waits are limited to five seconds. Body size is capped at 1 MiB. Redis rate limiting fails closed.

## Backup and reconciliation

Take regular PostgreSQL logical or managed backups and verify restoration into an isolated database. Redis holds authentication sessions; losing it signs users out and does not discard financial or inventory records. Redis persistence preserves normal sessions across restarts.

Compare inventory on-hand totals with the sum of immutable movement deltas per product/warehouse. Compare reserved totals with ACTIVE reservations joined to SO products. Compare invoice paid_amount with the sum of POSTED payments. These comparisons should be zero-difference; investigate drift before making corrective business transactions. No API exposes direct balance editing or posted-movement mutation.

## CI

[Backend CI runs](https://github.com/Tyomaaans/Mini-ERP/actions/workflows/ci.yml) publish unit, OpenAPI contract, PostgreSQL integration, concurrency and race-detector results, followed by the production image build. The workflow runs on main pushes and pull requests. Local equivalents are `make test`, `make vet`, `make openapi` and `make test-integration`.
