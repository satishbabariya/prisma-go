// Package builder implements the query builder.
package builder

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
)

// QueryBuilder implements the domain.QueryBuilder interface.
type QueryBuilder struct {
	model string
	query *domain.Query
}

// NewQueryBuilder creates a new query builder for a model.
func NewQueryBuilder(model string) *QueryBuilder {
	return &QueryBuilder{
		model: model,
		query: &domain.Query{
			Model:     model,
			Operation: domain.FindMany,
			Selection: domain.Selection{
				Fields:  []string{},
				Include: false,
			},
			Filter: domain.Filter{
				Conditions: []domain.Condition{},
				Operator:   domain.AND,
			},
			Relations:  []domain.RelationInclusion{},
			Ordering:   []domain.OrderBy{},
			Pagination: domain.Pagination{},
		},
	}
}

// FindMany sets the operation to FindMany.
func (b *QueryBuilder) FindMany() *QueryBuilder {
	b.query.Operation = domain.FindMany
	return b
}

// FindFirst sets the operation to FindFirst.
func (b *QueryBuilder) FindFirst() *QueryBuilder {
	b.query.Operation = domain.FindFirst
	return b
}

// FindUnique sets the operation to FindUnique.
func (b *QueryBuilder) FindUnique() *QueryBuilder {
	b.query.Operation = domain.FindUnique
	return b
}

// Delete marks the query as a DELETE operation.
func (b *QueryBuilder) Delete() *QueryBuilder {
	b.query.Operation = domain.Delete
	return b
}

// DeleteMany marks the query as a DELETE MANY operation.
func (b *QueryBuilder) DeleteMany() *QueryBuilder {
	b.query.Operation = domain.DeleteMany
	return b
}

// Create sets data for a CREATE operation.
func (b *QueryBuilder) Create(data map[string]interface{}) *QueryBuilder {
	b.query.Operation = domain.Create
	b.query.CreateData = data
	return b
}

// CreateMany sets data for a CREATE MANY operation.
func (b *QueryBuilder) CreateMany(data []map[string]interface{}) *QueryBuilder {
	b.query.Operation = domain.CreateMany
	b.query.CreateManyData = data
	return b
}

// Upsert sets data for an UPSERT operation (INSERT ON CONFLICT DO UPDATE).
func (b *QueryBuilder) Upsert(data map[string]interface{}, updateData map[string]interface{}, conflictKeys []string) *QueryBuilder {
	b.query.Operation = domain.Upsert
	b.query.UpsertData = data
	b.query.UpsertUpdate = updateData
	b.query.UpsertKeys = conflictKeys
	return b
}

// Update sets data for an UPDATE operation.
func (b *QueryBuilder) Update(data map[string]interface{}) *QueryBuilder {
	b.query.Operation = domain.Update
	b.query.UpdateData = data
	return b
}

// UpdateMany sets data for an UPDATE MANY operation.
func (b *QueryBuilder) UpdateMany(data map[string]interface{}) *QueryBuilder {
	b.query.Operation = domain.UpdateMany
	b.query.UpdateData = data
	return b
}

// Where adds filter conditions.
func (b *QueryBuilder) Where(conditions ...domain.Condition) *QueryBuilder {
	b.query.Filter.Conditions = append(b.query.Filter.Conditions, conditions...)
	return b
}

// And combines conditions with AND operator.
func (b *QueryBuilder) And(conditions ...domain.Condition) *QueryBuilder {
	b.query.Filter.Operator = domain.AND
	b.query.Filter.Conditions = append(b.query.Filter.Conditions, conditions...)
	return b
}

// Or combines conditions with OR operator.
func (b *QueryBuilder) Or(conditions ...domain.Condition) *QueryBuilder {
	b.query.Filter.Operator = domain.OR
	b.query.Filter.Conditions = append(b.query.Filter.Conditions, conditions...)
	return b
}

// Select specifies fields to select.
func (b *QueryBuilder) Select(fields ...string) *QueryBuilder {
	b.query.Selection.Fields = fields
	b.query.Selection.Include = false
	return b
}

// Include specifies relations to include.
func (b *QueryBuilder) Include(relations ...string) *QueryBuilder {
	for _, rel := range relations {
		b.query.Relations = append(b.query.Relations, domain.RelationInclusion{
			Relation: rel,
			Query:    nil,
		})
	}
	return b
}

// IncludeWith includes a relation with a nested query.
func (b *QueryBuilder) IncludeWith(relation string, nestedQuery *domain.Query) *QueryBuilder {
	b.query.Relations = append(b.query.Relations, domain.RelationInclusion{
		Relation: relation,
		Query:    nestedQuery,
	})
	return b
}

// OrderBy adds ordering.
func (b *QueryBuilder) OrderBy(field string, direction domain.SortDirection) *QueryBuilder {
	b.query.Ordering = append(b.query.Ordering, domain.OrderBy{
		Field:     field,
		Direction: direction,
	})
	return b
}

// Skip sets the number of records to skip.
func (b *QueryBuilder) Skip(skip int) *QueryBuilder {
	b.query.Pagination.Skip = &skip
	return b
}

// Take sets the number of records to take.
func (b *QueryBuilder) Take(take int) *QueryBuilder {
	b.query.Pagination.Take = &take
	return b
}

// Build builds the final query.
func (b *QueryBuilder) Build(ctx context.Context, query *domain.Query) (domain.SQL, error) {
	// This is called by the compiler, not directly by users
	// For now, return an error as this will be implemented in the compiler
	return domain.SQL{}, fmt.Errorf("use Compile() to generate SQL")
}

// GetQuery returns the built query.
func (b *QueryBuilder) GetQuery() *domain.Query {
	return b.query
}

// SetAggregations sets aggregations on the query (for internal use).
func (b *QueryBuilder) SetAggregations(aggs []domain.Aggregation) *QueryBuilder {
	b.query.Aggregations = aggs
	return b
}

// SetOperation sets the operation (for internal use).
func (b *QueryBuilder) SetOperation(op domain.QueryOperation) *QueryBuilder {
	b.query.Operation = op
	return b
}

// Ensure QueryBuilder implements the QueryBuilder interface.
var _ domain.QueryBuilder = (*QueryBuilder)(nil)
