package customer

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type CustomerHandler struct{ customerSvc CustomerService }

func NewCustomerHandler(customerSvc CustomerService) *CustomerHandler {
	return &CustomerHandler{customerSvc: customerSvc}
}

func (h *CustomerHandler) GetCustomers(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "name", "code"}, []string{"is_active"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.customerSvc.GetCustomers(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	response, err := h.customerSvc.GetCustomer(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.customerSvc.CreateCustomer(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	var req UpdateCustomerRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.customerSvc.UpdateCustomer(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *CustomerHandler) DeactivateCustomer(c *gin.Context) {
	active := false
	response, err := h.customerSvc.UpdateCustomer(c.Request.Context(), c.Param("id"), UpdateCustomerRequest{IsActive: &active})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
