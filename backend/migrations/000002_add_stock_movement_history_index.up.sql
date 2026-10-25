CREATE INDEX stock_movements_warehouse_product_occurred_at_idx ON stock_movements(warehouse_id, product_id, occurred_at DESC, id DESC);
