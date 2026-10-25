package pkg

import "context"

type contextKey int

const (
	actorKey contextKey = iota
	requestIDKey
)

type Actor struct {
	ID           string
	Name         string
	Department   string
	Roles        []string
	Permissions  []string
	WarehouseIDs []string
}

func (a Actor) HasRole(role string) bool {
	for _, code := range a.Roles {
		if code == role {
			return true
		}
	}
	return false
}

func (a Actor) Require(permission string) error {
	for _, code := range a.Permissions {
		if code == permission && a.ID != "" {
			return nil
		}
	}
	return ErrForbidden
}

func (a Actor) RequireWarehouse(id string) error {
	for _, warehouseID := range a.WarehouseIDs {
		if warehouseID == id {
			return nil
		}
	}
	return ErrForbidden
}

func (a Actor) WarehouseScope() []string {
	if a.HasRole("WAREHOUSE_STAFF") && !a.HasRole("MANAGER") && !a.HasRole("SUPER_ADMIN") {
		if len(a.WarehouseIDs) == 0 {
			return []string{"00000000-0000-0000-0000-000000000000"}
		}
		return a.WarehouseIDs
	}
	return nil
}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

func ActorFrom(ctx context.Context) Actor {
	actor, _ := ctx.Value(actorKey).(Actor)
	return actor
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
