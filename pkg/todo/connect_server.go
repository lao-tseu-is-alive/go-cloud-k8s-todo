// Package todo_app provides Connect RPC handlers for the TodoService.
package todo

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	todo_appv1 "github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo_app/v1"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo_app/v1/todo_appv1connect"
)

// TodoConnectServer implements the TodoServiceHandler interface.
// Authentication is handled by the AuthInterceptor, which injects user info into context.
type TodoConnectServer struct {
	BusinessService *BusinessService
	Log             *slog.Logger

	// Embed the unimplemented handler for forward compatibility
	todo_appv1connect.UnimplementedTodoServiceHandler
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

// List returns a list of todo_apps
func (s *TodoConnectServer) List(
	ctx context.Context,
	req *connect.Request[todo_appv1.ListRequest],
) (*connect.Response[todo_appv1.ListResponse], error) {
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
	response := &todo_appv1.ListResponse{
		Todos: DomainTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}

// Create creates a new todo_app
func (s *TodoConnectServer) Create(
	ctx context.Context,
	req *connect.Request[todo_appv1.CreateRequest],
) (*connect.Response[todo_appv1.CreateResponse], error) {
	s.Log.Info("Connect: Create called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Create", "userId", userId)

	// Convert proto to domain
	protoTodo := req.Msg.Todo
	if protoTodo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("todo_app is required"))
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
	response := &todo_appv1.CreateResponse{
		Todo: DomainTodoToProto(createdTodo),
	}
	return connect.NewResponse(response), nil
}

// Get retrieves a todo_app by ID
func (s *TodoConnectServer) Get(
	ctx context.Context,
	req *connect.Request[todo_appv1.GetRequest],
) (*connect.Response[todo_appv1.GetResponse], error) {
	s.Log.Info("Connect: Get called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Get", "userId", userId)

	// Parse UUID
	todo_appId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid todo_app ID format"))
	}

	// Call business logic
	todo_app, err := s.BusinessService.Get(ctx, todo_appId)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todo_appv1.GetResponse{
		Todo: DomainTodoToProto(todo_app),
	}
	return connect.NewResponse(response), nil
}

// Update updates a todo_app
func (s *TodoConnectServer) Update(
	ctx context.Context,
	req *connect.Request[todo_appv1.UpdateRequest],
) (*connect.Response[todo_appv1.UpdateResponse], error) {
	s.Log.Info("Connect: Update called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Update", "userId", userId)

	// Parse UUID
	todo_appId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid todo_app ID format"))
	}

	// Convert proto to domain
	protoTodo := req.Msg.Todo
	if protoTodo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("todo_app data is required"))
	}

	domainTodo, err := ProtoTodoToDomain(protoTodo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Call business logic
	updatedTodo, err := s.BusinessService.Update(ctx, userId, todo_appId, *domainTodo)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todo_appv1.UpdateResponse{
		Todo: DomainTodoToProto(updatedTodo),
	}
	return connect.NewResponse(response), nil
}

// Delete deletes a todo_app
func (s *TodoConnectServer) Delete(
	ctx context.Context,
	req *connect.Request[todo_appv1.DeleteRequest],
) (*connect.Response[todo_appv1.DeleteResponse], error) {
	s.Log.Info("Connect: Delete called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("Delete", "userId", userId)

	// Parse UUID
	todo_appId, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid todo_app ID format"))
	}

	// Call business logic
	err = s.BusinessService.Delete(ctx, userId, todo_appId)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	return connect.NewResponse(&todo_appv1.DeleteResponse{}), nil
}

// Search returns todo_apps based on search criteria
func (s *TodoConnectServer) Search(
	ctx context.Context,
	req *connect.Request[todo_appv1.SearchRequest],
) (*connect.Response[todo_appv1.SearchResponse], error) {
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

	response := &todo_appv1.SearchResponse{
		Todos: DomainTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}

// Count returns the number of todo_apps
func (s *TodoConnectServer) Count(
	ctx context.Context,
	req *connect.Request[todo_appv1.CountRequest],
) (*connect.Response[todo_appv1.CountResponse], error) {
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

	response := &todo_appv1.CountResponse{
		Count: count,
	}
	return connect.NewResponse(response), nil
}

// GeoJson returns a GeoJSON representation of todo_apps
func (s *TodoConnectServer) GeoJson(
	ctx context.Context,
	req *connect.Request[todo_appv1.GeoJsonRequest],
) (*connect.Response[todo_appv1.GeoJsonResponse], error) {
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

	response := &todo_appv1.GeoJsonResponse{
		Result: result,
	}
	return connect.NewResponse(response), nil
}

// ListByExternalId returns todo_apps filtered by external ID
func (s *TodoConnectServer) ListByExternalId(
	ctx context.Context,
	req *connect.Request[todo_appv1.ListByExternalIdRequest],
) (*connect.Response[todo_appv1.ListByExternalIdResponse], error) {
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
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no todo_apps found with this external ID"))
	}

	response := &todo_appv1.ListByExternalIdResponse{
		Todos: DomainTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}
