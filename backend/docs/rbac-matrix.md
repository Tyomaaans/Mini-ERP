# RBAC and resource authorization

Permissions are loaded from PostgreSQL role relationships on each request. Multiple roles contribute the union of their capabilities. Department does not grant access. Missing permissions deny the operation. SUPER_ADMIN has administrative grants rather than an implicit wildcard.

| Resource | ADMIN | SALES | PURCHASING | WAREHOUSE | FINANCE | MANAGER |
| --- | --- | --- | --- | --- | --- | --- |
| Users and custom roles | Manage | - | - | - | - | Read |
| Products and categories | Manage | Read | Read | Read | Read | Read |
| Warehouses | Manage | Read | Read | Assigned read | Read | Read |
| Customers | Read | Create/read/update | - | Read | Read | Read |
| Suppliers | Read | - | Create/read/update | Read | Read | Read |
| Purchase orders | Read | - | Create/read/update/submit/cancel | Scoped read | Read | Read/approve/reject |
| Goods receipts | Read | - | Read | Scoped create/read/post | Read | Read |
| Inventory | Read | Read | Read | Scoped read/reserve | Read | Read |
| Sales orders | Read | Create/read/update/confirm/cancel | - | Scoped read/reserve | Read | Read |
| Shipments | Read | Read | - | Scoped create/read/post | Read | Read |
| Invoices | Read | Read | Read | - | Create/read/update/issue | Read |
| Payments | Read | - | - | - | Create/read/post | Read |
| Reports | All | Sales | Purchasing | Assigned inventory | Finance | All |
| Audit logs | Read | - | - | - | - | Read |

Create/update master records includes activation changes when permitted. DELETE on those master endpoints deactivates the record through the update capability; it does not remove transaction history. Categories may be physically deleted only when unreferenced. Assigned custom roles and seeded system roles cannot be deleted. Seeded role definitions are maintained by the seed; custom roles support explicit additional grants.

## Rules that still apply with multiple roles

- Only the creator can update a draft SO or confirm/cancel that SO.
- Warehouse receiving, reserving and shipping always require the source warehouse in user_warehouses.
- Warehouse staff reads use assigned warehouses. MANAGER/SUPER_ADMIN combined with warehouse roles may read globally, but this never bypasses execution scope.
- PO creators cannot approve or reject their own PO, including when they also have MANAGER.
- Approval requires SUBMITTED; rejected POs require explicit revision before draft edits and resubmission.
- Posted, shipped, closed and cancelled documents cannot be edited through ordinary PATCH payloads.
- Posting and payment limits remain enforced independently of permission combinations.
- At least one active SUPER_ADMIN must remain. Administrators may assign multiple roles and warehouse scopes to other users through dedicated endpoints.

## Seeded permission codes

### SUPER_ADMIN

```text
users.read
users.create
users.update
users.assign_role
users.assign_warehouse
roles.manage
products.read
products.create
products.update
categories.read
categories.create
categories.update
categories.delete
customers.read
suppliers.read
warehouses.read
warehouses.create
warehouses.update
purchase_orders.read
goods_receipts.read
inventory.read
sales_orders.read
shipments.read
invoices.read
payments.read
reports.sales.read
reports.purchasing.read
reports.inventory.read
reports.finance.read
audit_logs.read
system.metrics.read
```

### SALES_STAFF

```text
products.read
categories.read
warehouses.read
customers.read
customers.create
customers.update
inventory.read
sales_orders.read
sales_orders.create
sales_orders.update
sales_orders.confirm
sales_orders.cancel
shipments.read
invoices.read
reports.sales.read
```

### PURCHASING_STAFF

```text
products.read
categories.read
warehouses.read
suppliers.read
suppliers.create
suppliers.update
purchase_orders.read
purchase_orders.create
purchase_orders.update
purchase_orders.submit
purchase_orders.cancel
goods_receipts.read
inventory.read
invoices.read
reports.purchasing.read
```

### WAREHOUSE_STAFF

```text
products.read
categories.read
warehouses.read
customers.read
suppliers.read
purchase_orders.read
goods_receipts.read
goods_receipts.create
goods_receipts.post
inventory.read
inventory.reserve
sales_orders.read
shipments.read
shipments.create
shipments.post
reports.inventory.read
```

### FINANCE_STAFF

```text
products.read
categories.read
warehouses.read
customers.read
suppliers.read
purchase_orders.read
goods_receipts.read
inventory.read
sales_orders.read
shipments.read
invoices.read
invoices.create
invoices.update
invoices.issue
payments.read
payments.create
payments.post
reports.finance.read
```

### MANAGER

```text
users.read
products.read
categories.read
warehouses.read
customers.read
suppliers.read
purchase_orders.read
purchase_orders.approve
goods_receipts.read
inventory.read
sales_orders.read
shipments.read
invoices.read
payments.read
reports.sales.read
reports.purchasing.read
reports.inventory.read
reports.finance.read
audit_logs.read
system.metrics.read
```
