# Mini ERP

A REST API for purchasing, inventory, sales, and finance, built with Go, Gin, PostgreSQL, and Redis. The system supports one company, multiple warehouses, IDR transactions, and access controls for each department.

## Features

- Login, JWT access tokens, refresh token rotation, and logout.
- Users with multiple roles, business permissions, and assigned warehouses.
- Product, category, supplier, customer, and warehouse management.
- Purchase orders with submission, manager approval, rejection, revision, and cancellation.
- Partial goods receipts with stock updates and movement records.
- Sales orders, stock reservations, and partial shipments.
- Customer and supplier invoices with partial payments.
- Sales, purchasing, stock, receivables, and payables reports.
- Idempotency keys for business actions and audit records for changes.
- PostgreSQL migrations, repeatable demo seeds, and OpenAPI 3.1 documentation.

## Main routes

All application routes use the `/api/v1` prefix.

| Method | Route | Access |
| --- | --- | --- |
| POST | `/auth/login` | Public |
| POST | `/auth/refresh` | Refresh token |
| POST | `/auth/logout` | Bearer token |
| GET | `/auth/me` | Bearer token |
| GET, POST | `/users` | User read/create permissions |
| PUT | `/users/:id/roles` | Role assignment permission |
| PUT | `/users/:id/warehouses` | Warehouse assignment permission |
| GET, POST | `/products` | Product read/create permissions |
| GET, PATCH | `/products/:id` | Product read/update permissions |
| GET, POST | `/purchase-orders` | Purchase order read/create permissions |
| GET, PATCH | `/purchase-orders/:id` | Purchase order read/update permissions |
| POST | `/purchase-orders/:id/submit` | Purchasing |
| POST | `/purchase-orders/:id/approve` | Manager; cannot approve own PO |
| POST | `/goods-receipts` | Warehouse; assigned warehouse |
| POST | `/goods-receipts/:id/post` | Warehouse; assigned warehouse |
| GET | `/inventory/balances` | Inventory read permission |
| GET | `/inventory/movements` | Inventory read permission |
| GET, POST | `/sales-orders` | Sales order read/create permissions |
| GET, PATCH | `/sales-orders/:id` | Read permission; creator for draft updates |
| POST | `/sales-orders/:id/confirm` | Sales; creator |
| POST | `/sales-orders/:id/reserve` | Warehouse; assigned warehouse |
| POST | `/shipments` | Warehouse; assigned warehouse |
| POST | `/shipments/:id/post` | Warehouse; assigned warehouse |
| GET, POST | `/invoices` | Invoice read/create permissions |
| POST | `/invoices/:id/issue` | Finance |
| POST | `/payments` | Finance |
| POST | `/payments/:id/post` | Finance |
| GET | `/reports/sales-summary` | Sales report permission |
| GET | `/reports/low-stock` | Inventory report permission |
| GET | `/reports/receivables` | Finance report permission |

`GET /health`, `GET /health/live`, `GET /docs`, and `GET /openapi.yaml` are public utility routes. `GET /metrics` requires a bearer token and `system.metrics.read`.

See the [OpenAPI specification](backend/docs/openapi.yaml) for all routes and the [RBAC matrix](backend/docs/rbac-matrix.md) for role permissions. Services also enforce ownership, warehouse scope, and document status.

## Project structure

```text
.
├── backend
│   ├── docs
│   ├── internal
│   │   ├── auth-session
│   │   ├── master-data
│   │   ├── purchasing
│   │   ├── inventory
│   │   ├── sales
│   │   ├── finance
│   │   ├── reporting
│   │   ├── config
│   │   ├── infrastructure
│   │   ├── middleware
│   │   └── router
│   ├── migrations
│   ├── seeds
│   ├── pkg
│   ├── main.go
│   ├── Dockerfile.dev
│   ├── Dockerfile
│   └── docker-compose.yml
└── README.md
```

HTTP handlers call services, services contain business rules, and repositories handle PostgreSQL access. Each feature keeps its DTOs, mappers, storage models, repositories, services, and handlers together. Redis stores sessions, refresh token hashes, and rate-limit state.

## Run locally

Requirements:

- Go 1.26.10
- Docker and Docker Compose

```bash
cd backend
cp .env.example .env
```

Set PostgreSQL and Redis passwords in `.env`, and set `JWT_SECRET_KEY` with `openssl rand -hex 32`. Use random hex passwords so the PostgreSQL URL needs no additional encoding. Change service ports if they are already in use.

```bash
docker compose up -d --build
docker compose run --rm seed
```

The migration service runs before the application starts. Air reloads the application during development. `docker compose down` keeps the database volumes.

- **API Base URL:** `http://localhost:8080/api/v1`
- **Swagger UI:** `http://localhost:8080/api/v1/docs`
- **OpenAPI:** `http://localhost:8080/api/v1/openapi.yaml`

For a direct Go run, start PostgreSQL and Redis first and update `.env` with addresses reachable from the host. From `backend/`, run:

```bash
make migrate-up
go run . seed
make run
```

`migrate up` applies pending migrations. `migrate down` reverts the latest migration. Migration checksums reject changes to files already applied.

## Configuration

| Variable | Description |
| --- | --- |
| `APP_ENV` | Runtime environment; production disables demo seeding |
| `APP_PORT` | HTTP server port |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_ADDR` | Redis host and port |
| `REDIS_PASSWORD` | Redis password |
| `JWT_SECRET_KEY` | JWT signing secret, at least 32 characters |
| `JWT_EXPIRY` | Access token duration, for example `15m` |
| `REFRESH_EXPIRY` | Absolute session duration, for example `168h` |
| `ALLOWED_ORIGINS` | Comma-separated frontend origins |
| `DEMO_PASSWORD` | Password used when creating demo accounts |
| `ADMIN_EMAIL` | First production administrator email |
| `ADMIN_PASSWORD` | First production administrator password |
| `APP_DATABASE_USER` | Production application database role |
| `APP_DATABASE_PASSWORD` | Production application database password |

Compose also reads `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT`, and `REDIS_PORT`. Production setup and administrator bootstrap commands are documented in [deployment](backend/docs/deployment.md).

## Demo accounts

The seed command creates these accounts with `DEMO_PASSWORD` from `.env`. The example password is `LocalDemoPassword123!`. Existing account passwords are retained when seeding again.

| Role | Email |
| --- | --- |
| SUPER_ADMIN | admin@example.test |
| SALES_STAFF | sales@example.test |
| PURCHASING_STAFF | purchasing@example.test |
| WAREHOUSE_STAFF | warehouse@example.test |
| FINANCE_STAFF | finance@example.test |
| MANAGER | manager@example.test |

The warehouse account is assigned to WH-JKT. Seeds include master data and opening stock recorded through a completed purchase order, receipt, and stock movement.

## Business workflows

Purchasing follows create PO, submit, manager approval, goods receipt, supplier invoice, and payment. A PO creator cannot approve their own order, including users with both Purchasing and Manager roles. Partial receiving is supported; the PO closes after all items are received.

Sales follows create SO, confirm, reserve stock, ship, customer invoice, and payment. Draft SO updates and confirmation require the creator. Warehouse actions require an assigned warehouse. Partial shipping and payments are supported.

Send an `Idempotency-Key` header for workflow actions. Retrying the same action with the same key returns the stored result. Reusing a key for another document or payload returns `409 IDEMPOTENCY_CONFLICT`. Successful results are kept for 24 hours.

Money and quantity use decimal strings. The backend calculates totals and controls status changes. For this version, one invoice covers the full order after receiving or shipping is complete. The scope excludes returns, payroll, tax calculations, full double-entry accounting, and multiple companies or currencies.

## Development checks

Run these commands from `backend/`:

```bash
make build
make test
make vet
make openapi
make test-integration
```

Integration checks use isolated PostgreSQL and Redis services. They cover migrations, constraints, repeatable seeds, API responses, authorization, rollback, and concurrent reservations, receipts, shipments, and payments. Other checks cover refresh token rotation, list query counts, rate limits, and monetary calculations.

## Documentation

- [OpenAPI specification](backend/docs/openapi.yaml)
- [Architecture](backend/docs/architecture.md)
- [Database ERD](backend/docs/erd.md)
- [RBAC matrix](backend/docs/rbac-matrix.md)
- [Deployment](backend/docs/deployment.md)
