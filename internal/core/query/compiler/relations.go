// Package compiler implements relation JOIN compilation.
package compiler

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/pkg/domain"
	"github.com/satishbabariya/prisma-go/pkg/schema"
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
		alias := c.dialect.Quote(fmt.Sprintf("%s_%s", baseAlias, rel.Relation))
		quotedBaseAlias := c.dialect.Quote(baseAlias)
		// Check if baseAlias was already quoted in caller?
		// Caller `compileSelect` passes `query.Model` as baseAlias. `compileSelect` quotes it in FROM clause.
		// But passes raw string.
		// So we should quote it here.
		// Wait, if alias is composed of quoted strings it breaks.

		// Build JOIN condition based on relation type and metadata
		var onCondition string
		if relationMeta.RelationType == domain.OneToMany {
			// For one-to-many: related_table.foreign_key = base_table.referenced_key
			// We need to look up the inverse relation (ManyToOne) to find the FKs
			inverse, err := registry.GetInverseRelation(relationMeta)
			if err == nil && len(inverse.FromFields) > 0 && len(inverse.ToFields) > 0 {
				foreignKeyCol := inverse.FromFields[0]
				refKeyCol := inverse.ToFields[0]
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, c.dialect.Quote(foreignKeyCol), quotedBaseAlias, c.dialect.Quote(refKeyCol))
			} else {
				// Fallback
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, c.dialect.Quote(strings.ToLower(relationMeta.FromModel)+"_id"), quotedBaseAlias, c.dialect.Quote("id"))
			}
		} else if relationMeta.RelationType == domain.ManyToOne {
			// For many-to-one: related_table.referenced_key = base_table.foreign_key
			if len(relationMeta.FromFields) > 0 && len(relationMeta.ToFields) > 0 {
				foreignKeyCol := relationMeta.FromFields[0]
				refKeyCol := relationMeta.ToFields[0]
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, c.dialect.Quote(refKeyCol), quotedBaseAlias, c.dialect.Quote(foreignKeyCol))
			} else {
				// Fallback to convention
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, c.dialect.Quote("id"), quotedBaseAlias, c.dialect.Quote(strings.ToLower(relationMeta.ToModel)+"_id"))
			}
		} else if relationMeta.RelationType == domain.ManyToMany {
			// ...
			return nil, fmt.Errorf("many-to-many relations not yet supported")
		} else {
			// OneToOne - similar to ManyToOne
			if len(relationMeta.FromFields) > 0 && len(relationMeta.ToFields) > 0 {
				foreignKeyCol := relationMeta.FromFields[0]
				refKeyCol := relationMeta.ToFields[0]
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, c.dialect.Quote(refKeyCol), quotedBaseAlias, c.dialect.Quote(foreignKeyCol))
			} else {
				onCondition = fmt.Sprintf("%s.%s = %s.%s", alias, c.dialect.Quote("id"), quotedBaseAlias, c.dialect.Quote(strings.ToLower(relationMeta.ToModel)+"_id"))
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
				columns = append(columns, fmt.Sprintf("%s.%s AS %s", alias, c.dialect.Quote(colName), c.dialect.Quote(fmt.Sprintf("%s_%s", rel.Relation, field))))
			}
		} else {
			// Select all scalar fields
			model, err := registry.GetModel(relationMeta.ToModel)
			if err == nil {
				for _, field := range model.Fields {
					// Skip relation fields (non-scalar)
					if isRelationField(field, registry) {
						continue
					}

					// Get actual column name
					colName, err := registry.GetColumnName(relationMeta.ToModel, field.Name)
					if err != nil {
						colName = field.Name
					}

					// Alias format: RelationName_FieldName
					// This matches hydrateResults expectation
					aliasName := fmt.Sprintf("%s_%s", rel.Relation, field.Name)

					columns = append(columns, fmt.Sprintf("%s.%s AS %s", alias, c.dialect.Quote(colName), c.dialect.Quote(aliasName)))
				}
			} else {
				// Fallback if model not found (shouldn't happen with valid registry)
				columns = append(columns, fmt.Sprintf("%s.*", alias))
			}
		}

		join := RelationJoin{
			JoinType:     "LEFT JOIN",
			Table:        c.dialect.Quote(relatedTable),
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

// isRelationField checks if a field is a relation.
func isRelationField(field domain.Field, registry *schema.MetadataRegistry) bool {
	// Simple check based on basic types
	scalars := map[string]bool{
		"String": true, "Boolean": true, "Int": true, "BigInt": true,
		"Float": true, "Decimal": true, "DateTime": true,
		"Json": true, "Bytes": true,
	}
	if scalars[field.Type.Name] {
		return false
	}
	// Check if enum
	if registry.IsEnum(field.Type.Name) {
		return false
	}
	return true
}
