// Package pslcore provides MySQL connector implementation.
package validation

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// MySqlConnector implements the ExtendedConnector interface for MySQL.
type MySqlConnector struct {
	*BaseExtendedConnector
}

// NewMySqlConnector creates a new MySQL connector.
func NewMySqlConnector() *MySqlConnector {
	return &MySqlConnector{
		BaseExtendedConnector: &BaseExtendedConnector{
			name:                "MySQL",
			providerName:        "mysql",
			flavour:             FlavourMySQL,
			capabilities:        ConnectorCapabilities(ConnectorCapabilityEnums | ConnectorCapabilityJson | ConnectorCapabilityFullTextIndex | ConnectorCapabilityAutoIncrement | ConnectorCapabilityAutoIncrementAllowedOnNonId | ConnectorCapabilityIndexColumnLengthPrefixing | ConnectorCapabilityImplicitManyToManyRelation),
			maxIdentifierLength: 64,
		},
	}
}

// Name returns the connector name.
func (c *MySqlConnector) Name() string {
	return "MySQL"
}

// HasCapability checks if the connector has a specific capability.
func (c *MySqlConnector) HasCapability(capability ConnectorCapability) bool {
	return c.Capabilities()&(1<<uint(capability)) != 0
}

// ReferentialActions returns the supported referential actions for the given relation mode.
func (c *MySqlConnector) ReferentialActions(relationMode RelationMode) []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// ForeignKeyReferentialActions returns the supported foreign key referential actions.
func (c *MySqlConnector) ForeignKeyReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// EmulatedReferentialActions returns the supported emulated referential actions.
func (c *MySqlConnector) EmulatedReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// SupportsReferentialAction checks if the connector supports the given referential action.
func (c *MySqlConnector) SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool {
	actions := c.ReferentialActions(relationMode)
	for _, supportedAction := range actions {
		if supportedAction == action {
			return true
		}
	}
	return false
}

// ValidateURL validates a MySQL connection URL.
func (c *MySqlConnector) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "mysql://") && !strings.HasPrefix(url, "env:") {
		return fmt.Errorf("MySQL URL must start with `mysql://`")
	}
	return nil
}

// AvailableNativeTypeConstructors returns available native type constructors.
func (c *MySqlConnector) AvailableNativeTypeConstructors() []*NativeTypeConstructor {
	return GetMySqlNativeTypeConstructors()
}

// ConstraintViolationScopes returns the constraint violation scopes.
func (c *MySqlConnector) ConstraintViolationScopes() []ConstraintScope {
	return []ConstraintScope{
		ConstraintScopeGlobalForeignKey,
		ConstraintScopeModelKeyIndex,
	}
}

// ConstraintViolationScopesExtended returns the extended constraint violation scopes.
func (c *MySqlConnector) ConstraintViolationScopesExtended() []ExtendedConstraintScope {
	return []ExtendedConstraintScope{
		ExtendedConstraintScopeGlobalForeignKey,
		ExtendedConstraintScopeModelKeyIndex,
	}
}

// SupportsIndexType checks if the connector supports the given index algorithm.
func (c *MySqlConnector) SupportsIndexType(algo IndexAlgorithm) bool {
	// MySQL supports BTree (default) and FullText indexes
	supported := []database.IndexAlgorithm{
		database.IndexAlgorithmBTree,
	}
	algoPD := database.IndexAlgorithm(algo)
	for _, supportedAlgo := range supported {
		if algoPD == supportedAlgo {
			return true
		}
	}
	return false
}

// ParseNativeType parses a native type from name and arguments.
func (c *MySqlConnector) ParseNativeType(name string, args []string, span Span, diagnostics *Diagnostics) *NativeTypeInstance {
	mysqlType := ParseMySqlNativeType(name, args, span, diagnostics)
	if mysqlType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: mysqlType}
}

// ScalarTypeForNativeType returns the default scalar type for a native type.
func (c *MySqlConnector) ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType {
	if nativeType == nil || nativeType.Data == nil {
		return nil
	}
	mysqlType, ok := nativeType.Data.(*MySqlNativeTypeInstance)
	if !ok {
		return nil
	}
	scalarType := MySqlNativeTypeToScalarType(mysqlType)
	if scalarType == nil {
		return nil
	}
	return &database.ScalarFieldType{BuiltInScalar: scalarType}
}

// DefaultNativeTypeForScalarType returns the default native type for a scalar type.
func (c *MySqlConnector) DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance {
	if scalarType == nil || scalarType.BuiltInScalar == nil {
		return nil
	}
	mysqlType := MySqlScalarTypeToDefaultNativeType(*scalarType.BuiltInScalar)
	if mysqlType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: mysqlType}
}

// NativeTypeToString converts a native type instance to string representation.
func (c *MySqlConnector) NativeTypeToString(instance *NativeTypeInstance) string {
	if instance == nil || instance.Data == nil {
		return ""
	}
	mysqlType, ok := instance.Data.(*MySqlNativeTypeInstance)
	if !ok {
		return ""
	}
	return MySqlNativeTypeToString(mysqlType)
}

// NativeTypeToParts returns the debug representation of a native type.
func (c *MySqlConnector) NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string) {
	if nativeType == nil || nativeType.Data == nil {
		return "", []string{}
	}
	mysqlType, ok := nativeType.Data.(*MySqlNativeTypeInstance)
	if !ok {
		return "", []string{}
	}

	name := mysqlType.Type.String()
	var parts []string
	if mysqlType.Precision != nil && mysqlType.Scale != nil {
		parts = append(parts, fmt.Sprintf("%d", *mysqlType.Precision), fmt.Sprintf("%d", *mysqlType.Scale))
	} else if mysqlType.Precision != nil {
		parts = append(parts, fmt.Sprintf("%d", *mysqlType.Precision))
	} else if mysqlType.Length != nil {
		parts = append(parts, fmt.Sprintf("%d", *mysqlType.Length))
	}
	return name, parts
}

// FindNativeTypeConstructor finds a native type constructor by name.
func (c *MySqlConnector) FindNativeTypeConstructor(name string) *NativeTypeConstructor {
	constructors := c.AvailableNativeTypeConstructors()
	for _, constructor := range constructors {
		if constructor != nil && constructor.Name == name {
			return constructor
		}
	}
	return nil
}

// NativeInstanceError creates a native type error factory.
func (c *MySqlConnector) NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory {
	nativeTypeStr := c.NativeTypeToString(instance)
	factory := diagnostics.NewNativeTypeErrorFactory(nativeTypeStr, c.Name())
	return &factory
}

// ValidateNativeTypeArguments validates native type attribute arguments.
func (c *MySqlConnector) ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics) {
	if nativeType == nil || nativeType.Data == nil {
		return
	}
	mysqlType, ok := nativeType.Data.(*MySqlNativeTypeInstance)
	if !ok {
		return
	}

	// Validate that the native type is compatible with the scalar type
	expectedScalarType := MySqlNativeTypeToScalarType(mysqlType)
	if expectedScalarType == nil {
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("MySQL", mysqlType.Type.String(), span))
		return
	}
}

// ValidateEnum performs enum-specific validation.
func (c *MySqlConnector) ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics) {
	// MySQL enums are validated by base validations
}

// ValidateModel performs model-specific validation.
func (c *MySqlConnector) ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics) {
	// Validate indexes for MySQL-specific constraints
	for _, index := range modelWalker.Indexes() {
		validateMySQLIndexFieldTypes(c, index, diags)
	}

	// Validate primary key for MySQL-specific constraints
	if pk := modelWalker.PrimaryKey(); pk != nil {
		validateMySQLPrimaryKeyFieldTypes(c, pk, diags)
	}

	// Validate referential actions if using foreign keys
	if relationMode == RelationMode("foreignKeys") {
		for _, field := range modelWalker.RelationFields() {
			validateMySQLReferentialActionSetDefault(c, field, diags)
		}
	}
}

// ValidateView performs view-specific validation.
func (c *MySqlConnector) ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics) {
	// MySQL views are validated by base validations
}

// ValidateRelationField performs relation field validation.
func (c *MySqlConnector) ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics) {
	// MySQL relation fields are validated by base validations
}

// ValidateDatasource performs datasource-specific validation.
func (c *MySqlConnector) ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics) {
	if datasource.URL != nil {
		url := *datasource.URL
		if url != "" && !strings.HasPrefix(url, "mysql://") && !strings.HasPrefix(url, "env:") {
			diags.PushWarning(diagnostics.NewDatamodelWarning(
				"MySQL datasource URL should start with `mysql://`.",
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}
	}
}

// ValidateScalarFieldUnknownDefaultFunctions validates scalar field default functions.
func (c *MySqlConnector) ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics) {
	// Use base implementation
	for _, model := range db.WalkModels() {
		for _, field := range model.ScalarFields() {
			if defaultValue := field.DefaultValue(); defaultValue != nil {
				if funcCall, ok := defaultValue.Value().AsFunction(); ok {
					switch funcCall.Name {
					case "now", "uuid", "cuid", "cuid2", "autoincrement", "dbgenerated", "env":
						// Known functions, do nothing
					default:
						pos := funcCall.Span()
						span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(funcCall.Name), diagnostics.FileIDZero)
						diags.PushError(diagnostics.NewDefaultUnknownFunctionError(funcCall.Name, span))
					}
				}
			}
		}
	}
}

// MySQL native types that cannot be used in key specifications without length prefix
var mysqlNativeTypesThatCannotBeUsedInKeySpecification = []string{
	"Text",
	"LongText",
	"MediumText",
	"TinyText",
	"Blob",
	"TinyBlob",
	"MediumBlob",
	"LongBlob",
}

const mysqlLengthGuide = " Please use the `length` argument to the field in the index definition to allow this."

// validateMySQLIndexFieldTypes validates that MySQL index fields use compatible native types.
func validateMySQLIndexFieldTypes(connector *MySqlConnector, index *database.IndexWalker, diags *diagnostics.Diagnostics) {
	for _, field := range index.ScalarFieldAttributes() {
		// Get native type instance from the scalar field
		scalarField := field.ScalarField()
		if scalarField == nil {
			continue
		}

		nativeTypeInfo := scalarField.NativeType()
		if nativeTypeInfo == nil {
			continue
		}

		// Parse the native type using the connector
		// We need to get the type name from the interner - access through the index's model
		model := index.Model()
		if model == nil {
			continue
		}
		// Access interner through the model's db (we'll need to add a helper or use reflection)
		// For now, let's get it from the scalar field's AST field directly
		astField := scalarField.AstField()
		if astField == nil {
			continue
		}

		// Find the @db attribute to get the type name
		var typeName string
		var typeArgs []string
		for _, attr := range astField.Attributes {
			if attr.Name.Name == "db" && len(attr.Arguments.Arguments) > 0 {
				if firstArg := attr.Arguments.Arguments[0]; firstArg.Value != nil {
					if strLit, ok := firstArg.Value.AsStringValue(); ok {
						typeName = strLit.Value
					} else if constVal, ok := firstArg.Value.AsConstantValue(); ok {
						typeName = constVal.Value
					}
					// Get additional arguments if any
					for i := 1; i < len(attr.Arguments.Arguments); i++ {
						if arg := attr.Arguments.Arguments[i]; arg.Value != nil {
							if strLit, ok := arg.Value.AsStringValue(); ok {
								typeArgs = append(typeArgs, strLit.Value)
							} else if numVal, ok := arg.Value.AsNumericValue(); ok {
								typeArgs = append(typeArgs, numVal.String())
							}
						}
					}
				}
				break
			}
		}

		if typeName == "" {
			continue
		}

		// Create a temporary diagnostics for parsing (errors will be handled separately)
		tempDiags := diagnostics.NewDiagnostics()
		nativeTypeInstance := connector.ParseNativeType(typeName, nativeTypeInfo.Arguments, nativeTypeInfo.Span, &tempDiags)
		if nativeTypeInstance == nil {
			continue
		}

		// Get native type name
		nativeTypeName, _ := connector.NativeTypeToParts(nativeTypeInstance)

		// Check if this native type is in the restricted list
		isRestricted := false
		for _, restrictedType := range mysqlNativeTypesThatCannotBeUsedInKeySpecification {
			if nativeTypeName == restrictedType {
				isRestricted = true
				break
			}
		}

		if !isRestricted {
			continue
		}

		// Length defined, so we allow the index
		if field.Length() != nil {
			continue
		}

		// Fulltext indexes are allowed
		if index.IsFulltext() {
			continue
		}

		// Create error
		errorFactory := connector.NativeInstanceError(nativeTypeInstance)
		var err diagnostics.DatamodelError
		span := diagnostics.EmptySpan()
		if astAttr := index.AstAttribute(); astAttr != nil {
			span = diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), diagnostics.FileIDZero)
		}
		if index.IsUnique() {
			err = errorFactory.NewIncompatibleNativeTypeWithUniqueError(mysqlLengthGuide, span)
		} else {
			err = errorFactory.NewIncompatibleNativeTypeWithIndexError(mysqlLengthGuide, span)
		}
		diags.PushError(err)
		break // Only report one error per index
	}
}

// validateMySQLPrimaryKeyFieldTypes validates that MySQL primary key fields use compatible native types.
func validateMySQLPrimaryKeyFieldTypes(connector *MySqlConnector, pk *database.PrimaryKeyWalker, diags *diagnostics.Diagnostics) {
	for _, field := range pk.ScalarFieldAttributes() {
		// Get native type instance from the scalar field
		scalarField := field.ScalarField()
		if scalarField == nil {
			continue
		}

		nativeTypeInfo := scalarField.NativeType()
		if nativeTypeInfo == nil {
			continue
		}

		// Parse the native type using the connector
		// Get type name from AST field
		astField := scalarField.AstField()
		if astField == nil {
			continue
		}

		// Find the @db attribute to get the type name
		var typeName string
		var typeArgs []string
		for _, attr := range astField.Attributes {
			if attr.Name.Name == "db" && len(attr.Arguments.Arguments) > 0 {
				if firstArg := attr.Arguments.Arguments[0]; firstArg.Value != nil {
					if strLit, ok := firstArg.Value.AsStringValue(); ok {
						typeName = strLit.Value
					} else if constVal, ok := firstArg.Value.AsConstantValue(); ok {
						typeName = constVal.Value
					}
					// Get additional arguments if any
					for i := 1; i < len(attr.Arguments.Arguments); i++ {
						if arg := attr.Arguments.Arguments[i]; arg.Value != nil {
							if strLit, ok := arg.Value.AsStringValue(); ok {
								typeArgs = append(typeArgs, strLit.Value)
							} else if numVal, ok := arg.Value.AsNumericValue(); ok {
								typeArgs = append(typeArgs, numVal.String())
							}
						}
					}
				}
				break
			}
		}

		if typeName == "" {
			continue
		}

		tempDiags := diagnostics.NewDiagnostics()
		nativeTypeInstance := connector.ParseNativeType(typeName, typeArgs, nativeTypeInfo.Span, &tempDiags)
		if nativeTypeInstance == nil {
			continue
		}

		// Get native type name
		nativeTypeName, _ := connector.NativeTypeToParts(nativeTypeInstance)

		// Check if this native type is in the restricted list
		isRestricted := false
		for _, restrictedType := range mysqlNativeTypesThatCannotBeUsedInKeySpecification {
			if nativeTypeName == restrictedType {
				isRestricted = true
				break
			}
		}

		if !isRestricted {
			continue
		}

		// Length defined, so we allow the primary key
		if field.Length() != nil {
			continue
		}

		// Create error
		span := diagnostics.EmptySpan()
		if astAttr := pk.AstAttribute(); astAttr != nil {
			span = diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), diagnostics.FileIDZero)
		}
		errorFactory := connector.NativeInstanceError(nativeTypeInstance)
		err := errorFactory.NewIncompatibleNativeTypeWithIDError(mysqlLengthGuide, span)
		diags.PushError(err)
		break // Only report one error per primary key
	}
}

// validateMySQLReferentialActionSetDefault validates that MySQL doesn't use SetDefault referential action.
func validateMySQLReferentialActionSetDefault(connector *MySqlConnector, field *database.RelationFieldWalker, diags *diagnostics.Diagnostics) {
	getSpan := func(referentialActionType string) diagnostics.Span {
		astField := field.AstField()
		if astField == nil {
			return diagnostics.EmptySpan()
		}
		// Try to find the @relation attribute and its argument
		for _, attr := range astField.Attributes {
			if attr.Name.Name == "relation" {
				if span := attr.SpanForArgument(referentialActionType); span != nil {
					return diagnostics.NewSpan(span.Offset, span.Offset+10, diagnostics.FileIDZero) // Rough length
				}
			}
		}
		return diagnostics.NewSpan(astField.Pos.Offset, astField.Pos.Offset+len(astField.Name.Name), diagnostics.FileIDZero)
	}

	warningMsg := fmt.Sprintf(
		"%s does not actually support the `SetDefault` referential action, so using it may result in unexpected errors. Read more at https://pris.ly/d/mysql-set-default",
		connector.Name(),
	)

	if onDelete := field.OnDelete(); onDelete != nil && onDelete.Action == database.ReferentialActionSetDefault {
		span := getSpan("onDelete")
		diags.PushWarning(diagnostics.NewDatamodelWarning(warningMsg, span))
	}

	if onUpdate := field.OnUpdate(); onUpdate != nil && onUpdate.Action == database.ReferentialActionSetDefault {
		span := getSpan("onUpdate")
		diags.PushWarning(diagnostics.NewDatamodelWarning(warningMsg, span))
	}
}
