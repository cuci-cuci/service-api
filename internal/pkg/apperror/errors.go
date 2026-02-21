package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

var (
	ErrNotFound     = &AppError{Code: http.StatusNotFound, Message: "resource not found"}
	ErrUnauthorized = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden    = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrConflict     = &AppError{Code: http.StatusConflict, Message: "resource already exists"}
	ErrValidation   = &AppError{Code: http.StatusUnprocessableEntity, Message: "validation error"}
	ErrInternal     = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
)

func NewAppError(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func WrapError(sentinel *AppError, err error) *AppError {
	return &AppError{
		Code:    sentinel.Code,
		Message: sentinel.Message,
		Err:     err,
	}
}

func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func NotFound(msg string) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: http.StatusConflict, Message: msg}
}

func Validation(msg string) *AppError {
	return &AppError{Code: http.StatusUnprocessableEntity, Message: msg}
}

func Internal(msg string, err error) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: msg, Err: err}
}
