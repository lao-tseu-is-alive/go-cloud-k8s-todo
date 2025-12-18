package todo

import "errors"

// Domain-specific errors for the Todo service
var (
	ErrNotFound         = errors.New("todo not found")
	ErrAlreadyExists    = errors.New("todo already exists")
	ErrTypeTodoNotFound = errors.New("type todo not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotOwner         = errors.New("user is not the owner")
	ErrAdminRequired    = errors.New("admin privileges required")
)
