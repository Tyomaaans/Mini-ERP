package purchaseorder

import (
	"context"
	"errors"
	"testing"

	"mini-erp/pkg"
)

func TestServiceAuthorizationIsEnforcedWithoutHTTPMiddleware(t *testing.T) {
	service := NewPurchaseOrderService(nil, nil, nil, nil, nil, nil)
	ctx := pkg.WithActor(context.Background(), pkg.Actor{
		ID:          "00000000-0000-0000-0000-000000000001",
		Roles:       []string{"SALES_STAFF"},
		Permissions: []string{"sales_orders.create"},
	})
	_, err := service.Approve(ctx, "00000000-0000-0000-0000-000000000002", "test", "")
	if !errors.Is(err, pkg.ErrForbidden) {
		t.Fatalf("approval authorization=%v", err)
	}
}
