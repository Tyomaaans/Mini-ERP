package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mini-erp/pkg"
)

type AuditLogger struct{ db *gorm.DB }

func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

func (a *AuditLogger) Record(ctx context.Context, actor pkg.Actor, action, entityType, entityID string, oldValues, newValues any) error {
	oldJSON, err := json.Marshal(oldValues)
	if err != nil {
		return err
	}
	newJSON, err := json.Marshal(newValues)
	if err != nil {
		return err
	}
	requestID := pkg.RequestID(ctx)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return HandleDBError(DB(ctx, a.db).Exec(`INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, old_values, new_values, request_id)
		VALUES (NULLIF(?, '')::UUID, ?, ?, ?::UUID, ?::JSONB, ?::JSONB, ?::UUID)`, actor.ID, action, entityType, entityID, string(oldJSON), string(newJSON), requestID).Error)
}
