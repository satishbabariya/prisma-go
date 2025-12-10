// Package executor provides optimized JOIN result mapping.
package executor

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// scanJoinResults maps JOIN query results to nested structs
func (e *Executor) scanJoinResults(
	rows *sql.Rows,
	mainTable string,
	joins []sqlgen.Join,
	relations map[string]RelationMetadata,
	dest interface{},
) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	// Get column names from query result
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	if len(columns) == 0 {
		return fmt.Errorf("query returned no columns")
	}

	// Build column mapping: prefix -> column indices
	columnMap := e.buildColumnMap(columns, mainTable, joins)

	// Validate that we have columns for main table
	if len(columnMap[mainTable]) == 0 {
		return fmt.Errorf("no columns found for main table %s", mainTable)
	}

	// Determine if we're scanning into a slice or single struct
	isSlice := destValue.Elem().Kind() == reflect.Slice
	elementType := destValue.Elem().Type()
	if isSlice {
		elementType = elementType.Elem()
		if elementType.Kind() == reflect.Ptr {
			elementType = elementType.Elem()
		}
	} else {
		elementType = elementType.Elem()
	}

	if elementType.Kind() != reflect.Struct {
		return fmt.Errorf("destination element type must be a struct, got %v", elementType.Kind())
	}

	// For one-to-many relations, we need to group rows
	// Check if we have any one-to-many relations
	hasOneToMany := false
	for _, relMeta := range relations {
		if relMeta.IsList {
			hasOneToMany = true
			break
		}
	}

	if hasOneToMany && isSlice {
		return e.scanJoinResultsGrouped(rows, columns, columnMap, mainTable, joins, relations, dest, elementType)
	}

	// Simple case: no one-to-many relations or single struct
	return e.scanJoinResultsSimple(rows, columns, columnMap, mainTable, joins, relations, dest, elementType, isSlice)
}

// buildColumnMap builds a map of table prefix -> column indices
func (e *Executor) buildColumnMap(columns []string, mainTable string, joins []sqlgen.Join) map[string][]int {
	columnMap := make(map[string][]int)

	// Build set of valid table names/aliases
	validTables := make(map[string]bool)
	validTables[mainTable] = true
	for _, join := range joins {
		tableName := join.Table
		if join.Alias != "" {
			validTables[join.Alias] = true
		} else {
			validTables[tableName] = true
		}
	}

	// Map columns to tables
	for i, col := range columns {
		if !strings.Contains(col, ".") {
			// No prefix - belongs to main table
			columnMap[mainTable] = append(columnMap[mainTable], i)
		} else {
			parts := strings.SplitN(col, ".", 2)
			if len(parts) == 2 {
				tablePrefix := strings.Trim(parts[0], `"`)
				// Only add if it's a valid table
				if validTables[tablePrefix] {
					columnMap[tablePrefix] = append(columnMap[tablePrefix], i)
				} else {
					// Fallback: might be main table with different casing
					columnMap[mainTable] = append(columnMap[mainTable], i)
				}
			}
		}
	}

	return columnMap
}

// scanJoinResultsSimple scans JOIN results for simple cases (no one-to-many grouping)
func (e *Executor) scanJoinResultsSimple(
	rows *sql.Rows,
	columns []string,
	columnMap map[string][]int,
	mainTable string,
	joins []sqlgen.Join,
	relations map[string]RelationMetadata,
	dest interface{},
	elementType reflect.Type,
	isSlice bool,
) error {
	destValue := reflect.ValueOf(dest).Elem()
	var sliceValue reflect.Value

	if isSlice {
		sliceValue = destValue
		destValue = reflect.New(reflect.TypeOf(nil)) // Will be set per row
	}

	for rows.Next() {
		// Create values array for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		// Create new element
		elem := reflect.New(elementType).Interface()

		// Map main table columns
		if err := e.mapColumnsToStruct(columns, values, columnMap[mainTable], mainTable, elem); err != nil {
			return err
		}

		// Map joined table columns
		for _, join := range joins {
			tableName := join.Table
			if join.Alias != "" {
				tableName = join.Alias
			}

			// Find relation for this join
			var relMeta RelationMetadata
			var relationName string
			for name, meta := range relations {
				if meta.RelatedTable == join.Table {
					relMeta = meta
					relationName = name
					break
				}
			}

			if relationName == "" {
				continue
			}

			// Map joined table to relation field
			if err := e.mapJoinToRelation(columns, values, columnMap[tableName], tableName, elem, relationName, relMeta); err != nil {
				return err
			}
		}

		if isSlice {
			sliceValue = reflect.Append(sliceValue, reflect.ValueOf(elem).Elem())
		} else {
			destValue = reflect.ValueOf(elem).Elem()
			break // Single row
		}
	}

	if isSlice {
		reflect.ValueOf(dest).Elem().Set(sliceValue)
	} else {
		reflect.ValueOf(dest).Elem().Set(destValue)
	}

	return rows.Err()
}

// scanJoinResultsGrouped scans JOIN results and groups one-to-many relations
func (e *Executor) scanJoinResultsGrouped(
	rows *sql.Rows,
	columns []string,
	columnMap map[string][]int,
	mainTable string,
	joins []sqlgen.Join,
	relations map[string]RelationMetadata,
	dest interface{},
	elementType reflect.Type,
) error {
	// Group rows by main table ID
	grouped := make(map[interface{}]*groupedRow)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		// Get main table ID - look for "id" field specifically
		mainIDIndex := -1
		for _, idx := range columnMap[mainTable] {
			if idx < len(columns) {
				colName := columns[idx]
				// Remove table prefix if present
				colNameLower := strings.ToLower(colName)
				if strings.Contains(colNameLower, ".") {
					parts := strings.SplitN(colNameLower, ".", 2)
					if len(parts) == 2 {
						colNameLower = strings.Trim(parts[1], `"`)
					}
				}
				// Check for "id" field (exact match or ends with "_id")
				if colNameLower == "id" || colNameLower == `"id"` {
					mainIDIndex = idx
					break
				}
			}
		}

		if mainIDIndex < 0 || mainIDIndex >= len(values) {
			// Fallback: use first column as ID
			if len(columnMap[mainTable]) > 0 {
				mainIDIndex = columnMap[mainTable][0]
			} else {
				continue
			}
		}

		mainID := values[mainIDIndex]
		if mainID == nil {
			continue // Skip rows with NULL ID
		}

		// Get or create grouped row
		group, exists := grouped[mainID]
		if !exists {
			group = &groupedRow{
				mainElement: reflect.New(elementType).Interface(),
				relations:   make(map[string][]interface{}),
			}
			grouped[mainID] = group

			// Map main table columns
			if err := e.mapColumnsToStruct(columns, values, columnMap[mainTable], mainTable, group.mainElement); err != nil {
				return err
			}
		}

		// Map one-to-many relations
		for _, join := range joins {
			tableName := join.Table
			if join.Alias != "" {
				tableName = join.Alias
			}

			var relMeta RelationMetadata
			var relationName string
			for name, meta := range relations {
				if meta.RelatedTable == join.Table && meta.IsList {
					relMeta = meta
					relationName = name
					break
				}
			}

			if relationName == "" || !relMeta.IsList {
				continue
			}

			// Check if this row has data for the joined table
			hasData := false
			for _, idx := range columnMap[tableName] {
				if idx < len(values) && values[idx] != nil {
					hasData = true
					break
				}
			}

			if hasData {
				// Create relation element
				relType := reflect.TypeOf(group.mainElement).Elem()
				relField, found := relType.FieldByName(toPascalCase(relationName))
				if found {
					relElementType := relField.Type.Elem() // []Post -> Post
					relElement := reflect.New(relElementType).Interface()

					if err := e.mapColumnsToStruct(columns, values, columnMap[tableName], tableName, relElement); err != nil {
						return err
					}

					// Check for duplicates by comparing ID values
					relElemValue := reflect.ValueOf(relElement).Elem()
					relIDField := relElemValue.FieldByName("Id")
					if !relIDField.IsValid() {
						relIDField = relElemValue.FieldByName("ID")
					}

					isDuplicate := false
					if relIDField.IsValid() {
						relID := relIDField.Interface()
						for _, existing := range group.relations[relationName] {
							existingValue := reflect.ValueOf(existing).Elem()
							existingIDField := existingValue.FieldByName("Id")
							if !existingIDField.IsValid() {
								existingIDField = existingValue.FieldByName("ID")
							}
							if existingIDField.IsValid() && existingIDField.Interface() == relID {
								isDuplicate = true
								break
							}
						}
					}

					if !isDuplicate {
						group.relations[relationName] = append(group.relations[relationName], relElement)
					}
				}
			}
		}
	}

	// Build final result
	sliceValue := reflect.MakeSlice(reflect.TypeOf(dest).Elem(), 0, len(grouped))
	for _, group := range grouped {
		elem := reflect.ValueOf(group.mainElement).Elem()

		// Set relation fields
		for relationName, relElements := range group.relations {
			field := elem.FieldByName(toPascalCase(relationName))
			if field.IsValid() {
				slice := reflect.MakeSlice(field.Type(), 0, len(relElements))
				for _, relElem := range relElements {
					slice = reflect.Append(slice, reflect.ValueOf(relElem).Elem())
				}
				field.Set(slice)
			}
		}

		// Initialize empty slices for relations that weren't included
		// This ensures empty relations are [] instead of nil
		for relationName := range relations {
			if _, exists := group.relations[relationName]; !exists {
				field := elem.FieldByName(toPascalCase(relationName))
				if field.IsValid() && field.Type().Kind() == reflect.Slice {
					field.Set(reflect.MakeSlice(field.Type(), 0, 0))
				}
			}
		}

		sliceValue = reflect.Append(sliceValue, elem)
	}

	reflect.ValueOf(dest).Elem().Set(sliceValue)
	return rows.Err()
}

type groupedRow struct {
	mainElement interface{}
	relations   map[string][]interface{}
}

// mapColumnsToStruct maps columns to a struct
func (e *Executor) mapColumnsToStruct(columns []string, values []interface{}, indices []int, tablePrefix string, dest interface{}) error {
	// Build column name map (without prefix)
	columnNameMap := make(map[string]int)
	for _, idx := range indices {
		if idx < len(columns) {
			colName := columns[idx]
			// Remove table prefix if present
			if strings.Contains(colName, ".") {
				parts := strings.SplitN(colName, ".", 2)
				if len(parts) == 2 {
					colName = strings.Trim(parts[1], `"`)
				}
			}
			columnNameMap[strings.ToLower(colName)] = idx
		}
	}

	// Use existing mapping logic
	return e.mapValuesToStructFromMap(columnNameMap, values, dest)
}

// mapJoinToRelation maps joined table columns to a relation field
func (e *Executor) mapJoinToRelation(
	columns []string,
	values []interface{},
	indices []int,
	tablePrefix string,
	dest interface{},
	relationName string,
	relMeta RelationMetadata,
) error {
	// Check if relation has data (check ID field specifically)
	hasData := false
	if len(indices) > 0 {
		// Look for ID field in joined table columns
		for _, idx := range indices {
			if idx < len(columns) && idx < len(values) {
				colName := columns[idx]
				// Remove table prefix
				if strings.Contains(colName, ".") {
					parts := strings.SplitN(colName, ".", 2)
					if len(parts) == 2 {
						colName = strings.Trim(parts[1], `"`)
					}
				}
				colNameLower := strings.ToLower(colName)
				// Check if this is an ID field and has a value
				if (colNameLower == "id" || colNameLower == `"id"`) && values[idx] != nil {
					hasData = true
					break
				}
			}
		}
		// Fallback: check if any column has data
		if !hasData {
			for _, idx := range indices {
				if idx < len(values) && values[idx] != nil {
					hasData = true
					break
				}
			}
		}
	}

	if !hasData {
		return nil // No data for this relation
	}

	// Get relation field
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	field := destValue.FieldByName(toPascalCase(relationName))
	if !field.IsValid() {
		return nil // Field not found
	}

	// Create relation element
	fieldType := field.Type()
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	relElement := reflect.New(fieldType).Interface()
	if err := e.mapColumnsToStruct(columns, values, indices, tablePrefix, relElement); err != nil {
		return err
	}

	// Set field - handle NULL values
	if !hasData {
		// No data for relation, set to nil/zero value
		if field.Type().Kind() == reflect.Ptr {
			field.Set(reflect.Zero(field.Type()))
		} else if field.Type().Kind() == reflect.Slice {
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		}
		return nil
	}

	// Set field with data
	if field.Type().Kind() == reflect.Ptr {
		field.Set(reflect.ValueOf(relElement))
	} else {
		field.Set(reflect.ValueOf(relElement).Elem())
	}

	return nil
}

// mapValuesToStructFromMap maps values using a column name map
func (e *Executor) mapValuesToStructFromMap(columnMap map[string]int, values []interface{}, dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get column name from tag or field name
		columnName := field.Tag.Get("db")
		if columnName == "" || columnName == "-" {
			columnName = e.toSnakeCase(field.Name)
		}

		colIndex, ok := columnMap[strings.ToLower(columnName)]
		if !ok {
			continue
		}

		if colIndex >= len(values) {
			continue
		}

		value := values[colIndex]
		if value == nil {
			if fieldValue.Kind() == reflect.Ptr {
				fieldValue.Set(reflect.Zero(fieldValue.Type()))
			}
			continue
		}

		if err := e.setFieldValue(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}
