package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini-erp/internal/auth-session/role"
	"mini-erp/internal/auth-session/session"
	"mini-erp/internal/auth-session/token"
	"mini-erp/internal/auth-session/user"
	"mini-erp/internal/config"
	"mini-erp/internal/finance/invoice"
	"mini-erp/internal/finance/payment"
	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/infrastructure/redis"
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

func main() {
	if err := run(); err != nil {
		log.Printf("application stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}

	db, err := postgres.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	defer sqlDB.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			if len(os.Args) != 3 || (os.Args[2] != "up" && os.Args[2] != "down") {
				return fmt.Errorf("usage: migrate up|down")
			}

			if err := postgres.Migrate(ctx, db, os.Args[2] == "down"); err != nil {
				return err
			}

			if os.Args[2] == "up" {
				return postgres.GrantApplication(ctx, db, cfg.DatabaseAppUser)
			}

			return nil
		case "bootstrap-admin":
			return seeds.BootstrapAdministrator(ctx, db, cfg.AdminEmail, cfg.AdminPassword)
		case "seed":
			if os.Getenv("APP_ENV") == "production" {
				return fmt.Errorf("demo seeding is disabled in production")
			}

			return seeds.Seed(ctx, db, cfg.DemoPassword)
		default:
			return fmt.Errorf("unknown command; use migrate up|down, seed, bootstrap-admin, or no arguments to serve")
		}
	}

	if err := cfg.ValidateServer(); err != nil {
		return err
	}

	redisClient, err := redis.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		return err
	}

	defer redisClient.Close()
	validate            := pkg.NewValidator()
	txManager           := postgres.NewTransactionManager(db)
	auditLogger         := postgres.NewAuditLogger(db)
	documentSequence    := postgres.NewDocumentSequence(db)
	idempotencyExecutor := postgres.NewIdempotencyExecutor(db, txManager)

	sessionSvc  := session.NewSessionService(redisClient, cfg.RefreshExpiry)
	tokenSvc    := token.NewJWTService(cfg.JWTSecretKey, cfg.JWTExpiry)
	userRepo    := user.NewUserRepository(db)
	userSvc     := user.NewUserService(userRepo, tokenSvc, sessionSvc, validate, txManager, auditLogger)
	userHandler := user.NewUserHandler(userSvc)

	roleRepo    := role.NewRoleRepository(db)
	roleSvc     := role.NewRoleService(roleRepo, validate, txManager, auditLogger)
	roleHandler := role.NewRoleHandler(roleSvc)

	stockRepo    := stock.NewStockRepository(db)
	stockSvc     := stock.NewStockService(stockRepo, txManager)
	stockHandler := stock.NewStockHandler(stockSvc)

	productRepo    := product.NewProductRepository(db)
	productSvc     := product.NewProductService(productRepo, validate, txManager, auditLogger)
	productHandler := product.NewProductHandler(productSvc)

	categoryRepo    := category.NewCategoryRepository(db)
	categorySvc     := category.NewCategoryService(categoryRepo, validate, txManager, auditLogger)
	categoryHandler := category.NewCategoryHandler(categorySvc)

	supplierRepo    := supplier.NewSupplierRepository(db)
	supplierSvc     := supplier.NewSupplierService(supplierRepo, validate, txManager, auditLogger)
	supplierHandler := supplier.NewSupplierHandler(supplierSvc)

	customerRepo    := customer.NewCustomerRepository(db)
	customerSvc     := customer.NewCustomerService(customerRepo, validate, txManager, auditLogger)
	customerHandler := customer.NewCustomerHandler(customerSvc)

	warehouseRepo    := warehouse.NewWarehouseRepository(db)
	warehouseSvc     := warehouse.NewWarehouseService(warehouseRepo, validate, txManager, auditLogger)
	warehouseHandler := warehouse.NewWarehouseHandler(warehouseSvc)

	purchaseOrderRepo    := purchaseorder.NewPurchaseOrderRepository(db)
	purchaseOrderSvc     := purchaseorder.NewPurchaseOrderService(purchaseOrderRepo, validate, txManager, auditLogger, documentSequence, idempotencyExecutor)
	purchaseOrderHandler := purchaseorder.NewPurchaseOrderHandler(purchaseOrderSvc)

	goodsReceiptRepo    := goodsreceipt.NewGoodsReceiptRepository(db)
	goodsReceiptSvc     := goodsreceipt.NewGoodsReceiptService(goodsReceiptRepo, purchaseOrderRepo, stockSvc, validate, txManager, auditLogger, documentSequence, idempotencyExecutor)
	goodsReceiptHandler := goodsreceipt.NewGoodsReceiptHandler(goodsReceiptSvc)

	salesOrderRepo    := salesorder.NewSalesOrderRepository(db)
	salesOrderSvc     := salesorder.NewSalesOrderService(salesOrderRepo, validate, txManager, auditLogger, documentSequence, idempotencyExecutor, stockSvc)
	salesOrderHandler := salesorder.NewSalesOrderHandler(salesOrderSvc)

	shipmentRepo    := shipment.NewShipmentRepository(db)
	shipmentSvc     := shipment.NewShipmentService(shipmentRepo, salesOrderRepo, stockSvc, validate, txManager, auditLogger, documentSequence, idempotencyExecutor)
	shipmentHandler := shipment.NewShipmentHandler(shipmentSvc)

	invoiceRepo    := invoice.NewInvoiceRepository(db)
	invoiceSvc     := invoice.NewInvoiceService(invoiceRepo, validate, txManager, auditLogger, documentSequence, idempotencyExecutor)
	invoiceHandler := invoice.NewInvoiceHandler(invoiceSvc)

	paymentRepo    := payment.NewPaymentRepository(db)
	paymentSvc     := payment.NewPaymentService(paymentRepo, invoiceRepo, validate, txManager, auditLogger, documentSequence, idempotencyExecutor)
	paymentHandler := payment.NewPaymentHandler(paymentSvc)

	reportRepo    := report.NewReportRepository(db)
	reportSvc     := report.NewReportService(reportRepo)
	reportHandler := report.NewReportHandler(reportSvc)

	authMiddleware := middleware.NewAuthMiddleware(userSvc)
	rateLimiter    := middleware.NewRateLimiter(redisClient)
	handlers       := router.Handlers{
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

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server_start", "port", cfg.AppPort)
		serverErrors <- srv.ListenAndServe()
	}()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result := db.WithContext(ctx).Exec("DELETE FROM idempotency_records WHERE expires_at <= CURRENT_TIMESTAMP")
				if result.Error != nil {
					logger.Error("idempotency_cleanup_failed")
				} else {
					logger.Info("idempotency_cleanup", "deleted", result.RowsAffected)
				}
			}
		}

	}()
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return err
		}

	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}
