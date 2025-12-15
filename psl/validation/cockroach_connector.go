// Package pslcore provides CockroachDB connector implementation.
package validation

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// CockroachConnector implements the ExtendedConnector interface for CockroachDB.
// CockroachDB is similar to PostgreSQL, so we can reuse PostgreSQL native types.
type CockroachConnector struct {
	*BaseExtendedConnector
}

// NewCockroachConnector creates a new CockroachDB connector.
func NewCockroachConnector() *CockroachConnector {
	return &CockroachConnector{
		BaseExtendedConnector: &BaseExtendedConnector{
			name:                "CockroachDB",
			providerName:        "cockroachdb",
			flavour:             FlavourCockroach,
			capabilities:        ConnectorCapabilities(ConnectorCapabilityEnums | ConnectorCapabilityCompositeTypes | ConnectorCapabilityJson | ConnectorCapabilityFullTextIndex | ConnectorCapabilityMultiSchema | ConnectorCapabilityViews | ConnectorCapabilityAutoIncrement | ConnectorCapabilityAutoIncrementMultipleAllowed | ConnectorCapabilityNamedDefaultValues | ConnectorCapabilityIndexColumnLengthPrefixing),
			maxIdentifierLength: 63,
		},
	}
}

// Name returns the connector name.
func (c *CockroachConnector) Name() string {
	return "CockroachDB"
}

// HasCapability checks if the connector has a specific capability.
func (c *CockroachConnector) HasCapability(capability ConnectorCapability) bool {
	return c.Capabilities()&(1<<uint(capability)) != 0
}

// ReferentialActions returns the supported referential actions.
func (c *CockroachConnector) ReferentialActions(relationMode RelationMode) []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// ForeignKeyReferentialActions returns the supported foreign key referential actions.
func (c *CockroachConnector) ForeignKeyReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// EmulatedReferentialActions returns the supported emulated referential actions.
func (c *CockroachConnector) EmulatedReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// SupportsReferentialAction checks if the connector supports the given referential action.
func (c *CockroachConnector) SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool {
	actions := c.ReferentialActions(relationMode)
	for _, supportedAction := range actions {
		if supportedAction == action {
			return true
		}
	}
	return false
}

// ScalarFilterName returns the filter name for a scalar type and native type combination.
func (c *CockroachConnector) ScalarFilterName(scalarTypeName string, nativeTypeName *string) string {
	if nativeTypeName != nil && strings.EqualFold(*nativeTypeName, "uuid") {
		return "Uuid"
	}
	return scalarTypeName
}

// StringFilters returns the string filters for a given input object name.
func (c *CockroachConnector) StringFilters(inputObjectName string) []StringFilter {
	if inputObjectName == "Uuid" {
		return []StringFilter{} // UUID doesn't support string filters
	}
	if inputObjectName == "String" {
		// Return all filters for String type
		return []StringFilter{
			StringFilterContains,
			StringFilterStartsWith,
			StringFilterEndsWith,
		}
	}
	panic(fmt.Sprintf("Unexpected scalar input object name for string filters: %s", inputObjectName))
}

// DatamodelCompletions provides IDE/LSP autocomplete support for schema positions.
func (c *CockroachConnector) DatamodelCompletions(db *database.ParserDatabase, position SchemaPosition, completions *CompletionList) {
	// CockroachDB-specific completions can be added here
	c.BaseExtendedConnector.DatamodelCompletions(db, position, completions)
}

// DatasourceCompletions provides IDE/LSP autocomplete support for datasource configuration.
func (c *CockroachConnector) DatasourceCompletions(config Configuration, completions *CompletionList) {
	// CockroachDB-specific datasource completions can be added here
	c.BaseExtendedConnector.DatasourceCompletions(config, completions)
}

// ValidateURL validates a CockroachDB connection URL.
func (c *CockroachConnector) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "postgresql://") && !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "env:") {
		return fmt.Errorf("CockroachDB URL should start with `postgresql://` or `postgres://`")
	}
	return nil
}

// AvailableNativeTypeConstructors returns available native type constructors.
func (c *CockroachConnector) AvailableNativeTypeConstructors() []*NativeTypeConstructor {
	return GetCockroachNativeTypeConstructors()
}

// ConstraintViolationScopes returns the constraint violation scopes.
func (c *CockroachConnector) ConstraintViolationScopes() []ConstraintScope {
	return []ConstraintScope{ConstraintScopeGlobal}
}

// ConstraintViolationScopesExtended returns the extended constraint violation scopes.
func (c *CockroachConnector) ConstraintViolationScopesExtended() []ExtendedConstraintScope {
	return []ExtendedConstraintScope{
		ExtendedConstraintScopeGlobalPrimaryKeyForeignKeyDefault,
	}
}

// SupportedIndexTypes returns the supported index types for CockroachDB.
func (c *CockroachConnector) SupportedIndexTypes() []IndexAlgorithm {
	return []IndexAlgorithm{
		IndexAlgorithmBTree,
		IndexAlgorithmGin,
	}
}

// SupportsIndexType checks if the connector supports the given index algorithm.
func (c *CockroachConnector) SupportsIndexType(algo IndexAlgorithm) bool {
	supported := []database.IndexAlgorithm{
		database.IndexAlgorithmBTree,
		database.IndexAlgorithmGin,
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
func (c *CockroachConnector) ParseNativeType(name string, args []string, span Span, diagnostics *Diagnostics) *NativeTypeInstance {
	crType := ParseCockroachNativeType(name, args, span, diagnostics)
	if crType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: crType}
}

// ScalarTypeForNativeType returns the default scalar type for a native type.
func (c *CockroachConnector) ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType {
	if nativeType == nil || nativeType.Data == nil {
		return nil
	}
	crType, ok := nativeType.Data.(*CockroachNativeTypeInstance)
	if !ok {
		return nil
	}
	scalarType := CockroachNativeTypeToScalarType(crType)
	if scalarType == nil {
		return nil
	}
	return &database.ScalarFieldType{BuiltInScalar: scalarType}
}

// DefaultNativeTypeForScalarType returns the default native type for a scalar type.
func (c *CockroachConnector) DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance {
	if scalarType == nil || scalarType.BuiltInScalar == nil {
		return nil
	}
	crType := CockroachScalarTypeToDefaultNativeType(*scalarType.BuiltInScalar)
	if crType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: crType}
}

// NativeTypeToString converts a native type instance to string representation.
func (c *CockroachConnector) NativeTypeToString(instance *NativeTypeInstance) string {
	if instance == nil || instance.Data == nil {
		return ""
	}
	crType, ok := instance.Data.(*CockroachNativeTypeInstance)
	if !ok {
		return ""
	}
	return CockroachNativeTypeToString(crType)
}

// NativeTypeToParts returns the debug representation of a native type.
func (c *CockroachConnector) NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string) {
	if nativeType == nil || nativeType.Data == nil {
		return "", []string{}
	}
	crType, ok := nativeType.Data.(*CockroachNativeTypeInstance)
	if !ok {
		return "", []string{}
	}
	name := crType.Type.String()
	var parts []string
	if crType.Precision != nil && crType.Scale != nil {
		parts = append(parts, fmt.Sprintf("%d", *crType.Precision), fmt.Sprintf("%d", *crType.Scale))
	} else if crType.Precision != nil {
		parts = append(parts, fmt.Sprintf("%d", *crType.Precision))
	} else if crType.Length != nil {
		parts = append(parts, fmt.Sprintf("%d", *crType.Length))
	}
	return name, parts
}

// FindNativeTypeConstructor finds a native type constructor by name.
func (c *CockroachConnector) FindNativeTypeConstructor(name string) *NativeTypeConstructor {
	constructors := c.AvailableNativeTypeConstructors()
	for _, constructor := range constructors {
		if constructor != nil && constructor.Name == name {
			return constructor
		}
	}
	return nil
}

// NativeInstanceError creates a native type error factory.
func (c *CockroachConnector) NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory {
	nativeTypeStr := c.NativeTypeToString(instance)
	factory := diagnostics.NewNativeTypeErrorFactory(nativeTypeStr, c.Name())
	return &factory
}

// ValidateNativeTypeArguments validates native type attribute arguments.
func (c *CockroachConnector) ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics) {
	if nativeType == nil || nativeType.Data == nil {
		return
	}
	crType, ok := nativeType.Data.(*CockroachNativeTypeInstance)
	if !ok {
		return
	}
	expectedScalarType := CockroachNativeTypeToScalarType(crType)
	if expectedScalarType == nil {
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("CockroachDB", crType.Type.String(), span))
		return
	}
	// Additional validation can be added here for specific type combinations
}

// ValidateEnum performs enum-specific validation.
func (c *CockroachConnector) ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics) {
}

// ValidateModel performs model-specific validation.
func (c *CockroachConnector) ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics) {
}

// ValidateView performs view-specific validation.
func (c *CockroachConnector) ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics) {
}

// ValidateRelationField performs relation field validation.
func (c *CockroachConnector) ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics) {
}

// ValidateDatasource performs datasource-specific validation.
func (c *CockroachConnector) ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics) {
	if datasource.URL != nil {
		url := *datasource.URL
		if url != "" && !strings.HasPrefix(url, "postgresql://") && !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "env:") {
			diags.PushWarning(diagnostics.NewDatamodelWarning(
				"CockroachDB datasource URL should start with `postgresql://` or `postgres://`.",
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}
	}
}

// ValidateScalarFieldUnknownDefaultFunctions validates scalar field default functions.
func (c *CockroachConnector) ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics) {
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
