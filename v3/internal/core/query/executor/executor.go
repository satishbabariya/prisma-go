// Package executor implements query execution.
package executor

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/mapper"
	"github.com/satishbabariya/prisma-go/v3/pkg/client"
)

// QueryExecutor implements the domain.QueryExecutor interface.
type QueryExecutor struct {
	db     database.Adapter
	mapper *mapper.ResultMapper
}

// NewQueryExecutor creates a new query executor.
func NewQueryExecutor(db database.Adapter) *QueryExecutor {
	return &QueryExecutor{
		db:     db,
		mapper: mapper.NewResultMapper(),
	}
}

// Execute executes a compiled query.
func (e *QueryExecutor) Execute(ctx context.Context, query *domain.CompiledQuery) (interface{}, error) {
	if e.db == nil {
		return nil, fmt.Errorf("database adapter not initialized")
	}

	// Execute the SQL query
	rows, err := e.db.Query(ctx, query.SQL.Query, query.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan results
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build result map
		result := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for text columns
			if b, ok := val.([]byte); ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Check if ThrowIfNotFound is set and no results were found
	if query.OriginalQuery != nil && query.OriginalQuery.ThrowIfNotFound && len(results) == 0 {
		return nil, client.NewNotFoundError(query.Mapping.Model)
	}

	return results, nil
}

// ExecuteBatch executes multiple queries in batch.
func (e *QueryExecutor) ExecuteBatch(ctx context.Context, queries []*domain.CompiledQuery) ([]interface{}, error) {
	results := make([]interface{}, len(queries))

	for i, query := range queries {
		result, err := e.Execute(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// ExecuteInto executes a query and maps results to a struct slice.
func (e *QueryExecutor) ExecuteInto(ctx context.Context, query *domain.CompiledQuery, dest interface{}) error {
	if e.db == nil {
		return fmt.Errorf("database adapter not initialized")
	}

	// Execute the SQL query
	rows, err := e.db.Query(ctx, query.SQL.Query, query.SQL.Args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan all results into maps first
	var mapResults []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}

		mapResults = append(mapResults, result)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Check if ThrowIfNotFound is set and no results were found
	if query.OriginalQuery != nil && query.OriginalQuery.ThrowIfNotFound && len(mapResults) == 0 {
		return client.NewNotFoundError(query.Mapping.Model)
	}

	// Map results to struct using the mapper
	return e.mapper.MapToStructSlice(mapResults, dest)
}

// ExecuteMutation executes a mutation query (UPDATE, DELETE) and returns rows affected.
func (e *QueryExecutor) ExecuteMutation(ctx context.Context, query *domain.CompiledQuery) (int64, error) {
	if e.db == nil {
		return 0, fmt.Errorf("database adapter not initialized")
	}

	// Execute the SQL
	result, err := e.db.Execute(ctx, query.SQL.Query, query.SQL.Args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute mutation: %w", err)
	}

	// Get rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return result, nil
}

// ExecuteNestedWrites executes nested write operations in a transaction.
// All statements must succeed or all will be rolled back.
func (e *QueryExecutor) ExecuteNestedWrites(ctx context.Context, statements []domain.SQL) error {
	if e.db == nil {
		return fmt.Errorf("database adapter not initialized")
	}

	// Begin transaction
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback on error
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("transaction failed and rollback failed: %w (rollback error: %v)", err, rbErr)
			}
		}
	}()

	// Execute all statements
	for i, stmt := range statements {
		_, err = tx.Execute(ctx, stmt.Query, stmt.Args...)
		if err != nil {
			return fmt.Errorf("failed to execute nested write statement %d: %w", i, err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ScanInto scans a single row into a struct (helper for future use).
func (e *QueryExecutor) ScanInto(rows *sql.Rows, dest interface{}) error {
	// This will be implemented when we add struct mapping
	// For now, return not implemented
	return fmt.Errorf("struct scanning not yet implemented")
}

// Ensure QueryExecutor implements QueryExecutor interface.
var _ domain.QueryExecutor = (*QueryExecutor)(nil)
```
