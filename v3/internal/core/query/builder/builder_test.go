package builder_test

import (
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/builder"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryBuilder_FindMany(t *testing.T) {
	b := builder.NewQueryBuilder("users").FindMany()
	query := b.GetQuery()

	assert.Equal(t, "users", query.Model)
	assert.Equal(t, domain.FindMany, query.Operation)
}

func TestQueryBuilder_FindFirst(t *testing.T) {
	b := builder.NewQueryBuilder("users").FindFirst()
	query := b.GetQuery()

	assert.Equal(t, "users", query.Model)
	assert.Equal(t, domain.FindFirst, query.Operation)
}

func TestQueryBuilder_Where(t *testing.T) {
	tests := []struct {
		name       string
		conditions []domain.Condition
		want       int
	}{
		{
			name: "single condition",
			conditions: []domain.Condition{
				{Field: "id", Operator: domain.Equals, Value: 1},
			},
			want: 1,
		},
		{
			name: "multiple conditions",
			conditions: []domain.Condition{
				{Field: "id", Operator: domain.Equals, Value: 1},
				{Field: "email", Operator: domain.Contains, Value: "test"},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := builder.NewQueryBuilder("users").
				FindMany().
				Where(tt.conditions...)

			query := b.GetQuery()
			assert.Len(t, query.Filter.Conditions, tt.want)
		})
	}
}

func TestQueryBuilder_OrderBy(t *testing.T) {
	b := builder.NewQueryBuilder("users").
		FindMany().
		OrderBy("created_at", domain.Desc)

	query := b.GetQuery()
	require.Len(t, query.Ordering, 1)
	assert.Equal(t, "created_at", query.Ordering[0].Field)
	assert.Equal(t, domain.Desc, query.Ordering[0].Direction)
}

func TestQueryBuilder_Pagination(t *testing.T) {
	b := builder.NewQueryBuilder("users").
		FindMany().
		Skip(10).
		Take(20)

	query := b.GetQuery()
	require.NotNil(t, query.Pagination.Skip)
	require.NotNil(t, query.Pagination.Take)
	assert.Equal(t, 10, *query.Pagination.Skip)
	assert.Equal(t, 20, *query.Pagination.Take)
}

func TestQueryBuilder_Create(t *testing.T) {
	data := map[string]interface{}{
		"email": "test@example.com",
		"name":  "Test User",
	}

	b := builder.NewQueryBuilder("users").Create(data)
	query := b.GetQuery()

	assert.Equal(t, domain.Create, query.Operation)
	assert.Equal(t, data, query.CreateData)
}

func TestQueryBuilder_CreateMany(t *testing.T) {
	data := []map[string]interface{}{
		{"email": "user1@example.com", "name": "User 1"},
		{"email": "user2@example.com", "name": "User 2"},
	}

	b := builder.NewQueryBuilder("users").CreateMany(data)
	query := b.GetQuery()

	assert.Equal(t, domain.CreateMany, query.Operation)
	assert.Equal(t, data, query.CreateManyData)
}

func TestQueryBuilder_Update(t *testing.T) {
	data := map[string]interface{}{
		"name": "Updated Name",
	}

	b := builder.NewQueryBuilder("users").
		Update(data).
		Where(domain.Condition{Field: "id", Operator: domain.Equals, Value: 1})

	query := b.GetQuery()

	assert.Equal(t, domain.Update, query.Operation)
	assert.Equal(t, data, query.UpdateData)
	assert.Len(t, query.Filter.Conditions, 1)
}

func TestQueryBuilder_Delete(t *testing.T) {
	b := builder.NewQueryBuilder("users").
		Delete().
		Where(domain.Condition{Field: "id", Operator: domain.Equals, Value: 1})

	query := b.GetQuery()

	assert.Equal(t, domain.Delete, query.Operation)
	assert.Len(t, query.Filter.Conditions, 1)
}

func TestQueryBuilder_Upsert(t *testing.T) {
	data := map[string]interface{}{
		"email": "test@example.com",
		"name":  "Test User",
	}
	updateData := map[string]interface{}{
		"name": "Updated User",
	}
	conflictKeys := []string{"email"}

	b := builder.NewQueryBuilder("users").Upsert(data, updateData, conflictKeys)
	query := b.GetQuery()

	assert.Equal(t, domain.Upsert, query.Operation)
	assert.Equal(t, data, query.UpsertData)
	assert.Equal(t, updateData, query.UpsertUpdate)
	assert.Equal(t, conflictKeys, query.UpsertKeys)
}

func TestQueryBuilder_Aggregations(t *testing.T) {
	b := builder.NewQueryBuilder("users")
	b.SetOperation(domain.Aggregate)
	b.SetAggregations([]domain.Aggregation{
		{Function: domain.Count, Field: "*"},
	})

	query := b.GetQuery()

	assert.Equal(t, domain.Aggregate, query.Operation)
	require.Len(t, query.Aggregations, 1)
	assert.Equal(t, domain.Count, query.Aggregations[0].Function)
}

func TestQueryBuilder_ChainedOperations(t *testing.T) {
	b := builder.NewQueryBuilder("users").
		FindMany().
		Where(domain.Condition{Field: "age", Operator: domain.Gte, Value: 18}).
		Where(domain.Condition{Field: "status", Operator: domain.Equals, Value: "active"}).
		OrderBy("created_at", domain.Desc).
		Skip(0).
		Take(10)

	query := b.GetQuery()

	assert.Equal(t, domain.FindMany, query.Operation)
	assert.Len(t, query.Filter.Conditions, 2)
	assert.Len(t, query.Ordering, 1)
	assert.NotNil(t, query.Pagination.Skip)
	assert.NotNil(t, query.Pagination.Take)
}
