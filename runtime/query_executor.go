// Package runtime provides query execution functionality for generated clients.
package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/internal/core/query/compiler"
	"github.com/satishbabariya/prisma-go/pkg/domain"
	"github.com/satishbabariya/prisma-go/pkg/schema"
)

// QueryExecutor provides query execution capabilities.
type QueryExecutor struct {
	db       *sql.DB
	dialect  domain.SQLDialect
	registry *schema.MetadataRegistry
}

// NewQueryExecutor creates a new query executor.
func NewQueryExecutor(db *sql.DB, dialect SQLDialect, registry *schema.MetadataRegistry) *QueryExecutor {
	return &QueryExecutor{
		db:       db,
		dialect:  dialect.toDomain(),
		registry: registry,
	}
}

// ExecuteFindMany executes a FindMany query and returns results.
func (e *QueryExecutor) ExecuteFindMany(ctx context.Context, query *domain.Query) ([]map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiler.SetRegistry(e.registry)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	rows, err := e.db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	results, err := e.scanRowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	return e.hydrateResults(results, compiled.Mapping), nil
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
	compiler.SetRegistry(e.registry)
	compiled, err := compiler.Compile(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	var createdItem map[string]interface{}
	if e.dialect == domain.PostgreSQL {
		// Postgres uses RETURNING
		rows, err := e.db.QueryContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute create: %w", err)
		}
		defer rows.Close()

		results, err := e.scanRowsToMaps(rows)
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("no result returned")
		}
		createdItem = results[0]
	} else {
		result, err := e.db.ExecContext(ctx, compiled.SQL.Query, compiled.SQL.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute create: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get insert ID: %w", err)
		}
		createdItem = map[string]interface{}{"id": id}
	}

	// 2. Process Nested Writes
	if len(compiled.OriginalQuery.NestedWrites) > 0 {
		for _, nw := range compiled.OriginalQuery.NestedWrites {
			if nw.Operation == domain.NestedCreate {
				// Resolve metadata for relation
				parentModel := compiled.OriginalQuery.Model
				relMeta, err := e.registry.GetRelation(parentModel, nw.Relation)
				if err != nil {
					return nil, fmt.Errorf("relation metadata not found for %s.%s: %w", parentModel, nw.Relation, err)
				}

				// Find inverse relation (to know FK field on child)
				inverseRel, err := e.registry.GetInverseRelation(relMeta)
				if err != nil {
					return nil, fmt.Errorf("inverse relation not found for %s.%s: %w", parentModel, nw.Relation, err)
				}

				// Determine FK field
				var fkField string
				if len(inverseRel.FromFields) > 0 {
					fkField = inverseRel.FromFields[0]
				} else {
					// Fallback to convention: relationName + "Id"
					fkField = inverseRel.Name + "Id"
				}

				// Prepare child data
				childData := make(map[string]interface{})
				for k, v := range nw.Data {
					childData[k] = v
				}
				// Inject FK
				// TODO: generic PK support (currently assumes "id")
				childData[fkField] = createdItem["id"]

				// Create child query
				childQuery := &domain.Query{
					Model:      relMeta.ToModel,
					Operation:  domain.Create,
					CreateData: childData,
				}

				// Execute recursive create
				// Note: using ExecuteCreate directly
				_, err = e.ExecuteCreate(ctx, childQuery)
				if err != nil {
					return nil, fmt.Errorf("failed to execute nested create for %s: %w", relMeta.ToModel, err)
				}
			}
		}
	}

	return createdItem, nil
}

// ExecuteUpdate executes an Update query and returns an updated record.
func (e *QueryExecutor) ExecuteUpdate(ctx context.Context, query *domain.Query) (map[string]interface{}, error) {
	compiler := compiler.NewSQLCompiler(e.dialect)
	compiler.SetRegistry(e.registry)
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
	compiler.SetRegistry(e.registry)
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
	compiler.SetRegistry(e.registry)
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
		compiler.SetRegistry(e.registry)
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
	compiler.SetRegistry(e.registry)
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

// hydrateResults transforms flattened results into nested structures based on mapping.
func (e *QueryExecutor) hydrateResults(rows []map[string]interface{}, mapping domain.ResultMapping) []map[string]interface{} {
	if len(mapping.Relations) == 0 {
		return rows
	}

	// Grouping by ID (assuming 'id' is the primary key for now)
	grouped := make(map[interface{}]map[string]interface{})
	var keys []interface{} // To preserve order

	for _, row := range rows {
		id, ok := row["id"]
		if !ok {
			return rows
		}

		mainItem, exists := grouped[id]
		if !exists {
			mainItem = make(map[string]interface{})
			// Copy all fields initially
			for k, v := range row {
				if !strings.Contains(k, "_") { // Simple heuristic for base fields? No, unsafe.
					// Copy everything. Relation fields will be processed.
					mainItem[k] = v
				}
				// Copy base fields (including foreign keys)
				// If mapping.Fields is populated, use it.
				// If not, copy everything.
			}

			// Initialize empty relation containers
			for _, rel := range mapping.Relations {
				if rel.Type == domain.OneToMany {
					mainItem[rel.Relation] = []map[string]interface{}{}
				}
			}

			grouped[id] = mainItem
			keys = append(keys, id)
		}

		// Process relations
		for _, rel := range mapping.Relations {
			prefix := rel.Relation + "_"
			relData := make(map[string]interface{})
			hasData := false

			// Extract fields for this relation (prefixed)
			for k, v := range row {
				if strings.HasPrefix(k, prefix) {
					cleanName := strings.TrimPrefix(k, prefix)
					relData[cleanName] = v
					if v != nil {
						hasData = true
					}
					// Clean up flat map if desired, but careful of side effects
					delete(mainItem, k)
				}
			}

			if !hasData {
				continue
			}

			// If relation has data, attach it
			if rel.Type == domain.OneToMany {
				relID := relData["id"]

				list := mainItem[rel.Relation].([]map[string]interface{})
				existsInList := false
				if relID != nil {
					for _, item := range list {
						if item["id"] == relID {
							existsInList = true
							break
						}
					}
				}

				if !existsInList && relID != nil {
					mainItem[rel.Relation] = append(list, relData)
				}
			} else {
				// OneToOne or ManyToOne
				mainItem[rel.Relation] = relData
			}
		}
	}

	// Reconstruct slice
	results := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		results = append(results, grouped[key])
	}

	return results
}
