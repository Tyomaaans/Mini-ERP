package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"mini-erp/docs"
	"mini-erp/internal/auth-session/role"
	"mini-erp/internal/auth-session/user"
	"mini-erp/internal/config"
	"mini-erp/internal/finance/invoice"
	"mini-erp/internal/finance/payment"
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
	salesorder "mini-erp/internal/sales/sales-order"
	"mini-erp/internal/sales/shipment"
	"mini-erp/pkg"
)

type Handlers struct {
	User          *user.UserHandler
	Role          *role.RoleHandler
	Product       *product.ProductHandler
	Category      *category.CategoryHandler
	Customer      *customer.CustomerHandler
	Supplier      *supplier.SupplierHandler
	Warehouse     *warehouse.WarehouseHandler
	PurchaseOrder *purchaseorder.PurchaseOrderHandler
	GoodsReceipt  *goodsreceipt.GoodsReceiptHandler
	Stock         *stock.StockHandler
	SalesOrder    *salesorder.SalesOrderHandler
	Shipment      *shipment.ShipmentHandler
	Invoice       *invoice.InvoiceHandler
	Payment       *payment.PaymentHandler
	Report        *report.ReportHandler
}

func NewRouter(
	h           Handlers,
	auth        *middleware.AuthMiddleware,
	limiter     *middleware.RateLimiter,
	db          *gorm.DB,
	redisClient *redis.Client,
	cfg         config.AppConfig,
	logger      *slog.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	metrics := &middleware.Metrics{}
	r.Use(middleware.Request(logger, metrics))
	if len(cfg.AllowedOrigins) > 0 {
		r.Use(func(c *gin.Context) {
			origin := c.GetHeader("Origin")
			allowed := origin == "" || origin == "http://"+c.Request.Host || origin == "https://"+c.Request.Host
			for _, value := range cfg.AllowedOrigins {
				if origin == value {
					allowed = true
				}
			}
			if !allowed {
				pkg.ErrorResponse(c, pkg.ErrForbidden)
				return
			}
			c.Next()
		})
		r.Use(cors.New(cors.Config{
			AllowOrigins:  cfg.AllowedOrigins,
			AllowMethods:  []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:  []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"},
			ExposeHeaders: []string{"X-Request-ID", "Retry-After"},
			MaxAge:        12 * time.Hour,
		}))
	}
	r.NoRoute(func(c *gin.Context) {
		pkg.ErrorResponse(c, pkg.ErrNotFound)
	})
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		pkg.ErrorResponse(c, pkg.NewError("METHOD_NOT_ALLOWED", "HTTP method is not supported", 405))
	})
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health/live", func(c *gin.Context) {
			pkg.SuccessResponse(c, http.StatusOK, gin.H{"status": "ok"})
		})
		v1.GET("/health", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			sqlDB, err := db.DB()
			if err != nil || sqlDB.PingContext(ctx) != nil || redisClient.Ping(ctx).Err() != nil {
				pkg.ErrorResponse(c, pkg.NewError("NOT_READY", "A required dependency is unavailable", 503))
				return
			}
			pkg.SuccessResponse(c, http.StatusOK, gin.H{"status": "ready"})
		})
		v1.GET("/openapi.yaml", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.Spec)
		})
		v1.GET("/docs", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", docs.UI)
		})
		v1.POST("/auth/login", limiter.Limit("auth.login", 10, time.Minute), h.User.Login)
		v1.POST("/auth/refresh", limiter.Limit("auth.refresh", 30, time.Minute), h.User.Refresh)
		secured := v1.Group("", auth.Authenticate(), limiter.Limit("api", 600, time.Minute))
		{
			secured.GET("/metrics", metrics.Handler)
			secured.GET("/auth/me", h.User.Me)
			secured.POST("/auth/logout", h.User.Logout)
			secured.GET("/users", h.User.GetUsers)
			secured.POST("/users", h.User.CreateUser)
			secured.GET("/users/:id", h.User.GetUser)
			secured.PATCH("/users/:id", h.User.UpdateUser)
			secured.DELETE("/users/:id", h.User.DeactivateUser)
			secured.PUT("/users/:id/roles", h.User.AssignRoles)
			secured.PUT("/users/:id/warehouses", h.User.AssignWarehouses)
			secured.GET("/roles", h.Role.GetRoles)
			secured.POST("/roles", h.Role.SaveRole)
			secured.GET("/roles/:id", h.Role.GetRole)
			secured.PATCH("/roles/:id", h.Role.SaveRole)
			secured.DELETE("/roles/:id", h.Role.DeleteRole)
			secured.GET("/departments", h.Role.Lookups("departments"))
			secured.GET("/permissions", h.Role.Lookups("permissions"))
			secured.GET("/categories", h.Category.GetCategories)
			secured.POST("/categories", h.Category.CreateCategory)
			secured.GET("/categories/:id", h.Category.GetCategory)
			secured.PATCH("/categories/:id", h.Category.UpdateCategory)
			secured.DELETE("/categories/:id", h.Category.DeleteCategory)
			secured.GET("/products", h.Product.GetProducts)
			secured.POST("/products", h.Product.CreateProduct)
			secured.GET("/products/:id", h.Product.GetProduct)
			secured.PATCH("/products/:id", h.Product.UpdateProduct)
			secured.DELETE("/products/:id", h.Product.DeactivateProduct)
			secured.GET("/customers", h.Customer.GetCustomers)
			secured.POST("/customers", h.Customer.CreateCustomer)
			secured.GET("/customers/:id", h.Customer.GetCustomer)
			secured.PATCH("/customers/:id", h.Customer.UpdateCustomer)
			secured.DELETE("/customers/:id", h.Customer.DeactivateCustomer)
			secured.GET("/suppliers", h.Supplier.GetSuppliers)
			secured.POST("/suppliers", h.Supplier.CreateSupplier)
			secured.GET("/suppliers/:id", h.Supplier.GetSupplier)
			secured.PATCH("/suppliers/:id", h.Supplier.UpdateSupplier)
			secured.DELETE("/suppliers/:id", h.Supplier.DeactivateSupplier)
			secured.GET("/warehouses", h.Warehouse.GetWarehouses)
			secured.POST("/warehouses", h.Warehouse.CreateWarehouse)
			secured.GET("/warehouses/:id", h.Warehouse.GetWarehouse)
			secured.PATCH("/warehouses/:id", h.Warehouse.UpdateWarehouse)
			secured.DELETE("/warehouses/:id", h.Warehouse.DeactivateWarehouse)
			secured.GET("/purchase-orders", h.PurchaseOrder.GetPurchaseOrders)
			secured.POST("/purchase-orders", h.PurchaseOrder.CreatePurchaseOrder)
			secured.GET("/purchase-orders/:id", h.PurchaseOrder.GetPurchaseOrder)
			secured.PATCH("/purchase-orders/:id", h.PurchaseOrder.UpdatePurchaseOrder)
			secured.POST("/purchase-orders/:id/submit", h.PurchaseOrder.Submit)
			secured.POST("/purchase-orders/:id/approve", h.PurchaseOrder.Approve)
			secured.POST("/purchase-orders/:id/reject", h.PurchaseOrder.Reject)
			secured.POST("/purchase-orders/:id/revise", h.PurchaseOrder.Revise)
			secured.POST("/purchase-orders/:id/cancel", h.PurchaseOrder.Cancel)
			secured.GET("/sales-orders", h.SalesOrder.GetSalesOrders)
			secured.POST("/sales-orders", h.SalesOrder.CreateSalesOrder)
			secured.GET("/sales-orders/:id", h.SalesOrder.GetSalesOrder)
			secured.PATCH("/sales-orders/:id", h.SalesOrder.UpdateSalesOrder)
			secured.POST("/sales-orders/:id/confirm", h.SalesOrder.Confirm)
			secured.POST("/sales-orders/:id/reserve", h.SalesOrder.Reserve)
			secured.POST("/sales-orders/:id/cancel", h.SalesOrder.Cancel)
			secured.GET("/goods-receipts", h.GoodsReceipt.GetGoodsReceipts)
			secured.POST("/goods-receipts", h.GoodsReceipt.CreateGoodsReceipt)
			secured.GET("/goods-receipts/:id", h.GoodsReceipt.GetGoodsReceipt)
			secured.POST("/goods-receipts/:id/post", h.GoodsReceipt.Post)
			secured.GET("/shipments", h.Shipment.GetShipments)
			secured.POST("/shipments", h.Shipment.CreateShipment)
			secured.GET("/shipments/:id", h.Shipment.GetShipment)
			secured.POST("/shipments/:id/post", h.Shipment.Post)
			secured.GET("/invoices", h.Invoice.GetInvoices)
			secured.POST("/invoices", h.Invoice.CreateInvoice)
			secured.GET("/invoices/:id", h.Invoice.GetInvoice)
			secured.POST("/invoices/:id/issue", h.Invoice.Issue)
			secured.PATCH("/invoices/:id", h.Invoice.UpdateInvoice)
			secured.GET("/payments", h.Payment.GetPayments)
			secured.POST("/payments", h.Payment.CreatePayment)
			secured.GET("/payments/:id", h.Payment.GetPayment)
			secured.POST("/payments/:id/post", h.Payment.Post)
			secured.GET("/inventory/balances", h.Stock.GetBalances)
			secured.GET("/inventory/movements", h.Stock.GetMovements)
			secured.GET("/reports/sales-summary", h.Report.Summary("sales"))
			secured.GET("/reports/purchasing-summary", h.Report.Summary("purchasing"))
			secured.GET("/reports/low-stock", h.Report.LowStock)
			secured.GET("/reports/receivables", h.Report.Outstanding("AR"))
			secured.GET("/reports/payables", h.Report.Outstanding("AP"))
			secured.GET("/audit-logs", h.Report.AuditLogs)
		}
	}
	return r
}
