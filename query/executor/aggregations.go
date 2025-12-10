// Package executor provides aggregation query execution.
package executor

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// AggregateResult holds the result of an aggregation query
type AggregateResult struct {
	Count *int64
	Sum   *float64
	Avg   *float64
	Min   *float64
	Max   *float64
}

// Count executes a COUNT query
func (e *Executor) Count(ctx context.Context, table string, where *sqlgen.WhereClause) (int64, error) {
	aggregates := []sqlgen.AggregateFunction{
		{Function: "COUNT", Field: "*", Alias: "count"},
	}
	
	query := e.generator.GenerateAggregate(table, aggregates, where, nil, nil)
	
	var count int64
	err := e.db.QueryRowContext(ctx, query.SQL, query.Args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}
	
	return count, nil
}

// Sum executes a SUM aggregation
func (e *Executor) Sum(ctx context.Context, table string, field string, where *sqlgen.WhereClause) (float64, error) {
	aggregates := []sqlgen.AggregateFunction{
		{Function: "SUM", Field: field, Alias: "sum"},
	}
	
	query := e.generator.GenerateAggregate(table, aggregates, where, nil, nil)
	
	var sum sql.NullFloat64
	err := e.db.QueryRowContext(ctx, query.SQL, query.Args...).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum query failed: %w", err)
	}
	
	if !sum.Valid {
		return 0, nil
	}
	
	return sum.Float64, nil
}

// Avg executes an AVG aggregation
func (e *Executor) Avg(ctx context.Context, table string, field string, where *sqlgen.WhereClause) (float64, error) {
	aggregates := []sqlgen.AggregateFunction{
		{Function: "AVG", Field: field, Alias: "avg"},
	}
	
	query := e.generator.GenerateAggregate(table, aggregates, where, nil, nil)
	
	var avg sql.NullFloat64
	err := e.db.QueryRowContext(ctx, query.SQL, query.Args...).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("avg query failed: %w", err)
	}
	
	if !avg.Valid {
		return 0, nil
	}
	
	return avg.Float64, nil
}

// Min executes a MIN aggregation
func (e *Executor) Min(ctx context.Context, table string, field string, where *sqlgen.WhereClause) (float64, error) {
	aggregates := []sqlgen.AggregateFunction{
		{Function: "MIN", Field: field, Alias: "min"},
	}
	
	query := e.generator.GenerateAggregate(table, aggregates, where, nil, nil)
	
	var min sql.NullFloat64
	err := e.db.QueryRowContext(ctx, query.SQL, query.Args...).Scan(&min)
	if err != nil {
		return 0, fmt.Errorf("min query failed: %w", err)
	}
	
	if !min.Valid {
		return 0, nil
	}
	
	return min.Float64, nil
}

// Max executes a MAX aggregation
func (e *Executor) Max(ctx context.Context, table string, field string, where *sqlgen.WhereClause) (float64, error) {
	aggregates := []sqlgen.AggregateFunction{
		{Function: "MAX", Field: field, Alias: "max"},
	}
	
	query := e.generator.GenerateAggregate(table, aggregates, where, nil, nil)
	
	var max sql.NullFloat64
	err := e.db.QueryRowContext(ctx, query.SQL, query.Args...).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("max query failed: %w", err)
	}
	
	if !max.Valid {
		return 0, nil
	}
	
	return max.Float64, nil
}

// Aggregate executes multiple aggregations in a single query
func (e *Executor) Aggregate(ctx context.Context, table string, aggregates []sqlgen.AggregateFunction, where *sqlgen.WhereClause, groupBy *sqlgen.GroupBy) ([]map[string]interface{}, error) {
	query := e.generator.GenerateAggregate(table, aggregates, where, groupBy, nil)
	
	rows, err := e.db.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return nil, fmt.Errorf("aggregate query failed: %w", err)
	}
	defer rows.Close()
	
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		
		// Build result map
		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}
		results = append(results, result)
	}
	
	return results, rows.Err()
}

