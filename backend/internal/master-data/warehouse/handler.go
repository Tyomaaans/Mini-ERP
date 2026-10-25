package warehouse

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type WarehouseHandler struct{ warehouseSvc WarehouseService }

func NewWarehouseHandler(warehouseSvc WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{warehouseSvc: warehouseSvc}
}

func (h *WarehouseHandler) GetWarehouses(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "name", "code"}, []string{"is_active"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.warehouseSvc.GetWarehouses(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *WarehouseHandler) GetWarehouse(c *gin.Context) {
	response, err := h.warehouseSvc.GetWarehouse(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *WarehouseHandler) CreateWarehouse(c *gin.Context) {
	var req CreateWarehouseRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.warehouseSvc.CreateWarehouse(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *WarehouseHandler) UpdateWarehouse(c *gin.Context) {
	var req UpdateWarehouseRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.warehouseSvc.UpdateWarehouse(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *WarehouseHandler) DeactivateWarehouse(c *gin.Context) {
	active := false
	response, err := h.warehouseSvc.UpdateWarehouse(c.Request.Context(), c.Param("id"), UpdateWarehouseRequest{IsActive: &active})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
