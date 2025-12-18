package todo

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/database"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/goHttpEcho"
)

type Permission int8 // enum
const (
	R Permission = iota // Read implies List (SELECT in DB, or GET in API)
	W                   // implies INSERT,UPDATE, DELETE
	M                   // Update or Put only
	D                   // Delete only
	C                   // Create only (Insert, Post)
	P                   // change Permissions of one todo
	O                   // change Owner of one Todo
	A                   // Audit log of changes of one todo and read only special _fields like _created_by
)

func (s Permission) String() string {
	switch s {
	case R:
		return "R"
	case W:
		return "W"
	case M:
		return "M"
	case D:
		return "D"
	case C:
		return "C"
	case P:
		return "P"
	case O:
		return "O"
	case A:
		return "A"
	}
	return "ErrorPermissionUnknown"
}

type Service struct {
	Log              *slog.Logger
	DbConn           database.DB
	Store            Storage
	Server           *goHttpEcho.Server
	ListDefaultLimit int
}

func (s Service) GeoJson(ctx echo.Context, params GeoJsonParams) error {
	handlerName := "GeoJson"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	limit := s.ListDefaultLimit
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	offset := 0
	if params.Offset != nil {
		offset = int(*params.Offset)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	jsonResult, err := s.Store.GeoJson(reqCtx, offset, limit, params)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("there was a problem when calling store.List :%v", err))
		} else {
			jsonResult = "empty"
			return ctx.JSONBlob(http.StatusOK, []byte(jsonResult))
		}
	}
	return ctx.JSONBlob(http.StatusOK, []byte(jsonResult))
}

// List sends a list of todos in the store based on the given parameters filters
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo?limit=3&ofset=0' |jq
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo?limit=3&type=112' |jq
func (s Service) List(ctx echo.Context, params ListParams) error {
	handlerName := "List"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	limit := s.ListDefaultLimit
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	offset := 0
	if params.Offset != nil {
		offset = int(*params.Offset)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	list, err := s.Store.List(reqCtx, offset, limit, params)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("there was a problem when calling store.List :%v", err))
		} else {
			list = make([]*TodoList, 0)
			return ctx.JSON(http.StatusOK, list)
		}
	}
	return ctx.JSON(http.StatusOK, list)
}

// Create allows to insert a new todo
// curl -s -XPOST -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"id": "3999971f-53d7-4eb6-8898-97f257ea5f27","type_id": 3,"name": "Gil-Parcelle","description": "just a nice parcelle test","external_id": 345678912,"inactivated": false,"managed_by": 999, "more_data": NULL,"pos_x":2537603.0 ,"pos_y":1152613.0   }' 'http://localhost:9090/goapi/v1/todo'
func (s Service) Create(ctx echo.Context) error {
	handlerName := "Create"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	/* TODO implement ACL & RBAC handling
	if !s.Store.IsUserAllowedToCreate(currentUserId, typeTodo) {
		return echo.NewHTTPError(http.StatusUnauthorized, "current user has no create role privilege")
	}
	*/
	newTodo := &Todo{
		CreatedBy: int32(currentUserId),
	}
	if err := ctx.Bind(newTodo); err != nil {
		msg := fmt.Sprintf("Create has invalid format [%v]", err)
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	s.Log.Info("Create Todo Bind ok", "todo", newTodo.Name)
	if len(strings.Trim(newTodo.Name, " ")) < 1 {

		msg := fmt.Sprintf(FieldCannotBeEmpty, "name")
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(newTodo.Name) < MinNameLength {
		msg := fmt.Sprintf(FieldMinLengthIsN, "name", MinNameLength)
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	if s.Store.Exist(reqCtx, newTodo.Id) {
		msg := fmt.Sprintf("This id (%v) already exist !", newTodo.Id)
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	todoCreated, err := s.Store.Create(reqCtx, *newTodo)
	if err != nil {
		msg := fmt.Sprintf("Create had an error saving todo:%#v, err:%#v", *newTodo, err)
		s.Log.Info(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	s.Log.Info("Create success", "todoId", todoCreated.Id)
	return ctx.JSON(http.StatusCreated, todoCreated)
}

// Count returns the number of todos found after filtering data with any given CountParams
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo/count' |jq
func (s Service) Count(ctx echo.Context, params CountParams) error {
	handlerName := "Count"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	numTodos, err := s.Store.Count(reqCtx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("problem counting todos :%v", err))
	}
	return ctx.JSON(http.StatusOK, numTodos)
}

// Delete will remove the given todoId entry from the store, and if not present will return 400 Bad Request
// curl -v -XDELETE -H "Content-Type: application/json" -H "Authorization: Bearer $token" 'http://localhost:8888/api/users/3' ->  204 No Content if present and delete it
// curl -v -XDELETE -H "Content-Type: application/json"  -H "Authorization: Bearer $token" 'http://localhost:8888/users/93333' -> 400 Bad Request
func (s Service) Delete(ctx echo.Context, todoId uuid.UUID) error {
	handlerName := "GeoJson"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := int32(claims.User.UserId)
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	if s.Store.Exist(reqCtx, todoId) == false {
		msg := fmt.Sprintf("Delete(%v) cannot delete this id, it does not exist !", todoId)
		s.Log.Warn(msg)
		return ctx.JSON(http.StatusNotFound, msg)
	}
	// IF USER IS NOT OWNER OF RECORD RETURN 401 Unauthorized
	if !s.Store.IsUserOwner(reqCtx, todoId, currentUserId) {
		return echo.NewHTTPError(http.StatusUnauthorized, "current user is not owner of this todo")
	}
	/* TODO implement ACL & RBAC handling
	if !s.Store.IsUserAllowedToDelete(currentUserId, typeTodo) {
		return echo.NewHTTPError(http.StatusUnauthorized, "current user has no create role privilege")
	}
	*/
	err := s.Store.Delete(reqCtx, todoId, currentUserId)
	if err != nil {
		msg := fmt.Sprintf("Delete(%v) got an error: %#v ", todoId, err)
		s.Log.Error(msg)
		return echo.NewHTTPError(http.StatusInternalServerError, msg)
	}
	return ctx.NoContent(http.StatusNoContent)

}

// Get will retrieve the Todo with the given id in the store and return it
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo/9999971f-53d7-4eb6-8898-97f257ea5f27' |jq
func (s Service) Get(ctx echo.Context, todoId uuid.UUID) error {
	handlerName := "Get"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	if s.Store.Exist(reqCtx, todoId) == false {
		msg := fmt.Sprintf("Get(%v) cannot get this id, it does not exist !", todoId)
		s.Log.Info(msg)
		return ctx.JSON(http.StatusNotFound, msg)
	}
	/* TODO implement ACL & RBAC handling
	if !s.Store.IsUserAllowedToGet(currentUserId, typeTodo) {
		return echo.NewHTTPError(http.StatusUnauthorized, "current user has no create role privilege")
	}
	*/
	todo, err := s.Store.Get(reqCtx, todoId)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("problem retrieving todo :%v", err))
		} else {
			msg := fmt.Sprintf("Get(%v) no rows found in db", todoId)
			s.Log.Info(msg)
			return ctx.JSON(http.StatusNotFound, msg)
		}
	}
	return ctx.JSON(http.StatusOK, todo)
}

// Update will change the attributes values for the todo identified by the given todoId
// curl -s -XPUT -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d '{"id": "3999971f-53d7-4eb6-8898-97f257ea5f27","type_id": 3,"name": "Gil-Parcelle","description": "just a nice parcelle test by GIL","external_id": 345678912,"inactivated": false,"managed_by": 999, "more_data": {"info_value": 3230 },"pos_x":2537603.0 ,"pos_y":1152613.0   }' 'http://localhost:9090/goapi/v1/todo/3999971f-53d7-4eb6-8898-97f257ea5f27' |jq
func (s Service) Update(ctx echo.Context, todoId uuid.UUID) error {
	handlerName := "GeoJson"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := int32(claims.User.UserId)
	s.Log.Info("handler called", "handler", handlerName, "todoId", todoId, "userId", currentUserId)
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	if s.Store.Exist(reqCtx, todoId) == false {
		msg := fmt.Sprintf("Update(%v) cannot update this id, it does not exist !", todoId)
		s.Log.Warn(msg)
		return ctx.JSON(http.StatusNotFound, msg)
	}
	if !s.Store.IsUserOwner(reqCtx, todoId, currentUserId) {
		return echo.NewHTTPError(http.StatusUnauthorized, "current user is not owner of this todo")
	}
	/* TODO implement ACL & RBAC handling
	if !s.Store.IsUserAllowedToUpdate(currentUserId, typeTodo) {
		return echo.NewHTTPError(http.StatusUnauthorized, "current user has no create role privilege")
	}
	*/

	updateTodo := new(Todo)
	if err := ctx.Bind(updateTodo); err != nil {
		msg := fmt.Sprintf("Update has invalid format error:[%v]", err)
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(strings.Trim(updateTodo.Name, " ")) < 1 {
		msg := fmt.Sprintf(FieldCannotBeEmpty, "name")
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(updateTodo.Name) < MinNameLength {

		msg := fmt.Sprintf(FieldMinLengthIsN+FoundNum, "name", MinNameLength, len(updateTodo.Name))
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	updateTodo.LastModifiedBy = &currentUserId
	//TODO handle update of validated field correctly by adding validated time & user
	// handle update of managed_by field correctly by checking if user is a valid active one
	todoUpdated, err := s.Store.Update(reqCtx, todoId, *updateTodo)
	if err != nil {
		msg := fmt.Sprintf("Update had an error saving todo:%#v, err:%#v", *updateTodo, err)
		s.Log.Info(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	s.Log.Info("Update success", "todoId", todoUpdated.Id)
	return ctx.JSON(http.StatusOK, todoUpdated)
}

// ListByExternalId sends a list of todos in the store as json based of the given filters
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo/by-external-id/345678912?limit=3&ofset=0' |jq
func (s Service) ListByExternalId(ctx echo.Context, externalId int32, params ListByExternalIdParams) error {
	handlerName := "ListByExternalId"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	limit := s.ListDefaultLimit
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	offset := 0
	if params.Offset != nil {
		offset = int(*params.Offset)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	list, err := s.Store.ListByExternalId(reqCtx, offset, limit, int(externalId))
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("there was a problem when calling store.ListByExternalId :%v", err))
		} else {
			list = make([]*TodoList, 0)
			return ctx.JSON(http.StatusNotFound, list)
		}
	}
	return ctx.JSON(http.StatusOK, list)
}

// Search returns a list of todos in the store as json based of the given search criteria
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo/search?limit=3&ofset=0' |jq
// curl -s -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 'http://localhost:9090/goapi/v1/todo/search?limit=3&type=112' |jq
func (s Service) Search(ctx echo.Context, params SearchParams) error {
	handlerName := "Search"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	limit := s.ListDefaultLimit
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	offset := 0
	if params.Offset != nil {
		offset = int(*params.Offset)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	list, err := s.Store.Search(reqCtx, offset, limit, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			list = make([]*TodoList, 0)
			return ctx.JSON(http.StatusOK, list)
		} else {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("there was a problem when calling store.Search :%v", err))
		}
	}
	return ctx.JSON(http.StatusOK, list)
}

// TypeTodoList sends a list of TypeTodo based on the given TypeTodoListParams parameters filters
func (s Service) TypeTodoList(ctx echo.Context, params TypeTodoListParams) error {
	handlerName := "TypeTodoList"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	limit := 250
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	offset := 0
	if params.Offset != nil {
		offset = int(*params.Offset)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	list, err := s.Store.ListTypeTodo(reqCtx, offset, limit, params)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("there was a problem when calling store.ListTypeTodo :%v", err))
		} else {
			list = make([]*TypeTodoList, 0)
			return ctx.JSON(http.StatusNotFound, list)
		}
	}
	return ctx.JSON(http.StatusOK, list)
}

// TypeTodoCreate will insert a new TypeTodo in the store
func (s Service) TypeTodoCreate(ctx echo.Context) error {
	handlerName := "TypeTodoCreate"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	if !claims.User.IsAdmin {
		return echo.NewHTTPError(http.StatusUnauthorized, OnlyAdminCanManageTypeTodos)
	}
	newTypeTodo := &TypeTodo{
		Comment:           nil,
		CreatedAt:         nil,
		CreatedBy:         int32(currentUserId),
		Deleted:           false,
		DeletedAt:         nil,
		DeletedBy:         nil,
		Description:       nil,
		ExternalId:        nil,
		GeometryType:      nil,
		Id:                0,
		Inactivated:       false,
		InactivatedBy:     nil,
		InactivatedReason: nil,
		InactivatedTime:   nil,
		LastModifiedAt:    nil,
		LastModifiedBy:    nil,
		ManagedBy:         nil,
		IconPath:          "",
		MoreDataSchema:    nil,
		Name:              "",
		TableName:         nil,
	}
	if err := ctx.Bind(newTypeTodo); err != nil {
		msg := fmt.Sprintf("TypeTodoCreate has invalid format [%v]", err)
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(strings.Trim(newTypeTodo.Name, " ")) < 1 {
		msg := fmt.Sprintf(FieldCannotBeEmpty, "name")
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(newTypeTodo.Name) < MinNameLength {
		msg := fmt.Sprintf(FieldMinLengthIsN+", found %d", "name", MinNameLength, len(newTypeTodo.Name))
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	//s.Log.Info("# Create() before Store.TypeTodoCreate newTodo : %#v\n", newTodo)
	typeTodoCreated, err := s.Store.CreateTypeTodo(reqCtx, *newTypeTodo)
	if err != nil {
		msg := fmt.Sprintf("TypeTodoCreate had an error saving todo:%#v, err:%#v", *newTypeTodo, err)
		s.Log.Info(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	s.Log.Info("TypeTodoCreate success", "typeTodoId", typeTodoCreated.Id)
	return ctx.JSON(http.StatusCreated, typeTodoCreated)
}

func (s Service) TypeTodoCount(ctx echo.Context, params TypeTodoCountParams) error {
	handlerName := "TypeTodoCount"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	// Use request context for cancellation and tracing support
	reqCtx := ctx.Request().Context()
	numTodos, err := s.Store.CountTypeTodo(reqCtx, params)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("problem counting todos :%v", err))
	}
	return ctx.JSON(http.StatusOK, numTodos)
}

// TypeTodoDelete will remove the given TypeTodo entry from the store, and if not present will return 400 Bad Request
func (s Service) TypeTodoDelete(ctx echo.Context, typeTodoId int32) error {
	handlerName := "TypeTodoDelete"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := int32(claims.User.UserId)
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	// IF USER IS NOT ADMIN  RETURN 401 Unauthorized
	if !claims.User.IsAdmin {
		return echo.NewHTTPError(http.StatusUnauthorized, OnlyAdminCanManageTypeTodos)
	}
	reqCtx := ctx.Request().Context()
	typeTodoCount, err := s.DbConn.GetQueryInt(reqCtx, existTypeTodo, typeTodoId)
	if err != nil || typeTodoCount < 1 {
		msg := fmt.Sprintf("TypeTodoDelete(%v) cannot delete this id, it does not exist !", typeTodoId)
		s.Log.Warn(msg)
		return ctx.JSON(http.StatusNotFound, msg)
	} else {
		err := s.Store.DeleteTypeTodo(reqCtx, typeTodoId, currentUserId)
		if err != nil {
			msg := fmt.Sprintf("TypeTodoDelete(%v) got an error: %#v ", typeTodoId, err)
			s.Log.Error(msg)
			return echo.NewHTTPError(http.StatusInternalServerError, msg)
		}
		return ctx.NoContent(http.StatusNoContent)
	}
}

// TypeTodoGet will retrieve the Todo with the given id in the store and return it
func (s Service) TypeTodoGet(ctx echo.Context, typeTodoId int32) error {
	handlerName := "TypeTodoGet"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := claims.User.UserId
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	if !claims.User.IsAdmin {
		return echo.NewHTTPError(http.StatusUnauthorized, OnlyAdminCanManageTypeTodos)
	}
	reqCtx := ctx.Request().Context()
	typeTodoCount, err := s.DbConn.GetQueryInt(reqCtx, existTypeTodo, typeTodoId)
	if err != nil || typeTodoCount < 1 {
		msg := fmt.Sprintf("TypeTodoGet(%v) cannot retrieve this id, it does not exist !", typeTodoId)
		s.Log.Warn(msg)
		return ctx.JSON(http.StatusNotFound, msg)
	}
	typeTodo, err := s.Store.GetTypeTodo(reqCtx, typeTodoId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("problem retrieving TypeTodo :%v", err))
	}
	return ctx.JSON(http.StatusOK, typeTodo)
}

func (s Service) TypeTodoUpdate(ctx echo.Context, typeTodoId int32) error {
	handlerName := "TypeTodoUpdate"
	goHttpEcho.TraceHttpRequest(handlerName, ctx.Request(), s.Log)
	// get the current user from JWT TOKEN
	claims := s.Server.JwtCheck.GetJwtCustomClaimsFromContext(ctx)
	currentUserId := int32(claims.User.UserId)
	s.Log.Info("handler called", "handler", handlerName, "userId", currentUserId)
	// IF USER IS NOT ADMIN  RETURN 401 Unauthorized
	if !claims.User.IsAdmin {
		return echo.NewHTTPError(http.StatusUnauthorized, OnlyAdminCanManageTypeTodos)
	}
	reqCtx := ctx.Request().Context()
	typeTodoCount, err := s.DbConn.GetQueryInt(reqCtx, existTypeTodo, typeTodoId)
	if err != nil || typeTodoCount < 1 {
		msg := fmt.Sprintf("TypeTodoUpdate(%v) cannot update this id, it does not exist !", typeTodoId)
		s.Log.Warn(msg)
		return ctx.JSON(http.StatusNotFound, msg)
	}
	uTypeTodo := new(TypeTodo)
	if err := ctx.Bind(uTypeTodo); err != nil {
		msg := fmt.Sprintf("TypeTodoUpdate has invalid format error:[%v]", err)
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(strings.Trim(uTypeTodo.Name, " ")) < 1 {
		msg := fmt.Sprintf(FieldCannotBeEmpty, "name")
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	if len(uTypeTodo.Name) < MinNameLength {
		msg := fmt.Sprintf(FieldMinLengthIsN+", found %d", "name", MinNameLength, len(uTypeTodo.Name))
		s.Log.Error(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	uTypeTodo.LastModifiedBy = &currentUserId
	todoUpdated, err := s.Store.UpdateTypeTodo(reqCtx, typeTodoId, *uTypeTodo)
	if err != nil {
		msg := fmt.Sprintf("TypeTodoUpdate had an error saving typeTodo:%#v, err:%#v", *uTypeTodo, err)
		s.Log.Info(msg)
		return ctx.JSON(http.StatusBadRequest, msg)
	}
	s.Log.Info("TypeTodoUpdate success", "typeTodoId", todoUpdated.Id)
	return ctx.JSON(http.StatusOK, todoUpdated)
}
