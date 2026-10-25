package pkg

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

type Response struct {
	Data any             `json:"data"`
	Meta *PaginationMeta `json:"meta,omitempty"`
}

func SuccessResponse(c *gin.Context, status int, data any) {
	c.JSON(status, Response{Data: data})
}

func ListResponse(c *gin.Context, data any, query ListQuery, total int64) {
	c.JSON(http.StatusOK, Response{
		Data: data,
		Meta: &PaginationMeta{
			Page:       query.Page,
			PerPage:    query.PerPage,
			Total:      total,
			TotalPages: (total + int64(query.PerPage) - 1) / int64(query.PerPage),
		},
	})
}

func ErrorResponse(c *gin.Context, err error) {
	appErr := ErrInternal
	var known *AppError
	if errors.As(err, &known) {
		appErr = known
	}
	if errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
		appErr = NewError("REQUEST_TIMEOUT", "Request timed out", http.StatusServiceUnavailable)
	}

	c.AbortWithStatusJSON(appErr.Status, gin.H{"error": gin.H{
		"code": appErr.Code, "message": appErr.Message,
		"details": appErr.Details, "request_id": RequestID(c.Request.Context()),
	}})
}
