// Package sqlgen provides MongoDB query generation.
// MongoDB is a NoSQL database, so queries are generated as MongoDB operations
// rather than SQL statements.
package sqlgen

import (
	"fmt"
	"strings"
)

// MongoDBGenerator generates MongoDB queries
// Note: MongoDB uses operations (find, insertOne, updateOne, etc.) rather than SQL
type MongoDBGenerator struct{}

func (g *MongoDBGenerator) GenerateSelect(table string, columns []string, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	// MongoDB uses find() operations
	// This is a foundation - full implementation would generate MongoDB query documents

	var filter map[string]interface{}
	if where != nil && !where.IsEmpty() {
		filter = g.buildMongoFilter(where)
	}

	// Build MongoDB find operation
	// In a full implementation, this would return a MongoDB operation document
	// For now, return a placeholder that indicates this needs MongoDB driver integration
	return &Query{
		SQL:  fmt.Sprintf("db.%s.find(%v)", table, filter),
		Args: []interface{}{filter},
	}
}

func (g *MongoDBGenerator) GenerateSelectWithJoins(table string, columns []string, joins []Join, where *WhereClause, orderBy []OrderBy, limit, offset *int) *Query {
	// MongoDB uses $lookup for joins (similar to SQL JOINs)
	// Foundation implementation
	return g.GenerateSelect(table, columns, where, orderBy, limit, offset)
}

// GenerateSelectWithAggregates for MongoDB (stub - MongoDB uses aggregation pipeline)
func (g *MongoDBGenerator) GenerateSelectWithAggregates(
	table string,
	columns []string,
	aggregates []AggregateFunction,
	joins []Join,
	where *WhereClause,
	groupBy *GroupBy,
	having *Having,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// MongoDB uses aggregation pipeline ($group, $match, etc.)
	// Foundation implementation - would need MongoDB aggregation pipeline builder
	return g.GenerateSelect(table, columns, where, orderBy, limit, offset)
}

// GenerateSelectWithCTE for MongoDB (stub - MongoDB doesn't support CTEs in the same way)
func (g *MongoDBGenerator) GenerateSelectWithCTE(
	table string,
	columns []string,
	ctes []CTE,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// MongoDB doesn't support CTEs in the traditional SQL sense
	// Would need to use $lookup or aggregation pipeline
	return g.GenerateSelect(table, columns, where, orderBy, limit, offset)
}

// GenerateSelectWithWindows for MongoDB (stub - MongoDB doesn't support window functions)
func (g *MongoDBGenerator) GenerateSelectWithWindows(
	table string,
	columns []string,
	windowFuncs []WindowFunction,
	joins []Join,
	where *WhereClause,
	orderBy []OrderBy,
	limit, offset *int,
) *Query {
	// MongoDB doesn't support window functions in the traditional SQL sense
	// Would need to use aggregation pipeline with $setWindowFields (MongoDB 5.0+)
	return g.GenerateSelect(table, columns, where, orderBy, limit, offset)
}

func (g *MongoDBGenerator) GenerateInsert(table string, columns []string, values []interface{}) *Query {
	// MongoDB uses insertOne() or insertMany()
	// Build document from columns and values
	doc := make(map[string]interface{})
	for i, col := range columns {
		if i < len(values) {
			doc[col] = values[i]
		}
	}

	return &Query{
		SQL:  fmt.Sprintf("db.%s.insertOne(%v)", table, doc),
		Args: []interface{}{doc},
	}
}

func (g *MongoDBGenerator) GenerateUpdate(table string, set map[string]interface{}, where *WhereClause) *Query {
	// MongoDB uses updateOne() or updateMany()
	var filter map[string]interface{}
	if where != nil && !where.IsEmpty() {
		filter = g.buildMongoFilter(where)
	}

	update := map[string]interface{}{
		"$set": set,
	}

	return &Query{
		SQL:  fmt.Sprintf("db.%s.updateMany(%v, %v)", table, filter, update),
		Args: []interface{}{filter, update},
	}
}

func (g *MongoDBGenerator) GenerateDelete(table string, where *WhereClause) *Query {
	// MongoDB uses deleteOne() or deleteMany()
	var filter map[string]interface{}
	if where != nil && !where.IsEmpty() {
		filter = g.buildMongoFilter(where)
	}

	return &Query{
		SQL:  fmt.Sprintf("db.%s.deleteMany(%v)", table, filter),
		Args: []interface{}{filter},
	}
}

func (g *MongoDBGenerator) GenerateUpsert(table string, columns []string, values []interface{}, updateColumns []string, conflictTarget []string) *Query {
	// MongoDB uses updateOne() with upsert: true
	var filter map[string]interface{}
	if len(conflictTarget) > 0 {
		filter = make(map[string]interface{})
		for i, col := range conflictTarget {
			if i < len(values) {
				filter[col] = values[i]
			}
		}
	}

	doc := make(map[string]interface{})
	for i, col := range columns {
		if i < len(values) {
			doc[col] = values[i]
		}
	}

	update := map[string]interface{}{
		"$set": doc,
	}

	return &Query{
		SQL:  fmt.Sprintf("db.%s.updateOne(%v, %v, {upsert: true})", table, filter, update),
		Args: []interface{}{filter, update},
	}
}

func (g *MongoDBGenerator) GenerateAggregate(table string, aggregates []AggregateFunction, where *WhereClause, groupBy *GroupBy, having *Having) *Query {
	// MongoDB uses aggregate() pipeline
	pipeline := []map[string]interface{}{}

	// Match stage (WHERE)
	if where != nil && !where.IsEmpty() {
		filter := g.buildMongoFilter(where)
		pipeline = append(pipeline, map[string]interface{}{"$match": filter})
	}

	// Group stage (GROUP BY)
	if groupBy != nil && len(groupBy.Fields) > 0 {
		group := map[string]interface{}{
			"_id": groupBy.Fields[0], // MongoDB groups by _id
		}
		for _, agg := range aggregates {
			group[agg.Alias] = map[string]interface{}{
				"$" + strings.ToLower(agg.Function): "$" + agg.Field,
			}
		}
		pipeline = append(pipeline, map[string]interface{}{"$group": group})
	}

	// Having stage (if needed)
	if having != nil && len(having.Conditions) > 0 {
		havingFilter := g.buildMongoFilterFromConditions(having.Conditions, having.Operator)
		pipeline = append(pipeline, map[string]interface{}{"$match": havingFilter})
	}

	return &Query{
		SQL:  fmt.Sprintf("db.%s.aggregate(%v)", table, pipeline),
		Args: []interface{}{pipeline},
	}
}

// buildMongoFilter converts a WHERE clause to MongoDB filter document
func (g *MongoDBGenerator) buildMongoFilter(where *WhereClause) map[string]interface{} {
	filter := make(map[string]interface{})

	for _, cond := range where.Conditions {
		switch cond.Operator {
		case "=":
			filter[cond.Field] = cond.Value
		case "!=":
			filter[cond.Field] = map[string]interface{}{"$ne": cond.Value}
		case ">":
			filter[cond.Field] = map[string]interface{}{"$gt": cond.Value}
		case "<":
			filter[cond.Field] = map[string]interface{}{"$lt": cond.Value}
		case ">=":
			filter[cond.Field] = map[string]interface{}{"$gte": cond.Value}
		case "<=":
			filter[cond.Field] = map[string]interface{}{"$lte": cond.Value}
		case "IN":
			filter[cond.Field] = map[string]interface{}{"$in": cond.Value}
		case "NOT IN":
			filter[cond.Field] = map[string]interface{}{"$nin": cond.Value}
		case "LIKE":
			filter[cond.Field] = map[string]interface{}{"$regex": cond.Value, "$options": "i"}
		case "IS NULL":
			filter[cond.Field] = nil
		case "IS NOT NULL":
			filter[cond.Field] = map[string]interface{}{"$ne": nil}
		}
	}

	return filter
}

// buildMongoFilterFromConditions builds a MongoDB filter from conditions
func (g *MongoDBGenerator) buildMongoFilterFromConditions(conditions []Condition, operator string) map[string]interface{} {
	if len(conditions) == 0 {
		return map[string]interface{}{}
	}

	if len(conditions) == 1 {
		return g.buildMongoFilter(&WhereClause{
			Conditions: conditions,
			Operator:   operator,
		})
	}

	// Multiple conditions - use $and or $or
	op := "$and"
	if operator == "OR" || operator == "or" {
		op = "$or"
	}

	clauses := make([]map[string]interface{}, len(conditions))
	for i, cond := range conditions {
		clauses[i] = g.buildMongoFilter(&WhereClause{
			Conditions: []Condition{cond},
			Operator:   "AND",
		})
	}

	return map[string]interface{}{op: clauses}
}
