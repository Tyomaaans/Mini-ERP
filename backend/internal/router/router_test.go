package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mini-erp/internal/auth-session/role"
	"mini-erp/internal/auth-session/session"
	"mini-erp/internal/auth-session/token"
	"mini-erp/internal/auth-session/user"
	"mini-erp/internal/config"
	"mini-erp/internal/finance/invoice"
	"mini-erp/internal/finance/payment"
	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/inventory/stock"
	"mini-erp/internal/master-data/category"
	"mini-erp/internal/master-data/customer"
	"mini-erp/internal/master-data/product"
	"mini-erp/internal/master-data/supplier"
	"mini-erp/internal/master-data/warehouse"
	"mini-erp/internal/middleware"
	goodsreceipt "mini-erp/internal/purchasing/goods-receipt"
	purchaseorder "mini-erp/internal/purchasing/purchase-order"
	"mini-erp/internal/reporting/report"
	"mini-erp/internal/router"
	salesorder "mini-erp/internal/sales/sales-order"
	"mini-erp/internal/sales/shipment"
	"mini-erp/pkg"
	"mini-erp/seeds"
)

type result struct {
	ContractError error
	Status        int
	Body          map[string]any
}

type fixture struct {
	RemoteAddr string
	Test       *testing.T
	Contract   *apiContract
	DB         *gorm.DB
	Handler    http.Handler
	Tokens     map[string]string
	Refresh    map[string]string
	IDs        map[string]string
}

func (f *fixture) request(method, path string, body any, account, key string) result {
	var buffer bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buffer).Encode(body)
	}
	req := httptest.NewRequest(method, "/api/v1"+path, &buffer)
	req.RemoteAddr = f.RemoteAddr
	req.Header.Set("Content-Type", "application/json")
	if account != "" {
		req.Header.Set("Authorization", "Bearer "+f.Tokens[account])
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	recorder := httptest.NewRecorder()
	f.Handler.ServeHTTP(recorder, req)
	var response map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	contractError := f.Contract.Validate(method, path, recorder.Code, response)
	if contractError != nil {
		f.Test.Error(contractError)
	}
	return result{
		Status:        recorder.Code,
		Body:          response,
		ContractError: contractError,
	}
}

func requireStatus(t *testing.T, response result, status int) map[string]any {
	t.Helper()
	if response.ContractError != nil {
		t.Fatal(response.ContractError)
	}
	if response.Status != status {
		t.Fatalf("status=%d, want %d; body=%v", response.Status, status, response.Body)
	}
	data, _ := response.Body["data"].(map[string]any)
	return data
}

func errorCode(response result) string {
	body, _ := response.Body["error"].(map[string]any)
	code, _ := body["code"].(string)
	return code
}

func (f *fixture) post(t *testing.T, path string, body any, account string) map[string]any {
	t.Helper()
	return requireStatus(t, f.request("POST", path, body, account, ""), 201)
}

func (f *fixture) action(t *testing.T, path, account string) map[string]any {
	t.Helper()
	return requireStatus(t, f.request("POST", path, nil, account, uuid.NewString()), 200)
}

func (f *fixture) product(t *testing.T) string {
	t.Helper()
	return f.post(t, "/products", map[string]any{"sku": "TEST-" + uuid.NewString(), "name": "Test Product", "category_id": f.IDs["category"], "unit": "PCS", "selling_price": "100.00", "reorder_level": "1.000"}, "admin")["id"].(string)
}

func (f *fixture) purchase(t *testing.T, productID, quantity, warehouseID string) map[string]any {
	t.Helper()
	order := f.post(t, "/purchase-orders", map[string]any{"supplier_id": f.IDs["supplier"], "warehouse_id": warehouseID, "order_date": "2026-10-01", "items": []any{map[string]any{"product_id": productID, "quantity": quantity, "unit_price": "100.00"}}}, "purchasing")
	id := order["id"].(string)
	f.action(t, "/purchase-orders/"+id+"/submit", "purchasing")
	f.action(t, "/purchase-orders/"+id+"/approve", "manager")
	return order
}

func (f *fixture) receipt(t *testing.T, order map[string]any, quantity string) map[string]any {
	t.Helper()
	item := order["items"].([]any)[0].(map[string]any)
	return f.post(t, "/goods-receipts", map[string]any{"purchase_order_id": order["id"], "items": []any{map[string]any{"purchase_order_item_id": item["id"], "quantity_received": quantity}}}, "warehouse")
}

func (f *fixture) stockedProduct(t *testing.T, quantity string) string {
	t.Helper()
	productID := f.product(t)
	order := f.purchase(t, productID, quantity, f.IDs["warehouse"])
	receipt := f.receipt(t, order, quantity)
	f.action(t, "/goods-receipts/"+receipt["id"].(string)+"/post", "warehouse")
	return productID
}

func (f *fixture) sale(t *testing.T, productID, quantity, price string) map[string]any {
	t.Helper()
	return f.post(t, "/sales-orders", map[string]any{"customer_id": f.IDs["customer"], "warehouse_id": f.IDs["warehouse"], "order_date": "2026-10-01", "items": []any{map[string]any{"product_id": productID, "quantity": quantity, "unit_price": price}}}, "sales")
}

func (f *fixture) shipment(t *testing.T, order map[string]any, quantity string) map[string]any {
	t.Helper()
	item := order["items"].([]any)[0].(map[string]any)
	return f.post(t, "/shipments", map[string]any{"sales_order_id": order["id"], "items": []any{map[string]any{"sales_order_item_id": item["id"], "quantity_shipped": quantity}}}, "warehouse")
}

func (f *fixture) invoice(t *testing.T, kind, orderID string) map[string]any {
	t.Helper()
	field := "sales_order_id"
	if kind == "AP" {
		field = "purchase_order_id"
	}
	inv := f.post(t, "/invoices", map[string]any{"invoice_type": kind, field: orderID, "issue_date": "2026-10-01", "due_date": "2026-10-31"}, "finance")
	f.action(t, "/invoices/"+inv["id"].(string)+"/issue", "finance")
	return inv
}

func (f *fixture) number(t *testing.T, query string, args ...any) string {
	t.Helper()
	var value string
	if err := f.DB.Raw(query, args...).Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	return value
}

func integrationFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	address := os.Getenv("TEST_REDIS_ADDR")
	if dsn == "" || address == "" {
		t.Skip("set TEST_DATABASE_URL and TEST_REDIS_ADDR to dedicated disposable services")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(strings.TrimPrefix(parsed.Path, "/"), "_test") {
		t.Fatal("TEST_DATABASE_URL must name a database ending in _test")
	}
	adminDB, err := postgres.NewPostgresDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminSQL, _ := adminDB.DB()
	t.Cleanup(func() {
		_ = adminSQL.Close()
	})
	databaseName := "erp_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := adminDB.Exec("CREATE DATABASE " + databaseName).Error; err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + databaseName
	db, err := postgres.NewPostgresDB(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = adminDB.Exec("DROP DATABASE " + databaseName + " WITH (FORCE)").Error
	})
	ctx := context.Background()
	for _, down := range []bool{false, false, true, true, false} {
		if err := postgres.Migrate(ctx, db, down); err != nil {
			t.Fatal(err)
		}
	}
	schema, err := os.ReadFile("../../migrations/migrations_test.sql")
	if err != nil {
		t.Fatal(err)
	}
	connection, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = connection.ExecContext(ctx, string(schema))
	_, _ = connection.ExecContext(ctx, "ROLLBACK")
	_ = connection.Close()
	if err != nil {
		t.Fatal(err)
	}
	rollbackFixture := errors.New("rollback bootstrap and grant verification")
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := seeds.BootstrapAdministrator(ctx, tx, "initial@example.test", "BootstrapPassword123!"); err != nil {
			return err
		}
		var users, products, audits int64
		if err := tx.Table("users").Count(&users).Error; err != nil {
			return err
		}
		if err := tx.Table("products").Count(&products).Error; err != nil {
			return err
		}
		if err := tx.Table("audit_logs").Where("action='users.bootstrap'").Count(&audits).Error; err != nil {
			return err
		}
		if users != 1 || products != 0 || audits != 1 {
			return fmt.Errorf("bootstrap created unexpected demo data or missed audit: users=%d products=%d audits=%d", users, products, audits)
		}
		if err := seeds.BootstrapAdministrator(ctx, tx, "another@example.test", "BootstrapPassword123!"); err == nil {
			return fmt.Errorf("bootstrap allowed a second initial administrator")
		}
		roleName := "runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		if err := tx.Exec("CREATE ROLE " + roleName + " NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE").Error; err != nil {
			return err
		}
		if err := postgres.GrantApplication(ctx, tx, roleName); err != nil {
			return err
		}
		var canInsert, canUpdate, canReadMigrations, canCreate bool
		if err := tx.Raw("SELECT has_table_privilege(?, 'audit_logs', 'INSERT')", roleName).Scan(&canInsert).Error; err != nil {
			return err
		}
		if err := tx.Raw("SELECT has_table_privilege(?, 'audit_logs', 'UPDATE')", roleName).Scan(&canUpdate).Error; err != nil {
			return err
		}
		if err := tx.Raw("SELECT has_table_privilege(?, 'schema_migrations', 'SELECT')", roleName).Scan(&canReadMigrations).Error; err != nil {
			return err
		}
		if err := tx.Raw("SELECT has_schema_privilege(?, 'public', 'CREATE')", roleName).Scan(&canCreate).Error; err != nil {
			return err
		}
		if !canInsert || canUpdate || canReadMigrations || canCreate {
			return fmt.Errorf("runtime database privileges are incorrect")
		}
		return rollbackFixture
	})
	if !errors.Is(err, rollbackFixture) {
		t.Fatal(err)
	}
	var checksum string
	if err := db.Raw("SELECT checksum FROM schema_migrations WHERE version=1").Scan(&checksum).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE schema_migrations SET checksum=REPEAT('0',64) WHERE version=1").Error; err != nil {
		t.Fatal(err)
	}
	if err := postgres.Migrate(ctx, db, false); err == nil || !strings.Contains(err.Error(), "modified") {
		t.Fatalf("modified migration was accepted: %v", err)
	}
	if err := db.Exec("UPDATE schema_migrations SET checksum=? WHERE version=1", checksum).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := seeds.Seed(ctx, db, "IntegrationPassword123!"); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Table("users").Count(&count).Error; err != nil || count != 6 {
		t.Fatalf("seed user count=%d err=%v", count, err)
	}
	if err := db.Table("stock_movements").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("seed movement count=%d err=%v", count, err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: os.Getenv("TEST_REDIS_PASSWORD"),
	})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})
	cfg := config.AppConfig{
		JWTSecretKey:  "integration-secret-key-at-least-32-characters",
		JWTExpiry:     15 * time.Minute,
		RefreshExpiry: time.Hour,
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	validate := pkg.NewValidator()
	tx := postgres.NewTransactionManager(db)
	audit := postgres.NewAuditLogger(db)
	sequence := postgres.NewDocumentSequence(db)
	idempotency := postgres.NewIdempotencyExecutor(db, tx)

	sessionSvc := session.NewSessionService(redisClient, cfg.RefreshExpiry)
	tokenSvc := token.NewJWTService(cfg.JWTSecretKey, cfg.JWTExpiry)
	userRepo := user.NewUserRepository(db)
	userSvc := user.NewUserService(userRepo, tokenSvc, sessionSvc, validate, tx, audit)
	userHandler := user.NewUserHandler(userSvc)
	roleRepo := role.NewRoleRepository(db)
	roleSvc := role.NewRoleService(roleRepo, validate, tx, audit)
	roleHandler := role.NewRoleHandler(roleSvc)
	stockRepo := stock.NewStockRepository(db)
	stockSvc := stock.NewStockService(stockRepo, tx)
	stockHandler := stock.NewStockHandler(stockSvc)
	productRepo := product.NewProductRepository(db)
	productSvc := product.NewProductService(productRepo, validate, tx, audit)
	productHandler := product.NewProductHandler(productSvc)
	categoryRepo := category.NewCategoryRepository(db)
	categorySvc := category.NewCategoryService(categoryRepo, validate, tx, audit)
	categoryHandler := category.NewCategoryHandler(categorySvc)
	supplierRepo := supplier.NewSupplierRepository(db)
	supplierSvc := supplier.NewSupplierService(supplierRepo, validate, tx, audit)
	supplierHandler := supplier.NewSupplierHandler(supplierSvc)
	customerRepo := customer.NewCustomerRepository(db)
	customerSvc := customer.NewCustomerService(customerRepo, validate, tx, audit)
	customerHandler := customer.NewCustomerHandler(customerSvc)
	warehouseRepo := warehouse.NewWarehouseRepository(db)
	warehouseSvc := warehouse.NewWarehouseService(warehouseRepo, validate, tx, audit)
	warehouseHandler := warehouse.NewWarehouseHandler(warehouseSvc)
	purchaseOrderRepo := purchaseorder.NewPurchaseOrderRepository(db)
	purchaseOrderSvc := purchaseorder.NewPurchaseOrderService(purchaseOrderRepo, validate, tx, audit, sequence, idempotency)
	purchaseOrderHandler := purchaseorder.NewPurchaseOrderHandler(purchaseOrderSvc)
	goodsReceiptRepo := goodsreceipt.NewGoodsReceiptRepository(db)
	goodsReceiptSvc := goodsreceipt.NewGoodsReceiptService(goodsReceiptRepo, purchaseOrderRepo, stockSvc, validate, tx, audit, sequence, idempotency)
	goodsReceiptHandler := goodsreceipt.NewGoodsReceiptHandler(goodsReceiptSvc)
	salesOrderRepo := salesorder.NewSalesOrderRepository(db)
	salesOrderSvc := salesorder.NewSalesOrderService(salesOrderRepo, validate, tx, audit, sequence, idempotency, stockSvc)
	salesOrderHandler := salesorder.NewSalesOrderHandler(salesOrderSvc)
	shipmentRepo := shipment.NewShipmentRepository(db)
	shipmentSvc := shipment.NewShipmentService(shipmentRepo, salesOrderRepo, stockSvc, validate, tx, audit, sequence, idempotency)
	shipmentHandler := shipment.NewShipmentHandler(shipmentSvc)
	invoiceRepo := invoice.NewInvoiceRepository(db)
	invoiceSvc := invoice.NewInvoiceService(invoiceRepo, validate, tx, audit, sequence, idempotency)
	invoiceHandler := invoice.NewInvoiceHandler(invoiceSvc)
	paymentRepo := payment.NewPaymentRepository(db)
	paymentSvc := payment.NewPaymentService(paymentRepo, invoiceRepo, validate, tx, audit, sequence, idempotency)
	paymentHandler := payment.NewPaymentHandler(paymentSvc)
	reportRepo := report.NewReportRepository(db)
	reportSvc := report.NewReportService(reportRepo)
	reportHandler := report.NewReportHandler(reportSvc)

	authMiddleware := middleware.NewAuthMiddleware(userSvc)
	rateLimiter := middleware.NewRateLimiter(redisClient)
	handlers := router.Handlers{
		User:          userHandler,
		Role:          roleHandler,
		Product:       productHandler,
		Category:      categoryHandler,
		Supplier:      supplierHandler,
		Customer:      customerHandler,
		Warehouse:     warehouseHandler,
		PurchaseOrder: purchaseOrderHandler,
		GoodsReceipt:  goodsReceiptHandler,
		Stock:         stockHandler,
		SalesOrder:    salesOrderHandler,
		Shipment:      shipmentHandler,
		Invoice:       invoiceHandler,
		Payment:       paymentHandler,
		Report:        reportHandler,
	}
	r := router.NewRouter(handlers, authMiddleware, rateLimiter, db, redisClient, cfg, logger)

	contract := newContract(t)
	contract.RequireRoutes(t, r)
	clientID := uuid.New()
	f := &fixture{
		RemoteAddr: net.JoinHostPort(net.IP(clientID[:]).String(), "12345"),
		Test:       t,
		Contract:   contract,
		DB:         db,
		Handler:    r,
		Tokens:     map[string]string{},
		Refresh:    map[string]string{},
		IDs:        map[string]string{},
	}
	for _, account := range []string{"admin", "sales", "purchasing", "warehouse", "finance", "manager"} {
		response := f.request("POST", "/auth/login", map[string]string{"email": account + "@example.test", "password": "IntegrationPassword123!"}, "", "")
		data := requireStatus(t, response, 200)
		f.Tokens[account] = data["access_token"].(string)
		f.Refresh[account] = data["refresh_token"].(string)
	}
	for key, query := range map[string]string{"category": "SELECT id FROM categories WHERE code='CAT-GENERAL'", "supplier": "SELECT id FROM suppliers WHERE code='SUP-001'", "customer": "SELECT id FROM customers WHERE code='CUST-001'", "warehouse": "SELECT id FROM warehouses WHERE code='WH-JKT'", "other_warehouse": "SELECT id FROM warehouses WHERE code='WH-SBY'"} {
		f.IDs[key] = f.number(t, query)
	}
	return f
}

func TestERPIntegration(t *testing.T) {
	f := integrationFixture(t)
	t.Run("purchase and customer workflows", func(t *testing.T) {
		productID := f.product(t)
		order := f.purchase(t, productID, "10.000", f.IDs["warehouse"])
		first := f.receipt(t, order, "4.000")
		key := uuid.NewString()
		path := "/goods-receipts/" + first["id"].(string) + "/post"
		posted := requireStatus(t, f.request("POST", path, nil, "warehouse", key), 200)
		replay := requireStatus(t, f.request("POST", path, nil, "warehouse", key), 200)
		a, _ := json.Marshal(posted)
		b, _ := json.Marshal(replay)
		if string(a) != string(b) {
			t.Fatal("idempotent replay changed the response")
		}
		second := f.receipt(t, order, "6.000")
		conflict := f.request("POST", "/goods-receipts/"+second["id"].(string)+"/post", nil, "warehouse", key)
		if conflict.Status != 409 || errorCode(conflict) != "IDEMPOTENCY_CONFLICT" {
			t.Fatalf("reused key: %+v", conflict)
		}
		f.action(t, "/goods-receipts/"+second["id"].(string)+"/post", "warehouse")
		po := requireStatus(t, f.request("GET", "/purchase-orders/"+order["id"].(string), nil, "manager", ""), 200)
		if po["status"] != "CLOSED" {
			t.Fatal("fully received PO did not close")
		}
		invAP := f.invoice(t, "AP", order["id"].(string))
		for _, amount := range []string{"400.00", "600.00"} {
			payment := f.post(t, "/payments", map[string]any{"invoice_id": invAP["id"], "amount": amount, "payment_method": "BANK_TRANSFER"}, "finance")
			f.action(t, "/payments/"+payment["id"].(string)+"/post", "finance")
		}
		paid := requireStatus(t, f.request("GET", "/invoices/"+invAP["id"].(string), nil, "finance", ""), 200)
		if paid["status"] != "PAID" || paid["paid_amount"] != "1000.00" {
			t.Fatal("AP partial payments do not reconcile")
		}
		sale := f.sale(t, productID, "10.000", "150.00")
		saleID := sale["id"].(string)
		f.action(t, "/sales-orders/"+saleID+"/confirm", "sales")
		f.action(t, "/sales-orders/"+saleID+"/reserve", "warehouse")
		for _, quantity := range []string{"3.000", "7.000"} {
			shipment := f.shipment(t, sale, quantity)
			f.action(t, "/shipments/"+shipment["id"].(string)+"/post", "warehouse")
		}
		shipped := requireStatus(t, f.request("GET", "/sales-orders/"+saleID, nil, "sales", ""), 200)
		if shipped["status"] != "SHIPPED" {
			t.Fatal("fully shipped order did not close")
		}
		invAR := f.invoice(t, "AR", saleID)
		for _, amount := range []string{"500.00", "1000.00"} {
			payment := f.post(t, "/payments", map[string]any{"invoice_id": invAR["id"], "amount": amount, "payment_method": "CASH"}, "finance")
			f.action(t, "/payments/"+payment["id"].(string)+"/post", "finance")
		}
		if balance := f.number(t, "SELECT quantity_on_hand::TEXT FROM inventory_balances WHERE product_id=? AND warehouse_id=?", productID, f.IDs["warehouse"]); balance != "0.000" {
			t.Fatalf("remaining balance=%s", balance)
		}
		if reserved := f.number(t, "SELECT quantity_reserved::TEXT FROM inventory_balances WHERE product_id=? AND warehouse_id=?", productID, f.IDs["warehouse"]); reserved != "0.000" {
			t.Fatalf("remaining reservation=%s", reserved)
		}
	})
	t.Run("authorization scope ownership and state machines", func(t *testing.T) {
		requireStatus(t, f.request("GET", "/products", nil, "", ""), 401)
		requireStatus(t, f.request("GET", "/users", nil, "sales", ""), 403)
		requireStatus(t, f.request("POST", "/purchase-orders/"+uuid.NewString()+"/approve", nil, "sales", uuid.NewString()), 403)
		requireStatus(t, f.request("GET", "/invoices", nil, "warehouse", ""), 403)
		requireStatus(t, f.request("GET", "/reports/receivables", nil, "sales", ""), 403)
		requireStatus(t, f.request("GET", "/products?per_page=101", nil, "sales", ""), 422)
		requireStatus(t, f.request("GET", "/products?sort=password_hash", nil, "sales", ""), 422)
		requireStatus(t, f.request("POST", "/products", map[string]any{"sku": "DENIED"}, "sales", ""), 403)
		productID := f.product(t)
		order := f.purchase(t, productID, "1.000", f.IDs["other_warehouse"])
		item := order["items"].([]any)[0].(map[string]any)
		requireStatus(t, f.request("POST", "/goods-receipts", map[string]any{"purchase_order_id": order["id"], "items": []any{map[string]any{"purchase_order_item_id": item["id"], "quantity_received": "1.000"}}}, "warehouse", ""), 403)
		otherReceipt := uuid.NewString()
		if err := f.DB.Exec("INSERT INTO goods_receipts (id,gr_number,purchase_order_id,warehouse_id,received_by) VALUES (?,?,?,?,(SELECT id FROM users WHERE email='warehouse@example.test'))", otherReceipt, "GR-SCOPE-"+uuid.NewString()[:8], order["id"], f.IDs["other_warehouse"]).Error; err != nil {
			t.Fatal(err)
		}
		requireStatus(t, f.request("POST", "/goods-receipts/"+otherReceipt+"/post", nil, "warehouse", uuid.NewString()), 403)
		requireStatus(t, f.request("GET", "/goods-receipts/"+otherReceipt, nil, "warehouse", ""), 403)
		departmentID := f.number(t, "SELECT id FROM departments WHERE code='IT'")
		roleID := f.number(t, "SELECT id FROM roles WHERE code='SALES_STAFF'")
		email := "sales2@example.test"
		created := f.post(t, "/users", map[string]any{"department_id": departmentID, "full_name": "Second Sales", "email": email, "password": "IntegrationPassword123!", "role_ids": []string{roleID}, "warehouse_ids": []string{}}, "admin")
		login := requireStatus(t, f.request("POST", "/auth/login", map[string]string{"email": email, "password": "IntegrationPassword123!"}, "", ""), 200)
		f.Tokens["other_sales"] = login["access_token"].(string)
		requireStatus(t, f.request("POST", "/products", map[string]any{"sku": "NO-IT-BYPASS"}, "other_sales", ""), 403)
		sale := f.sale(t, productID, "1.000", "100.00")
		id := sale["id"].(string)
		requireStatus(t, f.request("PATCH", "/sales-orders/"+id, map[string]any{"notes": "Unauthorized edit"}, "other_sales", ""), 403)
		updated := requireStatus(t, f.request("PATCH", "/sales-orders/"+id, map[string]any{"notes": "Owner edit"}, "sales", ""), 200)
		if updated["notes"] != "Owner edit" || updated["items"].([]any)[0].(map[string]any)["id"] != sale["items"].([]any)[0].(map[string]any)["id"] {
			t.Fatal("partial update replaced unrelated data")
		}
		requireStatus(t, f.request("PATCH", "/sales-orders/"+id, map[string]any{"status": "SHIPPED"}, "sales", ""), 400)
		f.action(t, "/sales-orders/"+id+"/confirm", "sales")
		requireStatus(t, f.request("PATCH", "/sales-orders/"+id, map[string]any{"notes": "Cannot edit confirmed"}, "sales", ""), 409)
		requireStatus(t, f.request("PATCH", "/users/"+created["id"].(string), map[string]any{"is_active": false}, "admin", ""), 200)
		requireStatus(t, f.request("GET", "/products", nil, "other_sales", ""), 401)
		requireStatus(t, f.request("PATCH", "/users/"+created["id"].(string), map[string]any{"is_active": true}, "admin", ""), 200)
		requireStatus(t, f.request("GET", "/products", nil, "other_sales", ""), 401)

		users := f.request("GET", "/users", nil, "manager", "")
		requireStatus(t, users, 200)
		encoded, _ := json.Marshal(users.Body)
		if strings.Contains(string(encoded), "password") {
			t.Fatal("user list leaked password material")
		}
	})

	t.Run("dual roles cannot approve their own purchase order", func(t *testing.T) {
		productID := f.product(t)
		order := f.post(t, "/purchase-orders", map[string]any{"supplier_id": f.IDs["supplier"], "warehouse_id": f.IDs["warehouse"], "order_date": "2026-10-01", "items": []any{map[string]any{"product_id": productID, "quantity": "1", "unit_price": "100"}}}, "purchasing")
		id := order["id"].(string)
		f.action(t, "/purchase-orders/"+id+"/submit", "purchasing")
		userID := f.number(t, "SELECT id FROM users WHERE email='purchasing@example.test'")
		purchasingRole := f.number(t, "SELECT id FROM roles WHERE code='PURCHASING_STAFF'")
		managerRole := f.number(t, "SELECT id FROM roles WHERE code='MANAGER'")
		requireStatus(t, f.request("PUT", "/users/"+userID+"/roles", map[string]any{"role_ids": []string{purchasingRole, managerRole}}, "admin", ""), 200)
		denied := f.request("POST", "/purchase-orders/"+id+"/approve", nil, "purchasing", uuid.NewString())
		if denied.Status != 403 || errorCode(denied) != "SEPARATION_OF_DUTIES" {
			t.Fatalf("self approval=%+v", denied)
		}
		requireStatus(t, f.request("PUT", "/users/"+userID+"/roles", map[string]any{"role_ids": []string{purchasingRole}}, "admin", ""), 200)
		requireStatus(t, f.request("POST", "/purchase-orders/"+id+"/reject", map[string]any{"note": "Please revise delivery details"}, "manager", uuid.NewString()), 200)
		requireStatus(t, f.request("POST", "/purchase-orders/"+id+"/submit", nil, "purchasing", uuid.NewString()), 409)
		requireStatus(t, f.request("POST", "/purchase-orders/"+id+"/revise", map[string]any{"note": "Delivery details corrected"}, "purchasing", uuid.NewString()), 200)
		requireStatus(t, f.request("PATCH", "/purchase-orders/"+id, map[string]any{"notes": "Revised"}, "purchasing", ""), 200)
		f.action(t, "/purchase-orders/"+id+"/submit", "purchasing")
		requireStatus(t, f.request("POST", "/purchase-orders/"+id+"/cancel", map[string]any{"note": "Supplier unavailable"}, "purchasing", uuid.NewString()), 200)
		if pending := f.number(t, "SELECT COUNT(*)::TEXT FROM approval_requests WHERE document_id=? AND status='PENDING'", id); pending != "0" {
			t.Fatal("cancelled order retained a pending approval")
		}
	})

	t.Run("posting rolls back ledger balances status audit and idempotency", func(t *testing.T) {
		productID := f.product(t)
		order := f.purchase(t, productID, "3.000", f.IDs["warehouse"])
		receipt := f.receipt(t, order, "3.000")
		id := receipt["id"].(string)
		if err := f.DB.Exec(`CREATE FUNCTION test_reject_receipt_audit() RETURNS TRIGGER LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='goods_receipts.post' THEN RAISE EXCEPTION 'forced test audit failure'; END IF; RETURN NEW; END; $$;
			CREATE TRIGGER test_reject_receipt_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION test_reject_receipt_audit();`).Error; err != nil {
			t.Fatal(err)
		}
		key := uuid.NewString()
		failed := f.request("POST", "/goods-receipts/"+id+"/post", nil, "warehouse", key)
		requireStatus(t, failed, 500)
		if f.number(t, "SELECT COUNT(*)::TEXT FROM stock_movements WHERE reference_id=?", id) != "0" || f.number(t, "SELECT status FROM goods_receipts WHERE id=?", id) != "DRAFT" || f.number(t, "SELECT status FROM purchase_orders WHERE id=?", order["id"]) != "APPROVED" || f.number(t, "SELECT COUNT(*)::TEXT FROM idempotency_records WHERE key=?", key) != "0" {
			t.Fatal("posting failure leaked partial writes")
		}
		if amount := f.number(t, "SELECT COALESCE(SUM(quantity_on_hand),0)::TEXT FROM inventory_balances WHERE product_id=?", productID); amount != "0" {
			t.Fatalf("balance survived rollback: %s", amount)
		}
		if err := f.DB.Exec("DROP TRIGGER test_reject_receipt_audit ON audit_logs; DROP FUNCTION test_reject_receipt_audit();").Error; err != nil {
			t.Fatal(err)
		}
		requireStatus(t, f.request("POST", "/goods-receipts/"+id+"/post", nil, "warehouse", key), 200)
		if f.number(t, "SELECT COUNT(*)::TEXT FROM stock_movements WHERE reference_id=?", id) != "1" {
			t.Fatal("retry did not post exactly once")
		}
	})

	t.Run("concurrent reservations cannot oversell", func(t *testing.T) {
		productID := f.stockedProduct(t, "10.000")
		first := f.sale(t, productID, "7.000", "100.00")
		second := f.sale(t, productID, "6.000", "100.00")
		for _, sale := range []map[string]any{first, second} {
			f.action(t, "/sales-orders/"+sale["id"].(string)+"/confirm", "sales")
		}
		results := raceRequests(func() result {
			return f.request("POST", "/sales-orders/"+first["id"].(string)+"/reserve", nil, "warehouse", uuid.NewString())
		}, func() result {
			return f.request("POST", "/sales-orders/"+second["id"].(string)+"/reserve", nil, "warehouse", uuid.NewString())
		})
		requireOneSuccess(t, results)
		reserved := f.number(t, "SELECT quantity_reserved::TEXT FROM inventory_balances WHERE product_id=?", productID)
		if reserved != "7.000" && reserved != "6.000" {
			t.Fatalf("reserved=%s", reserved)
		}
	})

	t.Run("concurrent receipts cannot exceed ordered quantities", func(t *testing.T) {
		productID := f.product(t)
		order := f.purchase(t, productID, "10.000", f.IDs["warehouse"])
		first := f.receipt(t, order, "7.000")
		second := f.receipt(t, order, "6.000")
		results := raceRequests(func() result {
			return f.request("POST", "/goods-receipts/"+first["id"].(string)+"/post", nil, "warehouse", uuid.NewString())
		}, func() result {
			return f.request("POST", "/goods-receipts/"+second["id"].(string)+"/post", nil, "warehouse", uuid.NewString())
		})
		requireOneSuccess(t, results)
		quantity := f.number(t, "SELECT quantity_on_hand::TEXT FROM inventory_balances WHERE product_id=?", productID)
		if quantity != "7.000" && quantity != "6.000" {
			t.Fatalf("on_hand=%s", quantity)
		}
	})

	t.Run("concurrent shipments cannot exceed order or reservation", func(t *testing.T) {
		productID := f.stockedProduct(t, "10.000")
		order := f.sale(t, productID, "10.000", "100.00")
		f.action(t, "/sales-orders/"+order["id"].(string)+"/confirm", "sales")
		f.action(t, "/sales-orders/"+order["id"].(string)+"/reserve", "warehouse")
		first := f.shipment(t, order, "7.000")
		second := f.shipment(t, order, "6.000")
		results := raceRequests(func() result {
			return f.request("POST", "/shipments/"+first["id"].(string)+"/post", nil, "warehouse", uuid.NewString())
		}, func() result {
			return f.request("POST", "/shipments/"+second["id"].(string)+"/post", nil, "warehouse", uuid.NewString())
		})
		requireOneSuccess(t, results)
		quantity := f.number(t, "SELECT quantity_on_hand::TEXT FROM inventory_balances WHERE product_id=?", productID)
		if quantity != "3.000" && quantity != "4.000" {
			t.Fatalf("remaining stock=%s", quantity)
		}
	})

	t.Run("concurrent payments cannot overpay", func(t *testing.T) {
		productID := f.stockedProduct(t, "1.000")
		order := f.sale(t, productID, "1.000", "100.00")
		f.action(t, "/sales-orders/"+order["id"].(string)+"/confirm", "sales")
		f.action(t, "/sales-orders/"+order["id"].(string)+"/reserve", "warehouse")
		shipment := f.shipment(t, order, "1.000")
		f.action(t, "/shipments/"+shipment["id"].(string)+"/post", "warehouse")
		inv := f.invoice(t, "AR", order["id"].(string))
		first := f.post(t, "/payments", map[string]any{"invoice_id": inv["id"], "amount": "70.00", "payment_method": "CASH"}, "finance")
		second := f.post(t, "/payments", map[string]any{"invoice_id": inv["id"], "amount": "60.00", "payment_method": "CASH"}, "finance")
		results := raceRequests(func() result {
			return f.request("POST", "/payments/"+first["id"].(string)+"/post", nil, "finance", uuid.NewString())
		}, func() result {
			return f.request("POST", "/payments/"+second["id"].(string)+"/post", nil, "finance", uuid.NewString())
		})
		requireOneSuccess(t, results)
		paid := f.number(t, "SELECT paid_amount::TEXT FROM invoices WHERE id=?", inv["id"])
		if paid != "70.00" && paid != "60.00" {
			t.Fatalf("paid=%s", paid)
		}
		if posted := f.number(t, "SELECT SUM(amount)::TEXT FROM payments WHERE invoice_id=? AND status='POSTED'", inv["id"]); posted != paid {
			t.Fatal("invoice paid amount does not match posted payments")
		}
	})

	t.Run("cancel releases reservations and reports honor division permissions", func(t *testing.T) {
		productID := f.stockedProduct(t, "2.000")
		order := f.sale(t, productID, "2.000", "100.00")
		id := order["id"].(string)
		f.action(t, "/sales-orders/"+id+"/confirm", "sales")
		f.action(t, "/sales-orders/"+id+"/reserve", "warehouse")
		shipment := f.shipment(t, order, "1.000")
		requireStatus(t, f.request("POST", "/sales-orders/"+id+"/cancel", map[string]any{"note": "Customer cancelled"}, "sales", uuid.NewString()), 200)
		if f.number(t, "SELECT quantity_reserved::TEXT FROM inventory_balances WHERE product_id=?", productID) != "0.000" || f.number(t, "SELECT status FROM shipments WHERE id=?", shipment["id"]) != "CANCELLED" {
			t.Fatal("order cancellation did not release pending work")
		}
		for _, path := range []string{"/reports/sales-summary", "/reports/purchasing-summary", "/reports/low-stock", "/reports/receivables", "/reports/payables", "/audit-logs", "/health", "/health/live", "/inventory/balances", "/inventory/movements"} {
			requireStatus(t, f.request("GET", path, nil, "manager", ""), 200)
		}
		requireStatus(t, f.request("GET", "/reports/sales-summary", nil, "warehouse", ""), 403)
		requireStatus(t, f.request("GET", "/reports/low-stock?warehouse_id="+f.IDs["other_warehouse"], nil, "warehouse", ""), 403)
	})

	t.Run("concurrent retries post a receipt once", func(t *testing.T) {
		productID := f.product(t)
		order := f.purchase(t, productID, "2.000", f.IDs["warehouse"])
		receipt := f.receipt(t, order, "2.000")
		id := receipt["id"].(string)
		key := uuid.NewString()
		call := func() result {
			return f.request("POST", "/goods-receipts/"+id+"/post", nil, "warehouse", key)
		}
		results := raceRequests(call, call)
		for _, response := range results {
			requireStatus(t, response, 200)
		}
		first, _ := json.Marshal(results[0].Body)
		second, _ := json.Marshal(results[1].Body)
		if string(first) != string(second) || f.number(t, "SELECT COUNT(*)::TEXT FROM stock_movements WHERE reference_id=?", id) != "1" {
			t.Fatal("concurrent retries did not replay one posting")
		}
	})

	t.Run("seeded read permissions match each division", func(t *testing.T) {
		checks := []struct {
			path    string
			allowed string
		}{
			{"/users", "admin manager"}, {"/roles", "admin manager"}, {"/departments", "admin manager"}, {"/permissions", "admin manager"},
			{"/products", "admin sales purchasing warehouse finance manager"}, {"/categories", "admin sales purchasing warehouse finance manager"},
			{"/customers", "admin sales warehouse finance manager"}, {"/suppliers", "admin purchasing warehouse finance manager"},
			{"/purchase-orders", "admin purchasing warehouse finance manager"}, {"/goods-receipts", "admin purchasing warehouse finance manager"},
			{"/sales-orders", "admin sales warehouse finance manager"}, {"/shipments", "admin sales warehouse finance manager"},
			{"/invoices", "admin sales purchasing finance manager"}, {"/payments", "admin finance manager"},
			{"/reports/sales-summary", "admin sales manager"}, {"/reports/purchasing-summary", "admin purchasing manager"},
			{"/reports/low-stock", "admin warehouse manager"}, {"/reports/receivables", "admin finance manager"}, {"/audit-logs", "admin manager"},
		}
		for _, check := range checks {
			for _, account := range []string{"admin", "sales", "purchasing", "warehouse", "finance", "manager"} {
				status := 403
				if strings.Contains(" "+check.allowed+" ", " "+account+" ") {
					status = 200
				}
				requireStatus(t, f.request("GET", check.path, nil, account, ""), status)
			}
		}
		requireStatus(t, f.request("GET", "/products?direction=desc;invalid=true", nil, "sales", ""), 400)
	})

	t.Run("master data trees credit limits and custom roles", func(t *testing.T) {
		root := f.post(t, "/categories", map[string]any{"code": "ROOT-" + uuid.NewString()[:8], "name": "Root category"}, "admin")
		child := f.post(t, "/categories", map[string]any{"code": "CHILD-" + uuid.NewString()[:8], "name": "Child category", "parent_id": root["id"]}, "admin")
		requireStatus(t, f.request("PATCH", "/categories/"+root["id"].(string), map[string]any{"parent_id": child["id"]}, "admin", ""), 409)
		requireStatus(t, f.request("DELETE", "/categories/"+root["id"].(string), nil, "admin", ""), 409)
		reset := requireStatus(t, f.request("PATCH", "/categories/"+child["id"].(string), map[string]any{"parent_id": nil}, "admin", ""), 200)
		if reset["parent_id"] != nil {
			t.Fatal("category parent could not be cleared")
		}
		requireStatus(t, f.request("DELETE", "/categories/"+root["id"].(string), nil, "admin", ""), 200)
		requireStatus(t, f.request("DELETE", "/categories/"+child["id"].(string), nil, "admin", ""), 200)
		customer := f.post(t, "/customers", map[string]any{"code": "CUST-" + uuid.NewString()[:8], "name": "Limited customer", "credit_limit": "50.00"}, "sales")
		supplier := f.post(t, "/suppliers", map[string]any{"code": "SUP-" + uuid.NewString()[:8], "name": "New supplier", "payment_terms_days": 30}, "purchasing")
		warehouse := f.post(t, "/warehouses", map[string]any{"code": "WH-" + uuid.NewString()[:8], "name": "New warehouse"}, "admin")
		requireStatus(t, f.request("GET", "/warehouses/"+warehouse["id"].(string), nil, "warehouse", ""), 403)
		requireStatus(t, f.request("PATCH", "/suppliers/"+supplier["id"].(string), map[string]any{"payment_terms_days": 45}, "purchasing", ""), 200)
		product := f.product(t)
		order := f.post(t, "/sales-orders", map[string]any{"customer_id": customer["id"], "warehouse_id": f.IDs["warehouse"], "order_date": "2026-10-01", "items": []any{map[string]any{"product_id": product, "quantity": "1", "unit_price": "100"}}}, "sales")
		limited := f.request("POST", "/sales-orders/"+order["id"].(string)+"/confirm", nil, "sales", uuid.NewString())
		requireStatus(t, limited, 409)
		if errorCode(limited) != "CREDIT_LIMIT_EXCEEDED" {
			t.Fatalf("credit response=%+v", limited)
		}
		requireStatus(t, f.request("PATCH", "/customers/"+customer["id"].(string), map[string]any{"credit_limit": "500.00"}, "sales", ""), 200)
		f.action(t, "/sales-orders/"+order["id"].(string)+"/confirm", "sales")
		permission := f.number(t, "SELECT id FROM permissions WHERE code='reports.sales.read'")
		roleCode := "REPORT_" + strings.ToUpper(uuid.NewString()[:8])
		role := f.post(t, "/roles", map[string]any{"code": roleCode, "name": "Report viewer", "permission_ids": []string{permission}}, "admin")
		userID := f.number(t, "SELECT id FROM users WHERE email='sales2@example.test'")
		salesRole := f.number(t, "SELECT id FROM roles WHERE code='SALES_STAFF'")
		requireStatus(t, f.request("PUT", "/users/"+userID+"/roles", map[string]any{"role_ids": []string{salesRole, role["id"].(string)}}, "admin", ""), 200)
		requireStatus(t, f.request("DELETE", "/roles/"+role["id"].(string), nil, "admin", ""), 409)
		requireStatus(t, f.request("PUT", "/users/"+userID+"/roles", map[string]any{"role_ids": []string{salesRole}}, "admin", ""), 200)
		requireStatus(t, f.request("DELETE", "/roles/"+role["id"].(string), nil, "admin", ""), 200)
	})

	t.Run("audit references match their concrete entity types", func(t *testing.T) {
		tables := map[string]string{"PRODUCT": "products", "CATEGORY": "categories", "SUPPLIER": "suppliers", "CUSTOMER": "customers", "WAREHOUSE": "warehouses", "USER": "users", "ROLE": "roles", "PURCHASE_ORDER": "purchase_orders", "GOODS_RECEIPT": "goods_receipts", "SALES_ORDER": "sales_orders", "SHIPMENT": "shipments", "INVOICE": "invoices", "PAYMENT": "payments"}
		for entityType, table := range tables {
			query := "SELECT COUNT(*)::TEXT FROM audit_logs a WHERE a.entity_type=? AND a.action NOT LIKE '%.delete' AND NOT EXISTS (SELECT 1 FROM audit_logs deleted WHERE deleted.entity_type=a.entity_type AND deleted.entity_id=a.entity_id AND deleted.action LIKE '%.delete') AND NOT EXISTS (SELECT 1 FROM " + table + " e WHERE e.id=a.entity_id)"
			if orphaned := f.number(t, query, entityType); orphaned != "0" {
				t.Fatalf("audit entity type %s contains %s unresolved references", entityType, orphaned)
			}
		}
	})

	t.Run("refresh tokens rotate once and logout revokes access", func(t *testing.T) {
		old := f.Refresh["finance"]
		response := requireStatus(t, f.request("POST", "/auth/refresh", map[string]string{"refresh_token": old}, "", ""), 200)
		f.Tokens["finance"] = response["access_token"].(string)
		requireStatus(t, f.request("POST", "/auth/refresh", map[string]string{"refresh_token": old}, "", ""), 401)
		requireStatus(t, f.request("POST", "/auth/logout", nil, "finance", ""), 200)
		requireStatus(t, f.request("GET", "/auth/me", nil, "finance", ""), 401)
	})

}

func raceRequests(operations ...func() result) []result {
	start := make(chan struct{})
	results := make([]result, len(operations))
	var group sync.WaitGroup
	for i, operation := range operations {
		group.Add(1)
		go func(i int, operation func() result) {
			defer group.Done()
			<-start
			results[i] = operation()
		}(i, operation)
	}
	close(start)
	group.Wait()
	return results
}

func requireOneSuccess(t *testing.T, results []result) {
	t.Helper()
	success, conflict := 0, 0
	for _, response := range results {
		if response.ContractError != nil {
			t.Fatal(response.ContractError)
		}
		if response.Status == 200 {
			success++
		} else if response.Status == 409 {
			conflict++
		} else {
			t.Fatalf("unexpected concurrent response: %+v", response)
		}
	}
	if success != 1 || conflict != len(results)-1 {
		t.Fatalf("concurrent results=%+v", results)
	}
}

type queryCounter struct {
	logger.Interface
	Count atomic.Int64
}

func (c *queryCounter) Trace(ctx context.Context, begin time.Time, query func() (string, int64), err error) {
	c.Count.Add(1)
}

func TestIdentityListsBatchAccessMetadata(t *testing.T) {
	f := integrationFixture(t)
	counter := &queryCounter{Interface: logger.Default.LogMode(logger.Silent)}
	db := f.DB.Session(&gorm.Session{Logger: counter})
	userRepo := user.NewUserRepository(db)
	roleRepo := role.NewRoleRepository(db)
	ctx := context.Background()
	for _, size := range []int{1, 100} {
		query := pkg.ListQuery{
			Page:      1,
			PerPage:   size,
			Sort:      "created_at",
			Direction: "asc",
		}
		counter.Count.Store(0)
		users, _, err := userRepo.GetUsers(ctx, query)
		if err != nil || len(users) == 0 {
			t.Fatalf("user page %d: count=%d error=%v", size, len(users), err)
		}
		if count := counter.Count.Load(); count != 4 {
			t.Fatalf("user page %d: queries=%d, want 4", size, count)
		}
		for _, entity := range users {
			actor, err := userRepo.GetActor(ctx, entity.ID)
			if err != nil || strings.Join(entity.Roles, ",") != strings.Join(actor.Roles, ",") || strings.Join(entity.WarehouseIDs, ",") != strings.Join(actor.WarehouseIDs, ",") {
				t.Fatalf("user access metadata differs for %s: %v", entity.ID, err)
			}
		}
		counter.Count.Store(0)
		roles, _, err := roleRepo.GetRoles(ctx, query)
		if err != nil || len(roles) == 0 {
			t.Fatalf("role page %d: count=%d error=%v", size, len(roles), err)
		}
		if count := counter.Count.Load(); count != 3 {
			t.Fatalf("role page %d: queries=%d, want 3", size, count)
		}
		for _, entity := range roles {
			detail, err := roleRepo.GetRole(ctx, entity.ID)
			if err != nil || strings.Join(entity.PermissionIDs, ",") != strings.Join(detail.PermissionIDs, ",") {
				t.Fatalf("role access metadata differs for %s: %v", entity.ID, err)
			}
		}
	}
}
