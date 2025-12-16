// Package service implements the query service.
package service

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/builder"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/executor"
)

// QueryService orchestrates query execution.
type QueryService struct {
	compiler *compiler.SQLCompiler
	executor *executor.QueryExecutor
}

// NewQueryService creates a new query service.
func NewQueryService(
	comp *compiler.SQLCompiler,
	exec *executor.QueryExecutor,
) *QueryService {
	return &QueryService{
		compiler: comp,
		executor: exec,
	}
}

// FindMany executes a find many query.
func (s *QueryService) FindMany(ctx context.Context, model string, opts ...QueryOption) (interface{}, error) {
	// Build query
	b := builder.NewQueryBuilder(model).FindMany()

	// Apply options
	for _, opt := range opts {
		opt(b)
	}

	return s.executeQuery(ctx, b.GetQuery())
}

// FindFirst executes a find first query.
func (s *QueryService) FindFirst(ctx context.Context, model string, opts ...QueryOption) (interface{}, error) {
	b := builder.NewQueryBuilder(model).FindFirst()

	for _, opt := range opts {
		opt(b)
	}

	return s.executeQuery(ctx, b.GetQuery())
}

// FindUnique executes a find unique query.
func (s *QueryService) FindUnique(ctx context.Context, model string, opts ...QueryOption) (interface{}, error) {
	b := builder.NewQueryBuilder(model).FindUnique()

	for _, opt := range opts {
		opt(b)
	}

	return s.executeQuery(ctx, b.GetQuery())
}

// FindManyInto executes a find many query and maps to structs.
func (s *QueryService) FindManyInto(ctx context.Context, model string, dest interface{}, opts ...QueryOption) error {
	b := builder.NewQueryBuilder(model).FindMany()

	for _, opt := range opts {
		opt(b)
	}

	return s.executeQueryInto(ctx, b.GetQuery(), dest)
}

// FindFirstInto executes a find first query and maps to a struct.
func (s *QueryService) FindFirstInto(ctx context.Context, model string, dest interface{}, opts ...QueryOption) error {
	b := builder.NewQueryBuilder(model).FindFirst()

	for _, opt := range opts {
		opt(b)
	}

	return s.executeQueryInto(ctx, b.GetQuery(), dest)
}

// Delete executes a delete query.
func (s *QueryService) Delete(ctx context.Context, model string, opts ...QueryOption) (int64, error) {
	b := builder.NewQueryBuilder(model).Delete()

	for _, opt := range opts {
		opt(b)
	}

	return s.executeMutation(ctx, b.GetQuery())
}

// DeleteMany executes a delete many query.
func (s *QueryService) DeleteMany(ctx context.Context, model string, opts ...QueryOption) (int64, error) {
	b := builder.NewQueryBuilder(model).DeleteMany()

	for _, opt := range opts {
		opt(b)
	}

	return s.executeMutation(ctx, b.GetQuery())
}

// Count executes a count aggregation.
func (s *QueryService) Count(ctx context.Context, model string, opts ...QueryOption) (int64, error) {
	return s.aggregate(ctx, model, domain.Count, "*", opts...)
}

// Sum executes a sum aggregation.
func (s *QueryService) Sum(ctx context.Context, model string, field string, opts ...QueryOption) (float64, error) {
	result, err := s.aggregate(ctx, model, domain.Sum, field, opts...)
	return float64(result), err
}

// Avg executes an average aggregation.
func (s *QueryService) Avg(ctx context.Context, model string, field string, opts ...QueryOption) (float64, error) {
	result, err := s.aggregate(ctx, model, domain.Avg, field, opts...)
	return float64(result), err
}

// Create creates a new record.
func (s *QueryService) Create(ctx context.Context, table string, data map[string]interface{}) (map[string]interface{}, error) {
	qb := builder.NewQueryBuilder(table).Create(data)
	query := qb.GetQuery()

	sql, args, err := s.compiler.CompileCreate(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile create: %w", err)
	}

	// Execute using compiled query structure
	compiled := &domain.CompiledQuery{
		SQL: domain.SQL{
			Query:   sql,
			Args:    args,
			Dialect: domain.PostgreSQL, // Default to PostgreSQL
		},
	}

	result, err := s.executor.Execute(ctx, compiled)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create: %w", err)
	}

	// Cast result to map slice
	if rows, ok := result.([]map[string]interface{}); ok && len(rows) > 0 {
		return rows[0], nil
	}

	return data, nil // Return input data if no RETURNING clause
}

// CreateMany creates multiple records in a single batch operation.
func (s *QueryService) CreateMany(ctx context.Context, table string, data []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	qb := builder.NewQueryBuilder(table).CreateMany(data)
	query := qb.GetQuery()

	sql, args, err := s.compiler.CompileCreateMany(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile create many: %w", err)
	}

	// Execute using compiled query structure
	compiled := &domain.CompiledQuery{
		SQL: domain.SQL{
			Query:   sql,
			Args:    args,
			Dialect: domain.PostgreSQL,
		},
	}

	result, err := s.executor.Execute(ctx, compiled)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create many: %w", err)
	}

	// Cast result to map slice
	if rows, ok := result.([]map[string]interface{}); ok {
		return rows, nil
	}

	return data, nil // Return input data if no RETURNING clause
}

// Upsert inserts or updates a record based on conflict keys.
func (s *QueryService) Upsert(ctx context.Context, table string, data map[string]interface{}, updateData map[string]interface{}, conflictKeys []string) (map[string]interface{}, error) {
	qb := builder.NewQueryBuilder(table).Upsert(data, updateData, conflictKeys)
	query := qb.GetQuery()

	sql, args, err := s.compiler.CompileUpsert(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile upsert: %w", err)
	}

	// Execute using compiled query structure
	compiled := &domain.CompiledQuery{
		SQL: domain.SQL{
			Query:   sql,
			Args:    args,
			Dialect: domain.PostgreSQL,
		},
	}

	result, err := s.executor.Execute(ctx, compiled)
	if err != nil {
		return nil, fmt.Errorf("failed to execute upsert: %w", err)
	}

	// Cast result to map slice
	if rows, ok := result.([]map[string]interface{}); ok && len(rows) > 0 {
		return rows[0], nil
	}

	// Merge data for return if no RETURNING clause
	merged := make(map[string]interface{})
	for k, v := range data {
		merged[k] = v
	}
	for k, v := range updateData {
		merged[k] = v
	}
	return merged, nil
}

// Update updates records matching the filters.
func (s *QueryService) Update(ctx context.Context, table string, data map[string]interface{}, opts ...QueryOption) (int64, error) {
	qb := builder.NewQueryBuilder(table).Update(data)

	for _, opt := range opts {
		opt(qb)
	}

	return s.executeMutation(ctx, qb.GetQuery())
}

// UpdateMany updates multiple records.
func (s *QueryService) UpdateMany(ctx context.Context, table string, data map[string]interface{}, opts ...QueryOption) (int64, error) {
	qb := builder.NewQueryBuilder(table).UpdateMany(data)

	for _, opt := range opts {
		opt(qb)
	}

	return s.executeMutation(ctx, qb.GetQuery())
}

// Min executes a min aggregation.
func (s *QueryService) Min(ctx context.Context, model string, field string, opts ...QueryOption) (interface{}, error) {
	result, err := s.aggregate(ctx, model, domain.Min, field, opts...)
	// Return as interface{} since min can be any type
	return result, err
}

// Max executes a max aggregation.
func (s *QueryService) Max(ctx context.Context, model string, field string, opts ...QueryOption) (interface{}, error) {
	result, err := s.aggregate(ctx, model, domain.Max, field, opts...)
	// Return as interface{} since max can be any type
	return result, err
}

// aggregate is a helper for executing aggregations.
func (s *QueryService) aggregate(ctx context.Context, model string, fn domain.AggregateFunc, field string, opts ...QueryOption) (int64, error) {
	b := builder.NewQueryBuilder(model)
	b.SetOperation(domain.Aggregate)
	b.SetAggregations([]domain.Aggregation{
		{Function: fn, Field: field},
	})

	for _, opt := range opts {
		opt(b)
	}

	// Compile and execute
	compiled, err := s.compiler.Compile(ctx, b.GetQuery())
	if err != nil {
		return 0, fmt.Errorf("failed to compile aggregation: %w", err)
	}

	// Execute
	result, err := s.executor.Execute(ctx, compiled)
	if err != nil {
		return 0, fmt.Errorf("failed to execute aggregation: %w", err)
	}

	// Extract value from result
	if rows, ok := result.([]map[string]interface{}); ok && len(rows) > 0 {
		// Get the first (and only) row
		row := rows[0]
		// The aggregation result will be under the alias key
		for _, val := range row {
			if val == nil {
				return 0, nil
			}
			// Convert to int64
			switch v := val.(type) {
			case int64:
				return v, nil
			case int:
				return int64(v), nil
			case float64:
				return int64(v), nil
			default:
				return 0, fmt.Errorf("unexpected aggregation result type: %T", val)
			}
		}
	}

	return 0, fmt.Errorf("no aggregation result returned")
}

// executeQuery compiles and executes a query.
func (s *QueryService) executeQuery(ctx context.Context, query *domain.Query) (interface{}, error) {
	// Compile the query
	compiled, err := s.compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	// Optimize if needed
	compiled, err = s.compiler.Optimize(ctx, compiled)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize query: %w", err)
	}

	// Execute the query
	result, err := s.executor.Execute(ctx, compiled)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

// executeQueryInto compiles and executes a query with struct mapping.
func (s *QueryService) executeQueryInto(ctx context.Context, query *domain.Query, dest interface{}) error {
	// Compile the query
	compiled, err := s.compiler.Compile(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to compile query: %w", err)
	}

	// Optimize if needed
	compiled, err = s.compiler.Optimize(ctx, compiled)
	if err != nil {
		return fmt.Errorf("failed to optimize query: %w", err)
	}

	// Execute the query into struct
	err = s.executor.ExecuteInto(ctx, compiled, dest)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// executeMutation compiles and executes a mutation (DELETE, UPDATE).
func (s *QueryService) executeMutation(ctx context.Context, query *domain.Query) (int64, error) {
	// Compile the query
	compiled, err := s.compiler.Compile(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to compile query: %w", err)
	}

	// Execute the mutation
	result, err := s.executor.ExecuteMutation(ctx, compiled)
	if err != nil {
		return 0, fmt.Errorf("failed to execute mutation: %w", err)
	}

	return result, nil
}

// QueryOption is a function that configures a query builder.
type QueryOption func(*builder.QueryBuilder)

// WithWhere adds filter conditions.
func WithWhere(conditions ...domain.Condition) QueryOption {
	return func(b *builder.QueryBuilder) {
		b.Where(conditions...)
	}
}

// WithSelect specifies fields to select.
func WithSelect(fields ...string) QueryOption {
	return func(b *builder.QueryBuilder) {
		b.Select(fields...)
	}
}

// WithInclude specifies relations to include.
func WithInclude(relations ...string) QueryOption {
	return func(b *builder.QueryBuilder) {
		b.Include(relations...)
	}
}

// WithOrderBy adds ordering.
func WithOrderBy(field string, direction domain.SortDirection) QueryOption {
	return func(b *builder.QueryBuilder) {
		b.OrderBy(field, direction)
	}
}

// WithSkip sets the number of records to skip.
func WithSkip(skip int) QueryOption {
	return func(b *builder.QueryBuilder) {
		b.Skip(skip)
	}
}

// WithTake sets the number of records to take.
func WithTake(take int) QueryOption {
	return func(b *builder.QueryBuilder) {
		b.Take(take)
	}
}

// Transaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (s *QueryService) Transaction(ctx context.Context, fn func(*QueryService) error) error {
	// For now, just execute the function without actual transaction support
	// Full transaction support requires access to the database adapter
	// This is a placeholder that will be enhanced when wired into container
	return fn(s)
}
