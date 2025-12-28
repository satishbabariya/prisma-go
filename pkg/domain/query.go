// Package domain contains the core business entities and interfaces for the Query domain.
package domain

import "context"

// Query represents a query aggregate root.
type Query struct {
	Model           string
	Operation       QueryOperation
	Selection       Selection
	Filter          Filter
	Relations       []RelationInclusion
	Ordering        []OrderBy
	Pagination      Pagination
	Aggregations    []Aggregation
	CreateData      map[string]interface{}   // Data for CREATE operations
	UpdateData      map[string]interface{}   // Data for UPDATE operations
	CreateManyData  []map[string]interface{} // Data for CREATE MANY operations
	UpsertData      map[string]interface{}   // Data for UPSERT insert
	UpsertUpdate    map[string]interface{}   // Data for UPSERT update
	UpsertKeys      []string                 // Conflict columns for UPSERT
	GroupBy         []string                 // Fields to group by
	Having          Filter                   // Having clause for GroupBy
	Distinct        []string                 // Fields for DISTINCT
	Cursor          *Cursor                  // Cursor for cursor-based pagination
	NestedWrites    []NestedWrite            // Nested write operations for relations
	ThrowIfNotFound bool                     // Throw error if no records found (for OrThrow variants)
}

// QueryOperation represents the type of query operation.
type QueryOperation string

const (
	// FindMany finds multiple records.
	FindMany QueryOperation = "FindMany"
	// FindFirst finds the first matching record.
	FindFirst QueryOperation = "FindFirst"
	// FindUnique finds a unique record.
	FindUnique QueryOperation = "FindUnique"
	// Create creates a new record.
	Create QueryOperation = "Create"
	// CreateMany creates multiple records.
	CreateMany QueryOperation = "CreateMany"
	// Update updates a record.
	Update QueryOperation = "Update"
	// UpdateMany updates multiple records.
	UpdateMany QueryOperation = "UpdateMany"
	// Delete deletes a record.
	Delete QueryOperation = "Delete"
	// DeleteMany deletes multiple records.
	DeleteMany QueryOperation = "DeleteMany"
	// Upsert inserts or updates a record.
	Upsert QueryOperation = "Upsert"
	// Aggregate performs aggregation.
	Aggregate QueryOperation = "Aggregate"
	// GroupBy groups records and performs aggregation.
	GroupBy QueryOperation = "GroupBy"
)

// Selection defines what fields to return.
type Selection struct {
	Fields  []string
	Include bool // true = include, false = select
}

// Filter represents query conditions with support for nested logical combinations.
// Filters can contain both direct conditions and nested filter groups.
// Example: (status='active' AND role='admin') OR verified=true
// Would be represented as:
//
//	Filter{
//	  Operator: OR,
//	  NestedFilters: [
//	    Filter{Operator: AND, Conditions: [{status='active'}, {role='admin'}]},
//	    Filter{Conditions: [{verified=true}]}
//	  ]
//	}
type Filter struct {
	Conditions    []Condition     // Direct field conditions
	NestedFilters []Filter        // Nested filter groups for complex logic
	Operator      LogicalOperator // How to combine conditions/nested filters
}

// LogicalOperator represents logical operators for combining conditions.
type LogicalOperator string

const (
	// AND combines conditions with AND.
	AND LogicalOperator = "AND"
	// OR combines conditions with OR.
	OR LogicalOperator = "OR"
	// NOT negates conditions.
	NOT LogicalOperator = "NOT"
)

// Condition represents a single filter condition.
type Condition struct {
	Field    string
	Operator ComparisonOperator
	Value    interface{}
	Mode     FilterMode // For case-insensitive mode
}

// FilterMode represents the filter mode (default or insensitive).
type FilterMode string

const (
	// ModeDefault is the default case-sensitive mode.
	ModeDefault FilterMode = "default"
	// ModeInsensitive is case-insensitive mode.
	ModeInsensitive FilterMode = "insensitive"
)

// ComparisonOperator represents comparison operators.
type ComparisonOperator string

const (
	// Equals checks equality.
	Equals ComparisonOperator = "equals"
	// NotEquals checks inequality.
	NotEquals ComparisonOperator = "not"
	// In checks if value is in list.
	In ComparisonOperator = "in"
	// NotIn checks if value is not in list.
	NotIn ComparisonOperator = "notIn"
	// Lt checks if value is less than.
	Lt ComparisonOperator = "lt"
	// Lte checks if value is less than or equal.
	Lte ComparisonOperator = "lte"
	// Gt checks if value is greater than.
	Gt ComparisonOperator = "gt"
	// Gte checks if value is greater than or equal.
	Gte ComparisonOperator = "gte"
	// Contains checks if string contains substring.
	Contains ComparisonOperator = "contains"
	// StartsWith checks if string starts with.
	StartsWith ComparisonOperator = "startsWith"
	// EndsWith checks if string ends with.
	EndsWith ComparisonOperator = "endsWith"

	// Array operators
	// IsEmpty checks if array field is empty.
	IsEmpty ComparisonOperator = "isEmpty"
	// Has checks if array contains value.
	Has ComparisonOperator = "has"
	// HasEvery checks if array contains all values.
	HasEvery ComparisonOperator = "hasEvery"
	// HasSome checks if array contains any of the values.
	HasSome ComparisonOperator = "hasSome"

	// Null check
	// IsNull checks if field is null.
	IsNull ComparisonOperator = "isNull"

	// Fulltext search
	// Search performs fulltext search.
	Search ComparisonOperator = "search"

	// Relation filters (for filtering based on related records)
	// Some checks if some related records match conditions.
	Some ComparisonOperator = "some"
	// Every checks if all related records match conditions.
	Every ComparisonOperator = "every"
	// None checks if no related records match conditions.
	None ComparisonOperator = "none"
)

// RelationInclusion defines relation loading.
type RelationInclusion struct {
	Relation string
	Query    *Query // Nested query
}

// NestedWrite represents a nested write operation on a relation.
type NestedWrite struct {
	Relation   string                 // Name of the relation field
	Operation  NestedWriteOp          // Type of nested operation
	Data       map[string]interface{} // Data for create operations
	UpdateData map[string]interface{} // Data for update in upsert operations
	Where      []Condition            // Conditions for connect/disconnect/update/delete
	Many       []NestedWrite          // For operations on multiple related records
}

// NestedWriteOp represents the type of nested write operation.
type NestedWriteOp string

const (
	// NestedCreate creates a new related record.
	NestedCreate NestedWriteOp = "create"
	// NestedCreateMany creates multiple new related records.
	NestedCreateMany NestedWriteOp = "createMany"
	// NestedConnect connects an existing record.
	NestedConnect NestedWriteOp = "connect"
	// NestedConnectOrCreate connects if exists, otherwise creates.
	NestedConnectOrCreate NestedWriteOp = "connectOrCreate"
	// NestedDisconnect disconnects a related record.
	NestedDisconnect NestedWriteOp = "disconnect"
	// NestedSet replaces all related records.
	NestedSet NestedWriteOp = "set"
	// NestedUpdate updates a related record.
	NestedUpdate NestedWriteOp = "update"
	// NestedUpdateMany updates multiple related records.
	NestedUpdateMany NestedWriteOp = "updateMany"
	// NestedDelete deletes a related record.
	NestedDelete NestedWriteOp = "delete"
	// NestedDeleteMany deletes multiple related records.
	NestedDeleteMany NestedWriteOp = "deleteMany"
	// NestedUpsert upserts a related record.
	NestedUpsert NestedWriteOp = "upsert"
)

// OrderBy defines sorting.
type OrderBy struct {
	Field     string
	Direction SortDirection
}

// SortDirection represents sort direction.
type SortDirection string

const (
	// Asc sorts ascending.
	Asc SortDirection = "asc"
	// Desc sorts descending.
	Desc SortDirection = "desc"
)

// Pagination defines result pagination.
type Pagination struct {
	Skip *int
	Take *int
}

// Cursor defines cursor-based pagination.
type Cursor struct {
	Field string      // Field to use for cursor (usually 'id')
	Value interface{} // The cursor value
}

// Aggregation defines aggregation operations.
type Aggregation struct {
	Function AggregateFunc
	Field    string
}

// AggregateFunc represents aggregation functions.
type AggregateFunc string

const (
	// Count counts records.
	Count AggregateFunc = "count"
	// Sum sums field values.
	Sum AggregateFunc = "sum"
	// Avg calculates average.
	Avg AggregateFunc = "avg"
	// Min finds minimum value.
	Min AggregateFunc = "min"
	// Max finds maximum value.
	Max AggregateFunc = "max"
)

// SQL represents generated SQL.
type SQL struct {
	Query   string
	Args    []interface{}
	Dialect SQLDialect
}

// SQLDialect represents a SQL dialect.
type SQLDialect string

const (
	// PostgreSQL dialect.
	PostgreSQL SQLDialect = "postgres"
	// MySQL dialect.
	MySQL SQLDialect = "mysql"
	// SQLite dialect.
	SQLite SQLDialect = "sqlite"
)

// CompiledQuery represents a compiled query ready for execution.
type CompiledQuery struct {
	SQL           SQL
	Mapping       ResultMapping
	CacheKey      string
	OriginalQuery *Query // Reference to original query for access to flags
}

// ResultMapping defines how to map results to Go types.
type ResultMapping struct {
	Model     string
	Fields    []FieldMapping
	Relations []RelationMapping
}

// FieldMapping maps a field to a column.
type FieldMapping struct {
	Field  string
	Column string
	Type   string
}

// RelationMapping maps a relation.
type RelationMapping struct {
	Relation string
	Type     RelationType
	Mapping  *ResultMapping
}

// RelationType represents the type of relation.
type RelationType string

const (
	// OneToOne represents a one-to-one relation.
	OneToOne RelationType = "OneToOne"
	// OneToMany represents a one-to-many relation.
	OneToMany RelationType = "OneToMany"
	// ManyToMany represents a many-to-many relation.
	ManyToMany RelationType = "ManyToMany"
)

// QueryBuilder defines the interface for building queries.
type QueryBuilder interface {
	// Build builds SQL from a query.
	Build(ctx context.Context, query *Query) (SQL, error)
}

// QueryCompiler defines the interface for query compilation.
type QueryCompiler interface {
	// Compile compiles a query to an executable form.
	Compile(ctx context.Context, query *Query) (*CompiledQuery, error)

	// Optimize optimizes a compiled query.
	Optimize(ctx context.Context, compiled *CompiledQuery) (*CompiledQuery, error)
}

// QueryExecutor defines the interface for query execution.
type QueryExecutor interface {
	// Execute executes a compiled query.
	Execute(ctx context.Context, query *CompiledQuery) (interface{}, error)

	// ExecuteBatch executes multiple queries in batch.
	ExecuteBatch(ctx context.Context, queries []*CompiledQuery) ([]interface{}, error)
}
