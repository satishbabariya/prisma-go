// Package pslcore provides model validation functions.
package validation

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateModelDatabaseNameClashes checks for database name clashes between models.
func validateModelDatabaseNameClashes(ctx *ValidationContext) {
	modelNames := make(map[string][]*database.ModelWalker)

	models := ctx.Db.WalkModels()
	for _, model := range models {
		if model.IsIgnored() {
			continue
		}
		dbName := model.DatabaseName()
		modelNames[dbName] = append(modelNames[dbName], model)
	}

	for dbName, models := range modelNames {
		if len(models) > 1 {
			modelNamesList := ""
			for i, m := range models {
				if i > 0 {
					modelNamesList += ", "
				}
				modelNamesList += m.Name()
			}
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Multiple models have the same database name '%s': %s", dbName, modelNamesList),
				diagnostics.NewSpan(0, 0, models[0].FileID()),
			))
		}
	}
}

// validateModelHasStrictUniqueCriteria checks that a model has a strict unique criteria.
func validateModelHasStrictUniqueCriteria(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	// Find a strict unique criteria (has only required fields and no unsupported fields)
	var strictCriteria *database.UniqueCriteriaWalker
	for _, criteria := range model.UniqueCriterias() {
		if criteria.IsStrictCriteria() && !criteria.HasUnsupportedFields() {
			strictCriteria = criteria
			break
		}
	}

	if strictCriteria != nil {
		return
	}

	// No strict criteria found - collect loose criterias for error message
	var looseCriterias []string
	for _, criteria := range model.UniqueCriterias() {
		fieldNames := make([]string, 0)
		for _, field := range criteria.Fields() {
			scalarField := field.ScalarField()
			if scalarField != nil {
				fieldNames = append(fieldNames, scalarField.Name())
			}
		}
		if len(fieldNames) > 0 {
			looseCriterias = append(looseCriterias, fmt.Sprintf("- %s", strings.Join(fieldNames, ", ")))
		}
	}

	containerType := "model"
	astModel := model.AstModel()
	if astModel != nil && astModel.IsView() {
		containerType = "view"
	}

	msg := fmt.Sprintf("Each %s must have at least one unique criteria that has only required fields. Either mark a single field with `@id`, `@unique` or add a multi field criterion with `@@id([])` or `@@unique([])` to the %s.", containerType, containerType)

	if len(looseCriterias) > 0 {
		suffix := fmt.Sprintf("The following unique criterias were not considered as they contain fields that are not required:\n%s", strings.Join(looseCriterias, "\n"))
		msg = fmt.Sprintf("%s %s", msg, suffix)
	}

	var span diagnostics.Span
	if astModel != nil {
		span = diagnostics.NewSpan(astModel.Pos.Offset, astModel.Pos.Offset+len(astModel.Name.Name), model.FileID())
	} else {
		span = diagnostics.NewSpan(0, 0, model.FileID())
	}

	ctx.PushError(diagnostics.NewValidationError(
		msg,
		span,
	))
}

// validateModelIdHasFields checks that @@id has fields specified.
func validateModelIdHasFields(model *database.ModelWalker, ctx *ValidationContext) {
	pk := model.PrimaryKey()
	if pk == nil {
		return
	}

	// Check if this is a @@id (not @id)
	if pk.SourceField() == nil {
		fields := pk.Fields()
		if len(fields) == 0 {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Model '%s' @@id must specify at least one field.", model.Name()),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
	}
}

// validateModelSchemaAttribute checks that @@schema is properly defined.
func validateModelSchemaAttribute(model *database.ModelWalker, ctx *ValidationContext) {
	if ctx.Datasource == nil {
		return
	}

	schema := model.Schema()
	if schema != nil {
		// Get schema name from StringId
		schemaName := ctx.Db.GetString(schema.Name)

		// Check if schema is in datasource schemas list
		if len(ctx.Datasource.Schemas) > 0 {
			found := false
			for _, dsSchema := range ctx.Datasource.Schemas {
				if dsSchema == schemaName {
					found = true
					break
				}
			}
			if !found {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Model '%s' schema '%s' is not defined in datasource schemas.", model.Name(), schemaName),
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}
		} else if !ctx.HasCapability(ConnectorCapabilityMultiSchema) {
			ctx.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Model '%s' has @@schema attribute but connector does not support multi-schema.", model.Name()),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
		}
	}
}

// validatePrimaryKeyConnectorSpecific validates primary key connector-specific requirements.
func validatePrimaryKeyConnectorSpecific(model *database.ModelWalker, ctx *ValidationContext) {
	pk := model.PrimaryKey()
	if pk == nil {
		return
	}

	containerType := "model"
	if model.IsView() {
		containerType = "view"
	}

	// Check if named primary keys are supported
	if pk.MappedName() != nil && !ctx.HasCapability(ConnectorCapabilityNamedPrimaryKeys) {
		ctx.PushError(diagnostics.NewModelValidationError(
			"You defined a database name for the primary key on the model. This is not supported by the provider.",
			containerType,
			model.Name(),
			diagnostics.NewSpan(0, 0, model.FileID()),
		))
	}

	// Check if compound primary keys are supported
	fields := pk.Fields()
	if len(fields) > 1 && !ctx.HasCapability(ConnectorCapabilityCompoundIds) {
		astModel := model.AstModel()
		if astModel != nil {
			ctx.PushError(diagnostics.NewModelValidationError(
				"The current connector does not support compound ids.",
				containerType,
				model.Name(),
				diagnostics.NewSpan(astModel.Pos.Offset, astModel.Pos.Offset+len(astModel.Name.Name), model.FileID()),
			))
		}
	}
}

// validatePrimaryKeyLengthPrefixSupported validates that primary key length prefix is supported.
func validatePrimaryKeyLengthPrefixSupported(model *database.ModelWalker, ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityIndexColumnLengthPrefixing) {
		return
	}

	pk := model.PrimaryKey()
	if pk == nil {
		return
	}

	// Check if any field has length()
	fields := pk.Fields()
	for _, field := range fields {
		if field.Length() != nil {
			containerType := "model"
			if model.IsView() {
				containerType = "view"
			}
			ctx.PushError(diagnostics.NewModelValidationError(
				"The current connector does not support length prefix on primary key fields.",
				containerType,
				model.Name(),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
			return
		}
	}
}

// validatePrimaryKeySortOrderSupported validates that primary key sort order is supported.
func validatePrimaryKeySortOrderSupported(model *database.ModelWalker, ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityPrimaryKeySortOrderDefinition) {
		return
	}

	pk := model.PrimaryKey()
	if pk == nil {
		return
	}

	// Check if any field has sort_order()
	fields := pk.Fields()
	for _, field := range fields {
		if field.SortOrder() != nil {
			containerType := "model"
			if model.IsView() {
				containerType = "view"
			}
			ctx.PushError(diagnostics.NewModelValidationError(
				"The current connector does not support sort order on primary key fields.",
				containerType,
				model.Name(),
				diagnostics.NewSpan(0, 0, model.FileID()),
			))
			return
		}
	}
}

// validatePrimaryKeyClientNameDoesNotClashWithField validates that primary key client name doesn't clash with field names.
func validatePrimaryKeyClientNameDoesNotClashWithField(model *database.ModelWalker, ctx *ValidationContext) {
	pk := model.PrimaryKey()
	if pk == nil {
		return
	}

	fields := pk.Fields()
	if len(fields) <= 1 {
		return
	}

	// Generate client name from field names
	fieldNames := []string{}
	for _, field := range fields {
		sf := field.ScalarField()
		if sf != nil {
			fieldNames = append(fieldNames, sf.Name())
		}
	}

	idClientName := ""
	for i, name := range fieldNames {
		if i > 0 {
			idClientName += "_"
		}
		idClientName += name
	}

	// Check if a field with this name exists
	for _, field := range model.ScalarFields() {
		if field.Name() == idClientName {
			containerType := "model"
			if model.IsView() {
				containerType = "view"
			}
			astModel := model.AstModel()
			if astModel != nil {
				ctx.PushError(diagnostics.NewModelValidationError(
					fmt.Sprintf("The field `%s` clashes with the `@@id` attribute's name. Please resolve the conflict by providing a custom id name: `@@id([...], name: \"custom_name\")`", idClientName),
					containerType,
					model.Name(),
					diagnostics.NewSpan(astModel.Pos.Offset, astModel.Pos.Offset+len(astModel.Name.Name), model.FileID()),
				))
			}
			return
		}
	}
}

// validateShardKeyIsSupported validates that shard key is supported by the connector.
func validateShardKeyIsSupported(model *database.ModelWalker, ctx *ValidationContext) {
	shardKey := model.ShardKey()
	if shardKey == nil {
		return
	}

	// Check if ShardKeys preview feature is enabled
	if !ctx.PreviewFeatures.Contains(PreviewFeatureShardKeys) {
		attr := shardKey.AstAttribute()
		if attr != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"Defining shard keys requires enabling the `shardKeys` preview feature",
				shardKey.AttributeName(),
				diagnostics.NewSpan(attr.Pos.Offset, attr.Pos.Offset+len(attr.String()), model.FileID()),
			))
		}
		return
	}

	// Check if connector supports shard keys
	if ctx.Connector != nil && !ctx.Connector.SupportsShardKeys() {
		attr := shardKey.AstAttribute()
		if attr != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("Shard keys are not currently supported for provider %s", ctx.Connector.ProviderName()),
				shardKey.AttributeName(),
				diagnostics.NewSpan(attr.Pos.Offset, attr.Pos.Offset+len(attr.String()), model.FileID()),
			))
		}
	}
}

// validateShardKeyHasFields validates that shard key has fields.
func validateShardKeyHasFields(model *database.ModelWalker, ctx *ValidationContext) {
	shardKey := model.ShardKey()
	if shardKey == nil {
		return
	}

	// @@shardKey must have at least one field
	if len(shardKey.Fields()) == 0 {
		attr := shardKey.AstAttribute()
		if attr != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				fmt.Sprintf("The list of fields in a `%s()` attribute cannot be empty. Please specify at least one field.", shardKey.AttributeName()),
				shardKey.AttributeName(),
				diagnostics.NewSpan(attr.Pos.Offset, attr.Pos.Offset+len(attr.String()), model.FileID()),
			))
		}
	}
}
