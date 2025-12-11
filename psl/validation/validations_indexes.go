// Package pslcore provides index validation functions.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateIndexNameClashes checks for index name clashes within a model.
func validateIndexNameClashes(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	indexes := model.Indexes()
	indexNames := make(map[string]bool)
	mappedNames := make(map[string]bool)

	for _, index := range indexes {
		// Check explicit names
		if name := index.Name(); name != nil {
			if indexNames[*name] {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Model '%s' has duplicate index name '%s'.", model.Name(), *name),
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}
			indexNames[*name] = true
		}

		// Check mapped names
		if mappedName := index.MappedName(); mappedName != nil {
			if mappedNames[*mappedName] {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Model '%s' has duplicate index mapped name '%s'.", model.Name(), *mappedName),
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}
			mappedNames[*mappedName] = true
		}
	}
}

// validateIndexFields validates that indexes have valid fields.
func validateIndexFields(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	fields := index.Fields()
	if len(fields) == 0 {
		astAttr := index.AstAttribute()
		if astAttr != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The list of fields in an index cannot be empty. Please specify at least one field.",
				index.AttributeName(),
				astAttr.Span,
			))
		} else {
			astModel := model.AstModel()
			if astModel != nil {
				// Use model span as fallback
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"The list of fields in an index cannot be empty. Please specify at least one field.",
					"@@index",
					astModel.Span(),
				))
			}
		}
		return
	}

	// Validate each field exists and is a scalar field
	for _, field := range fields {
		sf := field.ScalarField()
		if sf == nil {
			astAttr := index.AstAttribute()
			if astAttr != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					fmt.Sprintf("Index on model '%s' references invalid field.", model.Name()),
					index.AttributeName(),
					astAttr.Span,
				))
			} else {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Index on model '%s' references invalid field.", model.Name()),
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}
			continue
		}

		if sf.IsIgnored() {
			astField := sf.AstField()
			if astField != nil {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Index on model '%s' references ignored field '%s'.", model.Name(), sf.Name()),
					astField.Span(),
				))
			} else {
				ctx.PushError(diagnostics.NewValidationError(
					fmt.Sprintf("Index on model '%s' references ignored field '%s'.", model.Name(), sf.Name()),
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}
		}
	}
}

// validateIndexAlgorithm validates index algorithm support.
func validateIndexAlgorithm(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	algorithm := index.Algorithm()
	if algorithm == nil {
		return
	}

	// Convert database.IndexAlgorithm to pslcore.IndexAlgorithm
	algo := IndexAlgorithm(*algorithm)
	if ctx.Connector.SupportsIndexType(algo) {
		return
	}

	message := "The given index type is not supported with the current connector"
	astAttr := index.AstAttribute()
	if astAttr == nil {
		return
	}

	// Try to get span for "type" argument, otherwise use full attribute span
	span := astAttr.Span
	if typeSpan := index.SpanForArgument("type"); typeSpan != nil {
		span = *typeSpan
	}

	ctx.PushError(diagnostics.NewAttributeValidationError(
		message,
		index.AttributeName(),
		span,
	))
}

// validateFulltextIndexSupport validates that fulltext indexes are supported.
func validateFulltextIndexSupport(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !index.IsFulltext() {
		return
	}

	if !ctx.HasCapability(ConnectorCapabilityFullTextIndex) {
		astModel := model.AstModel()
		if astModel != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"Defining fulltext indexes is not supported with the current connector.",
				"@@fulltext",
				astModel.Span(),
			))
		}
	}
}

// validateFulltextColumnsShouldNotDefineLength validates that fulltext index columns don't define length.
func validateFulltextColumnsShouldNotDefineLength(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityFullTextIndex) {
		return
	}

	if !index.IsFulltext() {
		return
	}

	// Check if any field has length()
	for _, fieldAttr := range index.ScalarFieldAttributes() {
		if fieldAttr.Length() != nil {
			astAttr := index.AstAttribute()
			if astAttr == nil {
				return
			}

			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The length argument is not supported in a @@fulltext attribute.",
				index.AttributeName(),
				astAttr.Span,
			))
			return
		}
	}
}

// validateFulltextColumnSortIsSupported validates that fulltext index sort order is supported.
func validateFulltextColumnSortIsSupported(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityFullTextIndex) {
		return
	}

	if !index.IsFulltext() {
		return
	}

	if ctx.HasCapability(ConnectorCapabilitySortOrderInFullTextIndex) {
		return
	}

	// Check if any field has sort_order()
	for _, fieldAttr := range index.ScalarFieldAttributes() {
		if fieldAttr.SortOrder() != nil {
			astAttr := index.AstAttribute()
			if astAttr == nil {
				return
			}

			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The sort argument is not supported in a @@fulltext attribute in the current connector.",
				index.AttributeName(),
				astAttr.Span,
			))
			return
		}
	}
}

// validateFulltextTextColumnsShouldBeBundledTogether validates that text columns in fulltext indexes are bundled.
func validateFulltextTextColumnsShouldBeBundledTogether(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityFullTextIndex) {
		return
	}

	if !index.IsFulltext() {
		return
	}

	if !ctx.HasCapability(ConnectorCapabilitySortOrderInFullTextIndex) {
		return
	}

	// State machine to validate text columns are bundled together
	type State int
	const (
		StateInit State = iota
		StateSortParamHead
		StateTextFieldBundle
		StateSortParamTail
	)

	state := StateInit
	for _, fieldAttr := range index.ScalarFieldAttributes() {
		hasSortOrder := fieldAttr.SortOrder() != nil
		switch state {
		case StateInit:
			if hasSortOrder {
				state = StateSortParamHead
			} else {
				state = StateTextFieldBundle
			}
		case StateSortParamHead:
			if hasSortOrder {
				state = StateSortParamHead
			} else {
				state = StateTextFieldBundle
			}
		case StateTextFieldBundle:
			if hasSortOrder {
				state = StateSortParamTail
			} else {
				state = StateTextFieldBundle
			}
		case StateSortParamTail:
			if hasSortOrder {
				state = StateSortParamTail
			} else {
				// Error: text fields after sort params
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"All index fields must be listed adjacently in the fields argument.",
					index.AttributeName(),
					index.AstAttribute().Span,
				))
				return
			}
		}
	}
}

// validateHashIndexMustNotUseSortParam validates that hash indexes don't use sort parameters.
func validateHashIndexMustNotUseSortParam(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	// Check if connector supports hash index type
	if !ctx.Connector.SupportsIndexType(IndexAlgorithmHash) {
		return
	}

	// Check if index algorithm is hash
	algo := index.Algorithm()
	if algo == nil || *algo != database.IndexAlgorithmHash {
		return
	}

	// Check if any field has sort_order()
	for _, fieldAttr := range index.ScalarFieldAttributes() {
		if fieldAttr.SortOrder() != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"Hash type does not support sort option.",
				index.AttributeName(),
				index.AstAttribute().Span,
			))
			return
		}
	}
}

// validateIndexClusteringSetting validates clustering setting support.
func validateIndexClusteringSetting(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityClusteringSetting) {
		return
	}

	clustered := index.Clustered()
	if clustered == nil {
		return
	}

	// If clustering is explicitly set (true or false), but capability is not available, error
	astAttr := index.AstAttribute()
	if astAttr == nil {
		return
	}

	ctx.PushError(diagnostics.NewAttributeValidationError(
		"Defining clustering is not supported in the current connector.",
		index.AttributeName(),
		astAttr.Span,
	))
}

// validateClusteringCanBeDefinedOnlyOnce validates that clustering is defined only once per model.
func validateClusteringCanBeDefinedOnlyOnce(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityClusteringSetting) {
		return
	}

	clustered := index.Clustered()
	if clustered == nil || !*clustered {
		return
	}

	// Check if primary key is clustered
	if pk := model.PrimaryKey(); pk != nil {
		pkClustered := pk.Clustered()
		// If primary key is clustered (true) or not explicitly set (nil, defaults to clustered), error
		if pkClustered == nil || (pkClustered != nil && *pkClustered) {
			astAttr := index.AstAttribute()
			if astAttr == nil {
				return
			}

			ctx.PushError(diagnostics.NewAttributeValidationError(
				"A model can only hold one clustered index or key.",
				index.AttributeName(),
				astAttr.Span,
			))
			return
		}
	}

	// Check if any other index is clustered
	// Compare indexes by checking if they have the same fields and type
	indexFields := index.Fields()
	indexFieldIDs := make([]database.ScalarFieldId, len(indexFields))
	for i, f := range indexFields {
		indexFieldIDs[i] = f.FieldID()
	}

	for _, otherIndex := range model.Indexes() {
		// Skip if different type
		if index.Type() != otherIndex.Type() {
			continue
		}

		// Compare fields to see if it's the same index
		otherFields := otherIndex.Fields()
		if len(indexFieldIDs) != len(otherFields) {
			continue
		}

		isSame := true
		for i, f := range otherFields {
			if f.FieldID() != indexFieldIDs[i] {
				isSame = false
				break
			}
		}

		// Skip if this is the same index
		if isSame {
			continue
		}

		// Check if other index is clustered
		otherClustered := otherIndex.Clustered()
		if otherClustered != nil && *otherClustered {
			astAttr := index.AstAttribute()
			if astAttr == nil {
				return
			}

			ctx.PushError(diagnostics.NewAttributeValidationError(
				"A model can only hold one clustered index.",
				index.AttributeName(),
				astAttr.Span,
			))
			return
		}
	}
}

// validateOpclassesAreNotAllowedWithOtherThanNormalIndices validates that operator classes are only allowed in normal indices.
func validateOpclassesAreNotAllowedWithOtherThanNormalIndices(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	// Check if this is NOT a normal index (unique or fulltext)
	if index.IsUnique() || index.IsFulltext() {
		// Check if any field has operator_class()
		for _, fieldAttr := range index.ScalarFieldAttributes() {
			if fieldAttr.OperatorClass() != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"Operator classes are only allowed in normal indices, not in @@unique or @@fulltext.",
					index.AttributeName(),
					index.AstAttribute().Span,
				))
				return
			}
		}
	}
}

// validateCompositeTypeInCompoundUniqueIndex validates that composite types are not in compound unique indices.
func validateCompositeTypeInCompoundUniqueIndex(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !index.IsUnique() {
		return
	}

	fields := index.Fields()
	if len(fields) <= 1 {
		return
	}

	// Check if any field is a composite type
	for _, field := range fields {
		sf := field.ScalarField()
		if sf != nil {
			fieldType := sf.ScalarFieldType()
			if fieldType.CompositeTypeID != nil {
				astModel := model.AstModel()
				if astModel != nil {
					ctx.PushError(diagnostics.NewAttributeValidationError(
						fmt.Sprintf("Prisma does not currently support composite types in compound unique indices, please remove %s from the index. See https://pris.ly/d/mongodb-composite-compound-indices for more details", sf.Name()),
						"@@unique",
						astModel.Span(),
					))
				}
				return
			}
		}
	}
}

// validateUniqueIndexClientNameDoesNotClashWithField validates that unique index client names don't clash with field names.
func validateUniqueIndexClientNameDoesNotClashWithField(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if !index.IsUnique() {
		return
	}

	fields := index.Fields()
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

	idxClientName := ""
	for i, name := range fieldNames {
		if i > 0 {
			idxClientName += "_"
		}
		idxClientName += name
	}

	// Check if a field with this name exists
	for _, field := range model.ScalarFields() {
		if field.Name() == idxClientName {
			containerType := "model"
			if model.IsView() {
				containerType = "view"
			}
			astModel := model.AstModel()
			if astModel != nil {
				ctx.PushError(diagnostics.NewModelValidationError(
					fmt.Sprintf("The field `%s` clashes with the `@@unique` name. Please resolve the conflict by providing a custom id name: `@@unique([...], name: \"custom_name\")`", idxClientName),
					containerType,
					model.Name(),
					astModel.Span(),
				))
			}
			return
		}
	}
}

// validateIndexFieldLengthPrefix validates that index field length prefix is supported.
func validateIndexFieldLengthPrefix(index *database.IndexWalker, model *database.ModelWalker, ctx *ValidationContext) {
	if ctx.HasCapability(ConnectorCapabilityIndexColumnLengthPrefixing) {
		return
	}

	// Check if any field has length attribute
	for _, field := range index.Fields() {
		if field.Length() != nil {
			astAttr := index.AstAttribute()
			if astAttr != nil {
				ctx.PushError(diagnostics.NewAttributeValidationError(
					"The length argument is not supported in an index definition with the current connector",
					index.AttributeName(),
					astAttr.Span,
				))
			}
			return
		}
	}
	_ = model
}

// validateOnlyOneFulltextAttribute validates that only one fulltext index exists per model.
func validateOnlyOneFulltextAttribute(model *database.ModelWalker, ctx *ValidationContext) {
	if model.IsIgnored() {
		return
	}

	if !ctx.HasCapability(ConnectorCapabilityFullTextIndex) {
		return
	}

	if ctx.HasCapability(ConnectorCapabilityMultipleFullTextAttributesPerModel) {
		return
	}

	fulltextCount := 0
	indexes := model.Indexes()
	for _, index := range indexes {
		if index.IsFulltext() {
			fulltextCount++
		}
	}

	if fulltextCount > 1 {
		astModel := model.AstModel()
		if astModel != nil {
			ctx.PushError(diagnostics.NewAttributeValidationError(
				"The current connector only allows one fulltext attribute per model",
				"@@fulltext",
				astModel.Span(),
			))
		}
	}
}
