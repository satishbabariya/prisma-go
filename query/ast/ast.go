// Package ast defines the query AST (Abstract Syntax Tree).
package ast

// QueryNode represents a query operation
type QueryNode interface {
	Type() NodeType
}

// NodeType represents the type of query node
type NodeType string

const (
	NodeTypeFindMany   NodeType = "FindMany"
	NodeTypeFindFirst  NodeType = "FindFirst"
	NodeTypeFindUnique NodeType = "FindUnique"
	NodeTypeCreate     NodeType = "Create"
	NodeTypeUpdate     NodeType = "Update"
	NodeTypeDelete     NodeType = "Delete"
)

// FindManyQuery represents a findMany operation
type FindManyQuery struct {
	Model   string
	Where   *WhereClause
	OrderBy []OrderByClause
	Skip    *int
	Take    *int
	Include *IncludeClause
	Select  *SelectClause
}

func (q *FindManyQuery) Type() NodeType { return NodeTypeFindMany }

// WhereClause represents filtering conditions
type WhereClause struct {
	Conditions []Condition
	Operator   LogicalOperator
}

// Condition represents a single filter condition
type Condition struct {
	Field    string
	Operator ComparisonOperator
	Value    interface{}
}

// ComparisonOperator represents comparison operators
type ComparisonOperator string

const (
	OpEquals         ComparisonOperator = "equals"
	OpNotEquals      ComparisonOperator = "not"
	OpGreaterThan    ComparisonOperator = "gt"
	OpLessThan       ComparisonOperator = "lt"
	OpGreaterOrEqual ComparisonOperator = "gte"
	OpLessOrEqual    ComparisonOperator = "lte"
	OpIn             ComparisonOperator = "in"
	OpNotIn          ComparisonOperator = "notIn"
	OpContains       ComparisonOperator = "contains"
	OpStartsWith     ComparisonOperator = "startsWith"
	OpEndsWith       ComparisonOperator = "endsWith"
)

// LogicalOperator represents logical operators
type LogicalOperator string

const (
	OpAND LogicalOperator = "AND"
	OpOR  LogicalOperator = "OR"
	OpNOT LogicalOperator = "NOT"
)

// OrderByClause represents ordering
type OrderByClause struct {
	Field     string
	Direction SortDirection
}

// SortDirection represents sort direction
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// IncludeClause represents related data to include
type IncludeClause struct {
	Relations map[string]*RelationInclude
}

// RelationInclude represents including a relation
type RelationInclude struct {
	Include *IncludeClause
	Where   *WhereClause
}

// SelectClause represents fields to select
type SelectClause struct {
	Fields []string
}
