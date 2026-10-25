package product

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type ProductHandler struct{ productSvc ProductService }

func NewProductHandler(productSvc ProductService) *ProductHandler {
	return &ProductHandler{productSvc: productSvc}
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "name", "sku", "selling_price"}, []string{"is_active", "category_id"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.productSvc.GetProducts(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	response, err := h.productSvc.GetProduct(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.productSvc.CreateProduct(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	var req UpdateProductRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.productSvc.UpdateProduct(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *ProductHandler) DeactivateProduct(c *gin.Context) {
	active := false
	response, err := h.productSvc.UpdateProduct(c.Request.Context(), c.Param("id"), UpdateProductRequest{IsActive: &active})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
