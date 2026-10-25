package role

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type RoleHandler struct{ roleSvc RoleService }

func NewRoleHandler(roleSvc RoleService) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc}
}

func (h *RoleHandler) GetRoles(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "code", "name"}, nil)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.roleSvc.GetRoles(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	response, err := h.roleSvc.GetRole(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *RoleHandler) SaveRole(c *gin.Context) {
	var req RoleRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.roleSvc.SaveRole(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	status := http.StatusOK
	if c.Param("id") == "" {
		status = http.StatusCreated
	}

	pkg.SuccessResponse(c, status, response)
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	if err := h.roleSvc.DeleteRole(c.Request.Context(), c.Param("id")); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *RoleHandler) Lookups(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		filters := []string{}
		if resource == "permissions" {
			filters = []string{"module"}
		}

		query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"code", "created_at"}, filters)
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}

		response, total, err := h.roleSvc.Lookups(c.Request.Context(), resource, query)
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}

		pkg.ListResponse(c, response, query, total)
	}
}
