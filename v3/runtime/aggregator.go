// Package runtime provides aggregation functionality for Prisma queries.
package runtime

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// AggregateResult holds the results of aggregation operations.
type AggregateResult struct {
	Count *int64                 `json:"_count,omitempty"`
	Avg   map[string]float64     `json:"_avg,omitempty"`
	Sum   map[string]interface{} `json:"_sum,omitempty"`
	Min   map[string]interface{} `json:"_min,omitempty"`
	Max   map[string]interface{} `json:"_max,omitempty"`
}

// Aggregator provides aggregation operations.
type Aggregator struct {
	model    string
	executor *QueryExecutor
	filter   domain.Filter
	groupBy  []string
	having   domain.Filter
}

// NewAggregator creates a new aggregator.
func NewAggregator(model string, executor *QueryExecutor) *Aggregator {
	return &Aggregator{
		model:    model,
		executor: executor,
		filter:   domain.Filter{Operator: domain.AND},
	}
}

// Where adds filter conditions to the aggregation.
func (a *Aggregator) Where(conditions ...domain.Condition) *Aggregator {
	a.filter.Conditions = append(a.filter.Conditions, conditions...)
	return a
}

// GroupBy adds GROUP BY fields.
func (a *Aggregator) GroupBy(fields ...string) *Aggregator {
	a.groupBy = append(a.groupBy, fields...)
	return a
}

// Having adds HAVING conditions for grouped results.
func (a *Aggregator) Having(conditions ...domain.Condition) *Aggregator {
	a.having.Conditions = append(a.having.Conditions, conditions...)
	return a
}

// Count executes a COUNT aggregation.
func (a *Aggregator) Count(ctx context.Context) (int64, error) {
	query := &domain.Query{
		Model:     a.model,
		Operation: domain.Aggregate,
		Filter:    a.filter,
		GroupBy:   a.groupBy,
		Having:    a.having,
		Aggregations: []domain.Aggregation{
			{Function: "COUNT", Field: "*"},
		},
	}

	result, err := a.executor.ExecuteAggregate(ctx, query)
	if err != nil {
		return 0, err
	}

	// Extract count from result
	if countVal, ok := result["count"]; ok {
		switch v := countVal.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case float64:
			return int64(v), nil
		}
	}

	return 0, fmt.Errorf("count result not found in aggregation")
}

// Avg calculates the average of specified fields.
func (a *Aggregator) Avg(ctx context.Context, fields ...string) (map[string]float64, error) {
	aggregations := make([]domain.Aggregation, len(fields))
	for i, field := range fields {
		aggregations[i] = domain.Aggregation{
			Function: "AVG",
			Field:    field,
		}
	}

	query := &domain.Query{
		Model:        a.model,
		Operation:    domain.Aggregate,
		Filter:       a.filter,
		GroupBy:      a.groupBy,
		Having:       a.having,
		Aggregations: aggregations,
	}

	result, err := a.executor.ExecuteAggregate(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse average results
	avgResults := make(map[string]float64)
	for _, field := range fields {
		avgKey := fmt.Sprintf("avg_%s", field)
		if val, ok := result[avgKey]; ok {
			switch v := val.(type) {
			case float64:
				avgResults[field] = v
			case int64:
				avgResults[field] = float64(v)
			case int:
				avgResults[field] = float64(v)
			}
		}
	}

	return avgResults, nil
}

// Sum calculates the sum of specified fields.
func (a *Aggregator) Sum(ctx context.Context, fields ...string) (map[string]interface{}, error) {
	aggregations := make([]domain.Aggregation, len(fields))
	for i, field := range fields {
		aggregations[i] = domain.Aggregation{
			Function: "SUM",
			Field:    field,
		}
	}

	query := &domain.Query{
		Model:        a.model,
		Operation:    domain.Aggregate,
		Filter:       a.filter,
		GroupBy:      a.groupBy,
		Having:       a.having,
		Aggregations: aggregations,
	}

	result, err := a.executor.ExecuteAggregate(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse sum results
	sumResults := make(map[string]interface{})
	for _, field := range fields {
		sumKey := fmt.Sprintf("sum_%s", field)
		if val, ok := result[sumKey]; ok {
			sumResults[field] = val
		}
	}

	return sumResults, nil
}

// Min finds the minimum value of specified fields.
func (a *Aggregator) Min(ctx context.Context, fields ...string) (map[string]interface{}, error) {
	aggregations := make([]domain.Aggregation, len(fields))
	for i, field := range fields {
		aggregations[i] = domain.Aggregation{
			Function: "MIN",
			Field:    field,
		}
	}

	query := &domain.Query{
		Model:        a.model,
		Operation:    domain.Aggregate,
		Filter:       a.filter,
		GroupBy:      a.groupBy,
		Having:       a.having,
		Aggregations: aggregations,
	}

	result, err := a.executor.ExecuteAggregate(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse min results
	minResults := make(map[string]interface{})
	for _, field := range fields {
		minKey := fmt.Sprintf("min_%s", field)
		if val, ok := result[minKey]; ok {
			minResults[field] = val
		}
	}

	return minResults, nil
}

// Max finds the maximum value of specified fields.
func (a *Aggregator) Max(ctx context.Context, fields ...string) (map[string]interface{}, error) {
	aggregations := make([]domain.Aggregation, len(fields))
	for i, field := range fields {
		aggregations[i] = domain.Aggregation{
			Function: "MAX",
			Field:    field,
		}
	}

	query := &domain.Query{
		Model:        a.model,
		Operation:    domain.Aggregate,
		Filter:       a.filter,
		GroupBy:      a.groupBy,
		Having:       a.having,
		Aggregations: aggregations,
	}

	result, err := a.executor.ExecuteAggregate(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse max results
	maxResults := make(map[string]interface{})
	for _, field := range fields {
		maxKey := fmt.Sprintf("max_%s", field)
		if val, ok := result[maxKey]; ok {
			maxResults[field] = val
		}
	}

	return maxResults, nil
}

// Aggregate executes all aggregations at once.
func (a *Aggregator) Aggregate(ctx context.Context, opts AggregateOptions) (*AggregateResult, error) {
	var aggregations []domain.Aggregation

	// Add count if requested
	if opts.Count {
		aggregations = append(aggregations, domain.Aggregation{
			Function: "COUNT",
			Field:    "*",
		})
	}

	// Add avg for specified fields
	for _, field := range opts.AvgFields {
		aggregations = append(aggregations, domain.Aggregation{
			Function: "AVG",
			Field:    field,
		})
	}

	// Add sum for specified fields
	for _, field := range opts.SumFields {
		aggregations = append(aggregations, domain.Aggregation{
			Function: "SUM",
			Field:    field,
		})
	}

	// Add min for specified fields
	for _, field := range opts.MinFields {
		aggregations = append(aggregations, domain.Aggregation{
			Function: "MIN",
			Field:    field,
		})
	}

	// Add max for specified fields
	for _, field := range opts.MaxFields {
		aggregations = append(aggregations, domain.Aggregation{
			Function: "MAX",
			Field:    field,
		})
	}

	query := &domain.Query{
		Model:        a.model,
		Operation:    domain.Aggregate,
		Filter:       a.filter,
		GroupBy:      a.groupBy,
		Having:       a.having,
		Aggregations: aggregations,
	}

	result, err := a.executor.ExecuteAggregate(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse results into AggregateResult
	aggResult := &AggregateResult{
		Avg: make(map[string]float64),
		Sum: make(map[string]interface{}),
		Min: make(map[string]interface{}),
		Max: make(map[string]interface{}),
	}

	// Extract count
	if opts.Count {
		if countVal, ok := result["count"]; ok {
			count := int64(0)
			switch v := countVal.(type) {
			case int64:
				count = v
			case int:
				count = int64(v)
			case float64:
				count = int64(v)
			}
			aggResult.Count = &count
		}
	}

	// Extract avg values
	for _, field := range opts.AvgFields {
		avgKey := fmt.Sprintf("avg_%s", field)
		if val, ok := result[avgKey]; ok {
			switch v := val.(type) {
			case float64:
				aggResult.Avg[field] = v
			case int64:
				aggResult.Avg[field] = float64(v)
			case int:
				aggResult.Avg[field] = float64(v)
			}
		}
	}

	// Extract sum, min, max values
	for _, field := range opts.SumFields {
		sumKey := fmt.Sprintf("sum_%s", field)
		if val, ok := result[sumKey]; ok {
			aggResult.Sum[field] = val
		}
	}

	for _, field := range opts.MinFields {
		minKey := fmt.Sprintf("min_%s", field)
		if val, ok := result[minKey]; ok {
			aggResult.Min[field] = val
		}
	}

	for _, field := range opts.MaxFields {
		maxKey := fmt.Sprintf("max_%s", field)
		if val, ok := result[maxKey]; ok {
			aggResult.Max[field] = val
		}
	}

	return aggResult, nil
}

// AggregateOptions specifies what aggregations to perform.
type AggregateOptions struct {
	Count     bool
	AvgFields []string
	SumFields []string
	MinFields []string
	MaxFields []string
}
