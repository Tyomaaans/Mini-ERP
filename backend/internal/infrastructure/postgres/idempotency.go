package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"

	"mini-erp/pkg"
)

type IdempotencyExecutor struct {
	db        *gorm.DB
	txManager pkg.TransactionManager
}

func NewIdempotencyExecutor(db *gorm.DB, txManager pkg.TransactionManager) *IdempotencyExecutor {
	return &IdempotencyExecutor{
		db:        db,
		txManager: txManager,
	}
}

func (e *IdempotencyExecutor) Execute(ctx context.Context, actor pkg.Actor, operation, key string, request any, execute func(context.Context) (any, error)) (json.RawMessage, error) {
	if strings.TrimSpace(key) == "" || len(key) > 100 {
		return nil, pkg.NewError("IDEMPOTENCY_KEY_REQUIRED", "Provide an Idempotency-Key header of 1 to 100 characters", 400)
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	requestHash := hex.EncodeToString(hash[:])
	var response json.RawMessage

	err = e.txManager.Within(ctx, func(ctx context.Context) error {
		db := DB(ctx, e.db)
		if err := db.Exec(`DELETE FROM idempotency_records WHERE actor_id = ? AND operation = ? AND key = ? AND expires_at <= CURRENT_TIMESTAMP`, actor.ID, operation, key).Error; err != nil {
			return err
		}
		if err := db.Exec(`INSERT INTO idempotency_records (actor_id, operation, key, request_hash, expires_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP + INTERVAL '24 hours') ON CONFLICT (actor_id, operation, key) DO NOTHING`, actor.ID, operation, key, requestHash).Error; err != nil {
			return err
		}
		var record struct {
			RequestHash string
			Status      string
			Response    string
			CreatedAt   time.Time
		}
		if err := db.Raw(`SELECT request_hash, status, response_body::TEXT AS response, created_at FROM idempotency_records
			WHERE actor_id = ? AND operation = ? AND key = ? FOR UPDATE`, actor.ID, operation, key).Scan(&record).Error; err != nil {
			return err
		}
		if record.RequestHash != requestHash {
			return pkg.NewError("IDEMPOTENCY_CONFLICT", "Idempotency key was already used for a different request", 409)
		}
		if record.Status == "COMPLETED" {
			response = json.RawMessage(record.Response)
			return nil
		}
		result, err := execute(ctx)
		if err != nil {
			return err
		}
		response, err = json.Marshal(result)
		if err != nil {
			return err
		}
		return db.Exec(`UPDATE idempotency_records SET status = 'COMPLETED', response_code = 200, response_body = ?::JSONB, updated_at = CURRENT_TIMESTAMP
			WHERE actor_id = ? AND operation = ? AND key = ?`, string(response), actor.ID, operation, key).Error
	})
	return response, err
}
