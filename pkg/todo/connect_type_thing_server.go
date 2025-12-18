// Package todo provides Connect RPC handlers for the TypeTodoService.
package todo

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	todov1 "github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo/v1"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo/v1/todov1connect"
)

// TypeTodoConnectServer implements the TypeTodoServiceHandler interface.
// Authentication is handled by the AuthInterceptor, which injects user info into context.
type TypeTodoConnectServer struct {
	BusinessService *BusinessService
	Log             *slog.Logger

	// Embed the unimplemented handler for forward compatibility
	todov1connect.UnimplementedTypeTodoServiceHandler
}

// NewTypeTodoConnectServer creates a new TypeTodoConnectServer.
// Note: Authentication is handled by the AuthInterceptor, not by this server.
func NewTypeTodoConnectServer(business *BusinessService, log *slog.Logger) *TypeTodoConnectServer {
	return &TypeTodoConnectServer{
		BusinessService: business,
		Log:             log,
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

// mapErrorToConnect converts business errors to Connect errors
func (s *TypeTodoConnectServer) mapErrorToConnect(err error) *connect.Error {
	switch {
	case errors.Is(err, ErrTypeTodoNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
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
// TypeTodoService RPC Methods
// =============================================================================

// List returns a list of type todos
func (s *TypeTodoConnectServer) List(
	ctx context.Context,
	req *connect.Request[todov1.TypeTodoListRequest],
) (*connect.Response[todov1.TypeTodoListResponse], error) {
	s.Log.Info("Connect: TypeTodo.List called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("TypeTodo.List", "userId", userId)

	msg := req.Msg
	params := TypeTodoListParams{}
	if msg.Keywords != "" {
		params.Keywords = &msg.Keywords
	}
	if msg.CreatedBy != 0 {
		params.CreatedBy = &msg.CreatedBy
	}
	if msg.ExternalId != 0 {
		params.ExternalId = &msg.ExternalId
	}
	if msg.Inactivated {
		params.Inactivated = &msg.Inactivated
	}

	// Handle pagination
	limit := 250 // Default for TypeTodo as in HTTP handler
	if msg.Limit > 0 {
		limit = int(msg.Limit)
	}
	offset := 0
	if msg.Offset > 0 {
		offset = int(msg.Offset)
	}

	list, err := s.BusinessService.ListTypeTodos(ctx, offset, limit, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return empty list instead of error
			return connect.NewResponse(&todov1.TypeTodoListResponse{
				TypeTodos: []*todov1.TypeTodoList{},
			}), nil
		}
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.TypeTodoListResponse{
		TypeTodos: DomainTypeTodoListSliceToProto(list),
	}
	return connect.NewResponse(response), nil
}

// Create creates a new type todo
func (s *TypeTodoConnectServer) Create(
	ctx context.Context,
	req *connect.Request[todov1.TypeTodoCreateRequest],
) (*connect.Response[todov1.TypeTodoCreateResponse], error) {
	s.Log.Info("Connect: TypeTodo.Create called")

	// User info injected by AuthInterceptor
	userId, isAdmin := GetUserFromContext(ctx)
	s.Log.Info("TypeTodo.Create", "userId", userId, "isAdmin", isAdmin)

	protoTypeTodo := req.Msg.TypeTodo
	if protoTypeTodo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("type_todo is required"))
	}

	domainTypeTodo := ProtoTypeTodoToDomain(protoTypeTodo)

	createdTypeTodo, err := s.BusinessService.CreateTypeTodo(ctx, userId, isAdmin, *domainTypeTodo)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.TypeTodoCreateResponse{
		TypeTodo: DomainTypeTodoToProto(createdTypeTodo),
	}
	return connect.NewResponse(response), nil
}

// Get retrieves a type todo by ID
func (s *TypeTodoConnectServer) Get(
	ctx context.Context,
	req *connect.Request[todov1.TypeTodoGetRequest],
) (*connect.Response[todov1.TypeTodoGetResponse], error) {
	s.Log.Info("Connect: TypeTodo.Get called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	_, isAdmin := GetUserFromContext(ctx)

	typeTodo, err := s.BusinessService.GetTypeTodo(ctx, isAdmin, req.Msg.Id)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.TypeTodoGetResponse{
		TypeTodo: DomainTypeTodoToProto(typeTodo),
	}
	return connect.NewResponse(response), nil
}

// Update updates a type todo
func (s *TypeTodoConnectServer) Update(
	ctx context.Context,
	req *connect.Request[todov1.TypeTodoUpdateRequest],
) (*connect.Response[todov1.TypeTodoUpdateResponse], error) {
	s.Log.Info("Connect: TypeTodo.Update called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, isAdmin := GetUserFromContext(ctx)
	s.Log.Info("TypeTodo.Update", "userId", userId, "isAdmin", isAdmin)

	protoTypeTodo := req.Msg.TypeTodo
	if protoTypeTodo == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("type_todo data is required"))
	}

	domainTypeTodo := ProtoTypeTodoToDomain(protoTypeTodo)

	updatedTypeTodo, err := s.BusinessService.UpdateTypeTodo(ctx, userId, isAdmin, req.Msg.Id, *domainTypeTodo)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.TypeTodoUpdateResponse{
		TypeTodo: DomainTypeTodoToProto(updatedTypeTodo),
	}
	return connect.NewResponse(response), nil
}

// Delete deletes a type todo
func (s *TypeTodoConnectServer) Delete(
	ctx context.Context,
	req *connect.Request[todov1.TypeTodoDeleteRequest],
) (*connect.Response[todov1.TypeTodoDeleteResponse], error) {
	s.Log.Info("Connect: TypeTodo.Delete called", "id", req.Msg.Id)

	// User info injected by AuthInterceptor
	userId, isAdmin := GetUserFromContext(ctx)
	s.Log.Info("TypeTodo.Delete", "userId", userId, "isAdmin", isAdmin)

	err := s.BusinessService.DeleteTypeTodo(ctx, userId, isAdmin, req.Msg.Id)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	return connect.NewResponse(&todov1.TypeTodoDeleteResponse{}), nil
}

// Count returns the number of type todos
func (s *TypeTodoConnectServer) Count(
	ctx context.Context,
	req *connect.Request[todov1.TypeTodoCountRequest],
) (*connect.Response[todov1.TypeTodoCountResponse], error) {
	s.Log.Info("Connect: TypeTodo.Count called")

	// User info injected by AuthInterceptor
	userId, _ := GetUserFromContext(ctx)
	s.Log.Info("TypeTodo.Count", "userId", userId)

	msg := req.Msg
	params := TypeTodoCountParams{}
	if msg.Keywords != "" {
		params.Keywords = &msg.Keywords
	}
	if msg.CreatedBy != 0 {
		params.CreatedBy = &msg.CreatedBy
	}
	if msg.Inactivated {
		params.Inactivated = &msg.Inactivated
	}

	count, err := s.BusinessService.CountTypeTodos(ctx, params)
	if err != nil {
		return nil, s.mapErrorToConnect(err)
	}

	response := &todov1.TypeTodoCountResponse{
		Count: count,
	}
	return connect.NewResponse(response), nil
}
