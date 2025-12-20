// Package compiler implements relation JOIN compilation.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema"
	schemadomain "github.com/satishbabariya/prisma-go/v3/internal/core/schema/domain"
)

// RelationJoin represents a JOIN for a relation.
type RelationJoin struct {
	JoinType     string   // LEFT JOIN, INNER JOIN, etc.
	Table        string   // Table to join
	Alias        string   // Alias for the joined table
	OnConditions []string // JOIN conditions
	Columns      []string // Columns to select from this join
}

// buildRelationJoins builds JOIN clauses for relation inclusions using schema metadata.
// Now uses MetadataRegistry for accurate relation information.
func (c *SQLCompiler) buildRelationJoins(
	baseTable string,
	baseAlias string,
	relations []domain.RelationInclusion,
	registry *schema.MetadataRegistry,
) ([]RelationJoin, error) {
	if registry == nil {
		return nil, fmt.Errorf("schema metadata registry is required for relation loading")
	}

	var joins []RelationJoin

	for _, rel := range relations {
		// Get relation metadata from registry
		relationMeta, err := registry.GetRelation(baseTable, rel.Relation)
		if err != nil {
			// Relation not found in metadata - skip or return error
			continue
		}

		// Get table name for the related model
		relatedTable, err := registry.GetTableName(relationMeta.ToModel)
		if err != nil {
			relatedTable = relationMeta.ToModel // Fallback to model name
		}

		// Generate alias for this join
		alias := fmt.Sprintf("%s_%s", baseAlias, rel.Relation)

		// Build JOIN condition based on relation type and metadata
		var onCondition string
		if relationMeta.RelationType == schemadomain.OneToMany {
			// For one-to-many: related_table.foreign_key = base_table.referenced_key
			foreignKeyCol := relationMeta.FromFields[0] // Should have at least one
			refKeyCol := relationMeta.ToFields[0]
			onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, foreignKeyCol, baseAlias, refKeyCol)
		} else if relationMeta.RelationType == schemadomain.ManyToOne {
			// For many-to-one: related_table.referenced_key = base_table.foreign_key
			if len(relationMeta.FromFields) > 0 && len(relationMeta.ToFields) > 0 {
				foreignKeyCol := relationMeta.FromFields[0]
				refKeyCol := relationMeta.ToFields[0]
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, refKeyCol, baseAlias, foreignKeyCol)
			} else {
				// Fallback to convention
				onCondition = fmt.Sprintf("%s.id = %s.%s_id", alias, baseAlias, strings.ToLower(relationMeta.ToModel))
			}
		} else if relationMeta.RelationType == schemadomain.ManyToMany {
			// Many-to-many requires a junction table (not yet implemented)
			return nil, fmt.Errorf("many-to-many relations not yet supported")
		} else {
			// OneToOne - similar to ManyToOne
			if len(relationMeta.FromFields) > 0 && len(relationMeta.ToFields) > 0 {
				foreignKeyCol := relationMeta.FromFields[0]
				refKeyCol := relationMeta.ToFields[0]
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, refKeyCol, baseAlias, foreignKeyCol)
			} else {
				onCondition = fmt.Sprintf("%s.id = %s.%s_id", alias, baseAlias, strings.ToLower(relationMeta.ToModel))
			}
		}

		// Determine columns to select
		var columns []string
		// Check if there's a nested query with selection
		if rel.Query != nil && len(rel.Query.Selection.Fields) > 0 {
			// Select specific fields
			for _, field := range rel.Query.Selection.Fields {
				// Get actual column name using metadata
				colName, err := registry.GetColumnName(relationMeta.ToModel, field)
				if err != nil {
					colName = field // Fallback to field name
				}
				columns = append(columns, fmt.Sprintf("%s.%s AS %s_%s", alias, colName, rel.Relation, field))
			}
		} else {
			// Select all fields
			columns = append(columns, fmt.Sprintf("%s.*", alias))
		}

		join := RelationJoin{
			JoinType:     "LEFT JOIN", // Always LEFT JOIN to handle optional relations
			Table:        relatedTable,
			Alias:        alias,
			OnConditions: []string{onCondition},
			Columns:      columns,
		}

		joins = append(joins, join)

		// Recursively handle nested inclusions via Query.Relations
		if rel.Query != nil && len(rel.Query.Relations) > 0 {
			nestedJoins, err := c.buildRelationJoins(relationMeta.ToModel, alias, rel.Query.Relations, registry)
			if err != nil {
				return nil, err
			}
			joins = append(joins, nestedJoins...)
		}
	}

	return joins, nil
}

// generateJoinSQL generates the SQL for JOIN clauses.
func generateJoinSQL(joins []RelationJoin) string {
	if len(joins) == 0 {
		return ""
	}

	var parts []string
	for _, join := range joins {
		joinSQL := fmt.Sprintf(" %s %s AS %s ON %s",
			join.JoinType,
			join.Table,
			join.Alias,
			strings.Join(join.OnConditions, " AND "))
		parts = append(parts, joinSQL)
	}

	return strings.Join(parts, "")
}

// getJoinColumns extracts all columns to select from joins.
func getJoinColumns(joins []RelationJoin) []string {
	var columns []string
	for _, join := range joins {
		columns = append(columns, join.Columns...)
	}
	return columns
}
