package compiler_test

import (
	"strings"
	"testing"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompiler_FindMany_PostgreSQL(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.FindMany,
		Selection: domain.Selection{
			Fields: []string{"id", "email", "name"},
		},
	}

	sql, err := comp.Compile(nil, query)
	require.NoError(t, err)

	assert.Contains(t, sql.SQL.Query, "SELECT id, email, name")
	assert.Contains(t, sql.SQL.Query, "FROM users")
}

func TestCompiler_WhereClause_PostgreSQL(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	tests := []struct {
		name      string
		condition domain.Condition
		wantSQL   string
		wantArgs  int
	}{
		{
			name:      "equals operator",
			condition: domain.Condition{Field: "id", Operator: domain.Equals, Value: 1},
			wantSQL:   "id = $1",
			wantArgs:  1,
		},
		{
			name:      "greater than operator",
			condition: domain.Condition{Field: "age", Operator: domain.Gt, Value: 18},
			wantSQL:   "age > $1",
			wantArgs:  1,
		},
		{
			name:      "contains operator",
			condition: domain.Condition{Field: "email", Operator: domain.Contains, Value: "test"},
			wantSQL:   "email LIKE $1",
			wantArgs:  1,
		},
		{
			name:      "contains case insensitive",
			condition: domain.Condition{Field: "email", Operator: domain.Contains, Value: "test", Mode: domain.ModeInsensitive},
			wantSQL:   "LOWER(email) LIKE LOWER($1)",
			wantArgs:  1,
		},
		{
			name:      "isNull true",
			condition: domain.Condition{Field: "deleted_at", Operator: domain.IsNull, Value: true},
			wantSQL:   "deleted_at IS NULL",
			wantArgs:  0,
		},
		{
			name:      "isNull false",
			condition: domain.Condition{Field: "deleted_at", Operator: domain.IsNull, Value: false},
			wantSQL:   "deleted_at IS NOT NULL",
			wantArgs:  0,
		},
		{
			name:      "isEmpty true",
			condition: domain.Condition{Field: "tags", Operator: domain.IsEmpty, Value: true},
			wantSQL:   "COALESCE(array_length(tags, 1), 0) = 0",
			wantArgs:  0,
		},
		{
			name:      "has array element",
			condition: domain.Condition{Field: "tags", Operator: domain.Has, Value: "featured"},
			wantSQL:   "tags @> ARRAY[$1]",
			wantArgs:  1,
		},
		{
			name:      "fulltext search",
			condition: domain.Condition{Field: "content", Operator: domain.Search, Value: "golang"},
			wantSQL:   "to_tsvector(content) @@ to_tsquery($1)",
			wantArgs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &domain.Query{
				Model:     "users",
				Operation: domain.FindMany,
				Filter: domain.Filter{
					Conditions: []domain.Condition{tt.condition},
				},
			}

			sql, err := comp.Compile(nil, query)
			require.NoError(t, err)

			assert.Contains(t, sql.SQL.Query, "WHERE")
			assert.Contains(t, sql.SQL.Query, tt.condition.Field)
			assert.Len(t, sql.SQL.Args, tt.wantArgs)
		})
	}
}

func TestCompiler_OrderBy(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.FindMany,
		Ordering: []domain.OrderBy{
			{Field: "created_at", Direction: domain.Desc},
		},
	}

	sql, err := comp.Compile(nil, query)
	require.NoError(t, err)

	assert.Contains(t, strings.ToUpper(sql.SQL.Query), "ORDER BY CREATED_AT DESC")
}

func TestCompiler_Pagination(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	skip := 10
	take := 20
	query := &domain.Query{
		Model:     "users",
		Operation: domain.FindMany,
		Pagination: domain.Pagination{
			Skip: &skip,
			Take: &take,
		},
	}

	sql, err := comp.Compile(nil, query)
	require.NoError(t, err)

	assert.Contains(t, sql.SQL.Query, "LIMIT")
	assert.Contains(t, sql.SQL.Query, "OFFSET")
	assert.Contains(t, sql.SQL.Args, 20)
	assert.Contains(t, sql.SQL.Args, 10)
}

func TestCompiler_Create_PostgreSQL(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.Create,
		CreateData: map[string]interface{}{
			"email": "test@example.com",
			"name":  "Test User",
		},
	}

	sql, args, err := comp.CompileCreate(query)
	require.NoError(t, err)

	assert.Contains(t, sql, "INSERT INTO users")
	assert.Contains(t, sql, "VALUES")
	assert.Contains(t, sql, "RETURNING *") // PostgreSQL specific
	assert.Len(t, args, 2)
}

func TestCompiler_CreateMany(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.CreateMany,
		CreateManyData: []map[string]interface{}{
			{"email": "user1@example.com", "name": "User 1"},
			{"email": "user2@example.com", "name": "User 2"},
		},
	}

	sql, args, err := comp.CompileCreateMany(query)
	require.NoError(t, err)

	assert.Contains(t, sql, "INSERT INTO users")
	assert.Contains(t, sql, "VALUES")
	// Should have 2 value groups
	assert.Len(t, args, 4) // 2 rows * 2 fields
}

func TestCompiler_Update(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.Update,
		UpdateData: map[string]interface{}{
			"name": "Updated Name",
		},
		Filter: domain.Filter{
			Conditions: []domain.Condition{
				{Field: "id", Operator: domain.Equals, Value: 1},
			},
		},
	}

	sql, args, err := comp.CompileUpdate(query)
	require.NoError(t, err)

	assert.Contains(t, sql, "UPDATE users")
	assert.Contains(t, sql, "SET")
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 2) // 1 for SET, 1 for WHERE
}

func TestCompiler_Delete(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.Delete,
		Filter: domain.Filter{
			Conditions: []domain.Condition{
				{Field: "id", Operator: domain.Equals, Value: 1},
			},
		},
	}

	sql, args, err := comp.CompileDelete(query)
	require.NoError(t, err)

	assert.Contains(t, sql, "DELETE FROM users")
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 1)
}

func TestCompiler_Upsert_PostgreSQL(t *testing.T) {
	comp := compiler.NewSQLCompiler(domain.PostgreSQL)

	query := &domain.Query{
		Model:     "users",
		Operation: domain.Upsert,
		UpsertData: map[string]interface{}{
			"email": "test@example.com",
			"name":  "Test User",
		},
		UpsertUpdate: map[string]interface{}{
			"name": "Updated User",
		},
		UpsertKeys: []string{"email"},
	}

	sql, args, err := comp.CompileUpsert(query)
	require.NoError(t, err)

	assert.Contains(t, sql, "INSERT INTO users")
	assert.Contains(t, sql, "ON CONFLICT")
	assert.Contains(t, sql, "DO UPDATE SET")
	assert.Len(t, args, 3) // 2 for insert, 1 for update
}

func TestCompiler_MultiDialect(t *testing.T) {
	tests := []struct {
		name            string
		dialect         domain.SQLDialect
		wantPlaceholder string
	}{
		{
			name:            "PostgreSQL uses $1, $2",
			dialect:         domain.PostgreSQL,
			wantPlaceholder: "$",
		},
		{
			name:            "MySQL uses ?",
			dialect:         domain.MySQL,
			wantPlaceholder: "?",
		},
		{
			name:            "SQLite uses ?",
			dialect:         domain.SQLite,
			wantPlaceholder: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := compiler.NewSQLCompiler(tt.dialect)

			query := &domain.Query{
				Model:     "users",
				Operation: domain.FindMany,
				Filter: domain.Filter{
					Conditions: []domain.Condition{
						{Field: "id", Operator: domain.Equals, Value: 1},
					},
				},
			}

			sql, err := comp.Compile(nil, query)
			require.NoError(t, err)
			assert.Contains(t, sql.SQL.Query, tt.wantPlaceholder)
		})
	}
}
