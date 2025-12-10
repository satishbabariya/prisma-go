// Package executor provides JOIN query building for relations.
package executor

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/query/optimizer"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// buildJoinsFromIncludes builds JOIN clauses from include map and relation metadata
func buildJoinsFromIncludes(
	table string,
	includes map[string]bool,
	relations map[string]RelationMetadata,
	provider string,
) []sqlgen.Join {
	var joins []sqlgen.Join

	for relationName := range includes {
		if relMeta, ok := relations[relationName]; ok && relMeta.ForeignKey != "" {
			join := sqlgen.Join{
				Type:    "LEFT",
				Table:   relMeta.RelatedTable,
				Alias:   relMeta.RelatedTable, // Use table name as alias
				Columns: nil,                  // Will select all columns
			}

			// Build JOIN condition
			// For one-to-many: related_table.foreign_key = main_table.id
			// For many-to-one: main_table.foreign_key = related_table.id
			if relMeta.IsList {
				// One-to-many: foreign key is on the related table
				// Example: post.author_id = user.id
				join.Condition = fmt.Sprintf("%s.%s = %s.%s",
					quoteIdentifier(relMeta.RelatedTable),
					quoteIdentifier(relMeta.ForeignKey),
					quoteIdentifier(table),
					quoteIdentifier(relMeta.LocalKey))
			} else {
				// Many-to-one: foreign key is on the main table
				// Example: post.author_id = user.id
				join.Condition = fmt.Sprintf("%s.%s = %s.%s",
					quoteIdentifier(table),
					quoteIdentifier(relMeta.ForeignKey),
					quoteIdentifier(relMeta.RelatedTable),
					quoteIdentifier(relMeta.LocalKey))
			}

			joins = append(joins, join)
		}
	}

	// Optimize joins if optimizer is available
	if len(joins) > 0 {
		opt := optimizer.NewOptimizer(provider)
		joins = opt.OptimizeJoins(joins, includes)
	}

	return joins
}

// RelationMetadata contains metadata about a relation for JOIN generation
type RelationMetadata struct {
	RelatedTable string // Table name of the related model
	ForeignKey   string // Foreign key field name
	LocalKey     string // Local key field name (usually "id")
	IsList       bool   // true if one-to-many
}

func quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
