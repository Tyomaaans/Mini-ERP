package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type PaymentHandler struct{ paymentSvc PaymentService }

func NewPaymentHandler(paymentSvc PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.paymentSvc.CreatePayment(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	response, err := h.paymentSvc.GetPayment(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *PaymentHandler) GetPayments(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "paid_at", "amount"}, []string{"invoice_id", "status", "payment_method"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.paymentSvc.GetPayments(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *PaymentHandler) Post(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.paymentSvc.Post(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
