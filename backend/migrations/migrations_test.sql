-- Run against a disposable PostgreSQL database after applying all up migrations:
-- psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f migrations/migrations_test.sql

BEGIN;

CREATE FUNCTION pg_temp.assert_rejected(statement TEXT, expected_state TEXT) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    BEGIN
        EXECUTE statement;
    EXCEPTION WHEN OTHERS THEN
        IF SQLSTATE <> expected_state THEN
            RAISE EXCEPTION 'Expected SQLSTATE %, received %: %', expected_state, SQLSTATE, SQLERRM;
        END IF;
        RETURN;
    END;

    RAISE EXCEPTION 'Statement unexpectedly succeeded: %', statement;
END;
$$;

DO $$
DECLARE
    expected_tables TEXT[] := ARRAY[
        'departments', 'users', 'roles', 'permissions', 'user_roles', 'role_permissions',
        'categories', 'products', 'suppliers', 'customers', 'warehouses', 'user_warehouses',
        'purchase_orders', 'purchase_order_items', 'goods_receipts', 'goods_receipt_items',
        'inventory_balances', 'stock_movements', 'stock_reservations',
        'sales_orders', 'sales_order_items', 'shipments', 'shipment_items',
        'invoices', 'payments', 'approval_requests', 'audit_logs',
        'idempotency_records', 'document_sequences'
    ];
    expected_table TEXT;
BEGIN
    FOREACH expected_table IN ARRAY expected_tables LOOP
        IF to_regclass('public.' || expected_table) IS NULL THEN
            RAISE EXCEPTION 'Missing table: %', expected_table;
        END IF;
    END LOOP;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = ANY(expected_tables)
        AND data_type = 'numeric' AND (
            numeric_precision <> 18
            OR numeric_scale <> CASE
                WHEN column_name IN ('quantity', 'quantity_received', 'quantity_shipped', 'quantity_on_hand', 'quantity_reserved', 'quantity_delta', 'reorder_level') THEN 3
                ELSE 2
            END
        )
    ) THEN
        RAISE EXCEPTION 'Money and quantity precision does not match the API contract';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = ANY(expected_tables)
        AND column_name LIKE '%\_at' ESCAPE '\' AND data_type <> 'timestamp with time zone'
    ) THEN
        RAISE EXCEPTION 'Timestamp columns must preserve time zones';
    END IF;
END;
$$;

INSERT INTO departments (id, code, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'TEST', 'Migration Test');

INSERT INTO users (id, department_id, full_name, email, password_hash)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Purchaser', 'purchaser@example.test', 'test-only-not-a-login-password'),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Manager', 'manager@example.test', 'test-only-not-a-login-password');

INSERT INTO roles (id, code, name)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'PURCHASING_STAFF', 'Purchasing'),
    ('00000000-0000-0000-0000-000000000002', 'MANAGER', 'Manager');

INSERT INTO permissions (id, code, module, action)
VALUES ('00000000-0000-0000-0000-000000000001', 'purchase_orders.create', 'purchase_orders', 'create');

INSERT INTO user_roles (user_id, role_id)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001'),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002');

INSERT INTO role_permissions (role_id, permission_id)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001');

INSERT INTO categories (id, code, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'TEST', 'Test Category');

INSERT INTO products (id, sku, name, category_id, unit, selling_price)
VALUES ('00000000-0000-0000-0000-000000000001', 'TEST-001', 'Test Product', '00000000-0000-0000-0000-000000000001', 'KG', 80.00);

INSERT INTO suppliers (id, code, name)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'TEST-001', 'Test Supplier'),
    ('00000000-0000-0000-0000-000000000002', 'TEST-002', 'Other Supplier');

INSERT INTO customers (id, code, name)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'TEST-001', 'Test Customer'),
    ('00000000-0000-0000-0000-000000000002', 'TEST-002', 'Other Customer');

INSERT INTO warehouses (id, code, name)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'TEST-001', 'Jakarta'),
    ('00000000-0000-0000-0000-000000000002', 'TEST-002', 'Surabaya');

INSERT INTO user_warehouses (user_id, warehouse_id)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001');

INSERT INTO purchase_orders (id, po_number, supplier_id, warehouse_id, order_date, subtotal, discount_amount, total_amount, created_by)
VALUES ('00000000-0000-0000-0000-000000000001', 'PO-TEST', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '2026-10-01', 100.00, 10.00, 90.00, '00000000-0000-0000-0000-000000000001');

INSERT INTO purchase_order_items (id, purchase_order_id, product_id, quantity, unit_price, discount_amount, line_total)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 1.500, 80.00, 20.00, 100.00);

INSERT INTO goods_receipts (id, gr_number, purchase_order_id, warehouse_id, received_by)
VALUES ('00000000-0000-0000-0000-000000000001', 'GR-TEST', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002');

INSERT INTO goods_receipt_items (id, goods_receipt_id, purchase_order_item_id, quantity_received)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 0.500);

INSERT INTO inventory_balances (product_id, warehouse_id, quantity_on_hand, quantity_reserved)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 10.000, 3.000);

INSERT INTO stock_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'GOODS_RECEIPT', 0.500, 'GOODS_RECEIPT', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002');

INSERT INTO sales_orders (id, so_number, customer_id, warehouse_id, order_date, subtotal, total_amount, created_by)
VALUES ('00000000-0000-0000-0000-000000000001', 'SO-TEST', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '2026-10-01', 100.00, 100.00, '00000000-0000-0000-0000-000000000001');

INSERT INTO sales_order_items (id, sales_order_id, product_id, quantity, unit_price, line_total)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 2.000, 50.00, 100.00);

INSERT INTO stock_reservations (sales_order_item_id, warehouse_id, quantity, created_by)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 1.000, '00000000-0000-0000-0000-000000000002');

INSERT INTO shipments (id, shipment_number, sales_order_id, warehouse_id)
VALUES ('00000000-0000-0000-0000-000000000001', 'SHP-TEST', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001');

INSERT INTO shipment_items (shipment_id, sales_order_item_id, quantity_shipped)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 1.000);

INSERT INTO invoices (id, invoice_number, invoice_type, sales_order_id, purchase_order_id, customer_id, supplier_id, issue_date, due_date, total_amount, created_by)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'INV-AR-TEST', 'AR', '00000000-0000-0000-0000-000000000001', NULL, '00000000-0000-0000-0000-000000000001', NULL, '2026-10-01', '2026-10-31', 100.00, '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0000-000000000002', 'INV-AP-TEST', 'AP', NULL, '00000000-0000-0000-0000-000000000001', NULL, '00000000-0000-0000-0000-000000000001', '2026-10-01', '2026-10-31', 90.00, '00000000-0000-0000-0000-000000000002');

INSERT INTO payments (payment_number, invoice_id, amount, payment_method, paid_at, created_by)
VALUES ('PAY-TEST', '00000000-0000-0000-0000-000000000001', 25.00, 'BANK_TRANSFER', CURRENT_TIMESTAMP, '00000000-0000-0000-0000-000000000002');

INSERT INTO approval_requests (document_type, document_id, requested_by)
VALUES ('PURCHASE_ORDER', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001');

INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, request_id)
VALUES ('00000000-0000-0000-0000-000000000001', 'schema.test', 'PURCHASE_ORDER', '00000000-0000-0000-0000-000000000001', gen_random_uuid());

INSERT INTO idempotency_records (key, actor_id, operation, request_hash, expires_at)
VALUES ('schema-test', '00000000-0000-0000-0000-000000000001', 'goods_receipts.post', REPEAT('a', 64), CURRENT_TIMESTAMP + INTERVAL '1 day');

INSERT INTO document_sequences (document_type, period, last_number)
VALUES ('PO', '2026-10', 1);

DO $$
BEGIN
    -- Identity, foreign keys and warehouse assignments.
    PERFORM pg_temp.assert_rejected($q$INSERT INTO user_roles (user_id, role_id) SELECT user_id, role_id FROM user_roles LIMIT 1$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO role_permissions (role_id, permission_id) SELECT role_id, permission_id FROM role_permissions$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO user_warehouses (user_id, warehouse_id) SELECT user_id, warehouse_id FROM user_warehouses$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$UPDATE users SET department_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'$q$, '23503');
    PERFORM pg_temp.assert_rejected($q$DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001'$q$, '23503');
    PERFORM pg_temp.assert_rejected($q$UPDATE categories SET parent_id = id$q$, '23514');

    -- Money, fractional quantities and calculated order totals.
    PERFORM pg_temp.assert_rejected($q$UPDATE products SET selling_price = -0.01$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE products SET selling_price = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE products SET reorder_level = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE products SET selling_price = 10000000000000000$q$, '22003');
    PERFORM pg_temp.assert_rejected($q$UPDATE suppliers SET payment_terms_days = -1$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE customers SET credit_limit = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_orders SET total_amount = 999.00$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_orders SET subtotal = 'NaN', total_amount = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_orders SET status = 'UNKNOWN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_orders SET status = 'APPROVED', submitted_at = CURRENT_TIMESTAMP, approved_by = created_by, approved_at = CURRENT_TIMESTAMP$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_orders SET status = 'APPROVED', submitted_at = CURRENT_TIMESTAMP$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_order_items SET quantity = 0, discount_amount = 0, line_total = 0$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_order_items SET quantity = 'NaN', line_total = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_order_items SET unit_price = 'NaN', line_total = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE purchase_order_items SET line_total = 999.00$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE sales_orders SET total_amount = -1.00$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE sales_orders SET status = 'CONFIRMED'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE sales_order_items SET quantity = 'NaN', line_total = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE sales_order_items SET discount_amount = 101.00, line_total = -1.00$q$, '23514');

    -- Receiving and shipping must reference their order warehouse.
    PERFORM pg_temp.assert_rejected($q$UPDATE goods_receipts SET warehouse_id = '00000000-0000-0000-0000-000000000002'$q$, '23503');
    PERFORM pg_temp.assert_rejected($q$UPDATE goods_receipts SET status = 'POSTED'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE goods_receipt_items SET quantity_received = 0$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE goods_receipt_items SET quantity_received = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE shipments SET warehouse_id = '00000000-0000-0000-0000-000000000002'$q$, '23503');
    PERFORM pg_temp.assert_rejected($q$UPDATE shipments SET status = 'SHIPPED'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE shipment_items SET quantity_shipped = -1$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE shipment_items SET quantity_shipped = 'NaN'$q$, '23514');

    -- Balances cannot go negative or reserve more than physical stock.
    PERFORM pg_temp.assert_rejected($q$UPDATE inventory_balances SET quantity_on_hand = -1$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE inventory_balances SET quantity_reserved = -1$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE inventory_balances SET quantity_reserved = 11$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE inventory_balances SET quantity_on_hand = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO inventory_balances (product_id, warehouse_id) SELECT product_id, warehouse_id FROM inventory_balances$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$UPDATE stock_reservations SET quantity = 0$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE stock_reservations SET quantity = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE stock_reservations SET status = 'RELEASED'$q$, '23514');

    -- Posted movements and audit records cannot be rewritten or truncated.
    PERFORM pg_temp.assert_rejected($q$UPDATE stock_movements SET quantity_delta = 5$q$, '55000');
    PERFORM pg_temp.assert_rejected($q$DELETE FROM stock_movements$q$, '55000');
    PERFORM pg_temp.assert_rejected($q$TRUNCATE stock_movements$q$, '55000');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO stock_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by) SELECT product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by FROM stock_movements$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO stock_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by) SELECT product_id, warehouse_id, movement_type, -1, reference_type, gen_random_uuid(), gen_random_uuid(), performed_by FROM stock_movements$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO stock_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by) SELECT product_id, warehouse_id, movement_type, 'NaN', reference_type, gen_random_uuid(), gen_random_uuid(), performed_by FROM stock_movements$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE audit_logs SET action = 'changed'$q$, '55000');
    PERFORM pg_temp.assert_rejected($q$DELETE FROM audit_logs$q$, '55000');
    PERFORM pg_temp.assert_rejected($q$TRUNCATE audit_logs$q$, '55000');

    -- Each invoice belongs to exactly one order and its matching customer or supplier.
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET supplier_id = '00000000-0000-0000-0000-000000000001' WHERE invoice_type = 'AR'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET customer_id = NULL WHERE invoice_type = 'AR'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET sales_order_id = NULL WHERE invoice_type = 'AR'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET customer_id = '00000000-0000-0000-0000-000000000002' WHERE invoice_type = 'AR'$q$, '23503');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET customer_id = '00000000-0000-0000-0000-000000000001' WHERE invoice_type = 'AP'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET supplier_id = NULL WHERE invoice_type = 'AP'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET supplier_id = '00000000-0000-0000-0000-000000000002' WHERE invoice_type = 'AP'$q$, '23503');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO invoices (invoice_number, invoice_type, sales_order_id, customer_id, issue_date, due_date, total_amount, created_by) SELECT 'INV-DUPLICATE', invoice_type, sales_order_id, customer_id, issue_date, due_date, total_amount, created_by FROM invoices WHERE invoice_type = 'AR'$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO invoices (invoice_number, invoice_type, purchase_order_id, supplier_id, issue_date, due_date, total_amount, created_by) SELECT 'INV-DUPLICATE', invoice_type, purchase_order_id, supplier_id, issue_date, due_date, total_amount, created_by FROM invoices WHERE invoice_type = 'AP'$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET due_date = issue_date - 1$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET paid_amount = total_amount + 1, status = 'PARTIALLY_PAID'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET paid_amount = 1 WHERE status = 'DRAFT'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE invoices SET total_amount = 'NaN'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE payments SET amount = 0$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE payments SET amount = 'NaN'$q$, '23514');

    -- Governance keys and reusable idempotency responses.
    PERFORM pg_temp.assert_rejected($q$INSERT INTO approval_requests (document_type, document_id, requested_by) SELECT document_type, document_id, requested_by FROM approval_requests$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$UPDATE approval_requests SET status = 'APPROVED', approver_id = requested_by, decided_at = CURRENT_TIMESTAMP$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO idempotency_records (key, actor_id, operation, request_hash, expires_at) SELECT key, actor_id, operation, request_hash, expires_at FROM idempotency_records$q$, '23505');
    PERFORM pg_temp.assert_rejected($q$UPDATE idempotency_records SET status = 'COMPLETED'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE idempotency_records SET request_hash = 'invalid'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE idempotency_records SET expires_at = created_at - INTERVAL '1 second'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE document_sequences SET period = '2026-13'$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$UPDATE document_sequences SET last_number = -1$q$, '23514');
    PERFORM pg_temp.assert_rejected($q$INSERT INTO document_sequences (document_type, period) SELECT document_type, period FROM document_sequences$q$, '23505');

    -- Valid updates and isolation between different idempotency operations.
    UPDATE purchase_orders SET status = 'APPROVED', submitted_at = CURRENT_TIMESTAMP,
        approved_by = '00000000-0000-0000-0000-000000000002', approved_at = CURRENT_TIMESTAMP;
    UPDATE goods_receipts SET status = 'POSTED', received_at = CURRENT_TIMESTAMP;
    UPDATE sales_orders SET status = 'CONFIRMED', confirmed_at = CURRENT_TIMESTAMP;
    UPDATE shipments SET status = 'SHIPPED', shipped_at = CURRENT_TIMESTAMP, shipped_by = '00000000-0000-0000-0000-000000000002';
    UPDATE stock_reservations SET status = 'CONSUMED', released_at = CURRENT_TIMESTAMP;
    UPDATE invoices SET status = 'PARTIALLY_PAID', paid_amount = 25.00 WHERE invoice_type = 'AR';
    UPDATE invoices SET status = 'PAID', paid_amount = total_amount WHERE invoice_type = 'AP';
    UPDATE idempotency_records SET status = 'COMPLETED', response_code = 200, response_body = '{"data":{"status":"POSTED"}}';
    INSERT INTO idempotency_records (key, actor_id, operation, request_hash, expires_at)
        SELECT key, actor_id, 'shipments.post', request_hash, expires_at FROM idempotency_records;

    IF (SELECT COUNT(*) FROM user_roles WHERE user_id = '00000000-0000-0000-0000-000000000001') <> 2 THEN
        RAISE EXCEPTION 'A failed operation must preserve the multi-role user';
    END IF;

    IF (SELECT quantity_received FROM goods_receipt_items) <> 0.500 THEN
        RAISE EXCEPTION 'Fractional quantities must preserve their precision';
    END IF;

    RAISE NOTICE 'Migration schema and constraint tests passed';
END;
$$;

-- Repeated timestamps exercise the ID ordering used for stable history pagination.
INSERT INTO stock_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by, occurred_at)
SELECT '00000000-0000-0000-0000-000000000001'::UUID,
    CASE WHEN n % 2 = 0 THEN '00000000-0000-0000-0000-000000000001'::UUID ELSE '00000000-0000-0000-0000-000000000002'::UUID END,
    'GOODS_RECEIPT', 1.000, 'GOODS_RECEIPT', gen_random_uuid(), gen_random_uuid(),
    '00000000-0000-0000-0000-000000000002'::UUID, CURRENT_TIMESTAMP - (n % 100) * INTERVAL '1 day'
FROM generate_series(1, 40000) AS n;

ANALYZE stock_movements;

DO $$
DECLARE
    query_plan JSONB;
BEGIN
    EXECUTE $query$
        EXPLAIN (FORMAT JSON)
        SELECT id, quantity_delta, occurred_at FROM stock_movements
        WHERE warehouse_id = '00000000-0000-0000-0000-000000000001'
        AND product_id = '00000000-0000-0000-0000-000000000001'
        ORDER BY occurred_at DESC, id DESC LIMIT 20
    $query$ INTO query_plan;

    IF jsonb_path_exists(query_plan, '$.**."Node Type" ? (@ == "Sort" || @ == "Incremental Sort")') THEN
        RAISE EXCEPTION 'Stock history query requires an extra sort: %', query_plan;
    END IF;

    IF NOT jsonb_path_exists(query_plan, '$.**."Index Name" ? (@ == "stock_movements_warehouse_product_occurred_at_idx")') THEN
        RAISE EXCEPTION 'Stock history query does not use its composite index: %', query_plan;
    END IF;

    RAISE NOTICE 'Stock history query plan test passed';
END;
$$;

ROLLBACK;
