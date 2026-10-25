package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type UserHandler struct{ userSvc UserService }

func NewUserHandler(userSvc UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.userSvc.Login(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *UserHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.userSvc.Refresh(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *UserHandler) Logout(c *gin.Context) {
	if err := h.userSvc.Logout(c.Request.Context(), c.GetString("sessionID")); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, gin.H{"logged_out": true})
}

func (h *UserHandler) Me(c *gin.Context) {
	pkg.SuccessResponse(c, http.StatusOK, ToMeResponse(pkg.ActorFrom(c.Request.Context())))
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "full_name", "email"}, []string{"department_id", "is_active"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.userSvc.GetUsers(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	response, err := h.userSvc.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.userSvc.CreateUser(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.userSvc.UpdateUser(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *UserHandler) DeactivateUser(c *gin.Context) {
	active := false
	response, err := h.userSvc.UpdateUser(c.Request.Context(), c.Param("id"), UpdateUserRequest{IsActive: &active})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *UserHandler) AssignRoles(c *gin.Context) {
	var req AssignRolesRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.userSvc.AssignRoles(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *UserHandler) AssignWarehouses(c *gin.Context) {
	var req AssignWarehousesRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.userSvc.AssignWarehouses(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
