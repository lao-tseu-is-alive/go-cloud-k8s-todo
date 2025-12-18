package todo

import (
	"context"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/golog"
	todov1 "github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Test Helpers
// =============================================================================

// Helper to create a test Connect server
func createTestTodoConnectServer(mockStore *MockStorage, mockDB *MockDB) *TodoConnectServer {
	logger := golog.NewLogger("simple", os.Stdout, golog.InfoLevel, "test")
	businessService := NewBusinessService(mockStore, mockDB, logger, 50)
	return NewTodoConnectServer(businessService, logger)
}

// Helper to create a test TypeTodo Connect server
func createTestTypeTodoConnectServer(mockStore *MockStorage, mockDB *MockDB) *TypeTodoConnectServer {
	logger := golog.NewLogger("simple", os.Stdout, golog.InfoLevel, "test")
	businessService := NewBusinessService(mockStore, mockDB, logger, 50)
	return NewTypeTodoConnectServer(businessService, logger)
}

// Helper to create a context with user info (simulating what AuthInterceptor does)
func contextWithUser(userId int32, isAdmin bool) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, userIDKey, userId)
	ctx = context.WithValue(ctx, isAdminKey, isAdmin)
	return ctx
}

// Helper to create a Connect request (no auth header needed since we inject via context)
func createConnectRequest[T any](msg *T) *connect.Request[T] {
	return connect.NewRequest(msg)
}

// =============================================================================
// TodoConnectServer Tests
// =============================================================================

func TestTodoConnectServer_List(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		// Setup mock storage
		now := time.Now()
		expectedList := []*TodoList{
			{Id: uuid.New(), Name: "Todo 1", CreatedAt: &now},
			{Id: uuid.New(), Name: "Todo 2", CreatedAt: &now},
		}
		mockStore.On("List", mock.Anytodo, 0, 50, ListParams{}).Return(expectedList, nil)

		// Create request and context with user
		req := createConnectRequest(&todov1.ListRequest{Limit: 0, Offset: 0})
		ctx := contextWithUser(123, false)

		// Call handler
		resp, err := server.List(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Msg.Todos, 2)
		assert.Equal(t, "Todo 1", resp.Msg.Todos[0].Name)
		assert.Equal(t, "Todo 2", resp.Msg.Todos[1].Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("list with pagination", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		now := time.Now()
		expectedList := []*TodoList{
			{Id: uuid.New(), Name: "Todo 3", CreatedAt: &now},
		}
		mockStore.On("List", mock.Anytodo, 10, 5, ListParams{}).Return(expectedList, nil)

		req := createConnectRequest(&todov1.ListRequest{Limit: 5, Offset: 10})
		ctx := contextWithUser(123, false)

		resp, err := server.List(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Msg.Todos, 1)
		mockStore.AssertExpectations(t)
	})
}

func TestTodoConnectServer_Get(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		todoID := uuid.New()
		expectedTodo := &Todo{
			Id:   todoID,
			Name: "Test Todo",
		}

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("Get", mock.Anytodo, todoID).Return(expectedTodo, nil)

		req := createConnectRequest(&todov1.GetRequest{Id: todoID.String()})
		ctx := contextWithUser(123, false)

		resp, err := server.Get(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, todoID.String(), resp.Msg.Todo.Id)
		assert.Equal(t, "Test Todo", resp.Msg.Todo.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		todoID := uuid.New()

		mockStore.On("Exist", mock.Anytodo, todoID).Return(false)

		req := createConnectRequest(&todov1.GetRequest{Id: todoID.String()})
		ctx := contextWithUser(123, false)

		resp, err := server.Get(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		connectErr, ok := err.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeNotFound, connectErr.Code())
		mockStore.AssertExpectations(t)
	})

	t.Run("invalid UUID format", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		req := createConnectRequest(&todov1.GetRequest{Id: "not-a-uuid"})
		ctx := contextWithUser(123, false)

		resp, err := server.Get(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		connectErr, ok := err.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	})
}

func TestTodoConnectServer_Create(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		todoID := uuid.New()
		expectedTodo := &Todo{
			Id:        todoID,
			Name:      "New Todo",
			CreatedBy: 123,
		}

		mockDB.On("GetQueryInt", mock.Anytodo, existTypeTodo, mock.Anytodo).Return(1, nil)
		mockStore.On("Exist", mock.Anytodo, mock.AnytodoOfType("uuid.UUID")).Return(false)
		mockStore.On("Create", mock.Anytodo, mock.AnytodoOfType("Todo")).Return(expectedTodo, nil)

		req := createConnectRequest(&todov1.CreateRequest{
			Todo: &todov1.Todo{
				Name: "New Todo",
			},
		})
		ctx := contextWithUser(123, false)

		resp, err := server.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "New Todo", resp.Msg.Todo.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("validation error - missing todo", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		req := createConnectRequest(&todov1.CreateRequest{Todo: nil})
		ctx := contextWithUser(123, false)

		resp, err := server.Create(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		connectErr, ok := err.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	})
}

func TestTodoConnectServer_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(true)
		mockStore.On("Delete", mock.Anytodo, todoID, userID).Return(nil)

		req := createConnectRequest(&todov1.DeleteRequest{Id: todoID.String()})
		ctx := contextWithUser(userID, false)

		resp, err := server.Delete(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockStore.AssertExpectations(t)
	})

	t.Run("permission denied - not owner", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(false)

		req := createConnectRequest(&todov1.DeleteRequest{Id: todoID.String()})
		ctx := contextWithUser(userID, false)

		resp, err := server.Delete(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		connectErr, ok := err.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
		mockStore.AssertExpectations(t)
	})
}

func TestTodoConnectServer_Count(t *testing.T) {
	t.Run("successful count", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		mockStore.On("Count", mock.Anytodo, CountParams{}).Return(42, nil)

		req := createConnectRequest(&todov1.CountRequest{})
		ctx := contextWithUser(123, false)

		resp, err := server.Count(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(42), resp.Msg.Count)
		mockStore.AssertExpectations(t)
	})
}

// =============================================================================
// TypeTodoConnectServer Tests
// =============================================================================

func TestTypeTodoConnectServer_List(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTypeTodoConnectServer(mockStore, mockDB)

		now := time.Now()
		expectedList := []*TypeTodoList{
			{Id: 1, Name: "Type 1", CreatedAt: now},
			{Id: 2, Name: "Type 2", CreatedAt: now},
		}

		mockStore.On("ListTypeTodo", mock.Anytodo, 0, 250, TypeTodoListParams{}).Return(expectedList, nil)

		req := createConnectRequest(&todov1.TypeTodoListRequest{})
		ctx := contextWithUser(123, false)

		resp, err := server.List(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Msg.TypeTodos, 2)
		mockStore.AssertExpectations(t)
	})
}

func TestTypeTodoConnectServer_Create(t *testing.T) {
	t.Run("admin can create", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTypeTodoConnectServer(mockStore, mockDB)

		expectedTypeTodo := &TypeTodo{
			Id:        1,
			Name:      "New Type",
			CreatedBy: 123,
		}

		mockStore.On("CreateTypeTodo", mock.Anytodo, mock.AnytodoOfType("TypeTodo")).Return(expectedTypeTodo, nil)

		req := createConnectRequest(&todov1.TypeTodoCreateRequest{
			TypeTodo: &todov1.TypeTodo{
				Name: "New Type",
			},
		})
		ctx := contextWithUser(123, true) // isAdmin = true

		resp, err := server.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "New Type", resp.Msg.TypeTodo.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("non-admin rejected", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTypeTodoConnectServer(mockStore, mockDB)

		req := createConnectRequest(&todov1.TypeTodoCreateRequest{
			TypeTodo: &todov1.TypeTodo{
				Name: "New Type",
			},
		})
		ctx := contextWithUser(123, false) // isAdmin = false

		resp, err := server.Create(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		connectErr, ok := err.(*connect.Error)
		assert.True(t, ok)
		assert.Equal(t, connect.CodePermissionDenied, connectErr.Code())
	})
}

// =============================================================================
// AuthInterceptor Tests
// =============================================================================

func TestGetUserFromContext(t *testing.T) {
	t.Run("user present in context", func(t *testing.T) {
		ctx := contextWithUser(456, true)

		userId, isAdmin := GetUserFromContext(ctx)

		assert.Equal(t, int32(456), userId)
		assert.True(t, isAdmin)
	})

	t.Run("user not present in context", func(t *testing.T) {
		ctx := context.Background()

		userId, isAdmin := GetUserFromContext(ctx)

		assert.Equal(t, int32(0), userId)
		assert.False(t, isAdmin)
	})
}
