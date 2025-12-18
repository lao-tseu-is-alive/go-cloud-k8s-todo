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

// GeoJson returns a geoJson representation of todos based on the given parameters
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

// List returns the list of todos based on the given parameters
func (s *BusinessService) List(ctx context.Context, offset, limit int, params ListParams) ([]*TodoList, error) {
	list, err := s.Store.List(ctx, offset, limit, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows is not an error, return empty slice
			return make([]*TodoList, 0), nil
		}
		return nil, fmt.Errorf("error listing todos: %w", err)
	}
	if list == nil {
		return make([]*TodoList, 0), nil
	}
	return list, nil
}

// Create creates a new todo with the given data
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

	// Check if todo already exists
	if s.Store.Exist(ctx, newTodo.Id) {
		return nil, fmt.Errorf("%w: id %v", ErrAlreadyExists, newTodo.Id)
	}

	// Set creator
	newTodo.CreatedBy = currentUserId

	// Create in storage
	todoCreated, err := s.Store.Create(ctx, newTodo)
	if err != nil {
		return nil, fmt.Errorf("error creating todo: %w", err)
	}

	s.Log.Info("Created todo", "id", todoCreated.Id, "userId", currentUserId)
	return todoCreated, nil
}

// Count returns the number of todos based on the given parameters
func (s *BusinessService) Count(ctx context.Context, params CountParams) (int32, error) {
	numTodos, err := s.Store.Count(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error counting todos: %w", err)
	}
	return numTodos, nil
}

// Delete removes a todo with the given ID
func (s *BusinessService) Delete(ctx context.Context, currentUserId int32, todoId uuid.UUID) error {
	// Check if todo exists
	if !s.Store.Exist(ctx, todoId) {
		return fmt.Errorf("%w: id %v", ErrNotFound, todoId)
	}

	// Check if user is owner
	if !s.Store.IsUserOwner(ctx, todoId, currentUserId) {
		return fmt.Errorf("%w: user %d is not owner of todo %v", ErrUnauthorized, currentUserId, todoId)
	}

	// Delete from storage
	err := s.Store.Delete(ctx, todoId, currentUserId)
	if err != nil {
		return fmt.Errorf("error deleting todo: %w", err)
	}

	s.Log.Info("Deleted todo", "id", todoId, "userId", currentUserId)
	return nil
}

// Get retrieves a todo by its ID
func (s *BusinessService) Get(ctx context.Context, todoId uuid.UUID) (*Todo, error) {
	// Check if todo exists
	if !s.Store.Exist(ctx, todoId) {
		return nil, fmt.Errorf("%w: id %v", ErrNotFound, todoId)
	}

	// Get from storage
	todo, err := s.Store.Get(ctx, todoId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving todo: %w", err)
	}

	return todo, nil
}

// Update updates a todo with the given ID
func (s *BusinessService) Update(ctx context.Context, currentUserId int32, todoId uuid.UUID, updateTodo Todo) (*Todo, error) {
	// Check if todo exists
	if !s.Store.Exist(ctx, todoId) {
		return nil, fmt.Errorf("%w: id %v", ErrNotFound, todoId)
	}

	// Check if user is owner
	if !s.Store.IsUserOwner(ctx, todoId, currentUserId) {
		return nil, fmt.Errorf("%w: user %d is not owner of todo %v", ErrUnauthorized, currentUserId, todoId)
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
	todoUpdated, err := s.Store.Update(ctx, todoId, updateTodo)
	if err != nil {
		return nil, fmt.Errorf("error updating todo: %w", err)
	}

	s.Log.Info("Updated todo", "id", todoId, "userId", currentUserId)
	return todoUpdated, nil
}

// ListByExternalId returns todos filtered by external ID
func (s *BusinessService) ListByExternalId(ctx context.Context, offset, limit, externalId int) ([]*TodoList, error) {
	list, err := s.Store.ListByExternalId(ctx, offset, limit, externalId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows is not an error, return empty slice
			return make([]*TodoList, 0), nil
		}
		return nil, fmt.Errorf("error listing todos by external id: %w", err)
	}
	if list == nil {
		return make([]*TodoList, 0), nil
	}
	return list, nil
}

// Search returns todos based on search criteria
func (s *BusinessService) Search(ctx context.Context, offset, limit int, params SearchParams) ([]*TodoList, error) {
	list, err := s.Store.Search(ctx, offset, limit, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No rows is not an error, return empty slice
			return make([]*TodoList, 0), nil
		}
		return nil, fmt.Errorf("error searching todos: %w", err)
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
		return nil, fmt.Errorf("error listing type todos: %w", err)
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
		return nil, fmt.Errorf("error creating type todo: %w", err)
	}

	s.Log.Info("Created TypeTodo", "id", typeTodoCreated.Id, "userId", currentUserId)
	return typeTodoCreated, nil
}

// CountTypeTodos returns the count of TypeTodos based on parameters
func (s *BusinessService) CountTypeTodos(ctx context.Context, params TypeTodoCountParams) (int32, error) {
	numTodos, err := s.Store.CountTypeTodo(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("error counting type todos: %w", err)
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
		return fmt.Errorf("error deleting type todo: %w", err)
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
		return nil, fmt.Errorf("error retrieving type todo: %w", err)
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
	todoUpdated, err := s.Store.UpdateTypeTodo(ctx, typeTodoId, updateTypeTodo)
	if err != nil {
		return nil, fmt.Errorf("error updating type todo: %w", err)
	}

	s.Log.Info("Updated TypeTodo", "id", typeTodoId, "userId", currentUserId)
	return todoUpdated, nil
}
