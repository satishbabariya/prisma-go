// Package pslcore provides helper functions for index validation.
package validation

// validateModelIndexes validates all indexes across all models.
func validateModelIndexes(ctx *ValidationContext) {
	models := ctx.Db.WalkModels()
	for _, model := range models {
		if model.IsIgnored() {
			continue
		}

		validateIndexNameClashes(model, ctx)

		indexes := model.Indexes()
		for _, index := range indexes {
			validateIndexFields(index, model, ctx)
			validateIndexAlgorithm(index, model, ctx)
			validateFulltextIndexSupport(index, model, ctx)
			validateFulltextColumnsShouldNotDefineLength(index, model, ctx)
			validateFulltextColumnSortIsSupported(index, model, ctx)
			validateFulltextTextColumnsShouldBeBundledTogether(index, model, ctx)
			validateHashIndexMustNotUseSortParam(index, model, ctx)
			validateIndexClusteringSetting(index, model, ctx)
			validateClusteringCanBeDefinedOnlyOnce(index, model, ctx)
			validateOpclassesAreNotAllowedWithOtherThanNormalIndices(index, model, ctx)
			validateCompositeTypeInCompoundUniqueIndex(index, model, ctx)
			validateUniqueIndexClientNameDoesNotClashWithField(index, model, ctx)
			validateIndexFieldLengthPrefix(index, model, ctx)

			// View-specific index validations
			validateViewIndex(index, model, ctx)

			// Validate index field attributes for views
			for _, field := range index.Fields() {
				validateViewIndexFieldAttribute(index, field, ctx)
			}
		}
	}
}
