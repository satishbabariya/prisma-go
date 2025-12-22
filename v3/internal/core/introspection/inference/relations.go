// Package inference implements relation inference from database structures.
package inference

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/v3/internal/core/introspection/domain"
)

// RelationInferrer infers Prisma relations from database foreign keys.
type RelationInferrer struct{}

// NewRelationInferrer creates a new relation inferrer.
func NewRelationInferrer() *RelationInferrer {
	return &RelationInferrer{}
}

// InferRelations analyzes foreign keys and infers Prisma relations.
func (r *RelationInferrer) InferRelations(db *domain.IntrospectedDatabase) ([]domain.InferredRelation, error) {
	var relations []domain.InferredRelation

	// Build a map of tables for quick lookup
	tableMap := make(map[string]*domain.IntrospectedTable)
	for i := range db.Tables {
		tableMap[db.Tables[i].Name] = &db.Tables[i]
	}

	// Iterate through all tables and their foreign keys
	for _, table := range db.Tables {
		for _, fk := range table.ForeignKeys {
			// Infer relation from this foreign key
			rel, err := r.inferRelationFromFK(table, fk, tableMap)
			if err != nil {
				return nil, fmt.Errorf("failed to infer relation for FK %s: %w", fk.Name, err)
			}
			relations = append(relations, rel)

			// Also create the reverse relation
			reverseRel := r.createReverseRelation(rel, &table, tableMap[fk.ReferencedTable])
			relations = append(relations, reverseRel)
		}
	}

	return relations, nil
}

// inferRelationFromFK infers a relation from a foreign key (the "from" side).
func (r *RelationInferrer) inferRelationFromFK(
	table domain.IntrospectedTable,
	fk domain.IntrospectedForeignKey,
	tableMap map[string]*domain.IntrospectedTable,
) (domain.InferredRelation, error) {
	referencedTable := tableMap[fk.ReferencedTable]
	if referencedTable == nil {
		return domain.InferredRelation{}, fmt.Errorf("referenced table %s not found", fk.ReferencedTable)
	}

	// Determine relation type based on foreign key uniqueness
	relationType := r.determineRelationType(table, fk, referencedTable)

	// Generate relation name (e.g., "author" for "author_id" FK)
	relationName := r.generateRelationName(fk.ColumnNames[0], referencedTable.Name)

	return domain.InferredRelation{
		FromTable:    table.Name,
		ToTable:      fk.ReferencedTable,
		FromFields:   fk.ColumnNames,
		ToFields:     fk.ReferencedColumns,
		RelationType: relationType,
		RelationName: relationName,
		OnDelete:     fk.OnDelete,
		OnUpdate:     fk.OnUpdate,
	}, nil
}

// createReverseRelation creates the reverse side of a relation.
func (r *RelationInferrer) createReverseRelation(
	original domain.InferredRelation,
	fromTable *domain.IntrospectedTable,
	toTable *domain.IntrospectedTable,
) domain.InferredRelation {
	var reverseType domain.RelationType
	var reverseName string

	switch original.RelationType {
	case domain.OneToOne:
		reverseType = domain.OneToOne
		reverseName = pluralize(fromTable.Name)
	case domain.ManyToOne:
		reverseType = domain.OneToMany
		reverseName = pluralize(fromTable.Name)
	case domain.OneToMany:
		reverseType = domain.ManyToOne
		reverseName = singularize(fromTable.Name)
	default:
		reverseType = domain.ManyToMany
		reverseName = pluralize(fromTable.Name)
	}

	return domain.InferredRelation{
		FromTable:    original.ToTable,
		ToTable:      original.FromTable,
		FromFields:   original.ToFields,
		ToFields:     original.FromFields,
		RelationType: reverseType,
		RelationName: lowerFirst(reverseName),
		OnDelete:     original.OnDelete,
		OnUpdate:     original.OnUpdate,
	}
}

// determineRelationType determines if a relation is OneToOne or ManyToOne.
func (r *RelationInferrer) determineRelationType(
	table domain.IntrospectedTable,
	fk domain.IntrospectedForeignKey,
	referencedTable *domain.IntrospectedTable,
) domain.RelationType {
	// Check if the foreign key columns have a unique constraint
	isUnique := r.isUniqueConstraint(table, fk.ColumnNames)

	if isUnique {
		return domain.OneToOne
	}
	return domain.ManyToOne
}

// isUniqueConstraint checks if the given columns have a unique constraint.
func (r *RelationInferrer) isUniqueConstraint(table domain.IntrospectedTable, columns []string) bool {
	// Check indexes
	for _, idx := range table.Indexes {
		if idx.IsUnique && equalStringSlices(idx.Columns, columns) {
			return true
		}
	}

	// Check if columns themselves are marked as unique
	for _, col := range table.Columns {
		if col.IsUnique && contains(columns, col.Name) && len(columns) == 1 {
			return true
		}
	}

	return false
}

// generateRelationName generates a relation name from a foreign key column name.
// E.g., "author_id" -> "author", "user_profile_id" -> "userProfile"
func (r *RelationInferrer) generateRelationName(fkColumn, referencedTable string) string {
	// Remove common suffixes like "_id", "_fk"
	name := strings.TrimSuffix(strings.ToLower(fkColumn), "_id")
	name = strings.TrimSuffix(name, "_fk")

	// Convert to camelCase
	return toCamelCase(name)
}

// Helper functions

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func toCamelCase(s string) string {
	words := strings.Split(s, "_")
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			words[i] = strings.ToUpper(words[i][:1]) + words[i][1:]
		}
	}
	return strings.Join(words, "")
}

func lowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func pluralize(s string) string {
	// Simple pluralization (can be improved with a library)
	if strings.HasSuffix(s, "s") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

func singularize(s string) string {
	// Simple singularization
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

// Ensure RelationInferrer implements domain.RelationInferrer
var _ domain.RelationInferrer = (*RelationInferrer)(nil)
