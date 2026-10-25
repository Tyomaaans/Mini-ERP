package pkg

import "testing"

func TestPermissionsDenyByDefaultAndWarehouseScopeIsExplicit(t *testing.T) {
	actor := Actor{
		ID:         "00000000-0000-0000-0000-000000000001",
		Department: "IT",
		Roles:      []string{"SUPER_ADMIN"},
	}
	if actor.Require("purchase_orders.approve") == nil {
		t.Fatal("department or role name bypassed explicit permissions")
	}
	actor.Permissions = []string{"inventory.read"}
	if actor.Require("inventory.read") != nil || actor.Require("shipments.post") == nil {
		t.Fatal("permissions were not enforced")
	}
	actor.WarehouseIDs = []string{"warehouse-a"}
	if actor.RequireWarehouse("warehouse-b") == nil {
		t.Fatal("warehouse scope was bypassed")
	}
	actor.Roles = []string{"WAREHOUSE_STAFF"}
	actor.WarehouseIDs = nil
	if len(actor.WarehouseScope()) == 0 {
		t.Fatal("unassigned warehouse staff acquired global read access")
	}
}
