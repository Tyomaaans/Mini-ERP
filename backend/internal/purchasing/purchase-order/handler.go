package purchaseorder

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type PurchaseOrderHandler struct{ purchaseOrderSvc PurchaseOrderService }

func NewPurchaseOrderHandler(purchaseOrderSvc PurchaseOrderService) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{purchaseOrderSvc: purchaseOrderSvc}
}

func (h *PurchaseOrderHandler) CreatePurchaseOrder(c *gin.Context) {
	var req CreatePurchaseOrderRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.purchaseOrderSvc.CreatePurchaseOrder(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *PurchaseOrderHandler) UpdatePurchaseOrder(c *gin.Context) {
	var req UpdatePurchaseOrderRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.purchaseOrderSvc.UpdatePurchaseOrder(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PurchaseOrderHandler) GetPurchaseOrder(c *gin.Context) {
	response, err := h.purchaseOrderSvc.GetPurchaseOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PurchaseOrderHandler) GetPurchaseOrders(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "order_date", "total_amount", "po_number"}, []string{"status", "warehouse_id", "supplier_id", "created_by", "from", "to"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.purchaseOrderSvc.GetPurchaseOrders(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *PurchaseOrderHandler) Submit(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note := ""

	response, err := h.purchaseOrderSvc.Submit(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PurchaseOrderHandler) Approve(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note := ""

	response, err := h.purchaseOrderSvc.Approve(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PurchaseOrderHandler) Reject(c *gin.Context) {
	note := ""
	var req DecisionRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note = req.Note
	response, err := h.purchaseOrderSvc.Reject(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PurchaseOrderHandler) Revise(c *gin.Context) {
	note := ""
	var req DecisionRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note = req.Note
	response, err := h.purchaseOrderSvc.Revise(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PurchaseOrderHandler) Cancel(c *gin.Context) {
	note := ""
	var req DecisionRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	note = req.Note
	response, err := h.purchaseOrderSvc.Cancel(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"), note)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
