// Package pslcore provides MongoDB connector implementation.
package validation

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// MongoDbConnector implements the ExtendedConnector interface for MongoDB.
type MongoDbConnector struct {
	*BaseExtendedConnector
}

// NewMongoDbConnector creates a new MongoDB connector.
func NewMongoDbConnector() *MongoDbConnector {
	return &MongoDbConnector{
		BaseExtendedConnector: &BaseExtendedConnector{
			name:                "MongoDB",
			providerName:        "mongodb",
			flavour:             FlavourMongo,
			capabilities:        ConnectorCapabilities(ConnectorCapabilityJson | ConnectorCapabilityImplicitManyToManyRelation),
			maxIdentifierLength: 127,
		},
	}
}

// Name returns the connector name.
func (c *MongoDbConnector) Name() string {
	return "MongoDB"
}

// HasCapability checks if the connector has a specific capability.
func (c *MongoDbConnector) HasCapability(capability ConnectorCapability) bool {
	return c.Capabilities()&(1<<uint(capability)) != 0
}

// ReferentialActions returns the supported referential actions.
// MongoDB doesn't support foreign keys.
func (c *MongoDbConnector) ReferentialActions(relationMode RelationMode) []ReferentialAction {
	return []ReferentialAction{}
}

// ForeignKeyReferentialActions returns the supported foreign key referential actions.
func (c *MongoDbConnector) ForeignKeyReferentialActions() []ReferentialAction {
	return []ReferentialAction{}
}

// EmulatedReferentialActions returns the supported emulated referential actions.
func (c *MongoDbConnector) EmulatedReferentialActions() []ReferentialAction {
	return []ReferentialAction{}
}

// SupportsReferentialAction checks if the connector supports the given referential action.
func (c *MongoDbConnector) SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool {
	return false
}

// ValidateURL validates a MongoDB connection URL.
func (c *MongoDbConnector) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "mongodb://") && !strings.HasPrefix(url, "mongodb+srv://") && !strings.HasPrefix(url, "env:") {
		return fmt.Errorf("MongoDB URL must start with `mongodb://` or `mongodb+srv://`")
	}
	return nil
}

// AvailableNativeTypeConstructors returns available native type constructors.
func (c *MongoDbConnector) AvailableNativeTypeConstructors() []*NativeTypeConstructor {
	return GetMongoDbNativeTypeConstructors()
}

// ConstraintViolationScopes returns the constraint violation scopes.
func (c *MongoDbConnector) ConstraintViolationScopes() []ConstraintScope {
	return []ConstraintScope{
		ConstraintScopeModelKeyIndex,
	}
}

// ConstraintViolationScopesExtended returns the extended constraint violation scopes.
func (c *MongoDbConnector) ConstraintViolationScopesExtended() []ExtendedConstraintScope {
	return []ExtendedConstraintScope{
		ExtendedConstraintScopeModelKeyIndex,
	}
}

// SupportsIndexType checks if the connector supports the given index algorithm.
func (c *MongoDbConnector) SupportsIndexType(algo IndexAlgorithm) bool {
	return database.IndexAlgorithm(algo) == database.IndexAlgorithmBTree
}

// ParseNativeType parses a native type from name and arguments.
func (c *MongoDbConnector) ParseNativeType(name string, args []string, span Span, diags *Diagnostics) *NativeTypeInstance {
	mongoType := ParseMongoDbNativeType(name, args, span, diags)
	if mongoType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: mongoType}
}

// ScalarTypeForNativeType returns the default scalar type for a native type.
func (c *MongoDbConnector) ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType {
	if nativeType == nil || nativeType.Data == nil {
		return nil
	}
	mongoType, ok := nativeType.Data.(*MongoDbNativeTypeInstance)
	if !ok {
		return nil
	}
	scalarType := MongoDbNativeTypeToScalarType(mongoType)
	if scalarType == nil {
		return nil
	}
	return &database.ScalarFieldType{BuiltInScalar: scalarType}
}

// DefaultNativeTypeForScalarType returns the default native type for a scalar type.
func (c *MongoDbConnector) DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance {
	if scalarType == nil || scalarType.BuiltInScalar == nil {
		return nil
	}
	mongoType := MongoDbScalarTypeToDefaultNativeType(*scalarType.BuiltInScalar)
	if mongoType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: mongoType}
}

// NativeTypeToString converts a native type instance to string representation.
func (c *MongoDbConnector) NativeTypeToString(instance *NativeTypeInstance) string {
	if instance == nil || instance.Data == nil {
		return ""
	}
	mongoType, ok := instance.Data.(*MongoDbNativeTypeInstance)
	if !ok {
		return ""
	}
	return MongoDbNativeTypeToString(mongoType)
}

// NativeTypeToParts returns the debug representation of a native type.
func (c *MongoDbConnector) NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string) {
	if nativeType == nil || nativeType.Data == nil {
		return "", []string{}
	}
	mongoType, ok := nativeType.Data.(*MongoDbNativeTypeInstance)
	if !ok {
		return "", []string{}
	}
	return mongoType.Type.String(), []string{}
}

// FindNativeTypeConstructor finds a native type constructor by name.
func (c *MongoDbConnector) FindNativeTypeConstructor(name string) *NativeTypeConstructor {
	constructors := c.AvailableNativeTypeConstructors()
	for _, constructor := range constructors {
		if constructor != nil && constructor.Name == name {
			return constructor
		}
	}
	return nil
}

// NativeInstanceError creates a native type error factory.
func (c *MongoDbConnector) NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory {
	nativeTypeStr := c.NativeTypeToString(instance)
	factory := diagnostics.NewNativeTypeErrorFactory(nativeTypeStr, c.Name())
	return &factory
}

// ValidateNativeTypeArguments validates native type attribute arguments.
func (c *MongoDbConnector) ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics) {
	if nativeType == nil || nativeType.Data == nil {
		return
	}
	mongoType, ok := nativeType.Data.(*MongoDbNativeTypeInstance)
	if !ok {
		return
	}

	// Validate that the native type is compatible with the scalar type
	expectedScalarType := MongoDbNativeTypeToScalarType(mongoType)
	if expectedScalarType == nil {
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("MongoDB", mongoType.Type.String(), span))
		return
	}
	// Additional validation can be added here for specific type combinations
}

// ValidateEnum performs enum-specific validation.
func (c *MongoDbConnector) ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics) {
	// MongoDB enums are validated by base validations
}

// ValidateModel performs model-specific validation.
func (c *MongoDbConnector) ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics) {
	// MongoDB requires exactly one @id field
	if modelWalker.PrimaryKey() == nil {
		astModel := modelWalker.AstModel()
		if astModel != nil {
			modelPos := astModel.TopPos()
			modelSpan := diagnostics.NewSpan(modelPos.Offset, modelPos.Offset+len(astModel.GetName()), diagnostics.FileIDZero)
			diags.PushError(diagnostics.NewInvalidModelError(
				"MongoDB models require exactly one identity field annotated with @id",
				modelSpan,
			))
		}
		return
	}

	// Validate primary key has correct mapped name
	if pk := modelWalker.PrimaryKey(); pk != nil {
		validateMongoDBPrimaryKeyMappedName(pk, diags)
	}

	// Validate scalar fields
	for _, field := range modelWalker.ScalarFields() {
		validateMongoDBScalarField(field, diags)
	}

	// Validate indexes
	for _, index := range modelWalker.Indexes() {
		validateMongoDBIndex(index, diags)
	}
}

// ValidateView performs view-specific validation.
func (c *MongoDbConnector) ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics) {
	// MongoDB doesn't have views
}

// ValidateRelationField performs relation field validation.
func (c *MongoDbConnector) ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics) {
	validateMongoDBRelationFieldNativeTypes(fieldWalker, diags)
}

// ValidateDatasource performs datasource-specific validation.
func (c *MongoDbConnector) ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics) {
	if datasource.URL != nil {
		url := *datasource.URL
		if url != "" && !strings.HasPrefix(url, "mongodb://") && !strings.HasPrefix(url, "mongodb+srv://") && !strings.HasPrefix(url, "env:") {
			diags.PushWarning(diagnostics.NewDatamodelWarning(
				"MongoDB datasource URL should start with `mongodb://` or `mongodb+srv://`.",
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}
	}
}

// ValidateScalarFieldUnknownDefaultFunctions validates scalar field default functions.
func (c *MongoDbConnector) ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics) {
	for _, model := range db.WalkModels() {
		for _, field := range model.ScalarFields() {
			if defaultValue := field.DefaultValue(); defaultValue != nil {
				if funcCall, ok := defaultValue.Value().AsFunction(); ok {
					switch funcCall.Name {
					case "now", "uuid", "cuid", "cuid2", "autoincrement", "dbgenerated", "env", "objectid", "auto":
						// Known functions for MongoDB
					default:
						funcPos := funcCall.Pos
						funcSpan := diagnostics.NewSpan(funcPos.Offset, funcPos.Offset+len(funcCall.Name), diagnostics.FileIDZero)
						diags.PushError(diagnostics.NewDefaultUnknownFunctionError(funcCall.Name, funcSpan))
					}
				}
			}
		}
	}
}

// validateMongoDBScalarField validates MongoDB-specific scalar field rules.
func validateMongoDBScalarField(field *database.ScalarFieldWalker, diags *diagnostics.Diagnostics) {
	// Check @default(auto()) requires @db.ObjectId
	if defaultValue := field.DefaultValue(); defaultValue != nil {
		if value := defaultValue.Value(); value != nil {
			if funcCall, ok := value.AsFunction(); ok {
				if funcCall.Name == "auto" {
					// Check if native type is ObjectId
					nativeTypeInfo := field.NativeType()
					if nativeTypeInfo == nil {
						container := "model"
						if model := field.Model(); model != nil {
							if astModel := model.AstModel(); astModel != nil && astModel.IsView() {
								container = "view"
							}
						}
						astField := field.AstField()
						span := diagnostics.EmptySpan()
						if astField != nil {
							fieldPos := astField.Pos
							span = diagnostics.NewSpan(fieldPos.Offset, fieldPos.Offset+len(astField.GetName()), diagnostics.FileIDZero)
						}
						diags.PushError(diagnostics.NewFieldValidationError(
							"MongoDB `@default(auto())` fields must have `ObjectId` native type.",
							container,
							field.Model().Name(),
							field.Name(),
							span,
						))
					}

					// Check @default(auto()) requires @id
					if !field.IsSinglePK() && !field.IsPartOfCompoundPK() {
						container := "model"
						if model := field.Model(); model != nil {
							if astModel := model.AstModel(); astModel != nil && astModel.IsView() {
								container = "view"
							}
						}
						astField := field.AstField()
						span := diagnostics.EmptySpan()
						if astField != nil {
							fieldPos := astField.Pos
							span = diagnostics.NewSpan(fieldPos.Offset, fieldPos.Offset+len(astField.GetName()), diagnostics.FileIDZero)
						}
						diags.PushError(diagnostics.NewFieldValidationError(
							"MongoDB `@default(auto())` fields must have the `@id` attribute.",
							container,
							field.Model().Name(),
							field.Name(),
							span,
						))
					}
				}

				// Check @default(dbgenerated()) is not allowed
				if funcCall.Name == "dbgenerated" {
					container := "model"
					if model := field.Model(); model != nil {
						if astModel := model.AstModel(); astModel != nil && astModel.IsView() {
							container = "view"
						}
					}
					astField := field.AstField()
					span := diagnostics.EmptySpan()
					if astField != nil {
						fieldPos := astField.Pos
						span = diagnostics.NewSpan(fieldPos.Offset, fieldPos.Offset+len(astField.GetName()), diagnostics.FileIDZero)
					}
					diags.PushError(diagnostics.NewFieldValidationError(
						"The `dbgenerated()` function is not allowed with MongoDB. Please use `auto()` instead.",
						container,
						field.Model().Name(),
						field.Name(),
						span,
					))
				}
			}
		}
	}

	// Check field name characters (for @map)
	if mappedName := field.DatabaseName(); mappedName != field.Name() {
		astField := field.AstField()
		if astField != nil {
			span := diagnostics.EmptySpan()
			// Try to find @map attribute span
			for _, attr := range astField.Attributes {
				if attr.Name.Name == "map" {
					span = diagnostics.NewSpan(attr.Pos.Offset, attr.Pos.Offset+len(attr.String()), diagnostics.FileIDZero)
					break
				}
			}
			if mappedName != "" {
				if strings.HasPrefix(mappedName, "$") {
					diags.PushError(diagnostics.NewAttributeValidationError(
						"The field name cannot start with a `$` character",
						"@map",
						span,
					))
				}
				if strings.Contains(mappedName, ".") {
					diags.PushError(diagnostics.NewAttributeValidationError(
						"The field name cannot contain a `.` character",
						"@map",
						span,
					))
				}
			}
		}
	}
}

// validateMongoDBPrimaryKeyMappedName validates that MongoDB primary key has @map("_id").
func validateMongoDBPrimaryKeyMappedName(pk *database.PrimaryKeyWalker, diags *diagnostics.Diagnostics) {
	fields := pk.Fields()
	if len(fields) != 1 {
		return // Only validate single-field primary keys
	}

	field := fields[0]
	scalarField := field.ScalarField()
	if scalarField == nil {
		return
	}

	fieldName := scalarField.Name()
	mappedName := scalarField.DatabaseName()

	// Check if field name or mapped name is "_id"
	if fieldName == "_id" || mappedName == "_id" {
		return
	}

	container := "model"
	if model := scalarField.Model(); model != nil {
		if astModel := model.AstModel(); astModel != nil && astModel.IsView() {
			container = "view"
		}
	}

	astField := scalarField.AstField()
	span := diagnostics.EmptySpan()
	if astField != nil {
		fieldPos := astField.Pos
		span = diagnostics.NewSpan(fieldPos.Offset, fieldPos.Offset+len(astField.GetName()), diagnostics.FileIDZero)
	}

	var err diagnostics.DatamodelError
	if mappedName != fieldName {
		err = diagnostics.NewFieldValidationError(
			fmt.Sprintf("MongoDB model IDs must have an @map(\"_id\") annotation, found @map(\"%s\").", mappedName),
			container,
			scalarField.Model().Name(),
			fieldName,
			span,
		)
	} else {
		err = diagnostics.NewFieldValidationError(
			"MongoDB model IDs must have an @map(\"_id\") annotation.",
			container,
			scalarField.Model().Name(),
			fieldName,
			span,
		)
	}
	diags.PushError(err)
}

// validateMongoDBIndex validates MongoDB-specific index rules.
func validateMongoDBIndex(index *database.IndexWalker, diags *diagnostics.Diagnostics) {
	// Check for duplicate indexes with same fields
	model := index.Model()
	if model == nil {
		return
	}

	indexFields := index.Fields()
	for _, otherIndex := range model.Indexes() {
		if otherIndex == index {
			continue
		}
		otherFields := otherIndex.Fields()
		if len(indexFields) != len(otherFields) {
			continue
		}
		// Simple check - in full implementation would compare field IDs
		// For now, skip this complex validation
	}

	// Check unique cannot be on id field
	if index.IsUnique() && len(indexFields) == 1 {
		field := indexFields[0]
		scalarField := field.ScalarField()
		if scalarField != nil && scalarField.IsSinglePK() {
			span := diagnostics.EmptySpan()
			if astAttr := index.AstAttribute(); astAttr != nil {
				span = diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), diagnostics.FileIDZero)
			}
			diags.PushError(diagnostics.NewAttributeValidationError(
				"The same field cannot be an id and unique on MongoDB.",
				index.AttributeName(),
				span,
			))
		}
	}
}

// validateMongoDBRelationFieldNativeTypes validates that relation fields have matching native types.
func validateMongoDBRelationFieldNativeTypes(field *database.RelationFieldWalker, diags *diagnostics.Diagnostics) {
	referencingFields := field.ReferencingFields()
	referencedFields := field.ReferencedFields()

	if len(referencingFields) == 0 || len(referencedFields) == 0 {
		return
	}

	if len(referencingFields) != len(referencedFields) {
		return
	}

	for i := 0; i < len(referencingFields); i++ {
		refField := referencingFields[i]
		refdField := referencedFields[i]

		refNativeType := refField.NativeType()
		refdNativeType := refdField.NativeType()

		// Get native type names for comparison
		refTypeName := ""
		refdTypeName := ""

		if refNativeType != nil {
			astField := refField.AstField()
			if astField != nil {
				for _, attr := range astField.Attributes {
					if attr.Name.Name == "db" && len(attr.Arguments.Arguments) > 0 {
						if firstArg := attr.Arguments.Arguments[0]; firstArg.Value != nil {
							if strLit, ok := firstArg.Value.AsStringValue(); ok {
								refTypeName = strLit.Value
							} else if constVal, ok := firstArg.Value.AsConstantValue(); ok {
								refTypeName = constVal.Value
							}
						}
						break
					}
				}
			}
		}

		if refdNativeType != nil {
			astField := refdField.AstField()
			if astField != nil {
				for _, attr := range astField.Attributes {
					if attr.Name.Name == "db" && len(attr.Arguments.Arguments) > 0 {
						if firstArg := attr.Arguments.Arguments[0]; firstArg.Value != nil {
							if strLit, ok := firstArg.Value.AsStringValue(); ok {
								refdTypeName = strLit.Value
							} else if constVal, ok := firstArg.Value.AsConstantValue(); ok {
								refdTypeName = constVal.Value
							}
						}
						break
					}
				}
			}
		}

		// Compare native types
		if refTypeName != refdTypeName {
			var msg string
			if refTypeName != "" && refdTypeName != "" {
				msg = fmt.Sprintf(
					"Field %s.%s and %s.%s must have the same native type for MongoDB to join those collections correctly. Consider updating those fields to either use '@db.%s' or '@db.%s'.",
					refField.Model().Name(),
					refField.Name(),
					refdField.Model().Name(),
					refdField.Name(),
					refTypeName,
					refdTypeName,
				)
			} else if refTypeName == "" && refdTypeName != "" {
				msg = fmt.Sprintf(
					"Field %s.%s and %s.%s must have the same native type for MongoDB to join those collections correctly. Consider either removing %s.%s's native type attribute or adding '@db.%s' to %s.%s.",
					refField.Model().Name(),
					refField.Name(),
					refdField.Model().Name(),
					refdField.Name(),
					refdField.Model().Name(),
					refdField.Name(),
					refdTypeName,
					refField.Model().Name(),
					refField.Name(),
				)
			} else if refTypeName != "" && refdTypeName == "" {
				msg = fmt.Sprintf(
					"Field %s.%s and %s.%s must have the same native type for MongoDB to join those collections correctly. Consider either removing %s.%s's native type attribute or adding '@db.%s' to %s.%s.",
					refField.Model().Name(),
					refField.Name(),
					refdField.Model().Name(),
					refdField.Name(),
					refField.Model().Name(),
					refField.Name(),
					refTypeName,
					refdField.Model().Name(),
					refdField.Name(),
				)
			} else {
				continue
			}

			msg += " Beware that this will become an error in the future."

			astField := refField.AstField()
			span := diagnostics.EmptySpan()
			if astField != nil {
				span = diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), diagnostics.FileIDZero)
			}

			diags.PushWarning(diagnostics.NewFieldValidationWarning(
				msg,
				field.Model().Name(),
				field.Name(),
				span,
			))
		}
	}
}
