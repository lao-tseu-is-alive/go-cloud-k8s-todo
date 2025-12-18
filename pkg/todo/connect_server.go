// Package todo provides Connect RPC handlers for the TodoService.
package todo

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	todov1 "github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo/v1"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo/v1/todov1connect"
)

// TodoConnectServer implements the TodoServiceHandler interface.
// Authentication is handled by the AuthInterceptor, which injects user info into context.
type TodoConnectServer struct {
	BusinessService *BusinessService
	Log             *slog.Logger

	// Embed the unimplemented handler for forward compatibility
	todov1connect.UnimplementedTodoServiceHandler
}

// NewTodoConnectServer creates a new TodoConnectServer.
// Note: Authentication is handled by the AuthInterceptor, not by this server.
func NewTodoConnectServer(business *BusinessService, log *slog.Logger) *TodoConnectServer {
	return &TodoConnectServer{
		BusinessService: business,
		Log:             log,
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

// mapErrorToConnect converts business errors to Connect errors
func (s *TodoConnectServer) mapErrorToConnect(err error) *connect.Error {
	switch {
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrTypeTodoNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, ErrUnauthorized):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, ErrNotOwner):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, ErrAdminRequired):
		return connect.NewError(connect.CodePermissionDenied, errors.New(OnlyAdminCanManageTypeTodos))
	case errors.Is(err, ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, pgx.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, errors.New("not found"))
	default:
		s.Log.Error("internal error", "error", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

// =============================================================================
// TodoService RPC Methods
// =============================================================================

// List returns a list of todos
func (s *TodoConnectServer) List(
	ctx context.Context,
	req *connect.Request[todov1.ListRequest],
) (*connect.Response[todov1.ListResponse], error) {
	s.Log.Info("Connect: List called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("List", "userId", userId)

	// Build domain params from proto request
	msg := req.Msg
	params := ListParams{}
	if msg.Type != 0 {
		params.Type = &msg.Type
	}
	if msg.CreatedBy != 0 {
		params.CreatedBy = &msg.CreatedBy
	}
	if msg.Inactivated {
		params.Inactivated = &msg.Inactivated
	}
	if msg.Validated {
		params.Validated = &msg.Validated
	}

	// Handle pagination with defaults
	limit := s.BusinessService.ListDefaultLimit
	if msg.Limit > 0 {
		limit = int(msg.Limit)
	}
	offset := 0
	if msg.Offset > 0 {
		offset = int(msg.Offset)
	}

	// Call business logic
	list, err := s.BusinessService.List(ctx, offset, limit, params)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	// Convert to proto and return
	response := &todov1.ListResponse{
		Todos: DomainTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}

// Create creates a new todo
func (s *TodoConnectServer) Create(
	ctx context.Context,
	req *connect.Request[todov1.CreateRequest],
) (*connect.Response[todov1.CreateResponse], error) {
	s.Log.Info("Connect: Create called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Create", "userId", userId)

	// Convert proto to domain
	protoTodo := req.Msg.Todo
	if protoTodo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("todo is required"))
	}

	domainTodo, err := ProtoTodoToDomain(protoTodo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Call business logic
	createdTodo, err := s.BusinessService.Create(ctx, userId, *domainTodo)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	// Convert back to proto
	response := &todov1.CreateResponse{
		Todo: DomainTodoToProto(createdTodo),
	}
	return connect.NewResponse(response), nil
}

// Get retrieves a todo by ID
func (s *TodoConnectServer) Get(
	ctx context.Context,
	req *connect.Request[todov1.GetRequest],
) (*connect.Response[todov1.GetResponse], error) {
	s.Log.Info("Connect: Get called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Get", "userId", userId)

	// Parse UUID
	todoId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid todo ID format"))
	}

	// Call business logic
	todo, err := s.BusinessService.Get(ctx, todoId)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.GetResponse{
		Todo: DomainTodoToProto(todo),
	}
	return connect.NewResponse(response), nil
}

// Update updates a todo
func (s *TodoConnectServer) Update(
	ctx context.Context,
	req *connect.Request[todov1.UpdateRequest],
) (*connect.Response[todov1.UpdateResponse], error) {
	s.Log.Info("Connect: Update called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Update", "userId", userId)

	// Parse UUID
	todoId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid todo ID format"))
	}

	// Convert proto to domain
	protoTodo := req.Msg.Todo
	if protoTodo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("todo data is required"))
	}

	domainTodo, err := ProtoTodoToDomain(protoTodo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Call business logic
	updatedTodo, err := s.BusinessService.Update(ctx, userId, todoId, *domainTodo)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.UpdateResponse{
		Todo: DomainTodoToProto(updatedTodo),
	}
	return connect.NewResponse(response), nil
}

// Delete deletes a todo
func (s *TodoConnectServer) Delete(
	ctx context.Context,
	req *connect.Request[todov1.DeleteRequest],
) (*connect.Response[todov1.DeleteResponse], error) {
	s.Log.Info("Connect: Delete called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Delete", "userId", userId)

	// Parse UUID
	todoId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid todo ID format"))
	}

	// Call business logic
	err = s.BusinessService.Delete(ctx, userId, todoId)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	return connect.NewResponse(&todov1.DeleteResponse{}), nil
}

// Search returns todos based on search criteria
func (s *TodoConnectServer) Search(
	ctx context.Context,
	req *connect.Request[todov1.SearchRequest],
) (*connect.Response[todov1.SearchResponse], error) {
	s.Log.Info("Connect: Search called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Search", "userId", userId)

	msg := req.Msg
	params := SearchParams{}
	if msg.Keywords != "" {
		params.Keywords = &msg.Keywords
	}
	if msg.Type != 0 {
		params.Type = &msg.Type
	}
	if msg.CreatedBy != 0 {
		params.CreatedBy = &msg.CreatedBy
	}
	if msg.Inactivated {
		params.Inactivated = &msg.Inactivated
	}
	if msg.Validated {
		params.Validated = &msg.Validated
	}

	limit := s.BusinessService.ListDefaultLimit
	if msg.Limit > 0 {
		limit = int(msg.Limit)
	}
	offset := 0
	if msg.Offset > 0 {
		offset = int(msg.Offset)
	}

	list, err := s.BusinessService.Search(ctx, offset, limit, params)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.SearchResponse{
		Todos: DomainTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}

// Count returns the number of todos
func (s *TodoConnectServer) Count(
	ctx context.Context,
	req *connect.Request[todov1.CountRequest],
) (*connect.Response[todov1.CountResponse], error) {
	s.Log.Info("Connect: Count called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Count", "userId", userId)

	msg := req.Msg
	params := CountParams{}
	if msg.Keywords != "" {
		params.Keywords = &msg.Keywords
	}
	if msg.Type != 0 {
		params.Type = &msg.Type
	}
	if msg.CreatedBy != 0 {
		params.CreatedBy = &msg.CreatedBy
	}
	if msg.Inactivated {
		params.Inactivated = &msg.Inactivated
	}
	if msg.Validated {
		params.Validated = &msg.Validated
	}

	count, err := s.BusinessService.Count(ctx, params)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.CountResponse{
		Count: count,
	}
	return connect.NewResponse(response), nil
}

// GeoJson returns a GeoJSON representation of todos
func (s *TodoConnectServer) GeoJson(
	ctx context.Context,
	req *connect.Request[todov1.GeoJsonRequest],
) (*connect.Response[todov1.GeoJsonResponse], error) {
	s.Log.Info("Connect: GeoJson called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("GeoJson", "userId", userId)

	msg := req.Msg
	params := GeoJsonParams{}
	if msg.Type != 0 {
		params.Type = &msg.Type
	}
	if msg.CreatedBy != 0 {
		params.CreatedBy = &msg.CreatedBy
	}
	if msg.Inactivated {
		params.Inactivated = &msg.Inactivated
	}
	if msg.Validated {
		params.Validated = &msg.Validated
	}

	limit := s.BusinessService.ListDefaultLimit
	if msg.Limit > 0 {
		limit = int(msg.Limit)
	}
	offset := 0
	if msg.Offset > 0 {
		offset = int(msg.Offset)
	}

	result, err := s.BusinessService.GeoJson(ctx, offset, limit, params)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.GeoJsonResponse{
		Result: result,
	}
	return connect.NewResponse(response), nil
}

// ListByExternalId returns todos filtered by external ID
func (s *TodoConnectServer) ListByExternalId(
	ctx context.Context,
	req *connect.Request[todov1.ListByExternalIdRequest],
) (*connect.Response[todov1.ListByExternalIdResponse], error) {
	s.Log.Info("Connect: ListByExternalId called", "externalId", req.Msg.ExternalId)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("ListByExternalId", "userId", userId)

	msg := req.Msg
	limit := s.BusinessService.ListDefaultLimit
	if msg.Limit > 0 {
		limit = int(msg.Limit)
	}
	offset := 0
	if msg.Offset > 0 {
		offset = int(msg.Offset)
	}

	list, err := s.BusinessService.ListByExternalId(ctx, offset, limit, int(msg.ExternalId))
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	// Return NotFound if no results (matching HTTP handler behavior)
	if len(list) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no todos found with this external ID"))
	}

	response := &todov1.ListByExternalIdResponse{
		Todos: DomainTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}
