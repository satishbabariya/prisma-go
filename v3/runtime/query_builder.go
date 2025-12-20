// Package runtime provides query builders for generated Prisma clients.
package runtime

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// QueryBuilder provides Prisma-like query functionality.
type QueryBuilder struct {
	client          *Client
	model           string
	ctx             context.Context
	operation       domain.QueryOperation
	data            interface{}
	where           domain.Filter
	orderBy         []domain.OrderBy
	pagination      domain.Pagination
	cursor          *domain.Cursor
	include         []domain.RelationInclusion
	selects         domain.Selection
	aggregations    []domain.Aggregation
	groupBy         []string
	having          domain.Filter
	distinct        []string
	throwIfNotFound bool
}

// NewQueryBuilder creates a new query builder for a model.
func NewQueryBuilder(client *Client, model string) *QueryBuilder {
	return &QueryBuilder{
		client:  client,
		model:   model,
		ctx:     context.Background(),
		where:   domain.Filter{},
		selects: domain.Selection{},
	}
}

// WithContext sets the context for the query.
func (q *QueryBuilder) WithContext(ctx context.Context) *QueryBuilder {
	q.ctx = ctx
	return q
}

// FindMany configures the query to find multiple records.
func (q *QueryBuilder) FindMany() *QueryBuilder {
	q.operation = domain.FindMany
	return q
}

// FindFirst configures the query to find the first matching record.
func (q *QueryBuilder) FindFirst() *QueryBuilder {
	q.operation = domain.FindFirst
	q.pagination = domain.Pagination{Take: &[]int{1}[0]}
	return q
}

// FindUnique configures the query to find a unique record.
func (q *QueryBuilder) FindUnique() *QueryBuilder {
	q.operation = domain.FindUnique
	return q
}

// Create configures the query for creating records.
func (q *QueryBuilder) Create(data map[string]interface{}) *QueryBuilder {
	q.operation = domain.Create
	q.data = data
	return q
}

// CreateMany configures the query for creating multiple records.
func (q *QueryBuilder) CreateMany(data []map[string]interface{}) *QueryBuilder {
	q.operation = domain.CreateMany
	// Store as interface{} to handle different data types
	q.data = interface{}(data)
	return q
}

// Update configures the query for updating records.
func (q *QueryBuilder) Update(data map[string]interface{}) *QueryBuilder {
	q.operation = domain.Update
	q.data = data
	return q
}

// UpdateMany configures the query for updating multiple records.
func (q *QueryBuilder) UpdateMany(data map[string]interface{}) *QueryBuilder {
	q.operation = domain.UpdateMany
	q.data = data
	return q
}

// Delete configures the query for deleting records.
func (q *QueryBuilder) Delete() *QueryBuilder {
	q.operation = domain.Delete
	return q
}

// DeleteMany configures the query for deleting multiple records.
func (q *QueryBuilder) DeleteMany() *QueryBuilder {
	q.operation = domain.DeleteMany
	return q
}

// Upsert configures the query for upserting records.
func (q *QueryBuilder) Upsert(create, update map[string]interface{}) *QueryBuilder {
	q.operation = domain.Upsert
	q.data = map[string]interface{}{
		"create": create,
		"update": update,
	}
	return q
}

// Aggregate configures the query for aggregation.
func (q *QueryBuilder) Aggregate(aggregations ...domain.Aggregation) *QueryBuilder {
	q.operation = domain.Aggregate
	q.aggregations = aggregations
	return q
}

// GroupBy configures the query for grouping.
func (q *QueryBuilder) GroupBy(fields ...string) *QueryBuilder {
	q.groupBy = fields
	return q
}

// Where adds WHERE conditions to the query.
func (q *QueryBuilder) Where(conditions ...domain.Condition) *QueryBuilder {
	q.where.Conditions = append(q.where.Conditions, conditions...)
	return q
}

// And adds AND conditions to the query.
func (q *QueryBuilder) And(conditions ...domain.Condition) *QueryBuilder {
	// TODO: Implement logical AND
	q.where.Conditions = append(q.where.Conditions, conditions...)
	return q
}

// Or adds OR conditions to the query.
func (q *QueryBuilder) Or(conditions ...domain.Condition) *QueryBuilder {
	// TODO: Implement logical OR
	q.where.Conditions = append(q.where.Conditions, conditions...)
	return q
}

// Not adds NOT conditions to the query.
func (q *QueryBuilder) Not(conditions ...domain.Condition) *QueryBuilder {
	// TODO: Implement logical NOT
	q.where.Conditions = append(q.where.Conditions, conditions...)
	return q
}

// OrderBy adds ordering to the query.
func (q *QueryBuilder) OrderBy(order ...domain.OrderBy) *QueryBuilder {
	q.orderBy = append(q.orderBy, order...)
	return q
}

// OrderByAsc adds ascending order by field.
func (q *QueryBuilder) OrderByAsc(field string) *QueryBuilder {
	q.orderBy = append(q.orderBy, domain.OrderBy{
		Field:     field,
		Direction: domain.Asc,
	})
	return q
}

// OrderByDesc adds descending order by field.
func (q *QueryBuilder) OrderByDesc(field string) *QueryBuilder {
	q.orderBy = append(q.orderBy, domain.OrderBy{
		Field:     field,
		Direction: domain.Desc,
	})
	return q
}

// Skip adds pagination offset.
func (q *QueryBuilder) Skip(n int) *QueryBuilder {
	if q.pagination.Skip == nil {
		q.pagination.Skip = &n
	} else {
		*q.pagination.Skip = n
	}
	return q
}

// Take adds pagination limit.
func (q *QueryBuilder) Take(n int) *QueryBuilder {
	if q.pagination.Take == nil {
		q.pagination.Take = &n
	} else {
		*q.pagination.Take = n
	}
	return q
}

// Cursor adds cursor-based pagination.
func (q *QueryBuilder) Cursor(field string, value interface{}) *QueryBuilder {
	q.cursor = &domain.Cursor{
		Field: field,
		Value: value,
	}
	return q
}

// Include adds relations to include in the query.
func (q *QueryBuilder) Include(relations ...domain.RelationInclusion) *QueryBuilder {
	q.include = append(q.include, relations...)
	return q
}

// Select specifies which fields to select.
func (q *QueryBuilder) Select(fields ...string) *QueryBuilder {
	// For now, store field names as strings
	q.selects.Fields = append(q.selects.Fields, fields...)
	return q
}

// Distinct adds DISTINCT ON clause.
func (q *QueryBuilder) Distinct(fields ...string) *QueryBuilder {
	q.distinct = append(q.distinct, fields...)
	return q
}

// Having adds HAVING clause for GROUP BY queries.
func (q *QueryBuilder) Having(conditions ...domain.Condition) *QueryBuilder {
	q.having.Conditions = append(q.having.Conditions, conditions...)
	return q
}

// ThrowIfNotFound enables throwing error if no records found.
func (q *QueryBuilder) ThrowIfNotFound() *QueryBuilder {
	q.throwIfNotFound = true
	return q
}

// Execute executes the query and returns results.
func (q *QueryBuilder) Execute() (interface{}, error) {
	if q.client == nil || q.client.db == nil {
		return nil, fmt.Errorf("client not connected")
	}

	// Build the domain query
	domainQuery := &domain.Query{
		Model:           q.model,
		Operation:       q.operation,
		Selection:       q.selects,
		Filter:          q.where,
		Relations:       q.include,
		Ordering:        q.orderBy,
		Pagination:      q.pagination,
		Cursor:          q.cursor,
		Aggregations:    q.aggregations,
		GroupBy:         q.groupBy,
		Having:          q.having,
		Distinct:        q.distinct,
		ThrowIfNotFound: q.throwIfNotFound,
	}

	// Add create/update data
	switch q.operation {
	case domain.Create:
		if data, ok := q.data.(map[string]interface{}); ok {
			domainQuery.CreateData = data
		}
	case domain.Update, domain.UpdateMany:
		if data, ok := q.data.(map[string]interface{}); ok {
			domainQuery.UpdateData = data
		}
	case domain.Upsert:
		if upsertData, ok := q.data.(map[string]interface{}); ok {
			if create, ok := upsertData["create"].(map[string]interface{}); ok {
				domainQuery.UpsertData = create
			}
			if update, ok := upsertData["update"].(map[string]interface{}); ok {
				domainQuery.UpsertUpdate = update
			}
		}
	case domain.CreateMany:
		if data, ok := q.data.([]map[string]interface{}); ok {
			domainQuery.CreateManyData = data
		}
	}

	// Compile to SQL
	compiler := compiler.NewSQLCompiler(domain.PostgreSQL) // Should detect from config
	compiled, err := compiler.Compile(q.ctx, domainQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	// Execute based on operation
	return q.executeOperation(q.ctx, compiled)
}

// executeOperation executes the compiled query based on operation type.
func (q *QueryBuilder) executeOperation(ctx context.Context, compiled *domain.CompiledQuery) (interface{}, error) {
	db := q.client.db

	switch q.operation {
	case domain.FindMany, domain.FindFirst, domain.FindUnique, domain.Aggregate:
		rows, err := db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		defer rows.Close()

		// For now, return raw results
		return q.scanRows(rows)

	case domain.Create:
		result, err := db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute create: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get insert ID: %w", err)
		}

		return map[string]interface{}{"id": id}, nil

	case domain.CreateMany:
		result, err := db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute create many: %w", err)
		}

		count, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}

		return map[string]interface{}{"count": count}, nil

	case domain.Update, domain.UpdateMany:
		result, err := db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute update: %w", err)
		}

		count, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}

		return map[string]interface{}{"count": count}, nil

	case domain.Delete, domain.DeleteMany:
		result, err := db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute delete: %w", err)
		}

		count, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to get rows affected: %w", err)
		}

		return map[string]interface{}{"count": count}, nil

	default:
		return nil, fmt.Errorf("unsupported operation: %s", q.operation)
	}
}

// scanRows scans SQL rows into a slice of maps.
func (q *QueryBuilder) scanRows(rows *sql.Rows) (interface{}, error) {
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
			b, ok := val.([]byte)
			if ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// For FindFirst/FindUnique, return single result
	if q.operation == domain.FindFirst || q.operation == domain.FindUnique {
		if len(results) == 0 {
			if q.throwIfNotFound {
				return nil, fmt.Errorf("record not found")
			}
			return nil, nil
		}
		return results[0], nil
	}

	return results, nil
}

// Count executes a count query.
func (q *QueryBuilder) Count() (int64, error) {
	aggregations := []domain.Aggregation{
		{Function: domain.Count, Field: "id"},
	}
	q.aggregations = aggregations

	result, err := q.Execute()
	if err != nil {
		return 0, err
	}

	// Extract count from result
	if resultMap, ok := result.(map[string]interface{}); ok {
		if count, ok := resultMap["count"].(int64); ok {
			return count, nil
		}
	}

	return 0, fmt.Errorf("failed to get count from result")
}

// Transaction methods

// NestedCreate adds nested create operation.
func (q *QueryBuilder) NestedCreate(relation string, data map[string]interface{}) *QueryBuilder {
	// This would be implemented for nested writes
	return q
}

// NestedConnect adds nested connect operation.
func (q *QueryBuilder) NestedConnect(relation string, conditions ...domain.Condition) *QueryBuilder {
	// This would be implemented for nested writes
	return q
}

// NestedDisconnect adds nested disconnect operation.
func (q *QueryBuilder) NestedDisconnect(relation string, conditions ...domain.Condition) *QueryBuilder {
	// This would be implemented for nested writes
	return q
}

// NestedUpdate adds nested update operation.
func (q *QueryBuilder) NestedUpdate(relation string, data map[string]interface{}) *QueryBuilder {
	// This would be implemented for nested writes
	return q
}

// NestedDelete adds nested delete operation.
func (q *QueryBuilder) NestedDelete(relation string, conditions ...domain.Condition) *QueryBuilder {
	// This would be implemented for nested writes
	return q
}

// NestedSet adds nested set operation.
func (q *QueryBuilder) NestedSet(relation string, conditions ...domain.Condition) *QueryBuilder {
	// This would be implemented for nested writes
	return q
}

// Batch operations - use runtime.Batch instead

// Raw SQL operations

// QueryRaw executes a raw SQL query.
func (q *QueryBuilder) QueryRaw(query string, args ...interface{}) (interface{}, error) {
	if q.client == nil || q.client.db == nil {
		return nil, fmt.Errorf("client not connected")
	}

	rows, err := q.client.db.QueryContext(q.ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw query: %w", err)
	}
	defer rows.Close()

	return q.scanRows(rows)
}

// ExecuteRaw executes a raw SQL statement.
func (q *QueryBuilder) ExecuteRaw(query string, args ...interface{}) (int64, error) {
	if q.client == nil || q.client.db == nil {
		return 0, fmt.Errorf("client not connected")
	}

	result, err := q.client.db.ExecContext(q.ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute raw statement: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return affected, nil
}
