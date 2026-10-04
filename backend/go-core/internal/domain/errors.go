package domain

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidInput        = errors.New("invalid input")
	ErrConflict            = errors.New("conflict")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidStatus       = errors.New("invalid status for this operation")
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrTokenExpired        = errors.New("token has expired")
)
