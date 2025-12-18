// Package todo_app provides mappers between domain types and Proto types.
// This bridges the gap between the database layer (Domain) and the API layer (Proto).
package todo

import (
	"time"

	"github.com/google/uuid"
	todo_appv1 "github.com/lao-tseu-is-alive/go-cloud-k8s-todo/gen/todo_app/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// Helper Functions
// =============================================================================

// timeToTimestamp converts a *time.Time to *timestamppb.Timestamp
func timeToTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// timestampToTime converts a *timestamppb.Timestamp to *time.Time
func timestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// stringPtr returns a pointer to the string, or nil if empty
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefString safely dereferences a string pointer, returning empty string if nil
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// int32Ptr returns a pointer to the int32, or nil if zero
func int32Ptr(i int32) *int32 {
	if i == 0 {
		return nil
	}
	return &i
}

// derefInt32 safely dereferences an int32 pointer, returning 0 if nil
func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

// boolPtr returns a pointer to the bool
func boolPtr(b bool) *bool {
	return &b
}

// derefBool safely dereferences a bool pointer, returning false if nil
func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// mapToStruct converts a map[string]interface{} to *structpb.Struct
func mapToStruct(m *map[string]interface{}) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(*m)
	if err != nil {
		return nil
	}
	return s
}

// structToMap converts a *structpb.Struct to *map[string]interface{}
func structToMap(s *structpb.Struct) *map[string]interface{} {
	if s == nil {
		return nil
	}
	m := s.AsMap()
	return &m
}

// statusToString converts a *TodoStatus to string
func statusToString(s *TodoStatus) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// stringToStatus converts a string to *TodoStatus
func stringToStatus(s string) *TodoStatus {
	if s == "" {
		return nil
	}
	status := TodoStatus(s)
	return &status
}

// =============================================================================
// Todo Mappers
// =============================================================================

// DomainTodoToProto converts a domain Todo to a Proto Todo
func DomainTodoToProto(t *Todo) *todo_appv1.Todo {
	if t == nil {
		return nil
	}
	return &todo_appv1.Todo{
		Id:                t.Id.String(),
		TypeId:            t.TypeId,
		Name:              t.Name,
		Description:       derefString(t.Description),
		Comment:           derefString(t.Comment),
		ExternalId:        derefInt32(t.ExternalId),
		ExternalRef:       derefString(t.ExternalRef),
		BuildAt:           timeToTimestamp(t.BuildAt),
		Status:            statusToString(t.Status),
		ContainedBy:       derefString(t.ContainedBy),
		ContainedByOld:    derefInt32(t.ContainedByOld),
		Inactivated:       t.Inactivated,
		InactivatedTime:   timeToTimestamp(t.InactivatedTime),
		InactivatedBy:     derefInt32(t.InactivatedBy),
		InactivatedReason: derefString(t.InactivatedReason),
		Validated:         derefBool(t.Validated),
		ValidatedTime:     timeToTimestamp(t.ValidatedTime),
		ValidatedBy:       derefInt32(t.ValidatedBy),
		ManagedBy:         derefInt32(t.ManagedBy),
		CreatedAt:         timeToTimestamp(t.CreatedAt),
		CreatedBy:         t.CreatedBy,
		LastModifiedAt:    timeToTimestamp(t.LastModifiedAt),
		LastModifiedBy:    derefInt32(t.LastModifiedBy),
		Deleted:           t.Deleted,
		DeletedAt:         timeToTimestamp(t.DeletedAt),
		DeletedBy:         derefInt32(t.DeletedBy),
		MoreData:          mapToStruct(t.MoreData),
		PosX:              t.PosX,
		PosY:              t.PosY,
	}
}

// ProtoTodoToDomain converts a Proto Todo to a domain Todo.
// Returns an error if UUID parsing fails.
func ProtoTodoToDomain(t *todo_appv1.Todo) (*Todo, error) {
	if t == nil {
		return nil, nil
	}

	var id uuid.UUID
	var err error
	if t.Id != "" {
		id, err = uuid.Parse(t.Id)
		if err != nil {
			return nil, err
		}
	}

	return &Todo{
		Id:                id,
		TypeId:            t.TypeId,
		Name:              t.Name,
		Description:       stringPtr(t.Description),
		Comment:           stringPtr(t.Comment),
		ExternalId:        int32Ptr(t.ExternalId),
		ExternalRef:       stringPtr(t.ExternalRef),
		BuildAt:           timestampToTime(t.BuildAt),
		Status:            stringToStatus(t.Status),
		ContainedBy:       stringPtr(t.ContainedBy),
		ContainedByOld:    int32Ptr(t.ContainedByOld),
		Inactivated:       t.Inactivated,
		InactivatedTime:   timestampToTime(t.InactivatedTime),
		InactivatedBy:     int32Ptr(t.InactivatedBy),
		InactivatedReason: stringPtr(t.InactivatedReason),
		Validated:         boolPtr(t.Validated),
		ValidatedTime:     timestampToTime(t.ValidatedTime),
		ValidatedBy:       int32Ptr(t.ValidatedBy),
		ManagedBy:         int32Ptr(t.ManagedBy),
		CreatedAt:         timestampToTime(t.CreatedAt),
		CreatedBy:         t.CreatedBy,
		LastModifiedAt:    timestampToTime(t.LastModifiedAt),
		LastModifiedBy:    int32Ptr(t.LastModifiedBy),
		Deleted:           t.Deleted,
		DeletedAt:         timestampToTime(t.DeletedAt),
		DeletedBy:         int32Ptr(t.DeletedBy),
		MoreData:          structToMap(t.MoreData),
		PosX:              t.PosX,
		PosY:              t.PosY,
	}, nil
}

// DomainTodoListToProto converts a domain TodoList to a Proto TodoList
func DomainTodoListToProto(t *TodoList) *todo_appv1.TodoList {
	if t == nil {
		return nil
	}
	return &todo_appv1.TodoList{
		Id:          t.Id.String(),
		TypeId:      t.TypeId,
		Name:        t.Name,
		Description: derefString(t.Description),
		ExternalId:  derefInt32(t.ExternalId),
		Inactivated: t.Inactivated,
		Validated:   derefBool(t.Validated),
		Status:      statusToString(t.Status),
		CreatedBy:   t.CreatedBy,
		CreatedAt:   timeToTimestamp(t.CreatedAt),
		PosX:        t.PosX,
		PosY:        t.PosY,
	}
}

// DomainTodoListSliceToProto converts a slice of domain TodoList to Proto TodoList
func DomainTodoListSliceToProto(items []*TodoList) []*todo_appv1.TodoList {
	if items == nil {
		return nil
	}
	result := make([]*todo_appv1.TodoList, len(items))
	for i, item := range items {
		result[i] = DomainTodoListToProto(item)
	}
	return result
}

// =============================================================================
// TypeTodo Mappers
// =============================================================================

// DomainTypeTodoToProto converts a domain TypeTodo to a Proto TypeTodo
func DomainTypeTodoToProto(t *TypeTodo) *todo_appv1.TypeTodo {
	if t == nil {
		return nil
	}
	return &todo_appv1.TypeTodo{
		Id:                t.Id,
		Name:              t.Name,
		Description:       derefString(t.Description),
		Comment:           derefString(t.Comment),
		ExternalId:        derefInt32(t.ExternalId),
		TableName:         derefString(t.TableName),
		GeometryType:      derefString(t.GeometryType),
		Inactivated:       t.Inactivated,
		InactivatedTime:   timeToTimestamp(t.InactivatedTime),
		InactivatedBy:     derefInt32(t.InactivatedBy),
		InactivatedReason: derefString(t.InactivatedReason),
		ManagedBy:         derefInt32(t.ManagedBy),
		IconPath:          t.IconPath,
		CreatedAt:         timeToTimestamp(t.CreatedAt),
		CreatedBy:         t.CreatedBy,
		LastModifiedAt:    timeToTimestamp(t.LastModifiedAt),
		LastModifiedBy:    derefInt32(t.LastModifiedBy),
		Deleted:           t.Deleted,
		DeletedAt:         timeToTimestamp(t.DeletedAt),
		DeletedBy:         derefInt32(t.DeletedBy),
		MoreDataSchema:    mapToStruct(t.MoreDataSchema),
	}
}

// ProtoTypeTodoToDomain converts a Proto TypeTodo to a domain TypeTodo
func ProtoTypeTodoToDomain(t *todo_appv1.TypeTodo) *TypeTodo {
	if t == nil {
		return nil
	}
	return &TypeTodo{
		Id:                t.Id,
		Name:              t.Name,
		Description:       stringPtr(t.Description),
		Comment:           stringPtr(t.Comment),
		ExternalId:        int32Ptr(t.ExternalId),
		TableName:         stringPtr(t.TableName),
		GeometryType:      stringPtr(t.GeometryType),
		Inactivated:       t.Inactivated,
		InactivatedTime:   timestampToTime(t.InactivatedTime),
		InactivatedBy:     int32Ptr(t.InactivatedBy),
		InactivatedReason: stringPtr(t.InactivatedReason),
		ManagedBy:         int32Ptr(t.ManagedBy),
		IconPath:          t.IconPath,
		CreatedAt:         timestampToTime(t.CreatedAt),
		CreatedBy:         t.CreatedBy,
		LastModifiedAt:    timestampToTime(t.LastModifiedAt),
		LastModifiedBy:    int32Ptr(t.LastModifiedBy),
		Deleted:           t.Deleted,
		DeletedAt:         timestampToTime(t.DeletedAt),
		DeletedBy:         int32Ptr(t.DeletedBy),
		MoreDataSchema:    structToMap(t.MoreDataSchema),
	}
}

// DomainTypeTodoListToProto converts a domain TypeTodoList to a Proto TypeTodoList
func DomainTypeTodoListToProto(t *TypeTodoList) *todo_appv1.TypeTodoList {
	if t == nil {
		return nil
	}
	return &todo_appv1.TypeTodoList{
		Id:           t.Id,
		Name:         t.Name,
		ExternalId:   derefInt32(t.ExternalId),
		IconPath:     t.IconPath,
		CreatedAt:    timeToTimestamp(&t.CreatedAt),
		TableName:    derefString(t.TableName),
		GeometryType: derefString(t.GeometryType),
		Inactivated:  t.Inactivated,
	}
}

// DomainTypeTodoListSliceToProto converts a slice of domain TypeTodoList to Proto
func DomainTypeTodoListSliceToProto(items []*TypeTodoList) []*todo_appv1.TypeTodoList {
	if items == nil {
		return nil
	}
	result := make([]*todo_appv1.TypeTodoList, len(items))
	for i, item := range items {
		result[i] = DomainTypeTodoListToProto(item)
	}
	return result
}
