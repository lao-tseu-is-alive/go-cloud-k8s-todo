package todo

import (
	"context"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/golog"
	todo_appv1 "github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo_app/v1"
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
		mockStore.On("List", mock.Anytodo_app, 0, 50, ListParams{}).Return(expectedList, nil)

		// Create request and context with user
		req := createConnectRequest(&todo_appv1.ListRequest{Limit: 0, Offset: 0})
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
		mockStore.On("List", mock.Anytodo_app, 10, 5, ListParams{}).Return(expectedList, nil)

		req := createConnectRequest(&todo_appv1.ListRequest{Limit: 5, Offset: 10})
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

		todo_appID := uuid.New()
		expectedTodo := &Todo{
			Id:   todo_appID,
			Name: "Test Todo",
		}

		mockStore.On("Exist", mock.Anytodo_app, todo_appID).Return(true)
		mockStore.On("Get", mock.Anytodo_app, todo_appID).Return(expectedTodo, nil)

		req := createConnectRequest(&todo_appv1.GetRequest{Id: todo_appID.String()})
		ctx := contextWithUser(123, false)

		resp, err := server.Get(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, todo_appID.String(), resp.Msg.Todo.Id)
		assert.Equal(t, "Test Todo", resp.Msg.Todo.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		todo_appID := uuid.New()

		mockStore.On("Exist", mock.Anytodo_app, todo_appID).Return(false)

		req := createConnectRequest(&todo_appv1.GetRequest{Id: todo_appID.String()})
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

		req := createConnectRequest(&todo_appv1.GetRequest{Id: "not-a-uuid"})
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

		todo_appID := uuid.New()
		expectedTodo := &Todo{
			Id:        todo_appID,
			Name:      "New Todo",
			CreatedBy: 123,
		}

		mockDB.On("GetQueryInt", mock.Anytodo_app, existTypeTodo, mock.Anytodo_app).Return(1, nil)
		mockStore.On("Exist", mock.Anytodo_app, mock.Anytodo_appOfType("uuid.UUID")).Return(false)
		mockStore.On("Create", mock.Anytodo_app, mock.Anytodo_appOfType("Todo")).Return(expectedTodo, nil)

		req := createConnectRequest(&todo_appv1.CreateRequest{
			Todo: &todo_appv1.Todo{
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

	t.Run("validation error - missing todo_app", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		server := createTestTodoConnectServer(mockStore, mockDB)

		req := createConnectRequest(&todo_appv1.CreateRequest{Todo: nil})
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

		todo_appID := uuid.New()
		userID := int32(123)

		mockStore.On("Exist", mock.Anytodo_app, todo_appID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo_app, todo_appID, userID).Return(true)
		mockStore.On("Delete", mock.Anytodo_app, todo_appID, userID).Return(nil)

		req := createConnectRequest(&todo_appv1.DeleteRequest{Id: todo_appID.String()})
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

		todo_appID := uuid.New()
		userID := int32(123)

		mockStore.On("Exist", mock.Anytodo_app, todo_appID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo_app, todo_appID, userID).Return(false)

		req := createConnectRequest(&todo_appv1.DeleteRequest{Id: todo_appID.String()})
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

		mockStore.On("Count", mock.Anytodo_app, CountParams{}).Return(42, nil)

		req := createConnectRequest(&todo_appv1.CountRequest{})
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

		mockStore.On("ListTypeTodo", mock.Anytodo_app, 0, 250, TypeTodoListParams{}).Return(expectedList, nil)

		req := createConnectRequest(&todo_appv1.TypeTodoListRequest{})
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

		mockStore.On("CreateTypeTodo", mock.Anytodo_app, mock.Anytodo_appOfType("TypeTodo")).Return(expectedTypeTodo, nil)

		req := createConnectRequest(&todo_appv1.TypeTodoCreateRequest{
			TypeTodo: &todo_appv1.TypeTodo{
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

		req := createConnectRequest(&todo_appv1.TypeTodoCreateRequest{
			TypeTodo: &todo_appv1.TypeTodo{
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
