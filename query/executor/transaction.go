// Package executor provides transaction-aware query execution.
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

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
			db:        nil, // Not used in tx mode
			generator: generator,
			provider:  provider,
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

// CreateMany executes batch INSERT queries within a transaction
func (e *TxExecutor) CreateMany(ctx context.Context, table string, data []interface{}) ([]interface{}, error) {
	if len(data) == 0 {
		return []interface{}{}, nil
	}

	var results []interface{}

	// For PostgreSQL, we can use multi-row INSERT with RETURNING
	if e.provider == "postgresql" || e.provider == "postgres" {
		// Extract columns from first record
		columns, _, err := e.extractInsertData(data[0])
		if err != nil {
			return nil, fmt.Errorf("failed to extract insert data: %w", err)
		}

		// Build multi-row INSERT
		var parts []string
		var args []interface{}
		argIndex := 1

		parts = append(parts, fmt.Sprintf("INSERT INTO %s", e.quoteIdentifier(table)))
		quotedCols := make([]string, len(columns))
		for i, col := range columns {
			quotedCols[i] = e.quoteIdentifier(col)
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(quotedCols, ", ")))
		parts = append(parts, "VALUES")

		// Build VALUES for each row
		valueParts := make([]string, len(data))
		for i, record := range data {
			_, values, err := e.extractInsertData(record)
			if err != nil {
				return nil, fmt.Errorf("failed to extract insert data for record %d: %w", i, err)
			}

			placeholders := make([]string, len(values))
			for j := range values {
				placeholders[j] = fmt.Sprintf("$%d", argIndex)
				args = append(args, values[j])
				argIndex++
			}
			valueParts[i] = fmt.Sprintf("(%s)", strings.Join(placeholders, ", "))
		}

		parts = append(parts, strings.Join(valueParts, ", "))
		parts = append(parts, "RETURNING *")

		querySQL := strings.Join(parts, " ")
		rows, err := e.tx.QueryContext(ctx, querySQL, args...)
		if err != nil {
			return nil, fmt.Errorf("batch insert failed: %w", err)
		}
		defer rows.Close()

		// Scan all results
		for rows.Next() {
			record := reflect.New(reflect.TypeOf(data[0]).Elem()).Interface()
			columns, err := rows.Columns()
			if err != nil {
				return nil, fmt.Errorf("failed to get columns: %w", err)
			}
			if err := e.scanRowIntoStruct(rows, columns, record); err != nil {
				return nil, err
			}
			results = append(results, record)
		}

		return results, rows.Err()
	}

	// For other databases, insert one by one
	for _, record := range data {
		result, err := e.Create(ctx, table, record)
		if err != nil {
			return nil, fmt.Errorf("batch insert failed at record: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// UpdateMany executes batch UPDATE queries within a transaction
func (e *TxExecutor) UpdateMany(ctx context.Context, table string, set map[string]interface{}, where *sqlgen.WhereClause) (int64, error) {
	query := e.generator.GenerateUpdate(table, set, where)

	result, err := e.tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return 0, fmt.Errorf("batch update failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// DeleteMany executes batch DELETE queries within a transaction
func (e *TxExecutor) DeleteMany(ctx context.Context, table string, where *sqlgen.WhereClause) (int64, error) {
	query := e.generator.GenerateDelete(table, where)

	result, err := e.tx.ExecContext(ctx, query.SQL, query.Args...)
	if err != nil {
		return 0, fmt.Errorf("batch delete failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// quoteIdentifier quotes an identifier based on provider
func (e *TxExecutor) quoteIdentifier(name string) string {
	switch e.provider {
	case "postgresql", "postgres":
		return fmt.Sprintf(`"%s"`, name)
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "sqlite":
		return fmt.Sprintf(`"%s"`, name)
	default:
		return name
	}
}
