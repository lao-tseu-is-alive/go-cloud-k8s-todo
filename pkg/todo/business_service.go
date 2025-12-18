package todo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/database"
)

// BusinessService Business Service contains the transport-agnostic business logic for Todo operations
type BusinessService struct {
	Log              *slog.Logger
	DbConn           database.DB
	Store            Storage
	ListDefaultLimit int
}

// NewBusinessService creates a new instance of BusinessService
func NewBusinessService(store Storage, dbConn database.DB, log *slog.Logger, listDefaultLimit int) *BusinessService {
	return &BusinessService{
		Log:              log,
		DbConn:           dbConn,
		Store:            store,
		ListDefaultLimit: listDefaultLimit,
	}
}

// validateName validates the name field according to business rules
func validateName(name string) error {
	if len(strings.Trim(name, " ")) < 1 {
		return fmt.Errorf(FieldCannotBeEmpty, "name")
	}
	if len(name) < MinNameLength {
		return fmt.Errorf(FieldMinLengthIsN, "name", MinNameLength)
	}
	return nil
}

// GeoJson returns a geoJson representation of todo_apps based on the given parameters
func (s *BusinessService) GeoJson(ctx context.Context, offset, limit int, params GeoJsonParams) (string, error) {
	jsonResult, err := s.Store.GeoJson(ctx, offset, limit, params)
	if err != nil {
		return "", fmt.Errorf("error retrieving geoJson: %w", err)
	}
	if jsonResult == "" {
		return "empty", nil
	}
	return jsonResult, nil
}

// List returns the list of todo_apps based on the given parameters
func (s *BusinessService) List(ctx context.Context, offset, limit int, params ListParams) ([]*TodoList, error) {
	list, err := s.Store.List(ctx, offset, limit, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows is not an error, return empty slice
			return make([]*TodoList, 0), nil
		}
		return nil, fmt.Errorf("error listing todo_apps: %w", err)
	}
	if list == nil {
		return make([]*TodoList, 0), nil
	}
	return list, nil
}

// Create creates a new todo_app with the given data
func (s *BusinessService) Create(ctx context.Context, currentUserId int32, newTodo Todo) (*Todo, error) {
	// Validate name
	if err := validateName(newTodo.Name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Validate TypeId
	typeTodoCount, err := s.DbConn.GetQueryInt(ctx, existTypeTodo, newTodo.TypeId)
	if err != nil || typeTodoCount < 1 {
		return nil, fmt.Errorf("%w: typeId %v", ErrTypeTodoNotFound, newTodo.TypeId)
	}

	// Check if todo_app already exists
	if s.Store.Exist(ctx, newTodo.Id) {
		return nil, fmt.Errorf("%w: id %v", ErrAlreadyExists, newTodo.Id)
	}

	// Set creator
	newTodo.CreatedBy = currentUserId

	// Create in storage
	todo_appCreated, err := s.Store.Create(ctx, newTodo)
	if err != nil {
		return nil, fmt.Errorf("error creating todo_app: %w", err)
	}

	s.Log.Info("Created todo_app", "id", todo_appCreated.Id, "userId", currentUserId)
	return todo_appCreated, nil
}

// Count returns the number of todo_apps based on the given parameters
func (s *BusinessService) Count(ctx context.Context, params CountParams) (int32, error) {
	numTodos, err := s.Store.Count(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error counting todo_apps: %w", err)
	}
	return numTodos, nil
}

// Delete removes a todo_app with the given ID
func (s *BusinessService) Delete(ctx context.Context, currentUserId int32, todo_appId uuid.UUID) error {
	// Check if todo_app exists
	if !s.Store.Exist(ctx, todo_appId) {
		return fmt.Errorf("%w: id %v", ErrNotFound, todo_appId)
	}

	// Check if user is owner
	if !s.Store.IsUserOwner(ctx, todo_appId, currentUserId) {
		return fmt.Errorf("%w: user %d is not owner of todo_app %v", ErrUnauthorized, currentUserId, todo_appId)
	}

	// Delete from storage
	err := s.Store.Delete(ctx, todo_appId, currentUserId)
	if err != nil {
		return fmt.Errorf("error deleting todo_app: %w", err)
	}

	s.Log.Info("Deleted todo_app", "id", todo_appId, "userId", currentUserId)
	return nil
}

// Get retrieves a todo_app by its ID
func (s *BusinessService) Get(ctx context.Context, todo_appId uuid.UUID) (*Todo, error) {
	// Check if todo_app exists
	if !s.Store.Exist(ctx, todo_appId) {
		return nil, fmt.Errorf("%w: id %v", ErrNotFound, todo_appId)
	}

	// Get from storage
	todo_app, err := s.Store.Get(ctx, todo_appId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving todo_app: %w", err)
	}

	return todo_app, nil
}

// Update updates a todo_app with the given ID
func (s *BusinessService) Update(ctx context.Context, currentUserId int32, todo_appId uuid.UUID, updateTodo Todo) (*Todo, error) {
	// Check if todo_app exists
	if !s.Store.Exist(ctx, todo_appId) {
		return nil, fmt.Errorf("%w: id %v", ErrNotFound, todo_appId)
	}

	// Check if user is owner
	if !s.Store.IsUserOwner(ctx, todo_appId, currentUserId) {
		return nil, fmt.Errorf("%w: user %d is not owner of todo_app %v", ErrUnauthorized, currentUserId, todo_appId)
	}

	// Validate name
	if err := validateName(updateTodo.Name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Validate TypeId
	typeTodoCount, err := s.DbConn.GetQueryInt(ctx, existTypeTodo, updateTodo.TypeId)
	if err != nil || typeTodoCount < 1 {
		return nil, fmt.Errorf("%w: typeId %v", ErrTypeTodoNotFound, updateTodo.TypeId)
	}

	// Set last modifier
	updateTodo.LastModifiedBy = &currentUserId

	// Update in storage
	todo_appUpdated, err := s.Store.Update(ctx, todo_appId, updateTodo)
	if err != nil {
		return nil, fmt.Errorf("error updating todo_app: %w", err)
	}

	s.Log.Info("Updated todo_app", "id", todo_appId, "userId", currentUserId)
	return todo_appUpdated, nil
}

// ListByExternalId returns todo_apps filtered by external ID
func (s *BusinessService) ListByExternalId(ctx context.Context, offset, limit, externalId int) ([]*TodoList, error) {
	list, err := s.Store.ListByExternalId(ctx, offset, limit, externalId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows is not an error, return empty slice
			return make([]*TodoList, 0), nil
		}
		return nil, fmt.Errorf("error listing todo_apps by external id: %w", err)
	}
	if list == nil {
		return make([]*TodoList, 0), nil
	}
	return list, nil
}

// Search returns todo_apps based on search criteria
func (s *BusinessService) Search(ctx context.Context, offset, limit int, params SearchParams) ([]*TodoList, error) {
	list, err := s.Store.Search(ctx, offset, limit, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows is not an error, return empty slice
			return make([]*TodoList, 0), nil
		}
		return nil, fmt.Errorf("error searching todo_apps: %w", err)
	}
	if list == nil {
		return make([]*TodoList, 0), nil
	}
	return list, nil
}

// ListTypeTodos returns a list of TypeTodo based on parameters
func (s *BusinessService) ListTypeTodos(ctx context.Context, offset, limit int, params TypeTodoListParams) ([]*TypeTodoList, error) {
	list, err := s.Store.ListTypeTodo(ctx, offset, limit, params)
	if err != nil {
		return nil, fmt.Errorf("error listing type todo_apps: %w", err)
	}
	if list == nil {
		return make([]*TypeTodoList, 0), nil
	}
	return list, nil
}

// CreateTypeTodo creates a new TypeTodo
func (s *BusinessService) CreateTypeTodo(ctx context.Context, currentUserId int32, isAdmin bool, newTypeTodo TypeTodo) (*TypeTodo, error) {
	// Check admin privileges
	if !isAdmin {
		return nil, ErrAdminRequired
	}

	// Validate name
	if err := validateName(newTypeTodo.Name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Set creator
	newTypeTodo.CreatedBy = currentUserId

	// Create in storage
	typeTodoCreated, err := s.Store.CreateTypeTodo(ctx, newTypeTodo)
	if err != nil {
		return nil, fmt.Errorf("error creating type todo_app: %w", err)
	}

	s.Log.Info("Created TypeTodo", "id", typeTodoCreated.Id, "userId", currentUserId)
	return typeTodoCreated, nil
}

// CountTypeTodos returns the count of TypeTodos based on parameters
func (s *BusinessService) CountTypeTodos(ctx context.Context, params TypeTodoCountParams) (int32, error) {
	numTodos, err := s.Store.CountTypeTodo(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error counting type todo_apps: %w", err)
	}
	return numTodos, nil
}

// DeleteTypeTodo deletes a TypeTodo by ID
func (s *BusinessService) DeleteTypeTodo(ctx context.Context, currentUserId int32, isAdmin bool, typeTodoId int32) error {
	// Check admin privileges
	if !isAdmin {
		return ErrAdminRequired
	}

	// Check if TypeTodo exists
	typeTodoCount, err := s.DbConn.GetQueryInt(ctx, existTypeTodo, typeTodoId)
	if err != nil || typeTodoCount < 1 {
		return fmt.Errorf("%w: id %d", ErrTypeTodoNotFound, typeTodoId)
	}

	// Delete from storage
	err = s.Store.DeleteTypeTodo(ctx, typeTodoId, currentUserId)
	if err != nil {
		return fmt.Errorf("error deleting type todo_app: %w", err)
	}

	s.Log.Info("Deleted TypeTodo", "id", typeTodoId, "userId", currentUserId)
	return nil
}

// GetTypeTodo retrieves a TypeTodo by ID
func (s *BusinessService) GetTypeTodo(ctx context.Context, isAdmin bool, typeTodoId int32) (*TypeTodo, error) {
	// Check admin privileges
	if !isAdmin {
		return nil, ErrAdminRequired
	}

	// Check if TypeTodo exists
	typeTodoCount, err := s.DbConn.GetQueryInt(ctx, existTypeTodo, typeTodoId)
	if err != nil || typeTodoCount < 1 {
		return nil, fmt.Errorf("%w: id %d", ErrTypeTodoNotFound, typeTodoId)
	}

	// Get from storage
	typeTodo, err := s.Store.GetTypeTodo(ctx, typeTodoId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving type todo_app: %w", err)
	}

	return typeTodo, nil
}

// UpdateTypeTodo updates a TypeTodo
func (s *BusinessService) UpdateTypeTodo(ctx context.Context, currentUserId int32, isAdmin bool, typeTodoId int32, updateTypeTodo TypeTodo) (*TypeTodo, error) {
	// Check admin privileges
	if !isAdmin {
		return nil, ErrAdminRequired
	}

	// Check if TypeTodo exists
	typeTodoCount, err := s.DbConn.GetQueryInt(ctx, existTypeTodo, typeTodoId)
	if err != nil || typeTodoCount < 1 {
		return nil, fmt.Errorf("%w: id %d", ErrTypeTodoNotFound, typeTodoId)
	}

	// Validate name
	if err := validateName(updateTypeTodo.Name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Set last modifier
	updateTypeTodo.LastModifiedBy = &currentUserId

	// Update in storage
	todo_appUpdated, err := s.Store.UpdateTypeTodo(ctx, typeTodoId, updateTypeTodo)
	if err != nil {
		return nil, fmt.Errorf("error updating type todo_app: %w", err)
	}

	s.Log.Info("Updated TypeTodo", "id", typeTodoId, "userId", currentUserId)
	return todo_appUpdated, nil
}
