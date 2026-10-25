package pkg

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    string
	Message string
	Status  int
	Details any
	Cause   error
}

func (e *AppError) Error() string {
	return e.Message
}
func (e *AppError) Unwrap() error {
	return e.Cause
}

func NewError(code, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

var (
	ErrInvalidInput       = NewError("INVALID_INPUT", "Request contains invalid input", http.StatusUnprocessableEntity)
	ErrUnauthorized       = NewError("UNAUTHORIZED", "Authentication is required", http.StatusUnauthorized)
	ErrInvalidCredentials = NewError("INVALID_CREDENTIALS", "Email or password is incorrect", http.StatusUnauthorized)
	ErrForbidden          = NewError("FORBIDDEN", "You do not have permission for this operation", http.StatusForbidden)
	ErrNotFound           = NewError("NOT_FOUND", "Resource was not found", http.StatusNotFound)
	ErrConflict           = NewError("CONFLICT", "Resource conflicts with existing data", http.StatusConflict)
	ErrInvalidState       = NewError("INVALID_STATE", "Document cannot perform this action in its current state", http.StatusConflict)
	ErrInternal           = NewError("INTERNAL_ERROR", "An internal error occurred", http.StatusInternalServerError)
)

func ErrorCode(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrInternal.Code
}
