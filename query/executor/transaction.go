// Package executor provides transaction-aware query execution.
package executor

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// TxExecutor wraps an Executor to work within a transaction
type TxExecutor struct {
	*Executor
	tx *sql.Tx
}

// NewTxExecutor creates a new transaction-aware executor
func NewTxExecutor(tx *sql.Tx, provider string) *TxExecutor {
	generator := sqlgen.NewGenerator(provider)
	return &TxExecutor{
		Executor: &Executor{
			db:       nil, // Not used in tx mode
			generator: generator,
			provider: provider,
		},
		tx: tx,
	}
}

// Override query methods to use transaction

// FindManyWithRelations executes a SELECT query within a transaction
func (e *TxExecutor) FindManyWithRelations(ctx context.Context, table string, selectFields map[string]bool, where *sqlgen.WhereClause, orderBy []sqlgen.OrderBy, limit, offset *int, include map[string]bool, relations map[string]RelationMetadata, dest interface{}) error {
	// Convert selectFields map to slice
	var columns []string
	if selectFields != nil && len(selectFields) > 0 {
		for field := range selectFields {
			columns = append(columns, field)
		}
	}
	
	var query *sqlgen.Query
	
	// Build JOINs if relations are included
	var joins []sqlgen.Join
	if include != nil && len(include) > 0 && relations != nil {
		joins = buildJoinsFromIncludes(table, include, relations)
	}
	
	if len(joins) > 0 {
		query = e.generator.GenerateSelectWithJoins(table, columns, joins, where, orderBy, limit, offset)
	} else {
		query = e.generator.GenerateSelect(table, columns, where, orderBy, limit, offset)
	}
	
	rows, err := e.tx.QueryContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Use optimized JOIN mapping if we have JOINs
	if len(joins) > 0 && relations != nil {
		if err := validateRelations(relations); err != nil {
			return fmt.Errorf("invalid relations: %w", err)
		}
		err = e.scanJoinResults(rows, table, joins, relations, dest)
		if err != nil {
			return err
		}
	} else {
		err = e.scanRows(rows, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create executes an INSERT query within a transaction
func (e *TxExecutor) Create(ctx context.Context, table string, data interface{}) (interface{}, error) {
	columns, values, err := e.extractInsertData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract insert data: %w", err)
	}

	query := e.generator.GenerateInsert(table, columns, values)
	
	// For PostgreSQL, use RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		row := e.tx.QueryRowContext(ctx, query.SQL, query.Args...)
		return e.scanRowToStruct(row, data)
	}
	
	// For other databases, execute insert then query back
	result, err := e.tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	// Get the last insert ID if available
	id, err := result.LastInsertId()
	if err == nil {
		// Query back the record
		where := &sqlgen.WhereClause{
			Conditions: []sqlgen.Condition{
				{Field: "id", Operator: "=", Value: id},
			},
			Operator: "AND",
		}
		var found interface{} = data
		if err := e.FindFirstWithRelations(ctx, table, nil, where, nil, nil, nil, &found); err == nil {
			return found, nil
		}
	}

	return data, nil
}

// Update executes an UPDATE query within a transaction
func (e *TxExecutor) Update(ctx context.Context, table string, set map[string]interface{}, where *sqlgen.WhereClause, dest interface{}) error {
	query := e.generator.GenerateUpdate(table, set, where)
	
	// For PostgreSQL, use RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		row := e.tx.QueryRowContext(ctx, query.SQL, query.Args...)
		return e.scanRow(row, dest)
	}
	
	// For other databases, execute update
	_, err := e.tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

// Delete executes a DELETE query within a transaction
func (e *TxExecutor) Delete(ctx context.Context, table string, where *sqlgen.WhereClause) error {
	query := e.generator.GenerateDelete(table, where)
	
	_, err := e.tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

// Count executes a COUNT query within a transaction
func (e *TxExecutor) Count(ctx context.Context, table string, where *sqlgen.WhereClause) (int64, error) {
	aggregates := []sqlgen.AggregateFunction{
		{Function: "COUNT", Field: "*", Alias: "count"},
	}
	
	query := e.generator.GenerateAggregate(table, aggregates, where, nil, nil)
	
	var count int64
	err := e.tx.QueryRowContext(ctx, query.SQL, query.Args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count query failed: %w", err)
	}
	
	return count, nil
}

