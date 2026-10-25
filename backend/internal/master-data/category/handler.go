package category

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type CategoryHandler struct{ categorySvc CategoryService }

func NewCategoryHandler(categorySvc CategoryService) *CategoryHandler {
	return &CategoryHandler{categorySvc: categorySvc}
}

func (h *CategoryHandler) GetCategories(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "name", "code"}, []string{"parent_id"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.categorySvc.GetCategories(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *CategoryHandler) GetCategory(c *gin.Context) {
	response, err := h.categorySvc.GetCategory(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.categorySvc.CreateCategory(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	var req UpdateCategoryRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.categorySvc.UpdateCategory(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	if err := h.categorySvc.DeleteCategory(c.Request.Context(), c.Param("id")); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, gin.H{"deleted": true})
}
