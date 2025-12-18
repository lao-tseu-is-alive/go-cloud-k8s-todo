package todo

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/database"
)

// Storage is an interface to different implementation of persistence for Todos/TypeTodo
type Storage interface {
	// GeoJson returns a geoJson of existing todo_apps with the given offset and limit.
	GeoJson(ctx context.Context, offset, limit int, params GeoJsonParams) (string, error)
	// List returns the list of existing todo_apps with the given offset and limit.
	List(ctx context.Context, offset, limit int, params ListParams) ([]*TodoList, error)
	// ListByExternalId returns the list of existing todo_apps having the given externalId with the given offset and limit.
	ListByExternalId(ctx context.Context, offset, limit int, externalId int) ([]*TodoList, error)
	// Search returns the list of existing todo_apps filtered by search params with the given offset and limit.
	Search(ctx context.Context, offset, limit int, params SearchParams) ([]*TodoList, error)
	// Get returns the todo_app with the specified todo_apps ID.
	Get(ctx context.Context, id uuid.UUID) (*Todo, error)
	// Exist returns true only if a todo_apps with the specified id exists in store.
	Exist(ctx context.Context, id uuid.UUID) bool
	// Count returns the total number of todo_apps.
	Count(ctx context.Context, params CountParams) (int32, error)
	// Create saves a new todo_apps in the storage.
	Create(ctx context.Context, todo_app Todo) (*Todo, error)
	// Update updates the todo_apps with given ID in the storage.
	Update(ctx context.Context, id uuid.UUID, todo_app Todo) (*Todo, error)
	// Delete removes the todo_apps with given ID from the storage.
	Delete(ctx context.Context, id uuid.UUID, userId int32) error
	// IsTodoActive returns true if the todo_app with the specified id has the inactivated attribute set to false
	IsTodoActive(ctx context.Context, id uuid.UUID) bool
	// IsUserOwner returns true only if userId is the creator of the record (owner) of this todo_app in store.
	IsUserOwner(ctx context.Context, id uuid.UUID, userId int32) bool
	// CreateTypeTodo saves a new typeTodo in the storage.
	CreateTypeTodo(ctx context.Context, typeTodo TypeTodo) (*TypeTodo, error)
	// UpdateTypeTodo updates the typeTodo with given ID in the storage.
	UpdateTypeTodo(ctx context.Context, id int32, typeTodo TypeTodo) (*TypeTodo, error)
	// DeleteTypeTodo removes the typeTodo with given ID from the storage.
	DeleteTypeTodo(ctx context.Context, id int32, userId int32) error
	// ListTypeTodo returns the list of active typeTodos with the given offset and limit.
	ListTypeTodo(ctx context.Context, offset, limit int, params TypeTodoListParams) ([]*TypeTodoList, error)
	// GetTypeTodo returns the typeTodo with the specified todo_apps ID.
	GetTypeTodo(ctx context.Context, id int32) (*TypeTodo, error)
	// CountTypeTodo returns the number of TypeTodo based on search criteria
	CountTypeTodo(ctx context.Context, params TypeTodoCountParams) (int32, error)
}

func GetStorageInstanceOrPanic(ctx context.Context, dbDriver string, db database.DB, l *slog.Logger) Storage {
	var store Storage
	var err error
	switch dbDriver {
	case "pgx":
		store, err = NewPgxDB(ctx, db, l)
		if err != nil {
			l.Error("error doing NewPgxDB", "error", err)
			panic(err)
		}

	default:
		panic("unsupported DB driver type")
	}
	return store
}
