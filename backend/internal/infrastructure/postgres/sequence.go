package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DocumentSequence struct{ db *gorm.DB }

func NewDocumentSequence(db *gorm.DB) *DocumentSequence {
	return &DocumentSequence{db: db}
}

func (s *DocumentSequence) Next(ctx context.Context, documentType string) (string, error) {
	period := time.Now().UTC().Format("2006-01")
	var number int64
	err := DB(ctx, s.db).Raw(`INSERT INTO document_sequences (document_type, period, last_number) VALUES (?, ?, 1)
		ON CONFLICT (document_type, period) DO UPDATE SET last_number = document_sequences.last_number + 1, updated_at = CURRENT_TIMESTAMP
		RETURNING last_number`, documentType, period).Scan(&number).Error
	if err != nil {
		return "", HandleDBError(err)
	}
	return fmt.Sprintf("%s-%s-%06d", documentType, period, number), nil
}
