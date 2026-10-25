package invoice

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-erp/internal/finance/domain"
	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/pkg"
)

type invoiceRepository struct{ db *gorm.DB }

func NewInvoiceRepository(db *gorm.DB) domain.InvoiceRepository {
	return &invoiceRepository{db: db}
}

func (r *invoiceRepository) LoadSource(ctx context.Context, invoiceType, id string, lock bool) (*domain.InvoiceSource, error) {
	table, party := "purchase_orders", "supplier_id"
	if invoiceType == "AR" {
		table, party = "sales_orders", "customer_id"
	} else if invoiceType != "AP" {
		return nil, pkg.ErrInvalidInput
	}

	query := "SELECT id, " + party + " AS party_id, status, total_amount FROM " + table + " WHERE id = ?"
	if lock {
		query += " FOR UPDATE"
	}

	var source domain.InvoiceSource
	result := postgres.DB(ctx, r.db).Raw(query, id).Scan(&source)
	if result.Error != nil {
		return nil, postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, pkg.ErrNotFound
	}

	return &source, nil
}

func (r *invoiceRepository) CreateInvoice(ctx context.Context, entity *domain.InvoiceEntity) error {
	storage := ToInvoiceStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToInvoiceEntity(storage)
	return nil
}

func (r *invoiceRepository) GetInvoice(ctx context.Context, id string, lock bool) (*domain.InvoiceEntity, error) {
	db := postgres.DB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var storage InvoiceStorage
	if err := db.Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToInvoiceEntity(&storage), nil
}

func (r *invoiceRepository) GetInvoices(ctx context.Context, query pkg.ListQuery) ([]domain.InvoiceEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&InvoiceStorage{})
	if query.Search != "" {
		db = db.Where("invoice_number ILIKE ?", "%"+query.Search+"%")
	}

	for _, key := range []string{"status", "invoice_type", "customer_id", "supplier_id", "sales_order_id", "purchase_order_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	if value := query.Filters["from"]; value != "" {
		db = db.Where("issue_date >= ?", value)
	}

	if value := query.Filters["to"]; value != "" {
		db = db.Where("issue_date <= ?", value)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []InvoiceStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "due_date": "due_date", "total_amount": "total_amount", "invoice_number": "invoice_number"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.InvoiceEntity, len(rows))
	for i := range rows {
		entities[i] = *ToInvoiceEntity(&rows[i])
	}

	return entities, total, nil
}

func (r *invoiceRepository) UpdateInvoice(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&InvoiceStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}
