CREATE TABLE departments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(30) NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    full_name     VARCHAR(150) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX users_department_id_idx ON users(department_id);

CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permissions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(100) NOT NULL UNIQUE,
    module     VARCHAR(50) NOT NULL,
    action     VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT user_roles_user_role_key UNIQUE (user_id, role_id)
);

CREATE INDEX user_roles_role_id_idx ON user_roles(role_id);

CREATE TABLE role_permissions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT role_permissions_role_permission_key UNIQUE (role_id, permission_id)
);

CREATE INDEX role_permissions_permission_id_idx ON role_permissions(permission_id);

CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(30) NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    parent_id  UUID REFERENCES categories(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT categories_parent_check CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX categories_parent_id_idx ON categories(parent_id);

CREATE TABLE products (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku           VARCHAR(50) NOT NULL UNIQUE,
    name          VARCHAR(200) NOT NULL,
    category_id   UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    unit          VARCHAR(20) NOT NULL,
    selling_price NUMERIC(18,2) NOT NULL DEFAULT 0,
    reorder_level NUMERIC(18,3) NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT products_selling_price_check CHECK (selling_price >= 0 AND selling_price <> 'NaN'::NUMERIC),
    CONSTRAINT products_reorder_level_check CHECK (reorder_level >= 0 AND reorder_level <> 'NaN'::NUMERIC)
);

CREATE INDEX products_category_id_idx ON products(category_id);

CREATE TABLE suppliers (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code               VARCHAR(30) NOT NULL UNIQUE,
    name               VARCHAR(150) NOT NULL,
    email              VARCHAR(255),
    phone              VARCHAR(30),
    address            TEXT,
    payment_terms_days INT NOT NULL DEFAULT 0,
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT suppliers_payment_terms_days_check CHECK (payment_terms_days >= 0)
);

CREATE TABLE customers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code             VARCHAR(30) NOT NULL UNIQUE,
    name             VARCHAR(150) NOT NULL,
    email            VARCHAR(255),
    phone            VARCHAR(30),
    billing_address  TEXT,
    shipping_address TEXT,
    credit_limit     NUMERIC(18,2) NOT NULL DEFAULT 0,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT customers_credit_limit_check CHECK (credit_limit >= 0 AND credit_limit <> 'NaN'::NUMERIC)
);

CREATE TABLE warehouses (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(30) NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    address    TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_warehouses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT user_warehouses_user_warehouse_key UNIQUE (user_id, warehouse_id)
);

CREATE INDEX user_warehouses_warehouse_id_idx ON user_warehouses(warehouse_id);

CREATE TABLE purchase_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number       VARCHAR(40) NOT NULL UNIQUE,
    supplier_id     UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    warehouse_id    UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    status          VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    order_date      DATE NOT NULL,
    expected_date   DATE,
    subtotal        NUMERIC(18,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    total_amount    NUMERIC(18,2) NOT NULL DEFAULT 0,
    notes           TEXT,
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at    TIMESTAMPTZ,
    approved_by     UUID REFERENCES users(id) ON DELETE RESTRICT,
    approved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT purchase_orders_id_warehouse_key UNIQUE (id, warehouse_id),
    CONSTRAINT purchase_orders_id_supplier_key UNIQUE (id, supplier_id),
    CONSTRAINT purchase_orders_status_check CHECK (status IN ('DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED', 'CLOSED', 'CANCELLED')),
    CONSTRAINT purchase_orders_expected_date_check CHECK (expected_date IS NULL OR expected_date >= order_date),
    CONSTRAINT purchase_orders_subtotal_check CHECK (subtotal >= 0 AND subtotal <> 'NaN'::NUMERIC),
    CONSTRAINT purchase_orders_discount_amount_check CHECK (discount_amount >= 0 AND discount_amount <= subtotal),
    CONSTRAINT purchase_orders_total_amount_check CHECK (total_amount = subtotal - discount_amount),
    CONSTRAINT purchase_orders_approval_pair_check CHECK ((approved_by IS NULL) = (approved_at IS NULL)),
    CONSTRAINT purchase_orders_approver_check CHECK (approved_by IS NULL OR approved_by <> created_by),
    CONSTRAINT purchase_orders_submission_check CHECK (status NOT IN ('SUBMITTED', 'APPROVED', 'REJECTED', 'CLOSED') OR submitted_at IS NOT NULL),
    CONSTRAINT purchase_orders_approval_check CHECK (status NOT IN ('APPROVED', 'CLOSED') OR approved_by IS NOT NULL)
);

CREATE INDEX purchase_orders_supplier_id_idx ON purchase_orders(supplier_id);
CREATE INDEX purchase_orders_warehouse_id_idx ON purchase_orders(warehouse_id);
CREATE INDEX purchase_orders_created_by_idx ON purchase_orders(created_by);
CREATE INDEX purchase_orders_approved_by_idx ON purchase_orders(approved_by);
CREATE INDEX purchase_orders_status_order_date_idx ON purchase_orders(status, order_date DESC, id DESC);

CREATE TABLE purchase_order_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE RESTRICT,
    product_id        UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity          NUMERIC(18,3) NOT NULL,
    unit_price        NUMERIC(18,2) NOT NULL,
    discount_amount   NUMERIC(18,2) NOT NULL DEFAULT 0,
    line_total        NUMERIC(18,2) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT purchase_order_items_quantity_check CHECK (quantity > 0 AND quantity <> 'NaN'::NUMERIC),
    CONSTRAINT purchase_order_items_unit_price_check CHECK (unit_price >= 0 AND unit_price <> 'NaN'::NUMERIC),
    CONSTRAINT purchase_order_items_discount_amount_check CHECK (discount_amount >= 0 AND discount_amount <= ROUND(quantity * unit_price, 2)),
    CONSTRAINT purchase_order_items_line_total_check CHECK (line_total = ROUND(quantity * unit_price, 2) - discount_amount)
);

CREATE INDEX purchase_order_items_purchase_order_id_idx ON purchase_order_items(purchase_order_id);
CREATE INDEX purchase_order_items_product_id_idx ON purchase_order_items(product_id);

CREATE TABLE goods_receipts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gr_number         VARCHAR(40) NOT NULL UNIQUE,
    purchase_order_id UUID NOT NULL,
    warehouse_id      UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    status            VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    received_at       TIMESTAMPTZ,
    received_by       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT goods_receipts_purchase_order_warehouse_fkey FOREIGN KEY (purchase_order_id, warehouse_id) REFERENCES purchase_orders(id, warehouse_id) ON DELETE RESTRICT,
    CONSTRAINT goods_receipts_status_check CHECK (status IN ('DRAFT', 'POSTED', 'CANCELLED')),
    CONSTRAINT goods_receipts_received_at_check CHECK (status <> 'POSTED' OR received_at IS NOT NULL)
);

CREATE INDEX goods_receipts_purchase_order_status_idx ON goods_receipts(purchase_order_id, status);
CREATE INDEX goods_receipts_warehouse_id_idx ON goods_receipts(warehouse_id);
CREATE INDEX goods_receipts_received_by_idx ON goods_receipts(received_by);

CREATE TABLE goods_receipt_items (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_id       UUID NOT NULL REFERENCES goods_receipts(id) ON DELETE RESTRICT,
    purchase_order_item_id UUID NOT NULL REFERENCES purchase_order_items(id) ON DELETE RESTRICT,
    quantity_received      NUMERIC(18,3) NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT goods_receipt_items_receipt_order_item_key UNIQUE (goods_receipt_id, purchase_order_item_id),
    CONSTRAINT goods_receipt_items_quantity_received_check CHECK (quantity_received > 0 AND quantity_received <> 'NaN'::NUMERIC)
);

CREATE INDEX goods_receipt_items_purchase_order_item_id_idx ON goods_receipt_items(purchase_order_item_id);

CREATE TABLE inventory_balances (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    warehouse_id      UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    quantity_on_hand  NUMERIC(18,3) NOT NULL DEFAULT 0,
    quantity_reserved NUMERIC(18,3) NOT NULL DEFAULT 0,
    version           BIGINT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT inventory_balances_product_warehouse_key UNIQUE (product_id, warehouse_id),
    CONSTRAINT inventory_balances_quantity_on_hand_check CHECK (quantity_on_hand >= 0 AND quantity_on_hand <> 'NaN'::NUMERIC),
    CONSTRAINT inventory_balances_quantity_reserved_check CHECK (quantity_reserved >= 0 AND quantity_reserved <= quantity_on_hand),
    CONSTRAINT inventory_balances_version_check CHECK (version >= 0)
);

CREATE INDEX inventory_balances_warehouse_id_idx ON inventory_balances(warehouse_id);

CREATE TABLE stock_movements (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    warehouse_id      UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    movement_type     VARCHAR(30) NOT NULL,
    quantity_delta    NUMERIC(18,3) NOT NULL,
    reference_type    VARCHAR(40) NOT NULL,
    reference_id      UUID NOT NULL,
    reference_item_id UUID NOT NULL,
    idempotency_key   VARCHAR(100),
    performed_by      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    occurred_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT stock_movements_reference_key UNIQUE (reference_type, reference_id, reference_item_id),
    CONSTRAINT stock_movements_type_check CHECK (movement_type IN ('GOODS_RECEIPT', 'SHIPMENT', 'ADJUSTMENT')),
    CONSTRAINT stock_movements_quantity_delta_check CHECK (quantity_delta <> 'NaN'::NUMERIC AND (
        (movement_type = 'GOODS_RECEIPT' AND quantity_delta > 0)
        OR (movement_type = 'SHIPMENT' AND quantity_delta < 0)
        OR (movement_type = 'ADJUSTMENT' AND quantity_delta <> 0)
    )),
    CONSTRAINT stock_movements_reference_type_check CHECK (reference_type = movement_type)
);

CREATE INDEX stock_movements_product_id_idx ON stock_movements(product_id);
CREATE INDEX stock_movements_warehouse_id_idx ON stock_movements(warehouse_id);
CREATE INDEX stock_movements_performed_by_idx ON stock_movements(performed_by);
CREATE INDEX stock_movements_occurred_at_idx ON stock_movements(occurred_at);

CREATE TABLE sales_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    so_number       VARCHAR(40) NOT NULL UNIQUE,
    customer_id     UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    warehouse_id    UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    status          VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    order_date      DATE NOT NULL,
    subtotal        NUMERIC(18,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    total_amount    NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    confirmed_at    TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT sales_orders_id_warehouse_key UNIQUE (id, warehouse_id),
    CONSTRAINT sales_orders_id_customer_key UNIQUE (id, customer_id),
    CONSTRAINT sales_orders_status_check CHECK (status IN ('DRAFT', 'CONFIRMED', 'PARTIALLY_SHIPPED', 'SHIPPED', 'CANCELLED')),
    CONSTRAINT sales_orders_subtotal_check CHECK (subtotal >= 0 AND subtotal <> 'NaN'::NUMERIC),
    CONSTRAINT sales_orders_discount_amount_check CHECK (discount_amount >= 0 AND discount_amount <= subtotal),
    CONSTRAINT sales_orders_total_amount_check CHECK (total_amount = subtotal - discount_amount),
    CONSTRAINT sales_orders_confirmation_check CHECK (status NOT IN ('CONFIRMED', 'PARTIALLY_SHIPPED', 'SHIPPED') OR confirmed_at IS NOT NULL)
);

CREATE INDEX sales_orders_customer_id_idx ON sales_orders(customer_id);
CREATE INDEX sales_orders_warehouse_id_idx ON sales_orders(warehouse_id);
CREATE INDEX sales_orders_created_by_idx ON sales_orders(created_by);
CREATE INDEX sales_orders_status_order_date_idx ON sales_orders(status, order_date DESC, id DESC);

CREATE TABLE sales_order_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_id  UUID NOT NULL REFERENCES sales_orders(id) ON DELETE RESTRICT,
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity        NUMERIC(18,3) NOT NULL,
    unit_price      NUMERIC(18,2) NOT NULL,
    discount_amount NUMERIC(18,2) NOT NULL DEFAULT 0,
    line_total      NUMERIC(18,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT sales_order_items_quantity_check CHECK (quantity > 0 AND quantity <> 'NaN'::NUMERIC),
    CONSTRAINT sales_order_items_unit_price_check CHECK (unit_price >= 0 AND unit_price <> 'NaN'::NUMERIC),
    CONSTRAINT sales_order_items_discount_amount_check CHECK (discount_amount >= 0 AND discount_amount <= ROUND(quantity * unit_price, 2)),
    CONSTRAINT sales_order_items_line_total_check CHECK (line_total = ROUND(quantity * unit_price, 2) - discount_amount)
);

CREATE INDEX sales_order_items_sales_order_id_idx ON sales_order_items(sales_order_id);
CREATE INDEX sales_order_items_product_id_idx ON sales_order_items(product_id);

CREATE TABLE stock_reservations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_item_id UUID NOT NULL REFERENCES sales_order_items(id) ON DELETE RESTRICT,
    warehouse_id        UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    quantity            NUMERIC(18,3) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_by          UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    released_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT stock_reservations_quantity_check CHECK (quantity > 0 AND quantity <> 'NaN'::NUMERIC),
    CONSTRAINT stock_reservations_status_check CHECK (status IN ('ACTIVE', 'RELEASED', 'CONSUMED')),
    CONSTRAINT stock_reservations_released_at_check CHECK (
        (status = 'ACTIVE' AND released_at IS NULL)
        OR (status IN ('RELEASED', 'CONSUMED') AND released_at IS NOT NULL)
    )
);

CREATE INDEX stock_reservations_order_item_status_idx ON stock_reservations(sales_order_item_id, status);
CREATE INDEX stock_reservations_warehouse_id_idx ON stock_reservations(warehouse_id);
CREATE INDEX stock_reservations_created_by_idx ON stock_reservations(created_by);

CREATE TABLE shipments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_number VARCHAR(40) NOT NULL UNIQUE,
    sales_order_id  UUID NOT NULL,
    warehouse_id    UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    status          VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    shipped_at      TIMESTAMPTZ,
    shipped_by      UUID REFERENCES users(id) ON DELETE RESTRICT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT shipments_sales_order_warehouse_fkey FOREIGN KEY (sales_order_id, warehouse_id) REFERENCES sales_orders(id, warehouse_id) ON DELETE RESTRICT,
    CONSTRAINT shipments_status_check CHECK (status IN ('DRAFT', 'SHIPPED', 'CANCELLED')),
    CONSTRAINT shipments_shipping_pair_check CHECK ((shipped_by IS NULL) = (shipped_at IS NULL)),
    CONSTRAINT shipments_shipping_check CHECK (status <> 'SHIPPED' OR shipped_by IS NOT NULL)
);

CREATE INDEX shipments_sales_order_status_idx ON shipments(sales_order_id, status);
CREATE INDEX shipments_warehouse_id_idx ON shipments(warehouse_id);
CREATE INDEX shipments_shipped_by_idx ON shipments(shipped_by);

CREATE TABLE shipment_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipment_id         UUID NOT NULL REFERENCES shipments(id) ON DELETE RESTRICT,
    sales_order_item_id UUID NOT NULL REFERENCES sales_order_items(id) ON DELETE RESTRICT,
    quantity_shipped    NUMERIC(18,3) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT shipment_items_shipment_order_item_key UNIQUE (shipment_id, sales_order_item_id),
    CONSTRAINT shipment_items_quantity_shipped_check CHECK (quantity_shipped > 0 AND quantity_shipped <> 'NaN'::NUMERIC)
);

CREATE INDEX shipment_items_sales_order_item_id_idx ON shipment_items(sales_order_item_id);

CREATE TABLE invoices (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number    VARCHAR(40) NOT NULL UNIQUE,
    invoice_type      VARCHAR(20) NOT NULL,
    sales_order_id    UUID,
    purchase_order_id UUID,
    customer_id       UUID REFERENCES customers(id) ON DELETE RESTRICT,
    supplier_id       UUID REFERENCES suppliers(id) ON DELETE RESTRICT,
    status            VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    issue_date        DATE NOT NULL,
    due_date          DATE NOT NULL,
    total_amount      NUMERIC(18,2) NOT NULL,
    paid_amount       NUMERIC(18,2) NOT NULL DEFAULT 0,
    created_by        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT invoices_sales_order_customer_fkey FOREIGN KEY (sales_order_id, customer_id) REFERENCES sales_orders(id, customer_id) ON DELETE RESTRICT,
    CONSTRAINT invoices_purchase_order_supplier_fkey FOREIGN KEY (purchase_order_id, supplier_id) REFERENCES purchase_orders(id, supplier_id) ON DELETE RESTRICT,
    CONSTRAINT invoices_type_check CHECK (
        (invoice_type = 'AR' AND sales_order_id IS NOT NULL AND customer_id IS NOT NULL AND purchase_order_id IS NULL AND supplier_id IS NULL)
        OR (invoice_type = 'AP' AND purchase_order_id IS NOT NULL AND supplier_id IS NOT NULL AND sales_order_id IS NULL AND customer_id IS NULL)
    ),
    CONSTRAINT invoices_status_check CHECK (status IN ('DRAFT', 'ISSUED', 'PARTIALLY_PAID', 'PAID', 'CANCELLED')),
    CONSTRAINT invoices_due_date_check CHECK (due_date >= issue_date),
    CONSTRAINT invoices_total_amount_check CHECK (total_amount >= 0 AND total_amount <> 'NaN'::NUMERIC),
    CONSTRAINT invoices_paid_amount_check CHECK (paid_amount >= 0 AND paid_amount <= total_amount),
    CONSTRAINT invoices_payment_status_check CHECK (
        (status IN ('DRAFT', 'ISSUED', 'CANCELLED') AND paid_amount = 0)
        OR (status = 'PARTIALLY_PAID' AND paid_amount > 0 AND paid_amount < total_amount)
        OR (status = 'PAID' AND paid_amount = total_amount)
    )
);

CREATE UNIQUE INDEX invoices_sales_order_id_key ON invoices(sales_order_id) WHERE sales_order_id IS NOT NULL;
CREATE UNIQUE INDEX invoices_purchase_order_id_key ON invoices(purchase_order_id) WHERE purchase_order_id IS NOT NULL;
CREATE INDEX invoices_customer_id_idx ON invoices(customer_id);
CREATE INDEX invoices_supplier_id_idx ON invoices(supplier_id);
CREATE INDEX invoices_created_by_idx ON invoices(created_by);
CREATE INDEX invoices_type_status_due_date_idx ON invoices(invoice_type, status, due_date, id);

CREATE TABLE payments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_number   VARCHAR(40) NOT NULL UNIQUE,
    invoice_id       UUID NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    amount           NUMERIC(18,2) NOT NULL,
    payment_method   VARCHAR(30) NOT NULL,
    reference_number VARCHAR(100),
    status           VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    paid_at          TIMESTAMPTZ NOT NULL,
    created_by       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT payments_amount_check CHECK (amount > 0 AND amount <> 'NaN'::NUMERIC),
    CONSTRAINT payments_method_check CHECK (LENGTH(BTRIM(payment_method)) > 0),
    CONSTRAINT payments_status_check CHECK (status IN ('DRAFT', 'POSTED', 'CANCELLED'))
);

CREATE INDEX payments_invoice_status_idx ON payments(invoice_id, status);
CREATE INDEX payments_created_by_idx ON payments(created_by);
CREATE INDEX payments_paid_at_idx ON payments(paid_at DESC, id DESC);

CREATE TABLE approval_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type VARCHAR(30) NOT NULL,
    document_id   UUID NOT NULL,
    requested_by  UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approver_id   UUID REFERENCES users(id) ON DELETE RESTRICT,
    status        VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    requested_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    decided_at    TIMESTAMPTZ,
    decision_note TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT approval_requests_status_check CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'CANCELLED')),
    CONSTRAINT approval_requests_approver_check CHECK (approver_id IS NULL OR approver_id <> requested_by),
    CONSTRAINT approval_requests_decision_check CHECK (status NOT IN ('APPROVED', 'REJECTED') OR (approver_id IS NOT NULL AND decided_at IS NOT NULL)),
    CONSTRAINT approval_requests_decided_at_check CHECK (decided_at IS NULL OR decided_at >= requested_at)
);

CREATE UNIQUE INDEX approval_requests_pending_document_key ON approval_requests(document_type, document_id) WHERE status = 'PENDING';
CREATE INDEX approval_requests_document_idx ON approval_requests(document_type, document_id);
CREATE INDEX approval_requests_requested_by_idx ON approval_requests(requested_by);
CREATE INDEX approval_requests_approver_status_idx ON approval_requests(approver_id, status);

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID REFERENCES users(id) ON DELETE RESTRICT,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id   UUID NOT NULL,
    old_values  JSONB,
    new_values  JSONB,
    request_id  UUID NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX audit_logs_actor_id_idx ON audit_logs(actor_id);
CREATE INDEX audit_logs_entity_occurred_at_idx ON audit_logs(entity_type, entity_id, occurred_at DESC, id DESC);
CREATE INDEX audit_logs_request_id_idx ON audit_logs(request_id);
CREATE INDEX audit_logs_occurred_at_idx ON audit_logs(occurred_at DESC, id DESC);

CREATE TABLE idempotency_records (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key           VARCHAR(100) NOT NULL,
    actor_id      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    operation     VARCHAR(100) NOT NULL,
    request_hash  VARCHAR(64) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'PROCESSING',
    response_code INT,
    response_body JSONB,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT idempotency_records_actor_operation_key UNIQUE (actor_id, operation, key),
    CONSTRAINT idempotency_records_key_check CHECK (LENGTH(BTRIM(key)) > 0),
    CONSTRAINT idempotency_records_operation_check CHECK (LENGTH(BTRIM(operation)) > 0),
    CONSTRAINT idempotency_records_request_hash_check CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT idempotency_records_status_check CHECK (status IN ('PROCESSING', 'COMPLETED')),
    CONSTRAINT idempotency_records_response_check CHECK (
        (status = 'PROCESSING' AND response_code IS NULL AND response_body IS NULL)
        OR (status = 'COMPLETED' AND response_code BETWEEN 200 AND 599 AND response_code IS NOT NULL AND response_body IS NOT NULL)
    ),
    CONSTRAINT idempotency_records_expires_at_check CHECK (expires_at > created_at)
);

CREATE INDEX idempotency_records_expires_at_idx ON idempotency_records(expires_at);

CREATE TABLE document_sequences (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_type VARCHAR(30) NOT NULL,
    period        VARCHAR(7) NOT NULL,
    last_number   BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT document_sequences_type_period_key UNIQUE (document_type, period),
    CONSTRAINT document_sequences_period_check CHECK (period ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    CONSTRAINT document_sequences_last_number_check CHECK (last_number >= 0)
);

CREATE FUNCTION prevent_immutable_changes() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% records are immutable', TG_TABLE_NAME USING ERRCODE = '55000';
END;
$$;

CREATE TRIGGER stock_movements_immutable_rows
BEFORE UPDATE OR DELETE ON stock_movements
FOR EACH ROW EXECUTE FUNCTION prevent_immutable_changes();

CREATE TRIGGER stock_movements_immutable_truncate
BEFORE TRUNCATE ON stock_movements
FOR EACH STATEMENT EXECUTE FUNCTION prevent_immutable_changes();

CREATE TRIGGER audit_logs_immutable_rows
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_immutable_changes();

CREATE TRIGGER audit_logs_immutable_truncate
BEFORE TRUNCATE ON audit_logs
FOR EACH STATEMENT EXECUTE FUNCTION prevent_immutable_changes();

COMMENT ON TABLE purchase_orders IS 'Application services must enforce status transitions and close orders only after all receiving obligations are fulfilled.';
COMMENT ON TABLE goods_receipt_items IS 'Posting must lock the purchase order, validate item ownership and cumulative received quantities, and update movements, balances, status and audit in one transaction.';
COMMENT ON TABLE stock_movements IS 'Reference type, document and item must be validated by the posting service in the same transaction as the movement. Only posted movements are inserted.';
COMMENT ON TABLE inventory_balances IS 'Operational balance updated with stock movements or reservations in the same transaction, using row locks or conditional updates to prevent overselling.';
COMMENT ON TABLE stock_reservations IS 'Application services must verify the order warehouse and aggregate reserved and shipped quantities under inventory row locks.';
COMMENT ON TABLE shipment_items IS 'Posting must validate item ownership and cumulative shipped quantities against ordered and reserved quantities in one transaction.';
COMMENT ON TABLE invoices IS 'Issuing must lock and validate the source order and copy its total. Payment posting must lock the invoice before updating paid_amount.';
COMMENT ON TABLE payments IS 'Only POSTED payments contribute to invoice paid_amount. Posting must validate the aggregate payment amount under an invoice row lock.';
COMMENT ON TABLE approval_requests IS 'The approval service must validate document_type and document_id and enforce separation of duties against the original document creator in the same transaction.';
COMMENT ON TABLE audit_logs IS 'Append-only records. The application service validates polymorphic entity references and inserts audit records in the business transaction.';
COMMENT ON TABLE idempotency_records IS 'The application service must compare request hashes and serialize concurrent operations before replaying a completed response.';
COMMENT ON TABLE document_sequences IS 'Allocate document numbers with an atomic upsert or row lock inside the document creation transaction.';
