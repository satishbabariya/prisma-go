// Package pslcore provides SQL Server connector implementation.
package validation

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// MsSqlConnector implements the ExtendedConnector interface for SQL Server.
type MsSqlConnector struct {
	*BaseExtendedConnector
}

// NewMsSqlConnector creates a new SQL Server connector.
func NewMsSqlConnector() *MsSqlConnector {
	return &MsSqlConnector{
		BaseExtendedConnector: &BaseExtendedConnector{
			name:                "SQL Server",
			providerName:        "sqlserver",
			flavour:             FlavourSQLServer,
			capabilities:        ConnectorCapabilities(ConnectorCapabilityAutoIncrement | ConnectorCapabilityAutoIncrementAllowedOnNonId | ConnectorCapabilityAutoIncrementMultipleAllowed | ConnectorCapabilityMultiSchema | ConnectorCapabilityNamedDefaultValues | ConnectorCapabilityNamedForeignKeys | ConnectorCapabilityNamedPrimaryKeys | ConnectorCapabilityImplicitManyToManyRelation),
			maxIdentifierLength: 128,
		},
	}
}

// Name returns the connector name.
func (c *MsSqlConnector) Name() string {
	return "SQL Server"
}

// HasCapability checks if the connector has a specific capability.
func (c *MsSqlConnector) HasCapability(capability ConnectorCapability) bool {
	return c.Capabilities()&(1<<uint(capability)) != 0
}

// ReferentialActions returns the supported referential actions.
func (c *MsSqlConnector) ReferentialActions(relationMode RelationMode) []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// ForeignKeyReferentialActions returns the supported foreign key referential actions.
func (c *MsSqlConnector) ForeignKeyReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// EmulatedReferentialActions returns the supported emulated referential actions.
func (c *MsSqlConnector) EmulatedReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// SupportsReferentialAction checks if the connector supports the given referential action.
func (c *MsSqlConnector) SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool {
	actions := c.ReferentialActions(relationMode)
	for _, supportedAction := range actions {
		if supportedAction == action {
			return true
		}
	}
	return false
}

// ValidateURL validates a SQL Server connection URL.
func (c *MsSqlConnector) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "sqlserver://") && !strings.HasPrefix(url, "env:") {
		return fmt.Errorf("SQL Server URL must start with `sqlserver://`")
	}
	return nil
}

// AvailableNativeTypeConstructors returns available native type constructors.
func (c *MsSqlConnector) AvailableNativeTypeConstructors() []*NativeTypeConstructor {
	return GetMsSqlNativeTypeConstructors()
}

// ConstraintViolationScopes returns the constraint violation scopes.
func (c *MsSqlConnector) ConstraintViolationScopes() []ConstraintScope {
	return []ConstraintScope{
		ConstraintScopeGlobalPrimaryKeyForeignKeyDefault,
		ConstraintScopeModelPrimaryKeyKeyIndex,
	}
}

// ConstraintViolationScopesExtended returns the extended constraint violation scopes.
func (c *MsSqlConnector) ConstraintViolationScopesExtended() []ExtendedConstraintScope {
	return []ExtendedConstraintScope{
		ExtendedConstraintScopeGlobalPrimaryKeyForeignKeyDefault,
		ExtendedConstraintScopeModelPrimaryKeyKeyIndex,
	}
}

// SupportsIndexType checks if the connector supports the given index algorithm.
func (c *MsSqlConnector) SupportsIndexType(algo IndexAlgorithm) bool {
	return database.IndexAlgorithm(algo) == database.IndexAlgorithmBTree
}

// ParseNativeType parses a native type from name and arguments.
func (c *MsSqlConnector) ParseNativeType(name string, args []string, span Span, diags *Diagnostics) *NativeTypeInstance {
	mssqlType := ParseMsSqlNativeType(name, args, span, diags)
	if mssqlType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: mssqlType}
}

// ScalarTypeForNativeType returns the default scalar type for a native type.
func (c *MsSqlConnector) ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType {
	if nativeType == nil || nativeType.Data == nil {
		return nil
	}
	mssqlType, ok := nativeType.Data.(*MsSqlNativeTypeInstance)
	if !ok {
		return nil
	}
	scalarType := MsSqlNativeTypeToScalarType(mssqlType)
	if scalarType == nil {
		return nil
	}
	return &database.ScalarFieldType{BuiltInScalar: scalarType}
}

// DefaultNativeTypeForScalarType returns the default native type for a scalar type.
func (c *MsSqlConnector) DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance {
	if scalarType == nil || scalarType.BuiltInScalar == nil {
		return nil
	}
	mssqlType := MsSqlScalarTypeToDefaultNativeType(*scalarType.BuiltInScalar)
	if mssqlType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: mssqlType}
}

// NativeTypeToString converts a native type instance to string representation.
func (c *MsSqlConnector) NativeTypeToString(instance *NativeTypeInstance) string {
	if instance == nil || instance.Data == nil {
		return ""
	}
	mssqlType, ok := instance.Data.(*MsSqlNativeTypeInstance)
	if !ok {
		return ""
	}
	return MsSqlNativeTypeToString(mssqlType)
}

// NativeTypeToParts returns the debug representation of a native type.
func (c *MsSqlConnector) NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string) {
	if nativeType == nil || nativeType.Data == nil {
		return "", []string{}
	}
	mssqlType, ok := nativeType.Data.(*MsSqlNativeTypeInstance)
	if !ok {
		return "", []string{}
	}

	name := mssqlType.Type.String()
	var parts []string
	if mssqlType.Parameter != nil && mssqlType.Parameter.IsMax {
		parts = append(parts, "Max")
	} else if mssqlType.Precision != nil && mssqlType.Scale != nil {
		parts = append(parts, fmt.Sprintf("%d", *mssqlType.Precision), fmt.Sprintf("%d", *mssqlType.Scale))
	} else if mssqlType.Precision != nil {
		parts = append(parts, fmt.Sprintf("%d", *mssqlType.Precision))
	} else if mssqlType.Length != nil {
		parts = append(parts, fmt.Sprintf("%d", *mssqlType.Length))
	}
	return name, parts
}

// FindNativeTypeConstructor finds a native type constructor by name.
func (c *MsSqlConnector) FindNativeTypeConstructor(name string) *NativeTypeConstructor {
	constructors := c.AvailableNativeTypeConstructors()
	for _, constructor := range constructors {
		if constructor != nil && constructor.Name == name {
			return constructor
		}
	}
	return nil
}

// NativeInstanceError creates a native type error factory.
func (c *MsSqlConnector) NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory {
	nativeTypeStr := c.NativeTypeToString(instance)
	factory := diagnostics.NewNativeTypeErrorFactory(nativeTypeStr, c.Name())
	return &factory
}

// ValidateNativeTypeArguments validates native type attribute arguments.
func (c *MsSqlConnector) ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics) {
	if nativeType == nil || nativeType.Data == nil {
		return
	}
	mssqlType, ok := nativeType.Data.(*MsSqlNativeTypeInstance)
	if !ok {
		return
	}

	// Validate that the native type is compatible with the scalar type
	expectedScalarType := MsSqlNativeTypeToScalarType(mssqlType)
	if expectedScalarType == nil {
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("SQL Server", mssqlType.Type.String(), span))
		return
	}
	// Additional validation can be added here for specific type combinations
}

// ValidateEnum performs enum-specific validation.
func (c *MsSqlConnector) ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics) {
}

// ValidateModel performs model-specific validation.
func (c *MsSqlConnector) ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics) {
	// Validate indexes for SQL Server-specific constraints
	for _, index := range modelWalker.Indexes() {
		validateMsSqlIndexFieldTypes(c, index, diags)
	}

	// Validate primary key for SQL Server-specific constraints
	if pk := modelWalker.PrimaryKey(); pk != nil {
		validateMsSqlPrimaryKeyFieldTypes(c, pk, diags)
	}
}

// ValidateView performs view-specific validation.
func (c *MsSqlConnector) ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics) {
}

// ValidateRelationField performs relation field validation.
func (c *MsSqlConnector) ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics) {
}

// ValidateDatasource performs datasource-specific validation.
func (c *MsSqlConnector) ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics) {
	if datasource.URL != nil {
		url := *datasource.URL
		if url != "" && !strings.HasPrefix(url, "sqlserver://") && !strings.HasPrefix(url, "env:") {
			diags.PushWarning(diagnostics.NewDatamodelWarning(
				"SQL Server datasource URL should start with `sqlserver://`.",
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}
	}
}

// ValidateScalarFieldUnknownDefaultFunctions validates scalar field default functions.
func (c *MsSqlConnector) ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics) {
	for _, model := range db.WalkModels() {
		for _, field := range model.ScalarFields() {
			if defaultValue := field.DefaultValue(); defaultValue != nil {
				if funcCall, ok := defaultValue.Value().AsFunction(); ok {
					switch funcCall.Name {
					case "now", "uuid", "cuid", "cuid2", "autoincrement", "dbgenerated", "env":
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

// SQL Server heap-allocated types that cannot be used in indexes/primary keys
var msSqlHeapAllocatedTypes = []string{
	"Text",
	"NText",
	"Image",
}

// validateMsSqlIndexFieldTypes validates that SQL Server index fields don't use heap-allocated types.
func validateMsSqlIndexFieldTypes(connector *MsSqlConnector, index *database.IndexWalker, diags *diagnostics.Diagnostics) {
	for _, field := range index.ScalarFieldAttributes() {
		scalarField := field.ScalarField()
		if scalarField == nil {
			continue
		}

		nativeTypeInfo := scalarField.NativeType()
		if nativeTypeInfo == nil {
			continue
		}

		// Get type name from AST field
		astField := scalarField.AstField()
		if astField == nil {
			continue
		}

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

		// Check if this native type is heap-allocated
		isHeapAllocated := false
		for _, heapType := range msSqlHeapAllocatedTypes {
			if nativeTypeName == heapType {
				isHeapAllocated = true
				break
			}
		}

		if !isHeapAllocated {
			continue
		}

		// Create error
		errorFactory := connector.NativeInstanceError(nativeTypeInstance)
		span := diagnostics.EmptySpan()
		if astAttr := index.AstAttribute(); astAttr != nil {
			span = diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), diagnostics.FileIDZero)
		}
		var err diagnostics.DatamodelError
		if index.IsUnique() {
			err = errorFactory.NewIncompatibleNativeTypeWithUniqueError("", span)
		} else {
			err = errorFactory.NewIncompatibleNativeTypeWithIndexError("", span)
		}
		diags.PushError(err)
		break
	}
}

// validateMsSqlPrimaryKeyFieldTypes validates that SQL Server primary key fields don't use heap-allocated types or Bytes.
func validateMsSqlPrimaryKeyFieldTypes(connector *MsSqlConnector, pk *database.PrimaryKeyWalker, diags *diagnostics.Diagnostics) {
	span := diagnostics.EmptySpan()
	if astAttr := pk.AstAttribute(); astAttr != nil {
		span = diagnostics.NewSpan(astAttr.Pos.Offset, astAttr.Pos.Offset+len(astAttr.String()), diagnostics.FileIDZero)
	}

	for _, field := range pk.ScalarFieldAttributes() {
		scalarField := field.ScalarField()
		if scalarField == nil {
			continue
		}

		// Check for Bytes type (not allowed in primary key)
		if scalarType := scalarField.ScalarType(); scalarType != nil && *scalarType == database.ScalarTypeBytes {
			diags.PushError(diagnostics.NewInvalidModelError(
				"Using Bytes type is not allowed in the model's id.",
				span,
			))
			break
		}

		nativeTypeInfo := scalarField.NativeType()
		if nativeTypeInfo == nil {
			continue
		}

		// Get type name from AST field
		astField := scalarField.AstField()
		if astField == nil {
			continue
		}

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

		// Check if this native type is heap-allocated
		isHeapAllocated := false
		for _, heapType := range msSqlHeapAllocatedTypes {
			if nativeTypeName == heapType {
				isHeapAllocated = true
				break
			}
		}

		if !isHeapAllocated {
			continue
		}

		// Create error
		errorFactory := connector.NativeInstanceError(nativeTypeInstance)
		err := errorFactory.NewIncompatibleNativeTypeWithIDError("", span)
		diags.PushError(err)
		break
	}
}
