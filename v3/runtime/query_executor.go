// Package runtime provides query execution functionality for generated clients.
package runtime

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// QueryExecutor provides query execution capabilities.
type QueryExecutor struct {
	db      *sql.DB
	dialect domain.SQLDialect
}

// NewQueryExecutor creates a new query executor.
func NewQueryExecutor(db *sql.DB, dialect domain.SQLDialect) *QueryExecutor {
	return &QueryExecutor{
		db:      db,
		dialect: dialect,
	}
}

// ExecuteFindMany executes a FindMany query and returns results.
func (e *QueryExecutor) ExecuteFindMany(ctx context.Context, query *domain.Query) ([]map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	rows, err := e.db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return e.scanRowsToMaps(rows)
}

// ExecuteFindFirst executes a FindFirst query and returns a single result.
func (e *QueryExecutor) ExecuteFindFirst(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	queryCopy := *query
	queryCopy.Pagination = domain.Pagination{
		Take: &[]int{1}[0],
	}

	results, err := e.ExecuteFindMany(ctx, &queryCopy)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		if query.ThrowIfNotFound {
			return nil, fmt.Errorf("record not found")
		}
		return nil, nil
	}

	return results[0], nil
}

// ExecuteFindUnique executes a FindUnique query and returns a single result.
func (e *QueryExecutor) ExecuteFindUnique(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	results, err := e.ExecuteFindMany(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		if query.ThrowIfNotFound {
			return nil, fmt.Errorf("record not found")
		}
		return nil, nil
	}

	return results[0], nil
}

// ExecuteCreate executes a Create query and returns a created record.
func (e *QueryExecutor) ExecuteCreate(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	result, err := e.db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert ID: %w", err)
	}

	return map[string]interface{}{"id": id}, nil
}

// ExecuteUpdate executes an Update query and returns an updated record.
func (e *QueryExecutor) ExecuteUpdate(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	result, err := e.db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute update: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return map[string]interface{}{"count": count}, nil
}

// ExecuteDelete executes a Delete query and returns deleted record info.
func (e *QueryExecutor) ExecuteDelete(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	result, err := e.db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute delete: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return map[string]interface{}{"count": count}, nil
}

// ExecuteUpsert executes an Upsert query and returns a created/updated record.
func (e *QueryExecutor) ExecuteUpsert(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	result, err := e.db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute upsert: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return map[string]interface{}{"count": count}, nil
}

// Execute executes a compiled query and returns results.
func (e *QueryExecutor) Execute(ctx context.Context, compiled *domain.CompiledQuery) (interface{}, error) {
	switch compiled.OriginalQuery.Operation {
	case domain.FindMany:
		rows, err := e.db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		defer rows.Close()
		return e.scanRowsToMaps(rows)

	case domain.FindFirst, domain.FindUnique:
		queryCopy := *compiled.OriginalQuery
		queryCopy.Pagination = domain.Pagination{
			Take: &[]int{1}[0],
		}

		compiler := compiler.NewSQLCompiler(e.dialect)
		firstCompiled, err := compiler.Compile(ctx, &queryCopy)
		if err != nil {
			return nil, fmt.Errorf("failed to compile find first query: %w", err)
		}

		rows, err := e.db.QueryContext(ctx, firstCompiled.SQL.Query, firstCompiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		defer rows.Close()

		results, err := e.scanRowsToMaps(rows)
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			return nil, fmt.Errorf("no records found")
		}

		return results[0], nil

	case domain.Create, domain.CreateMany, domain.Update, domain.UpdateMany, domain.Delete, domain.DeleteMany, domain.Upsert:
		result, err := e.db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}

		return map[string]interface{}{"count": affected}, nil

	case domain.Aggregate:
		rows, err := e.db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute aggregate: %w", err)
		}
		defer rows.Close()

		return e.scanRowsToMaps(rows)

	default:
		return nil, fmt.Errorf("unsupported operation: %s", compiled.OriginalQuery.Operation)
	}
}

// ExecuteAggregate executes an Aggregate query and returns aggregation results.
func (e *QueryExecutor) ExecuteAggregate(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	rows, err := e.db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregate: %w", err)
	}
	defer rows.Close()

	results, err := e.scanRowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return map[string]interface{}{}, nil
	}

	return results[0], nil
}

// ExecuteRaw executes a raw SQL query.
func (e *QueryExecutor) ExecuteRaw(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw query: %w", err)
	}
	defer rows.Close()

	return e.scanRowsToMaps(rows)
}

// ExecuteRawStatement executes a raw SQL statement.
func (e *QueryExecutor) ExecuteRawStatement(ctx context.Context, query string, args ...interface{}) (int64, error) {
	result, err := e.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute raw statement: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return affected, nil
}

// scanRowsToMaps scans SQL rows into a slice of maps.
func (e *QueryExecutor) scanRowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
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

			// Handle different types properly
			switch v := val.(type) {
			case []byte:
				row[col] = string(v)
			default:
				if val != nil {
					row[col] = v
				}
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// BatchResult represents a result of a batch operation.
type BatchResult struct {
	Results []interface{} `json:"results"`
	Errors  []error       `json:"errors"`
}
