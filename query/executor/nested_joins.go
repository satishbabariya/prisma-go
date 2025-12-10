// Package executor provides nested JOIN query building for deep relations.
package executor

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/query/builder"
	"github.com/satishbabariya/prisma-go/query/sqlgen"
)

// buildNestedJoinsFromIncludes builds JOIN clauses for nested includes
// Example: Include().Posts().Author() creates:
//   - JOIN post ON user.id = post.author_id
//   - JOIN user AS post_author ON post.author_id = post_author.id
func buildNestedJoinsFromIncludes(
	table string,
	includes map[string]*builder.NestedInclude,
	relations map[string]RelationMetadata,
	allRelations map[string]map[string]RelationMetadata, // model -> relations
) []sqlgen.Join {
	var joins []sqlgen.Join
	processed := make(map[string]bool) // Track processed relations to avoid duplicates
	
	// Build joins for each top-level include
	for relationName, nestedInclude := range includes {
		joins = append(joins, buildJoinsForRelation(
			table,
			relationName,
			nestedInclude,
			relations,
			allRelations,
			"",
			processed,
		)...)
	}
	
	return joins
}

// buildJoinsForRelation recursively builds JOINs for a relation and its nested includes
func buildJoinsForRelation(
	parentTable string,
	relationName string,
	nestedInclude *builder.NestedInclude,
	parentRelations map[string]RelationMetadata,
	allRelations map[string]map[string]RelationMetadata,
	pathPrefix string,
	processed map[string]bool,
) []sqlgen.Join {
	var joins []sqlgen.Join
	
	// Get relation metadata
	relMeta, ok := parentRelations[relationName]
	if !ok || relMeta.ForeignKey == "" {
		return joins
	}
	
	// Build current path for alias
	currentPath := relationName
	if pathPrefix != "" {
		currentPath = pathPrefix + "_" + relationName
	}
	
	// Skip if already processed
	if processed[currentPath] {
		return joins
	}
	processed[currentPath] = true
	
	// Determine table alias
	tableAlias := currentPath
	if pathPrefix == "" {
		tableAlias = relMeta.RelatedTable
	}
	
	// Build JOIN for current relation
	join := sqlgen.Join{
		Type:  "LEFT",
		Table: relMeta.RelatedTable,
		Alias: tableAlias,
	}
	
	// Build JOIN condition
	if relMeta.IsList {
		// One-to-many: foreign key is on the related table
		join.Condition = fmt.Sprintf("%s.%s = %s.%s",
			quoteIdentifier(tableAlias),
			quoteIdentifier(relMeta.ForeignKey),
			quoteIdentifier(parentTable),
			quoteIdentifier(relMeta.LocalKey))
	} else {
		// Many-to-one: foreign key is on the parent table
		join.Condition = fmt.Sprintf("%s.%s = %s.%s",
			quoteIdentifier(parentTable),
			quoteIdentifier(relMeta.ForeignKey),
			quoteIdentifier(tableAlias),
			quoteIdentifier(relMeta.LocalKey))
	}
	
	joins = append(joins, join)
	
	// Process nested includes
	if nestedInclude.HasNested() {
		// Get relations for the related model
		relatedModelName := toPascalCase(relMeta.RelatedTable)
		relatedModelRelations, ok := allRelations[relatedModelName]
		if ok {
			for nestedRelationName, nestedIncludeChild := range nestedInclude.GetNestedIncludes() {
				joins = append(joins, buildJoinsForRelation(
					tableAlias, // Use current table alias as parent for nested joins
					nestedRelationName,
					nestedIncludeChild,
					relatedModelRelations,
					allRelations,
					currentPath,
					processed,
				)...)
			}
		}
	}
	
	return joins
}
