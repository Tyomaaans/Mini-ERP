package invoice

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type InvoiceHandler struct{ invoiceSvc InvoiceService }

func NewInvoiceHandler(invoiceSvc InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceSvc: invoiceSvc}
}

func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	var req CreateInvoiceRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.invoiceSvc.CreateInvoice(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	response, err := h.invoiceSvc.GetInvoice(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *InvoiceHandler) GetInvoices(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "due_date", "total_amount", "invoice_number"}, []string{"invoice_type", "status", "customer_id", "supplier_id", "sales_order_id", "purchase_order_id", "from", "to"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.invoiceSvc.GetInvoices(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *InvoiceHandler) UpdateInvoice(c *gin.Context) {
	var req UpdateInvoiceRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.invoiceSvc.UpdateInvoice(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *InvoiceHandler) Issue(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.invoiceSvc.Issue(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
