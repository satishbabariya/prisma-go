// Package pslcore provides validation functionality for Prisma schemas.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// ValidateOutput represents the output of validation.
type ValidateOutput struct {
	Db           *database.ParserDatabase
	Diagnostics  diagnostics.Diagnostics
	RelationMode RelationMode
	Connector    Connector
}

// ParseWithoutValidation parses a schema but skips validations.
func ParseWithoutValidation(
	db *database.ParserDatabase,
	sources []Datasource,
) ParseOutput {
	var connector Connector
	var relationMode RelationMode = RelationModePrisma

	if len(sources) > 0 {
		source := sources[0]
		// Select appropriate connector based on provider
		builtinConnectors := NewBuiltinConnectors()
		connector = builtinConnectors.GetConnector(source.Provider)

		// Extract relation mode from datasource configuration
		relationMode = source.RelationMode()
		if relationMode == "" {
			// Default to connector's default relation mode if available
			if connector != nil {
				relationMode = connector.DefaultRelationMode()
			} else {
				relationMode = RelationModePrisma
			}
		}
	}

	return ParseOutput{
		Db:           db,
		RelationMode: relationMode,
		Connector:    connector,
	}
}

// ParseOutput represents the output of parsing without validation.
type ParseOutput struct {
	Db           *database.ParserDatabase
	RelationMode RelationMode
	Connector    Connector
}

// Validate validates a Prisma schema.
func ValidateSchema(
	db *database.ParserDatabase,
	sources []Datasource,
	previewFeatures PreviewFeatures,
	diags diagnostics.Diagnostics,
	extensionTypes database.ExtensionTypes,
) ValidateOutput {
	parseOutput := ParseWithoutValidation(db, sources)

	output := ValidateOutput{
		Db:           parseOutput.Db,
		RelationMode: parseOutput.RelationMode,
		Connector:    parseOutput.Connector,
		Diagnostics:  diags,
	}

	// Early return if there are already errors
	if output.Diagnostics.HasErrors() {
		return output
	}

	var source *Datasource
	if len(sources) > 0 {
		source = &sources[0]
	}

	maxLength := 64 // Default
	if source != nil && parseOutput.Connector != nil {
		maxLength = parseOutput.Connector.MaxIdentifierLength()
		if maxLength == 0 {
			maxLength = 64 // Fallback to default
		}
	}

	ctx := &ValidationContext{
		Db:                  db,
		Datasource:          source,
		PreviewFeatures:     previewFeatures,
		Connector:           parseOutput.Connector,
		RelationMode:        parseOutput.RelationMode,
		Diagnostics:         &output.Diagnostics,
		ExtensionTypes:      extensionTypes,
		maxIdentifierLength: maxLength,
	}

	// Run validations
	runValidations(ctx)

	return output
}

// ValidationContext provides context for validation operations.
type ValidationContext struct {
	Db                  *database.ParserDatabase
	Datasource          *Datasource
	PreviewFeatures     PreviewFeatures
	Connector           Connector
	RelationMode        RelationMode
	Diagnostics         *diagnostics.Diagnostics
	ExtensionTypes      database.ExtensionTypes
	maxIdentifierLength int
}

// MaxIdentifierLength returns the maximum identifier length for the connector.
func (ctx *ValidationContext) MaxIdentifierLength() int {
	if ctx.maxIdentifierLength > 0 {
		return ctx.maxIdentifierLength
	}
	// Default max length if not set
	return 64
}

// PushError adds an error to the diagnostics.
func (ctx *ValidationContext) PushError(err diagnostics.DatamodelError) {
	ctx.Diagnostics.PushError(err)
}

// PushWarning adds a warning to the diagnostics.
func (ctx *ValidationContext) PushWarning(warn diagnostics.DatamodelWarning) {
	ctx.Diagnostics.PushWarning(warn)
}

// HasCapability checks if the connector has a specific capability.
func (ctx *ValidationContext) HasCapability(capability ConnectorCapability) bool {
	if ctx.Connector == nil {
		return false
	}
	return ctx.Connector.HasCapability(capability)
}

// PreviewFeatures is now defined in preview_features.go

// ConnectorCapability represents a connector capability.
type ConnectorCapability int

const (
	ConnectorCapabilityEnums ConnectorCapability = iota
	ConnectorCapabilityCompositeTypes
	ConnectorCapabilityJson
	ConnectorCapabilityJsonLists
	ConnectorCapabilityScalarLists
	ConnectorCapabilityFullTextIndex
	ConnectorCapabilityMultiSchema
	ConnectorCapabilityViews
	ConnectorCapabilityAutoIncrement
	ConnectorCapabilityAutoIncrementMultipleAllowed
	ConnectorCapabilityAutoIncrementAllowedOnNonId
	ConnectorCapabilityAutoIncrementNonIndexedAllowed
	ConnectorCapabilityNamedDefaultValues
	ConnectorCapabilityIndexColumnLengthPrefixing
	ConnectorCapabilitySortOrderInFullTextIndex
	ConnectorCapabilityClusteringSetting
	ConnectorCapabilityMultipleFullTextAttributesPerModel
	ConnectorCapabilityNamedPrimaryKeys
	ConnectorCapabilityNamedForeignKeys
	ConnectorCapabilityCompoundIds
	ConnectorCapabilityPrimaryKeySortOrderDefinition
	ConnectorCapabilityDefaultValueAuto
	ConnectorCapabilityImplicitManyToManyRelation
	ConnectorCapabilityTwoWayEmbeddedManyToManyRelation
)

// runValidations executes all validation passes.
func runValidations(ctx *ValidationContext) {
	// Initialize name tracking
	names := NewNames()

	// Basic validations
	// Datasource validations
	if ctx.Datasource != nil {
		validateDatasource(ctx)
	}

	// Validate scalar field unknown default functions (connector-specific)
	if ctx.Connector != nil {
		ctx.Connector.ValidateScalarFieldUnknownDefaultFunctions(ctx.Db, ctx.Diagnostics)
	}

	// Composite type validations
	validateCompositeTypeCycles(ctx)
	validateCompositeTypesSupport(ctx)
	validateCompositeTypes(ctx)

	// Model validations
	validateModelDatabaseNameClashes(ctx)
	validateModelsWithNames(ctx, names)

	// Index validations (after models are validated)
	validateModelIndexesWithNames(ctx, names)

	// Enum validations
	if ctx.HasCapability(ConnectorCapabilityEnums) {
		validateEnumDatabaseNameClashes(ctx)
	}
	validateEnums(ctx)

	// Relation validations
	validateRelationsWithNames(ctx, names)

	// View validations
	validateViewsSupport(ctx)
}

// validateModelsWithNames validates models with name tracking.
func validateModelsWithNames(ctx *ValidationContext, names *Names) {
	models := ctx.Db.WalkModels()
	for _, model := range models {
		validateModelWithNames(model, ctx, names)
	}
}

// validateModelWithNames validates a single model with name tracking.
func validateModelWithNames(model *database.ModelWalker, ctx *ValidationContext, names *Names) {
	// Basic validations
	if model.IsIgnored() {
		return
	}

	astModel := model.AstModel()
	isView := astModel != nil && astModel.IsView()

	// View-specific validations
	if isView {
		validateViewDefinitionWithoutPreviewFlag(model, ctx)
		validateViewConnectorSpecific(model, ctx)
	}

	// Model-specific validations (only for non-views)
	if !isView {
		validateModelHasStrictUniqueCriteria(model, ctx)
		validateModelIdHasFields(model, ctx)
		validateAutoIncrement(model, ctx)
		validateOnlyOneFulltextAttribute(model, ctx)
		validateShardKeyIsSupported(model, ctx)
		validateShardKeyHasFields(model, ctx)
	}

	// Common validations for both models and views
	validateModelSchemaAttribute(model, ctx)
	validateUniquePrimaryKeyName(model, names, ctx)
	validateModelDatabaseName(model, ctx)
	validateModelHasAtLeastOneField(model, ctx)
	validateModelUniqueConstraints(model, ctx)
	validateModelIndexConstraints(model, ctx)
	validateModelShardKeyConstraints(model, ctx)
	validateModelFieldNames(model, ctx)
	validatePrimaryKeyConnectorSpecific(model, ctx)
	validatePrimaryKeyLengthPrefixSupported(model, ctx)
	validatePrimaryKeySortOrderSupported(model, ctx)
	validatePrimaryKeyClientNameDoesNotClashWithField(model, ctx)

	// Validate primary key database name
	if pk := model.PrimaryKey(); pk != nil {
		validatePrimaryKeyDatabaseName(pk, model, ctx)
	}

	// Validate primary key
	pk := model.PrimaryKey()
	if pk != nil {
		if isView {
			// Views cannot have primary keys
			validateViewPrimaryKey(pk, ctx)
		} else {
			fields := pk.Fields()
			if len(fields) == 0 {
				ctx.PushError(diagnostics.NewValidationError(
					"Primary key must have at least one field.",
					diagnostics.NewSpan(0, 0, model.FileID()),
				))
			}

			// Validate primary key clustering settings
			validatePrimaryKeyClusteringSetting(pk, ctx)
			validatePrimaryKeyClusteringCanBeDefinedOnlyOnce(pk, ctx)

			// Validate primary key field attributes (length, etc.)
			for _, fieldAttr := range pk.ScalarFieldAttributes() {
				astAttr := pk.AstAttribute()
				if astAttr == nil {
					continue
				}
				span := diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), model.FileID())
				validateLengthUsedWithCorrectTypesForPrimaryKey(fieldAttr, pk.AttributeName(), span, ctx)
			}
		}
	}

	// Validate scalar fields
	for _, field := range model.ScalarFields() {
		validateScalarFieldConnectorSpecific(field, ctx)
		validateScalarFieldWithNames(field, model, ctx, names)
	}

	// Validate relation fields
	for _, field := range model.RelationFields() {
		validateRelationField(field, ctx)
		validateRelationFieldArity(field, ctx)
		validateRelationFieldSelfReference(field, ctx)
		validateRelationFieldBackReference(field, ctx)
		validateRelationFieldRequired(field, ctx)
		validateRelationFieldIndex(field, ctx)
		validateRelationFieldAmbiguity(field, names, ctx)
		validateRelationFieldIgnoredRelatedModel(field, ctx)
		validateRelationFieldReferentialActionsAdvanced(field, ctx)
		validateRelationFieldMap(field, ctx)
		validateRelationFieldMissingIndexes(field, ctx)
	}
}

// validateScalarFieldWithNames validates a scalar field with name tracking.
func validateScalarFieldWithNames(field *database.ScalarFieldWalker, model *database.ModelWalker, ctx *ValidationContext, names *Names) {
	// Basic validations
	if field.IsIgnored() {
		return
	}

	// Field-specific validations
	validateScalarFieldDefaultValue(field, ctx)
	validateScalarFieldNativeType(field, ctx)
	validateScalarFieldClientName(field, ctx)
	validateUniqueDefaultConstraintName(field, names, ctx)
	validateFieldClientName(field, model, names, ctx)
	validateDefaultValueType(field, ctx)
	validateNamedDefaultValues(field, ctx)
	validateFieldDatabaseName(field, model, ctx)
	validateDefaultDatabaseName(field, model, ctx)
	validateScalarTypeSupport(field, ctx)
	validateNativeTypeSupport(field, ctx)
	validateFieldArity(field, ctx)
	validateCompositeTypeFieldSupport(field, ctx)
	validateEnumFieldSupport(field, ctx)
	validateNativeTypeArguments(field, ctx)
	validateUnsupportedFieldType(field, ctx)
	validateAutoParam(field, ctx)
}

// validateModelIndexesWithNames validates indexes with name tracking.
func validateModelIndexesWithNames(ctx *ValidationContext, names *Names) {
	models := ctx.Db.WalkModels()
	for _, model := range models {
		if model.IsIgnored() {
			continue
		}

		astModel := model.AstModel()
		isView := astModel != nil && astModel.IsView()

		indexes := model.Indexes()
		for _, index := range indexes {
			if isView {
				// View-specific index validations
				validateViewIndex(index, model, ctx)
				validateUniqueIndexClientNameDoesNotClashWithField(index, model, ctx)
				// Validate index field attributes for views
				for _, fieldAttr := range index.ScalarFieldAttributes() {
					validateViewIndexFieldAttribute(index, fieldAttr, ctx)
				}
			} else {
				// Model-specific index validations
				validateUniqueIndexName(index, model, names, ctx)
				validateIndexFields(index, model, ctx)
				validateIndexAlgorithm(index, model, ctx)
				validateFulltextIndexSupport(index, model, ctx)
				validateIndexDatabaseName(index, model, ctx)
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
			}
		}
	}
}

// validateRelationsWithNames validates relations with name tracking.
func validateRelationsWithNames(ctx *ValidationContext, names *Names) {
	relations := ctx.Db.WalkRelations()
	for _, relation := range relations {
		validateRelationWithNames(relation, ctx, names)
	}
}

// validateRelationWithNames validates a single relation with name tracking.
func validateRelationWithNames(relation *database.RelationWalker, ctx *ValidationContext, names *Names) {
	// Basic validations
	if relation.IsIgnored() {
		return
	}

	// Validate models exist
	models := relation.Models()
	if len(models) != 2 {
		return
	}

	modelA := ctx.Db.WalkModel(models[0])
	modelB := ctx.Db.WalkModel(models[1])

	if modelA == nil || modelB == nil {
		ctx.PushError(diagnostics.NewValidationError(
			"Relation references non-existent models.",
			diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		))
		return
	}

	// Relation-specific validations
	validateRelationName(relation, names, ctx)
	validateRelationReferencesUniqueFields(relation, ctx)
	validateRelationSameLength(relation, ctx)
	validateRelationArity(relation, ctx)
	validateRelationRequiredCannotUseSetNull(relation, ctx)
	validateReferencingScalarFieldTypes(relation, ctx)
	validateRequiredRelationCannotUseSetNullDetailed(relation, ctx)
	validateRelationAmbiguity(relation, names, ctx)
	validateSelfRelationFields(relation, ctx)

	// Advanced relation validations
	refined := relation.Refine()
	if refined != nil {
		inline := refined.AsInline()
		if inline != nil {
			validateRelationReferencingScalarFieldTypes(inline, ctx)
			validateRelationHasUniqueConstraintName(inline, names, ctx)
			validateRelationFieldArityAdvanced(inline, ctx)
			validateRelationSameLengthAdvanced(inline, ctx)

			complete := inline.AsComplete()
			if complete != nil {
				validateRelationCycles(complete, ctx)
				validateRelationMultipleCascadingPaths(complete, ctx)
			}
		}
	}

	// One-to-one and one-to-many specific validations
	refined2 := relation.Refine()
	if refined2 != nil {
		inline := refined2.AsInline()
		if inline != nil {
			if inline.IsOneToOne() {
				validateOneToOneBothSidesAreDefined(inline, ctx)
				validateOneToOneFieldsAndReferencesAreDefined(inline, ctx)
				validateOneToOneFieldsAndReferencesOnOneSideOnly(inline, ctx)
				validateOneToOneReferentialActions(inline, ctx)
				validateOneToOneFieldsReferencesMixups(inline, ctx)
				validateOneToOneFieldsAndReferencesOnWrongSide(inline, ctx)
				validateOneToOneBackRelationArityIsOptional(inline, ctx)
				validateOneToOneFieldsMustBeUnique(inline, ctx)
			} else if inline.IsOneToMany() {
				validateOneToManyBothSidesAreDefined(inline, ctx)
				validateOneToManyFieldsAndReferencesAreDefined(inline, ctx)
				validateOneToManyReferentialActions(inline, ctx)
			}
		}

		// Many-to-many validations
		implicit := refined2.AsImplicitManyToMany()
		if implicit != nil {
			validateImplicitManyToManySingularId(implicit, ctx)
			validateImplicitManyToManySupports(implicit, ctx)
			validateImplicitManyToManyCannotDefineReferences(implicit, ctx)
		}

		embedded := refined2.AsTwoWayEmbeddedManyToMany()
		if embedded != nil {
			validateEmbeddedManyToManySupports(embedded, ctx)
			validateEmbeddedManyToManyDefinesReferences(embedded, ctx)
			validateEmbeddedManyToManyDefinesFields(embedded, ctx)
			validateEmbeddedManyToManyReferencesId(embedded, ctx)
		}
	}
}

// validateCompositeTypes validates all composite types.
func validateCompositeTypes(ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityCompositeTypes) {
		return
	}

	compositeTypes := ctx.Db.WalkCompositeTypes()
	for _, ct := range compositeTypes {
		validateCompositeType(ct, ctx)
	}
}

// validateCompositeType validates a single composite type.
func validateCompositeType(ct *database.CompositeTypeWalker, ctx *ValidationContext) {
	validateCompositeTypeMoreThanOneField(ct, ctx)

	fields := ct.Fields()
	for _, field := range fields {
		validateCompositeTypeFieldDefaultValue(field, ctx)
		validateCompositeTypeFieldTypes(field, ctx)
		validateCompositeTypeFieldArity(field, ctx)
	}
}

// validateRelationField validates a relation field.
func validateRelationField(field *database.RelationFieldWalker, ctx *ValidationContext) {
	// Basic validations
	if field.IsIgnored() {
		return
	}

	// Relation field-specific validations
	validateRelationFieldReferencedModel(field, ctx)
	validateRelationFieldReferentialActions(field, ctx)
}

// validateEnums performs enum-specific validations.
func validateEnums(ctx *ValidationContext) {
	if !ctx.HasCapability(ConnectorCapabilityEnums) {
		return
	}

	enums := ctx.Db.WalkEnums()
	for _, enum := range enums {
		validateEnum(enum, ctx)
	}
}

// validateEnum validates a single enum.
func validateEnum(enum *database.EnumWalker, ctx *ValidationContext) {
	validateEnumHasValues(enum, ctx)
	validateEnumConnectorSupport(enum, ctx)
	validateEnumSchemaAttribute(enum, ctx)
	validateEnumDatabaseName(enum, ctx)
	validateEnumValueNames(enum, ctx)
	validateEnumValueDatabaseNames(enum, ctx)
	validateEnumValueReservedNames(enum, ctx)
}
