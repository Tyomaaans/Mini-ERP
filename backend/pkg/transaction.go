package pkg

import (
	"context"
	"encoding/json"
)

type TransactionManager interface {
	Within(ctx context.Context, operation func(context.Context) error) error
}

type AuditLogger interface {
	Record(ctx context.Context, actor Actor, action, entityType, entityID string, oldValues, newValues any) error
}

type DocumentSequence interface {
	Next(ctx context.Context, documentType string) (string, error)
}

type IdempotencyExecutor interface {
	Execute(ctx context.Context, actor Actor, operation, key string, request any, execute func(context.Context) (any, error)) (json.RawMessage, error)
}
