package todo

import "errors"

// Domain-specific errors for the Todo service
var (
	ErrNotFound         = errors.New("todo_app not found")
	ErrAlreadyExists    = errors.New("todo_app already exists")
	ErrTypeTodoNotFound = errors.New("type todo_app not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotOwner         = errors.New("user is not the owner")
	ErrAdminRequired    = errors.New("admin privileges required")
)
