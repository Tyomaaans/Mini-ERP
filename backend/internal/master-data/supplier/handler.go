package supplier

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type SupplierHandler struct{ supplierSvc SupplierService }

func NewSupplierHandler(supplierSvc SupplierService) *SupplierHandler {
	return &SupplierHandler{supplierSvc: supplierSvc}
}

func (h *SupplierHandler) GetSuppliers(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "name", "code"}, []string{"is_active"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.supplierSvc.GetSuppliers(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *SupplierHandler) GetSupplier(c *gin.Context) {
	response, err := h.supplierSvc.GetSupplier(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *SupplierHandler) CreateSupplier(c *gin.Context) {
	var req CreateSupplierRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.supplierSvc.CreateSupplier(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *SupplierHandler) UpdateSupplier(c *gin.Context) {
	var req UpdateSupplierRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.supplierSvc.UpdateSupplier(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *SupplierHandler) DeactivateSupplier(c *gin.Context) {
	active := false
	response, err := h.supplierSvc.UpdateSupplier(c.Request.Context(), c.Param("id"), UpdateSupplierRequest{IsActive: &active})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
