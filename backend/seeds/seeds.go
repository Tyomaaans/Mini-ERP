package seeds

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"sort"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var RolePermissions = map[string][]string{
	"SUPER_ADMIN": {
		"users.read",
		"users.create",
		"users.update",
		"users.assign_role",
		"users.assign_warehouse",
		"roles.manage",
		"products.read",
		"products.create",
		"products.update",
		"categories.read",
		"categories.create",
		"categories.update",
		"categories.delete",
		"customers.read",
		"suppliers.read",
		"warehouses.read",
		"warehouses.create",
		"warehouses.update",
		"purchase_orders.read",
		"goods_receipts.read",
		"inventory.read",
		"sales_orders.read",
		"shipments.read",
		"invoices.read",
		"payments.read",
		"reports.sales.read",
		"reports.purchasing.read",
		"reports.inventory.read",
		"reports.finance.read",
		"audit_logs.read",
		"system.metrics.read",
	},
	"SALES_STAFF": {
		"products.read",
		"categories.read",
		"warehouses.read",
		"customers.read",
		"customers.create",
		"customers.update",
		"inventory.read",
		"sales_orders.read",
		"sales_orders.create",
		"sales_orders.update",
		"sales_orders.confirm",
		"sales_orders.cancel",
		"shipments.read",
		"invoices.read",
		"reports.sales.read",
	},
	"PURCHASING_STAFF": {
		"products.read",
		"categories.read",
		"warehouses.read",
		"suppliers.read",
		"suppliers.create",
		"suppliers.update",
		"purchase_orders.read",
		"purchase_orders.create",
		"purchase_orders.update",
		"purchase_orders.submit",
		"purchase_orders.cancel",
		"goods_receipts.read",
		"inventory.read",
		"invoices.read",
		"reports.purchasing.read",
	},
	"WAREHOUSE_STAFF": {
		"products.read",
		"categories.read",
		"warehouses.read",
		"customers.read",
		"suppliers.read",
		"purchase_orders.read",
		"goods_receipts.read",
		"goods_receipts.create",
		"goods_receipts.post",
		"inventory.read",
		"inventory.reserve",
		"sales_orders.read",
		"shipments.read",
		"shipments.create",
		"shipments.post",
		"reports.inventory.read",
	},
	"FINANCE_STAFF": {
		"products.read",
		"categories.read",
		"warehouses.read",
		"customers.read",
		"suppliers.read",
		"purchase_orders.read",
		"goods_receipts.read",
		"inventory.read",
		"sales_orders.read",
		"shipments.read",
		"invoices.read",
		"invoices.create",
		"invoices.update",
		"invoices.issue",
		"payments.read",
		"payments.create",
		"payments.post",
		"reports.finance.read",
	},
	"MANAGER": {
		"users.read",
		"products.read",
		"categories.read",
		"warehouses.read",
		"customers.read",
		"suppliers.read",
		"purchase_orders.read",
		"purchase_orders.approve",
		"goods_receipts.read",
		"inventory.read",
		"sales_orders.read",
		"shipments.read",
		"invoices.read",
		"payments.read",
		"reports.sales.read",
		"reports.purchasing.read",
		"reports.inventory.read",
		"reports.finance.read",
		"audit_logs.read",
		"system.metrics.read",
	},
}

var DemoAccounts = []struct{ Role, Department, Name, Email string }{
	{"SUPER_ADMIN", "IT", "System Administrator", "admin@example.test"},
	{"SALES_STAFF", "SALES", "Sales Demo", "sales@example.test"},
	{"PURCHASING_STAFF", "PURCHASING", "Purchasing Demo", "purchasing@example.test"},
	{"WAREHOUSE_STAFF", "WAREHOUSE", "Warehouse Demo", "warehouse@example.test"},
	{"FINANCE_STAFF", "FINANCE", "Finance Demo", "finance@example.test"},
	{"MANAGER", "MANAGEMENT", "Manager Demo", "manager@example.test"},
}

func Seed(ctx context.Context, db *gorm.DB, password string) error {
	if len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("DEMO_PASSWORD must contain 12 to 72 bytes")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(73410204)").Error; err != nil {
			return err
		}
		if err := seedAccess(tx); err != nil {
			return err
		}
		for _, account := range DemoAccounts {
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO users (department_id,full_name,email,password_hash)
				SELECT id,?,?,? FROM departments WHERE code=? ON CONFLICT (email) DO NOTHING`, account.Name, account.Email, string(hash), account.Department).Error; err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO user_roles (user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.email=? AND r.code=? ON CONFLICT DO NOTHING", account.Email, account.Role).Error; err != nil {
				return err
			}
		}
		for _, warehouse := range []struct{ Code, Name string }{{"WH-JKT", "Jakarta Warehouse"}, {"WH-SBY", "Surabaya Warehouse"}} {
			if err := tx.Exec("INSERT INTO warehouses (code,name) VALUES (?,?) ON CONFLICT (code) DO NOTHING", warehouse.Code, warehouse.Name).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("INSERT INTO user_warehouses (user_id,warehouse_id) SELECT u.id,w.id FROM users u CROSS JOIN warehouses w WHERE u.email='warehouse@example.test' AND w.code='WH-JKT' ON CONFLICT DO NOTHING").Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO categories (code,name) VALUES ('CAT-GENERAL','General Products') ON CONFLICT (code) DO NOTHING").Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO products (sku,name,category_id,unit,selling_price,reorder_level) SELECT 'PRD-001','Mechanical Keyboard',id,'PCS',450000,10 FROM categories WHERE code='CAT-GENERAL' ON CONFLICT (sku) DO NOTHING").Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO products (sku,name,category_id,unit,selling_price,reorder_level) SELECT 'PRD-002','Wireless Mouse',id,'PCS',150000,10 FROM categories WHERE code='CAT-GENERAL' ON CONFLICT (sku) DO NOTHING").Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO suppliers (code,name,email,payment_terms_days) VALUES ('SUP-001','Components','supplier@example.test',30) ON CONFLICT (code) DO NOTHING").Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO customers (code,name,email,credit_limit) VALUES ('CUST-001','Retail','customer@example.test',1000000000) ON CONFLICT (code) DO NOTHING").Error; err != nil {
			return err
		}
		return seedInventory(tx)
	})
}

func seedInventory(tx *gorm.DB) error {
	var existing int64
	if err := tx.Table("goods_receipts").Where("gr_number='GR-DEMO-000001'").Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	ids := map[string]string{}
	queries := map[string]string{
		"supplier":   "SELECT id FROM suppliers WHERE code='SUP-001'",
		"warehouse":  "SELECT id FROM warehouses WHERE code='WH-JKT'",
		"product":    "SELECT id FROM products WHERE sku='PRD-001'",
		"purchasing": "SELECT id FROM users WHERE email='purchasing@example.test'",
		"manager":    "SELECT id FROM users WHERE email='manager@example.test'",
		"receiver":   "SELECT id FROM users WHERE email='warehouse@example.test'",
	}
	for key, query := range queries {
		var id string
		if err := tx.Raw(query).Scan(&id).Error; err != nil {
			return err
		}
		ids[key] = id
	}
	po, item, receipt, receivedItem := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if err := tx.Exec(`INSERT INTO purchase_orders (id,po_number,supplier_id,warehouse_id,status,order_date,subtotal,total_amount,created_by,submitted_at,approved_by,approved_at)
		VALUES (?,'PO-DEMO-000001',?,?,'CLOSED',CURRENT_DATE,30000000,30000000,?,CURRENT_TIMESTAMP,?,CURRENT_TIMESTAMP)`, po, ids["supplier"], ids["warehouse"], ids["purchasing"], ids["manager"]).Error; err != nil {
		return err
	}
	if err := tx.Exec("INSERT INTO purchase_order_items (id,purchase_order_id,product_id,quantity,unit_price,line_total) VALUES (?,?,?,100,300000,30000000)", item, po, ids["product"]).Error; err != nil {
		return err
	}
	if err := tx.Exec("INSERT INTO goods_receipts (id,gr_number,purchase_order_id,warehouse_id,status,received_at,received_by,notes) VALUES (?,'GR-DEMO-000001',?,?,'POSTED',CURRENT_TIMESTAMP,?,'Demo opening stock')", receipt, po, ids["warehouse"], ids["receiver"]).Error; err != nil {
		return err
	}
	if err := tx.Exec("INSERT INTO goods_receipt_items (id,goods_receipt_id,purchase_order_item_id,quantity_received) VALUES (?,?,?,100)", receivedItem, receipt, item).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO stock_movements (product_id,warehouse_id,movement_type,quantity_delta,reference_type,reference_id,reference_item_id,performed_by)
		VALUES (?,?,'GOODS_RECEIPT',100,'GOODS_RECEIPT',?,?,?)`, ids["product"], ids["warehouse"], receipt, receivedItem, ids["receiver"]).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO inventory_balances (product_id,warehouse_id,quantity_on_hand) VALUES (?,?,100)
		ON CONFLICT (product_id,warehouse_id) DO UPDATE SET quantity_on_hand=inventory_balances.quantity_on_hand+100,version=inventory_balances.version+1,updated_at=CURRENT_TIMESTAMP`, ids["product"], ids["warehouse"]).Error; err != nil {
		return err
	}
	return tx.Exec("INSERT INTO audit_logs (actor_id,action,entity_type,entity_id,new_values,request_id) VALUES (?,'seed.inventory','GOODS_RECEIPT',?,'{\"quantity\":\"100.000\"}'::JSONB,?)", ids["receiver"], receipt, uuid.NewString()).Error
}

func seedAccess(tx *gorm.DB) error {
	for _, department := range []struct{ Code, Name string }{{"IT", "IT"}, {"SALES", "Sales"}, {"PURCHASING", "Purchasing"}, {"WAREHOUSE", "Warehouse"}, {"FINANCE", "Finance"}, {"MANAGEMENT", "Management"}} {
		if err := tx.Exec("INSERT INTO departments (code,name) VALUES (?,?) ON CONFLICT (code) DO NOTHING", department.Code, department.Name).Error; err != nil {
			return err
		}
	}
	permissions := map[string]bool{}
	roles := []string{}
	for role, grants := range RolePermissions {
		roles = append(roles, role)
		for _, grant := range grants {
			permissions[grant] = true
		}
	}
	sort.Strings(roles)
	permissionCodes := []string{}
	for code := range permissions {
		permissionCodes = append(permissionCodes, code)
	}
	sort.Strings(permissionCodes)
	for _, code := range permissionCodes {
		parts := strings.Split(code, ".")
		if err := tx.Exec("INSERT INTO permissions (code,module,action) VALUES (?,?,?) ON CONFLICT (code) DO NOTHING", code, parts[0], strings.Join(parts[1:], ".")).Error; err != nil {
			return err
		}
	}
	for _, role := range roles {
		if err := tx.Exec("INSERT INTO roles (code,name) VALUES (?,?) ON CONFLICT (code) DO NOTHING", role, strings.ReplaceAll(role, "_", " ")).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code = ?)", role).Error; err != nil {
			return err
		}
		for _, code := range RolePermissions[role] {
			if err := tx.Exec("INSERT INTO role_permissions (role_id,permission_id) SELECT r.id,p.id FROM roles r CROSS JOIN permissions p WHERE r.code=? AND p.code=? ON CONFLICT DO NOTHING", role, code).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func BootstrapAdministrator(ctx context.Context, db *gorm.DB, email, password string) error {
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("ADMIN_EMAIL must be an email address and ADMIN_PASSWORD must contain 12 to 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(73410204)").Error; err != nil {
			return err
		}
		if err := seedAccess(tx); err != nil {
			return err
		}
		var count int64
		if err := tx.Raw("SELECT COUNT(*) FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE u.is_active AND r.code='SUPER_ADMIN'").Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("an active administrator already exists; manage users through the API")
		}
		id := uuid.NewString()
		if err := tx.Exec("INSERT INTO users (id,department_id,full_name,email,password_hash) SELECT ?,id,'System Administrator',?,? FROM departments WHERE code='IT'", id, strings.ToLower(email), string(hash)).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO user_roles (user_id,role_id) SELECT ?,id FROM roles WHERE code='SUPER_ADMIN'", id).Error; err != nil {
			return err
		}
		values, err := json.Marshal(map[string]string{"email": strings.ToLower(email)})
		if err != nil {
			return err
		}
		return tx.Exec("INSERT INTO audit_logs (actor_id,action,entity_type,entity_id,new_values,request_id) VALUES (?,'users.bootstrap','USER',?,?::JSONB,?)", id, id, string(values), uuid.NewString()).Error
	})
}
