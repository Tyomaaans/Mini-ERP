package report

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/reporting/domain"
	"mini-erp/pkg"
)

type reportRepository struct{ db *gorm.DB }

func NewReportRepository(db *gorm.DB) domain.ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) read(ctx context.Context, sql string, args []any, query pkg.ListQuery, columns map[string]string, result any) (int64, error) {
	db := postgres.DB(ctx, r.db)
	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM ("+sql+") report_rows", args...).Scan(&total).Error; err != nil {
		return 0, err
	}

	args = append(args, query.PerPage, query.Offset())
	err := db.Raw("SELECT * FROM ("+sql+") report_rows ORDER BY "+query.Order(columns)+" LIMIT ? OFFSET ?", args...).Scan(result).Error
	return total, err
}

func (r *reportRepository) Summary(ctx context.Context, kind string, query pkg.ListQuery, scope []string) ([]domain.SummaryEntity, int64, error) {
	table := "sales_orders"
	states := "'CONFIRMED','PARTIALLY_SHIPPED','SHIPPED'"
	if kind == "purchasing" {
		table = "purchase_orders"
		states = "'APPROVED','CLOSED'"
	} else if kind != "sales" {
		return nil, 0, pkg.ErrInvalidInput
	}

	sql := "SELECT order_date::TEXT AS id, order_date::TEXT AS date, COUNT(*) AS order_count, SUM(total_amount) AS total_amount FROM " + table + " WHERE status IN (" + states + ")"
	args := []any{}
	for _, filter := range []struct{ key, operator string }{{"from", ">="}, {"to", "<="}} {
		if value := query.Filters[filter.key]; value != "" {
			sql += " AND order_date " + filter.operator + " ?"
			args = append(args, value)
		}
	}

	if value := query.Filters["warehouse_id"]; value != "" {
		sql += " AND warehouse_id = ?"
		args = append(args, value)
	}

	if len(scope) > 0 {
		sql += " AND warehouse_id IN ?"
		args = append(args, scope)
	}

	sql += " GROUP BY order_date"
	rows := []domain.SummaryEntity{}
	total, err := r.read(ctx, sql, args, query, map[string]string{"date": "date", "order_count": "order_count", "total_amount": "total_amount"}, &rows)
	return rows, total, err
}

func (r *reportRepository) LowStock(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.LowStockEntity, int64, error) {
	sql := `SELECT p.id::TEXT || ':' || w.id::TEXT AS id, p.id AS product_id, w.id AS warehouse_id, p.sku, p.name, w.name AS warehouse,
		COALESCE(b.quantity_on_hand,0) AS quantity_on_hand, COALESCE(b.quantity_reserved,0) AS quantity_reserved,
		COALESCE(b.quantity_on_hand-b.quantity_reserved,0) AS quantity_available,p.reorder_level
		FROM products p CROSS JOIN warehouses w LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=w.id
		WHERE p.is_active AND w.is_active AND COALESCE(b.quantity_on_hand-b.quantity_reserved,0) < p.reorder_level`
	args := []any{}
	if value := query.Filters["warehouse_id"]; value != "" {
		sql += " AND w.id = ?"
		args = append(args, value)
	}

	if value := query.Filters["product_id"]; value != "" {
		sql += " AND p.id = ?"
		args = append(args, value)
	}

	if query.Search != "" {
		sql += " AND (p.name ILIKE ? OR p.sku ILIKE ?)"
		args = append(args, "%"+query.Search+"%", "%"+query.Search+"%")
	}

	if len(scope) > 0 {
		sql += " AND w.id IN ?"
		args = append(args, scope)
	}

	rows := []domain.LowStockEntity{}
	total, err := r.read(ctx, sql, args, query, map[string]string{"quantity_available": "quantity_available", "sku": "sku", "reorder_level": "reorder_level"}, &rows)
	return rows, total, err
}

func (r *reportRepository) Outstanding(ctx context.Context, kind string, query pkg.ListQuery) ([]domain.OutstandingEntity, int64, error) {
	party, partyID := "customers", "customer_id"
	if kind == "AP" {
		party, partyID = "suppliers", "supplier_id"
	} else if kind != "AR" {
		return nil, 0, pkg.ErrInvalidInput
	}

	sql := `SELECT i.id,i.invoice_number,i.` + partyID + ` AS party_id,p.name AS party_name,i.issue_date,i.due_date,i.total_amount,i.paid_amount,i.total_amount-i.paid_amount AS outstanding_amount
		FROM invoices i JOIN ` + party + ` p ON p.id=i.` + partyID + ` WHERE i.invoice_type = ? AND i.status IN ('ISSUED','PARTIALLY_PAID') AND i.total_amount > i.paid_amount`
	args := []any{kind}
	if query.Filters["overdue"] == "true" {
		sql += " AND i.due_date < CURRENT_DATE"
	}

	if query.Filters["overdue"] == "false" {
		sql += " AND i.due_date >= CURRENT_DATE"
	}

	if value := query.Filters["party_id"]; value != "" {
		sql += " AND i." + partyID + " = ?"
		args = append(args, value)
	}

	if query.Search != "" {
		sql += " AND (i.invoice_number ILIKE ? OR p.name ILIKE ?)"
		args = append(args, "%"+query.Search+"%", "%"+query.Search+"%")
	}

	rows := []domain.OutstandingEntity{}
	total, err := r.read(ctx, sql, args, query, map[string]string{"due_date": "due_date", "outstanding_amount": "outstanding_amount", "invoice_number": "invoice_number"}, &rows)
	return rows, total, err
}

func (r *reportRepository) AuditLogs(ctx context.Context, query pkg.ListQuery) ([]domain.AuditEntity, int64, error) {
	parts := []string{"TRUE"}
	args := []any{}
	for _, key := range []string{"actor_id", "entity_type", "entity_id", "request_id", "action"} {
		if value := query.Filters[key]; value != "" {
			parts = append(parts, key+" = ?")
			args = append(args, value)
		}
	}

	if value := query.Filters["from"]; value != "" {
		parts = append(parts, "occurred_at >= ?::DATE")
		args = append(args, value)
	}

	if value := query.Filters["to"]; value != "" {
		parts = append(parts, "occurred_at < ?::DATE + INTERVAL '1 day'")
		args = append(args, value)
	}

	if query.Search != "" {
		parts = append(parts, "action ILIKE ?")
		args = append(args, "%"+query.Search+"%")
	}

	sql := "SELECT id,actor_id,action,entity_type,entity_id,old_values::TEXT AS old_values,new_values::TEXT AS new_values,request_id,occurred_at FROM audit_logs WHERE " + strings.Join(parts, " AND ")
	rows := []domain.AuditEntity{}
	total, err := r.read(ctx, sql, args, query, map[string]string{"occurred_at": "occurred_at", "action": "action"}, &rows)
	return rows, total, err
}
