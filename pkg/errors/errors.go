package errors

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrBadRequest          = errors.New("bad request")
	ErrInternalServer      = errors.New("internal server error")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrRoomNotFound        = errors.New("room not found")
	ErrParticipantNotFound = errors.New("participant not found")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
)

type APIError struct {
	Message string `json:"error"`
	Code    int    `json:"code"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(message string, code int) *APIError {
	return &APIError{
		Message: message,
		Code:    code,
	}
}

func HTTPStatusFromError(err error) int {
	switch err {
	case ErrNotFound, ErrRoomNotFound, ErrParticipantNotFound:
		return http.StatusNotFound
	case ErrUnauthorized, ErrInvalidCredentials, ErrInvalidToken, ErrTokenExpired:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrBadRequest, ErrUserAlreadyExists:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

