// Package pslcore provides constraint namespace validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// ConstraintScope represents the scope of a constraint.
type ConstraintScope int

const (
	ConstraintScopeGlobal ConstraintScope = iota
	ConstraintScopeSchema
	ConstraintScopeModel
	ConstraintScopeGlobalKeyIndex
	ConstraintScopeGlobalForeignKey
	ConstraintScopeGlobalPrimaryKeyForeignKeyDefault
	ConstraintScopeModelKeyIndex
	ConstraintScopeModelPrimaryKeyKeyIndex
)

// ConstraintNamespace tracks constraint names within their scopes.
type ConstraintNamespace struct {
	// Global constraints (across all schemas)
	global map[string]int
	// Schema-scoped constraints
	schema map[string]map[string]int
	// Model-scoped constraints
	local map[database.ModelId]map[string]int
}

// NewConstraintNamespace creates a new constraint namespace tracker.
func NewConstraintNamespace() *ConstraintNamespace {
	return &ConstraintNamespace{
		global: make(map[string]int),
		schema: make(map[string]map[string]int),
		local:  make(map[database.ModelId]map[string]int),
	}
}

// AddGlobalConstraint adds a global constraint name.
func (cn *ConstraintNamespace) AddGlobalConstraint(name string) {
	cn.global[name]++
}

// AddSchemaConstraint adds a schema-scoped constraint name.
func (cn *ConstraintNamespace) AddSchemaConstraint(schemaName, constraintName string) {
	if cn.schema[schemaName] == nil {
		cn.schema[schemaName] = make(map[string]int)
	}
	cn.schema[schemaName][constraintName]++
}

// AddLocalConstraint adds a model-scoped constraint name.
func (cn *ConstraintNamespace) AddLocalConstraint(modelID database.ModelId, constraintName string) {
	if cn.local[modelID] == nil {
		cn.local[modelID] = make(map[string]int)
	}
	cn.local[modelID][constraintName]++
}

// CheckGlobalConstraint checks if a global constraint name is unique.
func (cn *ConstraintNamespace) CheckGlobalConstraint(name string) bool {
	return cn.global[name] <= 1
}

// CheckSchemaConstraint checks if a schema-scoped constraint name is unique.
func (cn *ConstraintNamespace) CheckSchemaConstraint(schemaName, constraintName string) bool {
	if cn.schema[schemaName] == nil {
		return true
	}
	return cn.schema[schemaName][constraintName] <= 1
}

// CheckLocalConstraint checks if a model-scoped constraint name is unique.
func (cn *ConstraintNamespace) CheckLocalConstraint(modelID database.ModelId, constraintName string) bool {
	if cn.local[modelID] == nil {
		return true
	}
	return cn.local[modelID][constraintName] <= 1
}

// validateIndexConstraintNamespace validates index constraint names within their namespace.
func validateIndexConstraintNamespace(index *database.IndexWalker, model *database.ModelWalker, namespace *ConstraintNamespace, ctx *ValidationContext) {
	// Get constraint name (mapped name or generated name)
	constraintName := ""
	if mappedName := index.MappedName(); mappedName != nil {
		constraintName = *mappedName
	} else if name := index.Name(); name != nil {
		constraintName = *name
	} else {
		// Generated name - skip validation
		return
	}

	// Get schema name
	schemaName := ""
	if schema := model.Schema(); schema != nil {
		schemaName = ctx.Db.GetString(schema.Name)
	}

	// Check namespace based on connector requirements
	// For now, check model-scoped uniqueness
	// TODO: Get ModelID from walker when available
	modelID := database.ModelId{} // Placeholder
	if !namespace.CheckLocalConstraint(modelID, constraintName) {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Index constraint name '%s' in model '%s' is not unique.", constraintName, model.Name()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}

	// If schema-scoped, check schema namespace
	if schemaName != "" && !namespace.CheckSchemaConstraint(schemaName, constraintName) {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Index constraint name '%s' in schema '%s' is not unique.", constraintName, schemaName),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validatePrimaryKeyConstraintNamespace validates primary key constraint names within their namespace.
func validatePrimaryKeyConstraintNamespace(pk *database.PrimaryKeyWalker, model *database.ModelWalker, namespace *ConstraintNamespace, ctx *ValidationContext) {
	mappedName := pk.MappedName()
	if mappedName == nil {
		return
	}

	constraintName := *mappedName
	schemaName := ""
	if schema := model.Schema(); schema != nil {
		schemaName = ctx.Db.GetString(schema.Name)
	}

	// Check namespace based on connector requirements
	modelID := database.ModelId{} // Placeholder
	if !namespace.CheckLocalConstraint(modelID, constraintName) {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Primary key constraint name '%s' in model '%s' is not unique.", constraintName, model.Name()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}

	if schemaName != "" && !namespace.CheckSchemaConstraint(schemaName, constraintName) {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Primary key constraint name '%s' in schema '%s' is not unique.", constraintName, schemaName),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}

// validateDefaultConstraintNamespace validates default constraint names within their namespace.
func validateDefaultConstraintNamespace(field *database.ScalarFieldWalker, model *database.ModelWalker, namespace *ConstraintNamespace, ctx *ValidationContext) {
	if field.IsIgnored() {
		return
	}

	defaultValue := field.DefaultValue()
	if defaultValue == nil {
		return
	}

	mappedName := defaultValue.MappedName()
	if mappedName == nil {
		return
	}

	constraintName := *mappedName
	schemaName := ""
	if schema := model.Schema(); schema != nil {
		schemaName = ctx.Db.GetString(schema.Name)
	}

	// Check namespace based on connector requirements
	modelID := database.ModelId{} // Placeholder
	if !namespace.CheckLocalConstraint(modelID, constraintName) {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Default constraint name '%s' in model '%s' is not unique.", constraintName, model.Name()),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}

	if schemaName != "" && !namespace.CheckSchemaConstraint(schemaName, constraintName) {
		ctx.PushError(diagnostics.NewValidationError(
			fmt.Sprintf("Default constraint name '%s' in schema '%s' is not unique.", constraintName, schemaName),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}
}
