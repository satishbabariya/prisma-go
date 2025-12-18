// Package client implements raw SQL execution.
package client

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
)

// RawExecutor implements the RawClient interface.
type RawExecutor struct {
	db database.Adapter
}

// NewRawExecutor creates a new raw SQL executor.
func NewRawExecutor(db database.Adapter) *RawExecutor {
	return &RawExecutor{db: db}
}

// QueryRaw executes a raw SQL query and returns rows mapped to a generic slice.
func (r *RawExecutor) QueryRaw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	// Safe execution with parameter binding
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw query: %w", err)
	}

	return rows, nil
}

// QueryRawInto executes a raw SQL query and unmarshals results into dest.
// dest should be a pointer to a slice of structs or maps.
func (r *RawExecutor) QueryRawInto(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := r.QueryRaw(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return scanRowsInto(rows, dest)
}

// ExecuteRaw executes a raw SQL statement and returns the result.
func (r *RawExecutor) ExecuteRaw(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	// Safe execution with parameter binding
	result, err := r.db.Execute(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw statement: %w", err)
	}

	return result, nil
}

// QueryRawUnsafe executes a raw SQL query without parameter binding.
// WARNING: This is vulnerable to SQL injection if user input is not properly sanitized.
// Only use this when you need to execute dynamic SQL that cannot use placeholders.
func (r *RawExecutor) QueryRawUnsafe(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if len(args) > 0 {
		// For unsafe queries, we don't use parameter binding
		// This is intentionally dangerous and should be used with extreme caution
		query = fmt.Sprintf(query, args...)
	}

	if r.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute unsafe query: %w", err)
	}

	return rows, nil
}

// ExecuteRawUnsafe executes a raw SQL statement without parameter binding.
// WARNING: This is vulnerable to SQL injection if user input is not properly sanitized.
func (r *RawExecutor) ExecuteRawUnsafe(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if len(args) > 0 {
		query = fmt.Sprintf(query, args...)
	}

	if r.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	result, err := r.db.Execute(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute unsafe statement: %w", err)
	}

	return result, nil
}

// BuildResult converts sql.Rows to RawResult.
func BuildResult(rows *sql.Rows) (*RawResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var resultRows []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &RawResult{
		Columns: columns,
		Rows:    resultRows,
	}, nil
}

// scanRowsInto scans SQL rows into the destination struct/map slice.
func scanRowsInto(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceValue := destValue.Elem()
	elemType := sliceValue.Type().Elem()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Create row map
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		// Convert map to target type
		var elem reflect.Value
		if elemType.Kind() == reflect.Map {
			// Destination is slice of maps
			elem = reflect.ValueOf(rowMap)
		} else {
			// Destination is slice of structs - unmarshal via JSON
			elem = reflect.New(elemType).Elem()
			jsonData, err := json.Marshal(rowMap)
			if err != nil {
				return fmt.Errorf("failed to marshal row: %w", err)
			}
			if err := json.Unmarshal(jsonData, elem.Addr().Interface()); err != nil {
				return fmt.Errorf("failed to unmarshal into struct: %w", err)
			}
		}

		sliceValue.Set(reflect.Append(sliceValue, elem))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	return nil
}

// Ensure RawExecutor implements RawClient.
var _ RawClient = (*RawExecutor)(nil)
