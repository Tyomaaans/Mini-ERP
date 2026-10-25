package salesorder

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type SalesOrderHandler struct{ salesOrderSvc SalesOrderService }

func NewSalesOrderHandler(salesOrderSvc SalesOrderService) *SalesOrderHandler {
	return &SalesOrderHandler{salesOrderSvc: salesOrderSvc}
}

func (h *SalesOrderHandler) CreateSalesOrder(c *gin.Context) {
	var req CreateSalesOrderRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.salesOrderSvc.CreateSalesOrder(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *SalesOrderHandler) UpdateSalesOrder(c *gin.Context) {
	var req UpdateSalesOrderRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.salesOrderSvc.UpdateSalesOrder(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *SalesOrderHandler) GetSalesOrder(c *gin.Context) {
	response, err := h.salesOrderSvc.GetSalesOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *SalesOrderHandler) GetSalesOrders(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "order_date", "total_amount", "so_number"}, []string{"status", "warehouse_id", "customer_id", "created_by", "from", "to"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.salesOrderSvc.GetSalesOrders(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *SalesOrderHandler) Confirm(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note := ""

	response, err := h.salesOrderSvc.Confirm(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *SalesOrderHandler) Reserve(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note := ""

	response, err := h.salesOrderSvc.Reserve(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *SalesOrderHandler) Cancel(c *gin.Context) {
	note := ""
	var req DecisionRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note = req.Note
	response, err := h.salesOrderSvc.Cancel(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
