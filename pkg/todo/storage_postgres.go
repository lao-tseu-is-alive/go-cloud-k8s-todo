package todo

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/database"
)

type PGX struct {
	Conn *pgxpool.Pool
	dbi  database.DB
	log  *slog.Logger
}

// NewPgxDB will instantiate a new storage of type postgres and ensure schema exist
func NewPgxDB(ctx context.Context, db database.DB, log *slog.Logger) (Storage, error) {
	var psql PGX
	pgConn, err := db.GetPGConn()
	if err != nil {
		return nil, err
	}
	psql.Conn = pgConn
	psql.dbi = db
	psql.log = log
	var numberOfTypeTodos int
	errTypeTodoTable := pgConn.QueryRow(ctx, typeTodoCount).Scan(&numberOfTypeTodos)
	if errTypeTodoTable != nil {
		log.Error("Unable to retrieve the number of typeTodo", "error", err)
		return nil, errTypeTodoTable
	}

	if numberOfTypeTodos > 0 {
		log.Info("database contains records in todo.type_todo_app", "count", numberOfTypeTodos)
	} else {
		log.Warn("todo.type_todo_app is empty - it should contain at least one row")
		return nil, fmt.Errorf("«todo.type_todo_app» contains %w should not be empty", numberOfTypeTodos)
	}

	return &psql, err
}

func (db *PGX) GeoJson(ctx context.Context, offset, limit int, params GeoJsonParams) (string, error) {
	db.log.Debug("trace: entering GeoJson", "offset", offset, "limit", limit)
	if params.Type != nil {
		db.log.Info("param type", "type", *params.Type)
	}
	if params.CreatedBy != nil {
		db.log.Info("params.CreatedBy", "createdBy", *params.CreatedBy)
	}
	var (
		mayBeResultIsNull *string
		err               error
	)
	isInactive := false
	if params.Inactivated != nil {
		isInactive = *params.Inactivated
	}
	listTodos := baseGeoJsonTodoSearch + listTodosConditions
	if params.Validated != nil {
		db.log.Debug("params.Validated is not nil ")
		isValidated := *params.Validated
		listTodos += " AND validated = coalesce($6, validated) " + geoJsonListEndOfQuery
		err = pgxscan.Select(ctx, db.Conn, &mayBeResultIsNull, listTodos,
			limit, offset, &params.Type, &params.CreatedBy, isInactive, isValidated)
	} else {
		listTodos += geoJsonListEndOfQuery
		err = pgxscan.Select(ctx, db.Conn, &mayBeResultIsNull, listTodos,
			limit, offset, &params.Type, &params.CreatedBy, isInactive)
	}
	if err != nil {
		db.log.Error(SelectFailedInNWithErrorE, "List", err)
		return "", err
	}
	if mayBeResultIsNull == nil {
		db.log.Info("List returned no results")
		return "", pgx.ErrNoRows
	}
	return *mayBeResultIsNull, nil
}

// List returns the list of existing todo_apps with the given offset and limit.
func (db *PGX) List(ctx context.Context, offset, limit int, params ListParams) ([]*TodoList, error) {
	db.log.Debug("trace: entering List", "offset", offset, "limit", limit)
	if params.Type != nil {
		db.log.Info("param type", "type", *params.Type)
	}
	if params.CreatedBy != nil {
		db.log.Info("params.CreatedBy", "createdBy", *params.CreatedBy)
	}
	var (
		res []*TodoList
		err error
	)
	isInactive := false
	if params.Inactivated != nil {
		isInactive = *params.Inactivated
	}
	listTodos := baseTodoListQuery + listTodosConditions
	if params.Validated != nil {
		db.log.Debug("params.Validated is not nil ")
		isValidated := *params.Validated
		listTodos += " AND validated = coalesce($6, validated) " + todo_appListOrderBy
		err = pgxscan.Select(ctx, db.Conn, &res, listTodos,
			limit, offset, &params.Type, &params.CreatedBy, isInactive, isValidated)
	} else {
		listTodos += todo_appListOrderBy
		err = pgxscan.Select(ctx, db.Conn, &res, listTodos,
			limit, offset, &params.Type, &params.CreatedBy, isInactive)
	}
	if err != nil {
		db.log.Error(SelectFailedInNWithErrorE, "List", err)
		return nil, err
	}
	if res == nil {
		db.log.Info("List returned no results")
		return nil, pgx.ErrNoRows
	}
	return res, nil
}

// ListByExternalId returns the list of existing todo_apps having given externalId with the given offset and limit.
func (db *PGX) ListByExternalId(ctx context.Context, offset, limit int, externalId int) ([]*TodoList, error) {
	db.log.Debug("trace: entering ListByExternalId", "externalId", externalId)
	var res []*TodoList
	listByExternalIdTodos := baseTodoListQuery + listByExternalIdTodosCondition + todo_appListOrderBy
	err := pgxscan.Select(ctx, db.Conn, &res, listByExternalIdTodos, limit, offset, externalId)
	if err != nil {
		db.log.Error("ListByExternalId failed", "error", err)
		return nil, err
	}
	if res == nil {
		db.log.Info("ListByExternalId returned no results")
		return nil, pgx.ErrNoRows
	}
	return res, nil
}

func (db *PGX) Search(ctx context.Context, offset, limit int, params SearchParams) ([]*TodoList, error) {
	db.log.Debug("trace: entering Search", "offset", offset, "limit", limit)
	var (
		res []*TodoList
		err error
	)
	searchTodos := baseTodoListQuery + listTodosConditions
	if params.Keywords != nil {
		searchTodos += " AND text_search @@ plainto_tsquery('french', unaccent($6))"
		if params.Validated != nil {
			searchTodos += " AND validated = coalesce($7, validated) " + todo_appListOrderBy
			err = pgxscan.Select(ctx, db.Conn, &res, searchTodos,
				limit, offset, &params.Type, &params.CreatedBy, &params.Inactivated, &params.Keywords, &params.Validated)
		} else {
			searchTodos += todo_appListOrderBy
			err = pgxscan.Select(ctx, db.Conn, &res, searchTodos,
				limit, offset, &params.Type, &params.CreatedBy, &params.Inactivated, &params.Keywords)
		}
	} else {
		if params.Validated != nil {
			searchTodos += " AND validated = coalesce($6, validated) " + todo_appListOrderBy
			err = pgxscan.Select(ctx, db.Conn, &res, searchTodos,
				limit, offset, &params.Type, &params.CreatedBy, &params.Inactivated, &params.Validated)
		} else {
			searchTodos += todo_appListOrderBy
			err = pgxscan.Select(ctx, db.Conn, &res, searchTodos,
				limit, offset, &params.Type, &params.CreatedBy, &params.Inactivated)
		}
	}

	if err != nil {
		db.log.Error("Search failed", "error", err)
		return nil, err
	}
	if res == nil {
		db.log.Info("Search returned no results")
		return nil, pgx.ErrNoRows
	}
	return res, nil
}

// Get will retrieve the todo_app with given id
func (db *PGX) Get(ctx context.Context, id uuid.UUID) (*Todo, error) {
	db.log.Debug("trace: entering Get", "id", id)
	res := &Todo{}
	err := pgxscan.Get(ctx, db.Conn, res, getTodo, id)
	if err != nil {
		db.log.Error("Get failed", "error", err)
		return nil, err
	}
	if res == nil {
		db.log.Info("Get returned no results")
		return nil, pgx.ErrNoRows
	}
	return res, nil
}

// Exist returns true only if a todo_app with the specified id exists in store.
func (db *PGX) Exist(ctx context.Context, id uuid.UUID) bool {
	db.log.Debug("trace: entering Exist", "id", id)
	count, err := db.dbi.GetQueryInt(ctx, existTodo, id)
	if err != nil {
		db.log.Error("Exist could not be retrieved from DB", "id", id, "error", err)
		return false
	}
	if count > 0 {
		db.log.Info("Exist: id does exist", "id", id, "count", count)
		return true
	} else {
		db.log.Info("Exist: id does not exist", "id", id, "count", count)
		return false
	}
}

// Count returns the number of todo_app stored in DB
func (db *PGX) Count(ctx context.Context, params CountParams) (int32, error) {
	db.log.Debug("trace : entering Count()")
	var (
		count int
		err   error
	)
	queryCount := countTodo + " WHERE _deleted = false AND position IS NOT NULL "
	withoutSearchParameters := true
	if params.Keywords != nil {
		withoutSearchParameters = false
		queryCount += `AND text_search @@ plainto_tsquery('french', unaccent($1))
		AND type_id = coalesce($2, type_id)
		AND _created_by = coalesce($3, _created_by)
		AND inactivated = coalesce($4, inactivated)
`
		if params.Validated != nil {
			db.log.Debug("params.Validated is not nil ")
			isValidated := *params.Validated
			queryCount += " AND validated = coalesce($4, validated) "
			count, err = db.dbi.GetQueryInt(ctx, queryCount, &params.Keywords, &params.Type, &params.CreatedBy, &params.Inactivated, isValidated)

		} else {
			count, err = db.dbi.GetQueryInt(ctx, queryCount, &params.Keywords, &params.Type, &params.CreatedBy, &params.Inactivated)
		}
	}
	if withoutSearchParameters {
		queryCount += `
		AND type_id = coalesce($1, type_id)
		AND _created_by = coalesce($2, _created_by)
		AND inactivated = coalesce($3, inactivated)
`
		if params.Validated != nil {
			db.log.Debug("params.Validated is not nil ")
			isValidated := *params.Validated
			queryCount += " AND validated = coalesce($4, validated) "
			count, err = db.dbi.GetQueryInt(ctx, queryCount, &params.Type, &params.CreatedBy, &params.Inactivated, isValidated)

		} else {
			count, err = db.dbi.GetQueryInt(ctx, queryCount, &params.Type, &params.CreatedBy, &params.Inactivated)
		}

	}

	if err != nil {
		db.log.Error("Count failed", "error", err)
		return 0, err
	}
	return int32(count), nil
}

// Create will store the new Todo in the database
func (db *PGX) Create(ctx context.Context, t Todo) (*Todo, error) {
	db.log.Debug("trace: entering Create", "name", t.Name, "id", t.Id)

	rowsAffected, err := db.dbi.ExecActionQuery(ctx, createTodo,
		t.Id, t.TypeId, t.Name, &t.Description, &t.Comment, &t.ExternalId, &t.ExternalRef, //$7
		&t.BuildAt, &t.Status, &t.ContainedBy, &t.ContainedByOld, t.Validated, &t.ValidatedTime, &t.ValidatedBy, //$14
		&t.ManagedBy, t.CreatedBy, &t.MoreData, t.PosX, t.PosY)
	if err != nil {
		db.log.Error("Create unexpectedly failed", "name", t.Name, "error", err)
		return nil, err
	}
	if rowsAffected < 1 {
		db.log.Error("Create no row was created", "name", t.Name)
		return nil, err
	}
	db.log.Info("Create success", "name", t.Name, "id", t.Id)

	// if we get to here all is good, so let's retrieve a fresh copy to send it back
	createdTodo, err := db.Get(ctx, t.Id)
	if err != nil {
		return nil, fmt.Errorf("error %w: todo_app was created, but can not be retrieved", err)
	}
	return createdTodo, nil
}

// Update the todo_app stored in DB with given id and other information in struct
func (db *PGX) Update(ctx context.Context, id uuid.UUID, t Todo) (*Todo, error) {
	db.log.Debug("trace: entering Update", "id", t.Id)

	rowsAffected, err := db.dbi.ExecActionQuery(ctx, updateTodo,
		t.Id, t.TypeId, t.Name, &t.Description, &t.Comment, &t.ExternalId, &t.ExternalRef, //$7
		&t.BuildAt, &t.Status, &t.ContainedBy, &t.ContainedByOld, t.Inactivated, &t.InactivatedTime, &t.InactivatedBy, &t.InactivatedReason, //$15
		t.Validated, &t.ValidatedTime, &t.ValidatedBy, //$18
		&t.ManagedBy, &t.LastModifiedBy, &t.MoreData, t.PosX, t.PosY) //$23
	if err != nil {

		db.log.Error("Update unexpectedly failed", "id", t.Id, "error", err)
		return nil, err
	}
	if rowsAffected < 1 {
		db.log.Error("Update no row was updated", "id", t.Id)
		return nil, err
	}

	// if we get to here all is good, so let's retrieve a fresh copy to send it back
	updatedTodo, err := db.Get(ctx, t.Id)
	if err != nil {
		return nil, fmt.Errorf("error %w: todo_app was updated, but can not be retrieved", err)
	}
	return updatedTodo, nil
}

// Delete the todo_app stored in DB with given id
func (db *PGX) Delete(ctx context.Context, id uuid.UUID, userId int32) error {
	db.log.Debug("trace: entering Delete", "id", id)
	rowsAffected, err := db.dbi.ExecActionQuery(ctx, deleteTodo, userId, id)
	if err != nil {
		db.log.Error("todo_app could not be deleted", "id", id, "error", err)
		return fmt.Errorf("todo_app could not be deleted: %w", err)
	}
	if rowsAffected < 1 {
		db.log.Error("todo_app was not deleted", "id", id)
		return fmt.Errorf("todo_app was not marked for deletetion")
	}
	return nil
}

// IsTodoActive returns true if the todo_app with the specified id has the inactivated attribute set to false
func (db *PGX) IsTodoActive(ctx context.Context, id uuid.UUID) bool {
	db.log.Debug("trace: entering IsTodoActive", "id", id)
	count, err := db.dbi.GetQueryInt(ctx, isActiveTodo, id)
	if err != nil {
		db.log.Error("IsTodoActive could not be retrieved from DB", "id", id, "error", err)
		return false
	}
	if count > 0 {
		db.log.Info("IsTodoActive is true", "id", id, "count", count)
		return true
	} else {
		db.log.Info("IsTodoActive is false", "id", id, "count", count)
		return false
	}
}

// IsUserOwner returns true only if userId is the creator of the record (owner) of this todo_app in store.
func (db *PGX) IsUserOwner(ctx context.Context, id uuid.UUID, userId int32) bool {
	db.log.Debug("trace: entering IsUserOwner", "id", id, "userId", userId)
	count, err := db.dbi.GetQueryInt(ctx, existTodoOwnedBy, id, userId)
	if err != nil {
		db.log.Error("IsUserOwner could not be retrieved from DB", "id", id, "userId", userId, "error", err)
		return false
	}
	if count > 0 {
		db.log.Info("IsUserOwner is true", "id", id, "userId", userId, "count", count)
		return true
	} else {
		db.log.Info("IsUserOwner is false", "id", id, "userId", userId, "count", count)
		return false
	}
}

// CreateTypeTodo will store the new TypeTodo in the database
func (db *PGX) CreateTypeTodo(ctx context.Context, tt TypeTodo) (*TypeTodo, error) {
	db.log.Debug("trace: entering CreateTypeTodo", "name", tt.Name, "createdBy", tt.CreatedBy)
	var lastInsertId int = 0
	err := db.Conn.QueryRow(ctx, createTypeTodo,
		tt.Name, &tt.Description, &tt.Comment, &tt.ExternalId, &tt.TableName, &tt.GeometryType, //$6
		&tt.ManagedBy, tt.IconPath, tt.CreatedBy, &tt.MoreDataSchema).Scan(&lastInsertId)
	if err != nil {
		db.log.Error("CreateTypeTodo unexpectedly failed", "name", tt.Name, "error", err)
		return nil, err
	}
	db.log.Info("CreateTypeTodo success", "name", tt.Name, "id", lastInsertId)

	// if we get to here all is good, so let's retrieve a fresh copy to send it back
	createdTypeTodo, err := db.GetTypeTodo(ctx, int32(lastInsertId))
	if err != nil {
		return nil, fmt.Errorf("error %w: typeTodo was created, but can not be retrieved", err)
	}
	return createdTypeTodo, nil
}

// UpdateTypeTodo updates the TypeTodo stored in DB with given id and other information in struct
func (db *PGX) UpdateTypeTodo(ctx context.Context, id int32, tt TypeTodo) (*TypeTodo, error) {
	db.log.Debug("trace: entering UpdateTypeTodo", "id", id)

	rowsAffected, err := db.dbi.ExecActionQuery(ctx, updateTypeTing,
		id, tt.Name, &tt.Description, &tt.Comment, &tt.ExternalId, &tt.TableName, //$6
		&tt.GeometryType, tt.Inactivated, &tt.InactivatedTime, &tt.InactivatedBy, &tt.InactivatedReason, //$11
		&tt.ManagedBy, tt.IconPath, &tt.LastModifiedBy, &tt.MoreDataSchema) //$14
	if err != nil {

		db.log.Error("UpdateTypeTodo unexpectedly failed", "id", id, "error", err)
		return nil, err
	}
	if rowsAffected < 1 {
		db.log.Error("UpdateTypeTodo no row was updated", "id", id)
		return nil, err
	}

	// if we get to here all is good, so let's retrieve a fresh copy to send it back
	updatedTypeTodo, err := db.GetTypeTodo(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error %w: todo_app was updated, but can not be retrieved", err)
	}
	return updatedTypeTodo, nil
}

// DeleteTypeTodo deletes the TypeTodo stored in DB with given id
func (db *PGX) DeleteTypeTodo(ctx context.Context, id int32, userId int32) error {
	db.log.Debug("trace: entering DeleteTypeTodo", "id", id)
	rowsAffected, err := db.dbi.ExecActionQuery(ctx, deleteTypeTodo, userId, id)
	if err != nil {
		db.log.Error("typetodo_app could not be deleted", "id", id, "error", err)
		return fmt.Errorf("typetodo_app could not be deleted: %w", err)
	}
	if rowsAffected < 1 {
		db.log.Error("typetodo_app was not deleted", "id", id)
		return fmt.Errorf("typetodo_app was not marked for deletion")
	}
	return nil
}

// ListTypeTodo returns the list of existing TypeTodo with the given offset and limit.
func (db *PGX) ListTypeTodo(ctx context.Context, offset, limit int, params TypeTodoListParams) ([]*TypeTodoList, error) {
	db.log.Debug("trace : entering ListTypeTodo")
	var (
		res []*TypeTodoList
		err error
	)
	listTypeTodos := typeTodoListQuery
	if params.Keywords != nil {
		listTypeTodos += listTypeTodosConditionsWithKeywords + typeTodoListOrderBy
		err = pgxscan.Select(ctx, db.Conn, &res, listTypeTodos,
			limit, offset, &params.Keywords, &params.CreatedBy, &params.ExternalId, &params.Inactivated)
	} else {
		listTypeTodos += listTypeTodosConditionsWithoutKeywords + typeTodoListOrderBy
		err = pgxscan.Select(ctx, db.Conn, &res, listTypeTodos,
			limit, offset, &params.CreatedBy, &params.ExternalId, &params.Inactivated)
	}

	if err != nil {
		db.log.Error("ListTypeTodo failed", "error", err)
		return nil, err
	}
	if res == nil {
		db.log.Info("ListTypeTodo returned no results")
		return nil, pgx.ErrNoRows
	}
	return res, nil
}

// GetTypeTodo will retrieve the TypeTodo with given id
func (db *PGX) GetTypeTodo(ctx context.Context, id int32) (*TypeTodo, error) {
	db.log.Debug("trace: entering GetTypeTodo", "id", id)
	res := &TypeTodo{}
	err := pgxscan.Get(ctx, db.Conn, res, getTypeTodo, id)
	if err != nil {
		db.log.Error("GetTypeTodo failed", "error", err)
		return nil, err
	}
	if res == nil {
		db.log.Info("GetTypeTodo returned no results", "id", id)
		return nil, pgx.ErrNoRows
	}
	return res, nil
}

// CountTypeTodo returns the number of TypeTodo based on search criteria
func (db *PGX) CountTypeTodo(ctx context.Context, params TypeTodoCountParams) (int32, error) {
	db.log.Debug("trace : entering CountTypeTodo()")
	var (
		count int
		err   error
	)
	queryCount := countTypeTodo + " WHERE 1 = 1 "
	withoutSearchParameters := true
	if params.Keywords != nil {
		withoutSearchParameters = false
		queryCount += `AND text_search @@ plainto_tsquery('french', unaccent($1))
		AND _created_by = coalesce($2, _created_by)
		AND inactivated = coalesce($3, inactivated)
`
		count, err = db.dbi.GetQueryInt(ctx, queryCount, &params.Keywords, &params.CreatedBy, &params.Inactivated)
	}
	if withoutSearchParameters {
		queryCount += `
		AND _created_by = coalesce($1, _created_by)
		AND inactivated = coalesce($2, inactivated)
`
		count, err = db.dbi.GetQueryInt(ctx, queryCount, &params.CreatedBy, &params.Inactivated)

	}
	if err != nil {
		db.log.Error("CountTypeTodo failed", "error", err)
		return 0, err
	}
	return int32(count), nil
}

// GetTypeTodoMaxId will retrieve maximum value of TypeTodo id existing in store.
func (db *PGX) GetTypeTodoMaxId(ctx context.Context) (int32, error) {
	db.log.Debug("trace : entering GetTypeTodoMaxId")
	existingMaxId, err := db.dbi.GetQueryInt(ctx, typeTodoMaxId)
	if err != nil {
		db.log.Error("GetTypeTodoMaxId() failed", "error", err)
		return 0, err
	}
	return int32(existingMaxId), nil
}
