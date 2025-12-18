package todo

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/golog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of the Storage interface for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) GeoJson(ctx context.Context, offset, limit int, params GeoJsonParams) (string, error) {
	args := m.Called(ctx, offset, limit, params)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) List(ctx context.Context, offset, limit int, params ListParams) ([]*TodoList, error) {
	args := m.Called(ctx, offset, limit, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TodoList), args.Error(1)
}

func (m *MockStorage) ListByExternalId(ctx context.Context, offset, limit int, externalId int) ([]*TodoList, error) {
	args := m.Called(ctx, offset, limit, externalId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TodoList), args.Error(1)
}

func (m *MockStorage) Search(ctx context.Context, offset, limit int, params SearchParams) ([]*TodoList, error) {
	args := m.Called(ctx, offset, limit, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TodoList), args.Error(1)
}

func (m *MockStorage) Get(ctx context.Context, id uuid.UUID) (*Todo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Todo), args.Error(1)
}

func (m *MockStorage) Exist(ctx context.Context, id uuid.UUID) bool {
	args := m.Called(ctx, id)
	return args.Bool(0)
}

func (m *MockStorage) Count(ctx context.Context, params CountParams) (int32, error) {
	args := m.Called(ctx, params)
	return int32(args.Int(0)), args.Error(1)
}

func (m *MockStorage) Create(ctx context.Context, todo Todo) (*Todo, error) {
	args := m.Called(ctx, todo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Todo), args.Error(1)
}

func (m *MockStorage) Update(ctx context.Context, id uuid.UUID, todo Todo) (*Todo, error) {
	args := m.Called(ctx, id, todo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Todo), args.Error(1)
}

func (m *MockStorage) Delete(ctx context.Context, id uuid.UUID, userId int32) error {
	args := m.Called(ctx, id, userId)
	return args.Error(0)
}

func (m *MockStorage) IsTodoActive(ctx context.Context, id uuid.UUID) bool {
	args := m.Called(ctx, id)
	return args.Bool(0)
}

func (m *MockStorage) IsUserOwner(ctx context.Context, id uuid.UUID, userId int32) bool {
	args := m.Called(ctx, id, userId)
	return args.Bool(0)
}

func (m *MockStorage) CreateTypeTodo(ctx context.Context, typeTodo TypeTodo) (*TypeTodo, error) {
	args := m.Called(ctx, typeTodo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TypeTodo), args.Error(1)
}

func (m *MockStorage) UpdateTypeTodo(ctx context.Context, id int32, typeTodo TypeTodo) (*TypeTodo, error) {
	args := m.Called(ctx, id, typeTodo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TypeTodo), args.Error(1)
}

func (m *MockStorage) DeleteTypeTodo(ctx context.Context, id int32, userId int32) error {
	args := m.Called(ctx, id, userId)
	return args.Error(0)
}

func (m *MockStorage) ListTypeTodo(ctx context.Context, offset, limit int, params TypeTodoListParams) ([]*TypeTodoList, error) {
	args := m.Called(ctx, offset, limit, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TypeTodoList), args.Error(1)
}

func (m *MockStorage) GetTypeTodo(ctx context.Context, id int32) (*TypeTodo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TypeTodo), args.Error(1)
}

func (m *MockStorage) CountTypeTodo(ctx context.Context, params TypeTodoCountParams) (int32, error) {
	args := m.Called(ctx, params)
	return int32(args.Int(0)), args.Error(1)
}

// MockDB is a minimal mock for database connection
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetQueryInt(ctx context.Context, query string, args ...interface{}) (int, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Int(0), callArgs.Error(1)
}

func (m *MockDB) GetVersion(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockDB) Close() {
	m.Called()
}

func (m *MockDB) HealthCheck(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockDB) GetQueryBool(ctx context.Context, query string, args ...interface{}) (bool, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Bool(0), callArgs.Error(1)
}

func (m *MockDB) ExecActionQuery(ctx context.Context, query string, args ...interface{}) (int, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Int(0), callArgs.Error(1)
}

func (m *MockDB) DoesTableExist(ctx context.Context, schema, table string) bool {
	args := m.Called(ctx, schema, table)
	return args.Bool(0)
}

func (m *MockDB) GetPGConn() (*pgxpool.Pool, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pgxpool.Pool), args.Error(1)
}

func (m *MockDB) GetQueryString(ctx context.Context, query string, args ...interface{}) (string, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.String(0), callArgs.Error(1)
}

func (m *MockDB) Insert(ctx context.Context, query string, args ...interface{}) (int, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Int(0), callArgs.Error(1)
}

// Helper function to create a test business service
func createTestBusinessService(mockStore *MockStorage, mockDB *MockDB) *BusinessService {
	logger := golog.NewLogger("simple", os.Stdout, golog.InfoLevel, "test")
	return NewBusinessService(mockStore, mockDB, logger, 50)
}

// Test Create operation
func TestBusinessService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		newTodo := Todo{
			Id:   todoID,
			Name: "Test Todo",
		}

		expectedTodo := newTodo
		expectedTodo.CreatedBy = 123

		// Mock TypeTodo existence check
		mockDB.On("GetQueryInt", mock.Anytodo, existTypeTodo, []interface{}{newTodo.TypeId}).Return(1, nil)
		mockStore.On("Exist", mock.Anytodo, todoID).Return(false)
		mockStore.On("Create", mock.Anytodo, mock.AnytodoOfType("Todo")).Return(&expectedTodo, nil)

		result, err := service.Create(ctx, 123, newTodo)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(123), result.CreatedBy)
		mockStore.AssertExpectations(t)
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		newTodo := Todo{
			Id:   uuid.New(),
			Name: "  ", // Empty/whitespace name
		}

		result, err := service.Create(ctx, 123, newTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("validation error - name too short", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		newTodo := Todo{
			Id:   uuid.New(),
			Name: "ab", // Less than MinNameLength (5)
		}

		result, err := service.Create(ctx, 123, newTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("validation error - invalid type id", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		newTodo := Todo{
			Id:     uuid.New(),
			Name:   "Test Todo",
			TypeId: 999,
		}

		// Mock TypeTodo existence check failure
		mockDB.On("GetQueryInt", mock.Anytodo, existTypeTodo, []interface{}{newTodo.TypeId}).Return(0, nil)

		result, err := service.Create(ctx, 123, newTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrTypeTodoNotFound)
	})

	t.Run("already exists error", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		newTodo := Todo{
			Id:   todoID,
			Name: "Test Todo",
		}

		// Mock TypeTodo existence check
		mockDB.On("GetQueryInt", mock.Anytodo, existTypeTodo, []interface{}{newTodo.TypeId}).Return(1, nil)
		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)

		result, err := service.Create(ctx, 123, newTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrAlreadyExists)
		mockStore.AssertExpectations(t)
	})
}

// Test Get operation
func TestBusinessService_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		expectedTodo := &Todo{
			Id:   todoID,
			Name: "Test Todo",
		}

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("Get", mock.Anytodo, todoID).Return(expectedTodo, nil)

		result, err := service.Get(ctx, todoID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Todo", result.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("todo not found", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		mockStore.On("Exist", mock.Anytodo, todoID).Return(false)

		result, err := service.Get(ctx, todoID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrNotFound)
		mockStore.AssertExpectations(t)
	})
}

// Test Update operation
func TestBusinessService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)
		updateTodo := Todo{
			Id:   todoID,
			Name: "Updated Todo",
		}

		expectedTodo := updateTodo
		expectedTodo.LastModifiedBy = &userID

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(true)
		// Mock TypeTodo existence check
		mockDB.On("GetQueryInt", mock.Anytodo, existTypeTodo, []interface{}{updateTodo.TypeId}).Return(1, nil)
		mockStore.On("Update", mock.Anytodo, todoID, mock.AnytodoOfType("Todo")).Return(&expectedTodo, nil)

		result, err := service.Update(ctx, userID, todoID, updateTodo)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Todo", result.Name)
		mockStore.AssertExpectations(t)
	})

	t.Run("not owner error", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)
		updateTodo := Todo{
			Id:   todoID,
			Name: "Updated Todo",
		}

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(false)

		result, err := service.Update(ctx, userID, todoID, updateTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrUnauthorized)
		mockStore.AssertExpectations(t)
	})
	t.Run("validation error - invalid type id", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)
		updateTodo := Todo{
			Id:     todoID,
			Name:   "Updated Todo",
			TypeId: 999,
		}

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(true)
		// Mock TypeTodo existence check failure
		mockDB.On("GetQueryInt", mock.Anytodo, existTypeTodo, []interface{}{updateTodo.TypeId}).Return(0, nil)

		result, err := service.Update(ctx, userID, todoID, updateTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrTypeTodoNotFound)
		mockStore.AssertExpectations(t)
	})
}

// Test Delete operation
func TestBusinessService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(true)
		mockStore.On("Delete", mock.Anytodo, todoID, userID).Return(nil)

		err := service.Delete(ctx, userID, todoID)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("not owner error", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		todoID := uuid.New()
		userID := int32(123)

		mockStore.On("Exist", mock.Anytodo, todoID).Return(true)
		mockStore.On("IsUserOwner", mock.Anytodo, todoID, userID).Return(false)

		err := service.Delete(ctx, userID, todoID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUnauthorized)
		mockStore.AssertExpectations(t)
	})
}

// Test List operation
func TestBusinessService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		expectedList := []*TodoList{
			{Id: uuid.New(), Name: "Todo 1"},
			{Id: uuid.New(), Name: "Todo 2"},
		}
		params := ListParams{}

		mockStore.On("List", mock.Anytodo, 0, 10, params).Return(expectedList, nil)

		result, err := service.List(ctx, 0, 10, params)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockStore.AssertExpectations(t)
	})

	t.Run("empty list with pgx.ErrNoRows", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		params := ListParams{}
		mockStore.On("List", mock.Anytodo, 0, 10, params).Return(nil, pgx.ErrNoRows)

		result, err := service.List(ctx, 0, 10, params)

		assert.NoError(t, err)
		assert.Empty(t, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		params := ListParams{}
		dbError := errors.New("database connection failed")
		mockStore.On("List", mock.Anytodo, 0, 10, params).Return(nil, dbError)

		result, err := service.List(ctx, 0, 10, params)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockStore.AssertExpectations(t)
	})
}

// Test Count operation
func TestBusinessService_Count(t *testing.T) {
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		params := CountParams{}
		mockStore.On("Count", mock.Anytodo, params).Return(42, nil)

		result, err := service.Count(ctx, params)

		assert.NoError(t, err)
		assert.Equal(t, int32(42), result)
		mockStore.AssertExpectations(t)
	})
}

// Test CreateTypeTodo operation
func TestBusinessService_CreateTypeTodo(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation by admin", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		newTypeTodo := TypeTodo{
			Name: "Test Type",
		}

		expectedTypeTodo := newTypeTodo
		expectedTypeTodo.Id = 1
		expectedTypeTodo.CreatedBy = 123

		mockStore.On("CreateTypeTodo", mock.Anytodo, mock.AnytodoOfType("TypeTodo")).Return(&expectedTypeTodo, nil)

		result, err := service.CreateTypeTodo(ctx, 123, true, newTypeTodo)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(123), result.CreatedBy)
		mockStore.AssertExpectations(t)
	})

	t.Run("non-admin rejection", func(t *testing.T) {
		mockStore := new(MockStorage)
		mockDB := new(MockDB)
		service := createTestBusinessService(mockStore, mockDB)

		newTypeTodo := TypeTodo{
			Name: "Test Type",
		}

		result, err := service.CreateTypeTodo(ctx, 123, false, newTypeTodo)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrAdminRequired)
	})
}

// Test validation function
func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid name", "Valid Name", false},
		{"empty string", "", true},
		{"only spaces", "   ", true},
		{"too short", "ab", true},
		{"exactly min length", "12345", false},
		{"longer than min", "Long Enough Name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
