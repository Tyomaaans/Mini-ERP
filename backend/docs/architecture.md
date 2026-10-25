# Architecture

## Feature-first structure

`main.go` wires dependencies. Configuration, database connections, middleware and routes live in `internal/config`, `internal/infrastructure`, `internal/middleware` and `internal/router`. Business features live under `auth-session`, `master-data`, `purchasing`, `inventory`, `sales`, `finance` and `reporting`.

Each business group stores entities and repository interfaces in `domain/`. Features use `dto.go`, `mapper.go`, `storage.go`, `repository.go`, `service.go` and `handler.go`. Session and token services handle authentication helpers. Reports query existing tables and map the results to response DTOs.

Handlers decode DTOs and send responses. Services enforce permissions, ownership, state, quantities and transaction boundaries. Repositories implement database access and map persistence models to domain entities. Response DTOs exclude internal persistence/authentication fields. Dedicated action endpoints control transaction status changes.

## Consistency boundaries

`TransactionManager.Within` stores a transaction in the request context. Repositories transparently use that transaction. Nested service operations join the existing transaction; inventory operations also establish their own transaction when called independently. Audit and document-sequence writes participate in the same transaction.

Receipt posting locks the PO before the receipt and inventory balances. Shipment posting locks the SO before the shipment and balances. Balance rows are inserted if absent and locked in sorted product order. Conditional balance updates and PostgreSQL CHECK constraints additionally prevent negative balances or excessive reservations. Reservations are consumed or split on partial shipments; cancellation releases active reservations.

Payments lock the invoice before the payment row. A payment is posted only if the aggregate of POSTED payments remains within the invoice total. Updating the payment, invoice paid cache/status and audit records is atomic. Confirmation locks the customer while checking credit exposure, so concurrent confirmations cannot independently consume the same credit allowance.

## Idempotency

The executor inserts or locks `(actor_id, operation, key)` in the same transaction as the business effect. It compares a SHA-256 hash of the canonical document/payload before executing or replaying stored JSON. Concurrent requests serialize on the unique key. On failure, the idempotency record and all business writes roll back together. Receipt/shipment warehouse authorization is checked before replay, so changed warehouse assignments cannot expose a cached result.

Keys expire after 24 hours. Expired keys are removed on reuse and by hourly cleanup. Document state checks still prevent double posting after a key expires or when another key is supplied. Deadlocks/lock timeouts return a retryable `CONCURRENT_OPERATION` error; clients should retry with the same key.

## Authorization

The department identifies an organizational unit. Permissions come from the union of assigned database roles. Role names do not grant implicit wildcard access, including SUPER_ADMIN. System administrators manage access/master records but do not automatically approve, receive, ship or pay.

Authorization is checked inside services as well as authentication middleware. Ownership and warehouse rules apply after capability checks. The PO creator restriction applies even when that user also has MANAGER. Warehouse staff reads are limited to assigned warehouses unless they also have global management/system read responsibility; execution always requires the assigned warehouse.

JWT access tokens validate algorithm, issuer, audience, expiry, subject and session ID. Redis holds only a hash of each refresh token, an absolute TTL, and a credential fingerprint. Rotation is atomic, logout removes the session, user activation and role changes are observed on the next request, and administrative user updates invalidate existing session fingerprints. Redis failure never grants access.

## API and validation

OpenAPI 3.1 is the published contract. HTTP tests compile its JSON Schemas, check route coverage, and validate live responses. JSON bodies reject unknown fields and multiple JSON values. Money uses NUMERIC(18,2); quantity uses NUMERIC(18,3). DTOs accept decimal strings with bounded precision. Each gross line is rounded half away from zero to two decimals, then line and order discounts are applied.

One product appears at most once in an API order. PATCH merges supplied draft fields and retains matching item IDs. Master record deletion generally deactivates records to preserve history; referenced categories and assigned roles cannot be deleted. Seeded system role definitions are managed by the versioned seed; administrators create custom roles for additional combinations.

## MVP business decisions

AR invoices require a fully SHIPPED SO; AP invoices require a CLOSED, fully received PO. One invoice per order avoids partial-invoice amount allocation. Payments may be partial. A zero customer credit limit means no configured cap. PO cancellation is allowed only before receiving or issued financial obligations; SO cancellation is allowed only while draft or confirmed and before shipping/issued invoices.

Polymorphic references in stock movements, approvals and audit logs are built only by services from validated, locked source documents and item IDs. Normal FK constraints protect concrete links, including matching order/warehouse and invoice/order-party combinations. These references point to different tables, so services enforce their integrity within the transaction.

## Tests and deployment

Unit tests cover precision, authorization, token verification and pagination. PostgreSQL/Redis integration tests exercise complete workflows, rollback and concurrent competing operations. The SQL suite validates constraints and immutable records. CI adds the race detector, static checks and a production container build.

The production application runs as a non-root user with a DML database role. Migrations/bootstrap use the database owner. Runtime has no DDL or migration metadata privileges and cannot update/delete immutable ledger/audit tables. Health checks, structured logs, request IDs and bounded-cardinality Prometheus metrics provide operational visibility.
