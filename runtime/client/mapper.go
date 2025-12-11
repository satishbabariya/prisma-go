// Package client provides result mapping utilities.
package client

import (
	"database/sql"
	"reflect"
	"strings"
)

// ScanRows scans SQL rows into a slice of structs
func ScanRows[T any](rows *sql.Rows) ([]T, error) {
	var results []T
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var result T
		val := reflect.ValueOf(&result).Elem()
		typ := val.Type()

		// Create a slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		// Map columns to struct fields
		for i, colName := range columns {
			field := findFieldByName(typ, colName)
			if field.Name != "" {
				fieldVal := val.FieldByIndex(field.Index)
				if fieldVal.CanAddr() {
					valuePtrs[i] = fieldVal.Addr().Interface()
				} else {
					// Fallback: use sql.NullString for unmapped columns
					var nullStr sql.NullString
					valuePtrs[i] = &nullStr
				}
			} else {
				// Unmapped column - use sql.NullString
				var nullStr sql.NullString
				valuePtrs[i] = &nullStr
			}
			values[i] = valuePtrs[i]
		}

		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// ScanRow scans a single SQL row into a struct
func ScanRow[T any](row *sql.Row) (*T, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()
	typ := val.Type()

	// We need to know the columns - this is a simplified version
	// In practice, you'd pass columns or use reflection on the struct
	// For now, we'll use a basic approach
	columns := getStructColumns(typ)
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for i, colName := range columns {
		field := findFieldByName(typ, colName)
		if field.Name != "" {
			fieldVal := val.FieldByIndex(field.Index)
			if fieldVal.CanAddr() {
				valuePtrs[i] = fieldVal.Addr().Interface()
			} else {
				var nullStr sql.NullString
				valuePtrs[i] = &nullStr
			}
		} else {
			var nullStr sql.NullString
			valuePtrs[i] = &nullStr
		}
		values[i] = valuePtrs[i]
	}

	if err := row.Scan(values...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &result, nil
}

// findFieldByName finds a struct field by database column name (db tag or field name)
func findFieldByName(typ reflect.Type, colName string) reflect.StructField {
	// First try exact match
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == colName {
			return field
		}
		// Check db tag
		dbTag := field.Tag.Get("db")
		if dbTag != "" {
			// Handle tags like "db:\"column_name\""
			tagParts := strings.Split(dbTag, ",")
			if len(tagParts) > 0 && tagParts[0] == colName {
				return field
			}
		}
		// Case-insensitive match
		if strings.EqualFold(field.Name, colName) {
			return field
		}
	}
	return reflect.StructField{}
}

// getStructColumns extracts column names from struct tags
func getStructColumns(typ reflect.Type) []string {
	var columns []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" {
			tagParts := strings.Split(dbTag, ",")
			if len(tagParts) > 0 && tagParts[0] != "" {
				columns = append(columns, tagParts[0])
			} else {
				columns = append(columns, field.Name)
			}
		} else {
			columns = append(columns, field.Name)
		}
	}
	return columns
}

