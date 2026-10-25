package payment

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-erp/internal/finance/domain"
	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/pkg"
)

type paymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) domain.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreatePayment(ctx context.Context, entity *domain.PaymentEntity) error {
	storage := ToPaymentStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToPaymentEntity(storage)
	return nil
}

func (r *paymentRepository) GetPayment(ctx context.Context, id string, lock bool) (*domain.PaymentEntity, error) {
	db := postgres.DB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var storage PaymentStorage
	if err := db.Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToPaymentEntity(&storage), nil
}

func (r *paymentRepository) GetPayments(ctx context.Context, query pkg.ListQuery) ([]domain.PaymentEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&PaymentStorage{})
	if query.Search != "" {
		db = db.Where("payment_number ILIKE ?", "%"+query.Search+"%")
	}

	for _, key := range []string{"invoice_id", "status", "payment_method"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []PaymentStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "paid_at": "paid_at", "amount": "amount"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.PaymentEntity, len(rows))
	for i := range rows {
		entities[i] = *ToPaymentEntity(&rows[i])
	}

	return entities, total, nil
}

func (r *paymentRepository) PostedAmount(ctx context.Context, invoiceID string) (decimal.Decimal, error) {
	var amount decimal.Decimal
	err := postgres.DB(ctx, r.db).Raw("SELECT COALESCE(SUM(amount), 0) FROM payments WHERE invoice_id = ? AND status = 'POSTED'", invoiceID).Scan(&amount).Error
	return amount, err
}

func (r *paymentRepository) Post(ctx context.Context, id string) error {
	return postgres.DB(ctx, r.db).Exec("UPDATE payments SET status = 'POSTED', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id).Error
}
