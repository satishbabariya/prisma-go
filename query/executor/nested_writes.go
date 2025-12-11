// Package executor provides nested write operation execution.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/satishbabariya/prisma-go/query/builder"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// ExecuteNestedWrites executes nested write operations within a transaction
// This is a foundation implementation - full execution requires relation metadata
func (e *Executor) ExecuteNestedWrites(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, operations []*builder.NestedWriteOperation, relations map[string]RelationMetadata) error {
	if len(operations) == 0 {
		return nil
	}

	// Group operations by relation
	opsByRelation := make(map[string][]*builder.NestedWriteOperation)
	for _, op := range operations {
		opsByRelation[op.Relation] = append(opsByRelation[op.Relation], op)
	}

	// Execute operations for each relation
	for relationName, ops := range opsByRelation {
		relMeta, ok := relations[relationName]
		if !ok {
			return fmt.Errorf("relation %s not found", relationName)
		}

		for _, op := range ops {
			if err := e.executeNestedOperation(ctx, tx, parentTable, parentID, op, relMeta); err != nil {
				return fmt.Errorf("failed to execute nested operation %s on relation %s: %w", op.Type, relationName, err)
			}
		}
	}

	return nil
}

// executeNestedOperation executes a single nested write operation
func (e *Executor) executeNestedOperation(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	switch op.Type {
	case builder.NestedWriteCreate:
		return e.executeNestedCreate(ctx, tx, parentTable, parentID, op, relMeta)
	case builder.NestedWriteUpdate:
		return e.executeNestedUpdate(ctx, tx, parentTable, parentID, op, relMeta)
	case builder.NestedWriteDelete:
		return e.executeNestedDelete(ctx, tx, parentTable, parentID, op, relMeta)
	case builder.NestedWriteConnect:
		return e.executeNestedConnect(ctx, tx, parentTable, parentID, op, relMeta)
	case builder.NestedWriteDisconnect:
		return e.executeNestedDisconnect(ctx, tx, parentTable, parentID, op, relMeta)
	case builder.NestedWriteSet:
		return e.executeNestedSet(ctx, tx, parentTable, parentID, op, relMeta)
	case builder.NestedWriteUpsert:
		return e.executeNestedUpsert(ctx, tx, parentTable, parentID, op, relMeta)
	default:
		return fmt.Errorf("unsupported nested write operation: %s", op.Type)
	}
}

// executeNestedCreate creates related records
func (e *Executor) executeNestedCreate(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// Handle many-to-many relations differently
	if relMeta.IsManyToMany {
		return e.executeNestedCreateManyToMany(ctx, tx, parentTable, parentID, op, relMeta)
	}

	// Extract data - op.Data can be a single item or a slice
	dataValue := reflect.ValueOf(op.Data)
	if dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	var items []interface{}
	if dataValue.Kind() == reflect.Slice {
		for i := 0; i < dataValue.Len(); i++ {
			items = append(items, dataValue.Index(i).Interface())
		}
	} else {
		items = []interface{}{op.Data}
	}

	// Create each item
	for _, item := range items {
		// Extract columns and values
		columns, values, err := e.extractInsertData(item)
		if err != nil {
			return fmt.Errorf("failed to extract insert data: %w", err)
		}

		// Set foreign key based on relation type
		if relMeta.IsList {
			// One-to-many: FK is on the related table
			// Find FK column and set it to parentID
			fkIndex := -1
			for i, col := range columns {
				if col == relMeta.ForeignKey || strings.EqualFold(col, relMeta.ForeignKey) {
					fkIndex = i
					break
				}
			}
			if fkIndex >= 0 {
				values[fkIndex] = parentID
			} else {
				// Add FK column if not present
				columns = append(columns, relMeta.ForeignKey)
				values = append(values, parentID)
			}
		} else {
			// One-to-one or many-to-one: FK is on the parent table
			// This shouldn't happen in nested create, but handle it
			return fmt.Errorf("nested create for one-to-one/many-to-one relation not supported")
		}

		// Generate INSERT query
		query := e.generator.GenerateInsert(relMeta.RelatedTable, columns, values)

		// Execute INSERT
		_, err = tx.ExecContext(ctx, query.SQL, query.Args...)
		if err != nil {
			return fmt.Errorf("failed to insert related record: %w", err)
		}
	}

	return nil
}

// executeNestedCreateManyToMany creates records in many-to-many junction table
func (e *Executor) executeNestedCreateManyToMany(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// For many-to-many, we need to insert into junction table
	// op.Data should contain IDs of related records to connect

	dataValue := reflect.ValueOf(op.Data)
	if dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	var relatedIDs []interface{}
	if dataValue.Kind() == reflect.Slice {
		for i := 0; i < dataValue.Len(); i++ {
			relatedIDs = append(relatedIDs, dataValue.Index(i).Interface())
		}
	} else {
		// Try to extract ID from struct
		if dataValue.Kind() == reflect.Struct {
			idField := dataValue.FieldByName("Id")
			if !idField.IsValid() {
				idField = dataValue.FieldByName("ID")
			}
			if idField.IsValid() {
				relatedIDs = []interface{}{idField.Interface()}
			}
		} else {
			relatedIDs = []interface{}{op.Data}
		}
	}

	// Insert into junction table
	for _, relatedID := range relatedIDs {
		columns := []string{relMeta.JunctionFKToSelf, relMeta.JunctionFKToOther}
		values := []interface{}{parentID, relatedID}

		query := e.generator.GenerateInsert(relMeta.JunctionTable, columns, values)
		_, err := tx.ExecContext(ctx, query.SQL, query.Args...)
		if err != nil {
			// Ignore duplicate key errors (record already exists)
			if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "UNIQUE") {
				return fmt.Errorf("failed to insert into junction table: %w", err)
			}
		}
	}

	return nil
}

// executeNestedUpdate updates related records
func (e *Executor) executeNestedUpdate(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// op.Data should be a map with update data and optional where clause
	dataMap, ok := op.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("nested update data must be a map")
	}

	// Extract update data
	set := make(map[string]interface{})
	var where *sqlgen.WhereClause

	for key, value := range dataMap {
		if key == "where" {
			// Build WHERE clause from where data
			whereData, ok := value.(map[string]interface{})
			if ok {
				where = e.buildWhereFromMap(whereData)
			}
		} else {
			set[key] = value
		}
	}

	// Build WHERE clause to filter by relation
	if where == nil {
		where = &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{},
			Operator:   "AND",
		}
	}

	// Add relation filter to WHERE clause
	if relMeta.IsList {
		// One-to-many: filter by FK = parentID
		where.Conditions = append(where.Conditions, sqlgen.Condition{
			Field:    relMeta.ForeignKey,
			Operator: "=",
			Value:    parentID,
		})
	} else {
		// Many-to-one: filter by FK on parent table (shouldn't happen in nested update)
		return fmt.Errorf("nested update for many-to-one relation not supported")
	}

	// Generate UPDATE query
	query := e.generator.GenerateUpdate(relMeta.RelatedTable, set, where)

	// Execute UPDATE
	_, err := tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("failed to update related records: %w", err)
	}

	return nil
}

// buildWhereFromMap builds a WHERE clause from a map
func (e *Executor) buildWhereFromMap(whereMap map[string]interface{}) *sqlgen.WhereClause {
	where := &sqlgen.WhereClause{
		Conditions: []sqlgen.Condition{},
		Operator:   "AND",
	}

	for field, value := range whereMap {
		where.Conditions = append(where.Conditions, sqlgen.Condition{
			Field:    field,
			Operator: "=",
			Value:    value,
		})
	}

	return where
}

// executeNestedDelete deletes related records
func (e *Executor) executeNestedDelete(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// Handle many-to-many relations differently
	if relMeta.IsManyToMany {
		return e.executeNestedDeleteManyToMany(ctx, tx, parentTable, parentID, op, relMeta)
	}

	// Build WHERE clause
	where := &sqlgen.WhereClause{
		Conditions: []sqlgen.Condition{},
		Operator:   "AND",
	}

	// Add relation filter
	if relMeta.IsList {
		// One-to-many: filter by FK = parentID
		where.Conditions = append(where.Conditions, sqlgen.Condition{
			Field:    relMeta.ForeignKey,
			Operator: "=",
			Value:    parentID,
		})
	} else {
		return fmt.Errorf("nested delete for many-to-one relation not supported")
	}

	// Add additional WHERE conditions from op.Data if provided
	if op.Data != nil {
		if whereMap, ok := op.Data.(map[string]interface{}); ok {
			additionalWhere := e.buildWhereFromMap(whereMap)
			where.Conditions = append(where.Conditions, additionalWhere.Conditions...)
		}
	}

	// Generate DELETE query
	query := e.generator.GenerateDelete(relMeta.RelatedTable, where)

	// Execute DELETE
	_, err := tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("failed to delete related records: %w", err)
	}

	return nil
}

// executeNestedDeleteManyToMany deletes records from many-to-many junction table
func (e *Executor) executeNestedDeleteManyToMany(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	where := &sqlgen.WhereClause{
		Conditions: []sqlgen.Condition{
			{
				Field:    relMeta.JunctionFKToSelf,
				Operator: "=",
				Value:    parentID,
			},
		},
		Operator: "AND",
	}

	// If op.Data contains specific IDs, filter by them
	if op.Data != nil {
		dataValue := reflect.ValueOf(op.Data)
		if dataValue.Kind() == reflect.Ptr {
			dataValue = dataValue.Elem()
		}

		if dataValue.Kind() == reflect.Slice && dataValue.Len() > 0 {
			var ids []interface{}
			for i := 0; i < dataValue.Len(); i++ {
				ids = append(ids, dataValue.Index(i).Interface())
			}
			where.Conditions = append(where.Conditions, sqlgen.Condition{
				Field:    relMeta.JunctionFKToOther,
				Operator: "IN",
				Value:    ids,
			})
		}
	}

	query := e.generator.GenerateDelete(relMeta.JunctionTable, where)
	_, err := tx.ExecContext(ctx, query.SQL, query.Args...)
	return err
}

// executeNestedConnect connects existing records to parent
func (e *Executor) executeNestedConnect(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// Handle many-to-many relations differently
	if relMeta.IsManyToMany {
		return e.executeNestedCreateManyToMany(ctx, tx, parentTable, parentID, op, relMeta)
	}

	// Extract IDs to connect
	dataValue := reflect.ValueOf(op.Data)
	if dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	var relatedIDs []interface{}
	if dataValue.Kind() == reflect.Slice {
		for i := 0; i < dataValue.Len(); i++ {
			item := dataValue.Index(i).Interface()
			// Extract ID from struct or use value directly
			if id := e.extractID(item); id != nil {
				relatedIDs = append(relatedIDs, id)
			} else {
				relatedIDs = append(relatedIDs, item)
			}
		}
	} else {
		if id := e.extractID(op.Data); id != nil {
			relatedIDs = []interface{}{id}
		} else {
			relatedIDs = []interface{}{op.Data}
		}
	}

	// Update foreign key on related records
	for _, relatedID := range relatedIDs {
		set := map[string]interface{}{
			relMeta.ForeignKey: parentID,
		}

		where := &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{
				{
					Field:    relMeta.LocalKey,
					Operator: "=",
					Value:    relatedID,
				},
			},
			Operator: "AND",
		}

		query := e.generator.GenerateUpdate(relMeta.RelatedTable, set, where)
		_, err := tx.ExecContext(ctx, query.SQL, query.Args...)
		if err != nil {
			return fmt.Errorf("failed to connect record: %w", err)
		}
	}

	return nil
}

// extractID extracts ID field from a struct or returns the value if it's already an ID
func (e *Executor) extractID(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		// Try to get Id or ID field
		idField := v.FieldByName("Id")
		if !idField.IsValid() {
			idField = v.FieldByName("ID")
		}
		if idField.IsValid() && idField.CanInterface() {
			return idField.Interface()
		}
	}

	// Return as-is if it's a primitive type
	return data
}

// executeNestedDisconnect disconnects records from parent
func (e *Executor) executeNestedDisconnect(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// Handle many-to-many relations differently
	if relMeta.IsManyToMany {
		return e.executeNestedDeleteManyToMany(ctx, tx, parentTable, parentID, op, relMeta)
	}

	// Build WHERE clause
	where := &sqlgen.WhereClause{
		Conditions: []sqlgen.Condition{
			{
				Field:    relMeta.ForeignKey,
				Operator: "=",
				Value:    parentID,
			},
		},
		Operator: "AND",
	}

	// If op.Data contains specific IDs, filter by them
	if op.Data != nil {
		dataValue := reflect.ValueOf(op.Data)
		if dataValue.Kind() == reflect.Ptr {
			dataValue = dataValue.Elem()
		}

		if dataValue.Kind() == reflect.Slice && dataValue.Len() > 0 {
			var ids []interface{}
			for i := 0; i < dataValue.Len(); i++ {
				item := dataValue.Index(i).Interface()
				if id := e.extractID(item); id != nil {
					ids = append(ids, id)
				} else {
					ids = append(ids, item)
				}
			}
			where.Conditions = append(where.Conditions, sqlgen.Condition{
				Field:    relMeta.LocalKey,
				Operator: "IN",
				Value:    ids,
			})
		}
	}

	// Set foreign key to NULL
	set := map[string]interface{}{
		relMeta.ForeignKey: nil,
	}

	query := e.generator.GenerateUpdate(relMeta.RelatedTable, set, where)
	_, err := tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("failed to disconnect records: %w", err)
	}

	return nil
}

// executeNestedSet replaces all relations (disconnect all, then connect new ones)
func (e *Executor) executeNestedSet(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// Foundation: First disconnect all, then connect new ones
	disconnectOp := &builder.NestedWriteOperation{
		Relation: op.Relation,
		Type:     builder.NestedWriteDisconnect,
		Data:     nil, // Disconnect all
	}
	if err := e.executeNestedDisconnect(ctx, tx, parentTable, parentID, disconnectOp, relMeta); err != nil {
		return err
	}
	return e.executeNestedConnect(ctx, tx, parentTable, parentID, op, relMeta)
}

// executeNestedUpsert upserts related records
func (e *Executor) executeNestedUpsert(ctx context.Context, tx *sql.Tx, parentTable string, parentID interface{}, op *builder.NestedWriteOperation, relMeta RelationMetadata) error {
	// op.Data should be a map with "where", "create", and "update" keys
	dataMap, ok := op.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("nested upsert data must be a map with where, create, and update keys")
	}

	whereData, _ := dataMap["where"].(map[string]interface{})
	createData := dataMap["create"]
	updateData, _ := dataMap["update"].(map[string]interface{})

	// Build WHERE clause
	where := e.buildWhereFromMap(whereData)
	if where == nil {
		where = &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{},
			Operator:   "AND",
		}
	}

	// Add relation filter
	if relMeta.IsList {
		where.Conditions = append(where.Conditions, sqlgen.Condition{
			Field:    relMeta.ForeignKey,
			Operator: "=",
			Value:    parentID,
		})
	}

	// Check if record exists
	checkQuery := e.generator.GenerateSelect(relMeta.RelatedTable, []string{"id"}, where, nil, nil, nil)
	var existingID interface{}
	err := tx.QueryRowContext(ctx, checkQuery.SQL, checkQuery.Args...).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Record doesn't exist, create it
		createOp := &builder.NestedWriteOperation{
			Relation: op.Relation,
			Type:     builder.NestedWriteCreate,
			Data:     createData,
		}
		return e.executeNestedCreate(ctx, tx, parentTable, parentID, createOp, relMeta)
	} else if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	// Record exists, update it
	if updateData != nil && len(updateData) > 0 {
		updateWhere := e.buildWhereFromMap(whereData)
		updateWhere.Conditions = append(updateWhere.Conditions, sqlgen.Condition{
			Field:    relMeta.LocalKey,
			Operator: "=",
			Value:    existingID,
		})

		query := e.generator.GenerateUpdate(relMeta.RelatedTable, updateData, updateWhere)
		_, err = tx.ExecContext(ctx, query.SQL, query.Args...)
		if err != nil {
			return fmt.Errorf("failed to update record: %w", err)
		}
	}

	return nil
}
