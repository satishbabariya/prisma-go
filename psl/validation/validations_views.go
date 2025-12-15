// Package pslcore provides view validation functions.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateViewsSupport validates that views are supported by the connector.
func validateViewsSupport(ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityViews) {
		return
	}

	// Check all models to see if any are views
	models := ctx.Db.WalkModels()
	for _, model := range models {
		astModel := model.AstModel()
		if astModel == nil {
			continue
		}

		if astModel.IsView() {
			ctx.PushError(diagnostics.NewValidationError(
				"View definitions are not supported with the current connector.",
				diagnostics.NewSpan(astModel.Pos.Offset, astModel.Pos.Offset+len(astModel.Name.Name), model.FileID()),
			))
		}
	}
}

// validateViewDefinitionWithoutPreviewFlag validates that views require preview feature.
func validateViewDefinitionWithoutPreviewFlag(model *database.ModelWalker, ctx *ValidationContext) {
	astModel := model.AstModel()
	if astModel == nil {
		return
	}

	if !astModel.IsView() {
		return
	}

	// Check if Views preview feature is enabled
	if ctx.PreviewFeatures.Contains(PreviewFeatureViews) {
		return
	}

	ctx.PushError(diagnostics.NewValidationError(
		"View definitions are only available with the `views` preview feature.",
		diagnostics.NewSpan(astModel.Pos.Offset, astModel.Pos.Offset+len(astModel.Name.Name), model.FileID()),
	))
}

// validateViewPrimaryKey validates that views cannot have primary keys.
func validateViewPrimaryKey(pk *database.PrimaryKeyWalker, ctx *ValidationContext) {
	model := pk.Model()
	if model == nil {
		return
	}

	astModel := model.AstModel()
	if astModel == nil {
		return
	}

	if !astModel.IsView() {
		return
	}

	astAttr := pk.AstAttribute()
	if astAttr != nil {
		ctx.PushError(diagnostics.NewValidationError(
			"Views cannot have primary keys.",
			diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), model.FileID()),
		))
	}
}

// validateViewIndex validates that views cannot have indexes (except unique).
func validateViewIndex(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if model == nil {
		return
	}

	astModel := model.AstModel()
	if astModel == nil {
		return
	}

	if !astModel.IsView() {
		return
	}

	astAttr := index.AstAttribute()
	if astAttr == nil {
		return
	}

	// Non-unique indexes are not allowed
	if !index.IsUnique() {
		ctx.PushError(diagnostics.NewValidationError(
			"Views cannot have indexes.",
			diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), model.FileID()),
		))
		return
	}

	// Unique indexes cannot have mapped names
	if mappedName := index.MappedName(); mappedName != nil {
		ctx.PushError(diagnostics.NewValidationError(
			"@@unique annotations on views are not backed by unique indexes in the database and cannot specify a mapped database name.",
			diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), model.FileID()),
		))
	}

	// Unique indexes cannot be clustered
	if clustered := index.Clustered(); clustered != nil {
		ctx.PushError(diagnostics.NewValidationError(
			"@@unique annotations on views are not backed by unique indexes in the database and cannot be clustered.",
			diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), model.FileID()),
		))
	}
}

// validateViewIndexFieldAttribute validates index field attributes on views.
func validateViewIndexFieldAttribute(index *database.IndexWalker, field *database.IndexFieldWalker, ctx *ValidationContext) {
	if index == nil || field == nil {
		return
	}

	model := index.Model()
	if model == nil {
		return
	}

	astModel := model.AstModel()
	if astModel == nil {
		return
	}

	if !astModel.IsView() {
		return
	}

	// Check if field has length, operator class, or sort order
	hasLength := field.Length() != nil
	hasOperatorClass := field.OperatorClass() != nil
	hasSortOrder := field.SortOrder() != nil

	if hasLength || hasOperatorClass || hasSortOrder {
		astAttr := index.AstAttribute()
		if astAttr != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"Scalar fields in @@unique attributes in views cannot have arguments.",
				index.AttributeName(),
				diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), model.FileID()),
			))
		}
	}
}

// validateViewConnectorSpecific validates view-specific connector requirements.
func validateViewConnectorSpecific(model *database.ModelWalker, ctx *ValidationContext) {
	astModel := model.AstModel()
	if astModel == nil {
		return
	}

	if !astModel.IsView() {
		return
	}

	// Call connector-specific view validation
	if ctx.Connector != nil {
		diags := diagnostics.NewDiagnostics()
		modelWalker := model
		ctx.Connector.ValidateView(modelWalker, &diags)
		// Push any errors from connector validation
		for _, err := range diags.Errors() {
			ctx.PushError(err)
		}
	}
}
