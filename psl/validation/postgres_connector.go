// Package pslcore provides PostgreSQL connector implementation.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// PostgresConnector implements the ExtendedConnector interface for PostgreSQL.
type PostgresConnector struct {
	*BaseExtendedConnector
}

// NewPostgresConnector creates a new PostgreSQL connector.
func NewPostgresConnector() *PostgresConnector {
	return &PostgresConnector{
		BaseExtendedConnector: &BaseExtendedConnector{
			name:                "PostgreSQL",
			providerName:        "postgresql",
			flavour:             FlavourPostgres,
			capabilities:        ConnectorCapabilities(ConnectorCapabilityEnums | ConnectorCapabilityCompositeTypes | ConnectorCapabilityJson | ConnectorCapabilityFullTextIndex | ConnectorCapabilityMultiSchema | ConnectorCapabilityViews | ConnectorCapabilityAutoIncrement | ConnectorCapabilityAutoIncrementMultipleAllowed | ConnectorCapabilityNamedDefaultValues | ConnectorCapabilityIndexColumnLengthPrefixing),
			maxIdentifierLength: 63,
		},
	}
}

// Name returns the connector name.
func (c *PostgresConnector) Name() string {
	return "PostgreSQL"
}

// HasCapability checks if the connector has a specific capability.
func (c *PostgresConnector) HasCapability(capability ConnectorCapability) bool {
	return c.Capabilities()&(1<<uint(capability)) != 0
}

// ReferentialActions returns the supported referential actions for the given relation mode.
func (c *PostgresConnector) ReferentialActions(relationMode RelationMode) []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// ForeignKeyReferentialActions returns the supported foreign key referential actions.
func (c *PostgresConnector) ForeignKeyReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// EmulatedReferentialActions returns the supported emulated referential actions.
func (c *PostgresConnector) EmulatedReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// SupportsReferentialAction checks if the connector supports the given referential action for the relation mode.
func (c *PostgresConnector) SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool {
	// PostgreSQL supports all referential actions
	actions := c.ReferentialActions(relationMode)
	for _, supportedAction := range actions {
		if supportedAction == action {
			return true
		}
	}
	return false
}

// ScalarFilterName returns the filter name for a scalar type and native type combination.
func (c *PostgresConnector) ScalarFilterName(scalarTypeName string, nativeTypeName *string) string {
	if nativeTypeName != nil && strings.EqualFold(*nativeTypeName, "uuid") {
		return "Uuid"
	}
	return scalarTypeName
}

// StringFilters returns the string filters for a given input object name.
func (c *PostgresConnector) StringFilters(inputObjectName string) []StringFilter {
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
func (c *PostgresConnector) DatamodelCompletions(db *database.ParserDatabase, position SchemaPosition, completions *CompletionList) {
	// PostgreSQL-specific completions can be added here
	// For example: index types, operator classes, etc.
	c.BaseExtendedConnector.DatamodelCompletions(db, position, completions)
}

// DatasourceCompletions provides IDE/LSP autocomplete support for datasource configuration.
func (c *PostgresConnector) DatasourceCompletions(config Configuration, completions *CompletionList) {
	// PostgreSQL-specific datasource completions can be added here
	c.BaseExtendedConnector.DatasourceCompletions(config, completions)
}

// ValidateURL validates a PostgreSQL connection URL.
func (c *PostgresConnector) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://") && !strings.HasPrefix(url, "prisma+postgres://") && !strings.HasPrefix(url, "env:") {
		return fmt.Errorf("must start with the protocol `postgresql://` or `postgres://`")
	}
	return nil
}

// AvailableNativeTypeConstructors returns available native type constructors.
func (c *PostgresConnector) AvailableNativeTypeConstructors() []*NativeTypeConstructor {
	return GetPostgresNativeTypeConstructors()
}

// ConstraintViolationScopes returns the constraint violation scopes (for Connector interface).
func (c *PostgresConnector) ConstraintViolationScopes() []ConstraintScope {
	return []ConstraintScope{
		ConstraintScopeGlobal,
	}
}

// ConstraintViolationScopesExtended returns the extended constraint violation scopes.
func (c *PostgresConnector) ConstraintViolationScopesExtended() []ExtendedConstraintScope {
	return []ExtendedConstraintScope{
		ExtendedConstraintScopeGlobalPrimaryKeyForeignKeyDefault,
	}
}

// ConstraintViolationScopesCompat returns constraint scopes compatible with the base Connector interface.
func (c *PostgresConnector) ConstraintViolationScopesCompat() []ConstraintScope {
	extended := c.ConstraintViolationScopes()
	result := make([]ConstraintScope, len(extended))
	for i := range extended {
		// Map ExtendedConstraintScope to ConstraintScope
		// For now, map all to Global scope as a simple conversion
		result[i] = ConstraintScopeGlobal
	}
	return result
}

// SupportedIndexTypes returns the supported index types for PostgreSQL.
func (c *PostgresConnector) SupportedIndexTypes() []IndexAlgorithm {
	return []IndexAlgorithm{
		IndexAlgorithmBTree,
		IndexAlgorithmHash,
		IndexAlgorithmGist,
		IndexAlgorithmGin,
		IndexAlgorithmSpGist,
		IndexAlgorithmBrin,
	}
}

// SupportsIndexType checks if the connector supports the given index algorithm.
func (c *PostgresConnector) SupportsIndexType(algo IndexAlgorithm) bool {
	// PostgreSQL supports Normal, Hash, Gist, Gin, SpGist, Brin, and Btree (default)
	// Map database.IndexAlgorithm to pslcore.IndexAlgorithm
	supported := []database.IndexAlgorithm{
		database.IndexAlgorithmBTree, // Normal/BTree is the default
		database.IndexAlgorithmHash,
		database.IndexAlgorithmGist,
		database.IndexAlgorithmGin,
		database.IndexAlgorithmSpGist,
		database.IndexAlgorithmBrin,
	}
	// Convert algo to database.IndexAlgorithm for comparison
	algoPD := database.IndexAlgorithm(algo)
	for _, supportedAlgo := range supported {
		if algoPD == supportedAlgo {
			return true
		}
	}
	return false
}

// NativeTypeToParts returns the debug representation of a native type.
func (c *PostgresConnector) NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string) {
	if nativeType == nil || nativeType.Data == nil {
		return "", []string{}
	}
	pgType, ok := nativeType.Data.(*PostgresNativeTypeInstance)
	if !ok {
		return "", []string{}
	}

	name := pgType.Type.String()
	var parts []string
	if pgType.Precision != nil && pgType.Scale != nil {
		parts = append(parts, fmt.Sprintf("%d", *pgType.Precision), fmt.Sprintf("%d", *pgType.Scale))
	} else if pgType.Precision != nil {
		parts = append(parts, fmt.Sprintf("%d", *pgType.Precision))
	} else if pgType.Length != nil {
		parts = append(parts, fmt.Sprintf("%d", *pgType.Length))
	}
	return name, parts
}

// FindNativeTypeConstructor finds a native type constructor by name.
func (c *PostgresConnector) FindNativeTypeConstructor(name string) *NativeTypeConstructor {
	constructors := c.AvailableNativeTypeConstructors()
	for _, constructor := range constructors {
		if constructor != nil && constructor.Name == name {
			return constructor
		}
	}
	return nil
}

// ParseNativeType parses a native type from name and arguments.
func (c *PostgresConnector) ParseNativeType(name string, args []string, span Span, diagnostics *Diagnostics) *NativeTypeInstance {
	pgType := ParsePostgresNativeType(name, args, span, diagnostics)
	if pgType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: pgType}
}

// ScalarTypeForNativeType returns the default scalar type for a native type.
func (c *PostgresConnector) ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType {
	if nativeType == nil || nativeType.Data == nil {
		return nil
	}
	pgType, ok := nativeType.Data.(*PostgresNativeTypeInstance)
	if !ok {
		return nil
	}
	scalarType := PostgresNativeTypeToScalarType(pgType)
	if scalarType == nil {
		return nil
	}
	return &database.ScalarFieldType{BuiltInScalar: scalarType}
}

// DefaultNativeTypeForScalarType returns the default native type for a scalar type.
func (c *PostgresConnector) DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance {
	if scalarType == nil || scalarType.BuiltInScalar == nil {
		return nil
	}
	pgType := PostgresScalarTypeToDefaultNativeType(*scalarType.BuiltInScalar)
	if pgType == nil {
		return nil
	}
	return &NativeTypeInstance{Data: pgType}
}

// NativeTypeToString converts a native type instance to string representation.
func (c *PostgresConnector) NativeTypeToString(instance *NativeTypeInstance) string {
	if instance == nil || instance.Data == nil {
		return ""
	}
	pgType, ok := instance.Data.(*PostgresNativeTypeInstance)
	if !ok {
		return ""
	}
	return PostgresNativeTypeToString(pgType)
}

// NativeInstanceError creates a native type error factory.
func (c *PostgresConnector) NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory {
	nativeTypeStr := c.NativeTypeToString(instance)
	factory := diagnostics.NewNativeTypeErrorFactory(nativeTypeStr, c.Name())
	return &factory
}

// ValidateNativeTypeArguments validates native type attribute arguments.
func (c *PostgresConnector) ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics) {
	if nativeType == nil || nativeType.Data == nil {
		return
	}
	pgType, ok := nativeType.Data.(*PostgresNativeTypeInstance)
	if !ok {
		return
	}

	// Validate that the native type is compatible with the scalar type
	expectedScalarType := PostgresNativeTypeToScalarType(pgType)
	if expectedScalarType == nil {
		diags.PushError(diagnostics.NewNativeTypeNameUnknownError("PostgreSQL", pgType.Type.String(), span))
		return
	}

	// Additional validation can be added here for specific type combinations
	// For example, Decimal requires precision and scale, VarChar requires length, etc.
}

// ValidateEnum performs enum-specific validation.
// PostgreSQL doesn't have specific enum validations beyond base validations.
func (c *PostgresConnector) ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics) {
	// PostgreSQL enums are validated by base validations
	// No connector-specific enum validations needed
}

// ValidateModel performs model-specific validation.
func (c *PostgresConnector) ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics) {
	// Validate indexes for PostgreSQL-specific constraints
	for _, index := range modelWalker.Indexes() {
		// Check SP-GiST index column count (must be single column)
		if algo := index.Algorithm(); algo != nil && *algo == database.IndexAlgorithmSpGist {
			if len(index.Fields()) > 1 {
				// Get span from AST attribute
				span := diagnostics.NewSpan(0, 0, diagnostics.FileIDZero)
				if astAttr := index.AstAttribute(); astAttr != nil {
					span = astAttr.Span
				}
				diags.PushError(diagnostics.NewAttributeValidationError(
					"SpGist does not support multi-column indices.",
					index.AttributeName(),
					span,
				))
			}
		}

		// Additional PostgreSQL-specific index validations would go here
		// For example: checking compatible native types, operator classes, etc.
		// These are complex and can be added incrementally
	}
}

// ValidateView performs view-specific validation.
// PostgreSQL views are validated by base validations.
func (c *PostgresConnector) ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics) {
	// PostgreSQL doesn't have specific view validations beyond base validations
	// Views are validated by the base validation pipeline
}

// ValidateRelationField performs relation field validation.
// PostgreSQL doesn't have specific relation field validations.
func (c *PostgresConnector) ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics) {
	// PostgreSQL relation fields are validated by base validations
	// No connector-specific relation field validations needed
}

// ValidateDatasource performs datasource-specific validation.
func (c *PostgresConnector) ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics) {
	// Check if extensions are used without postgresqlExtensions preview feature
	// Note: This is a simplified check - full implementation would parse datasource properties
	// For now, base validations handle most datasource validation
	if datasource.URL != nil {
		// Basic URL validation
		url := *datasource.URL
		if url != "" && !strings.HasPrefix(url, "postgresql://") && !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "env:") {
			// Allow env: prefix for environment variables
			if !strings.HasPrefix(url, "postgresql://") {
				diags.PushWarning(diagnostics.NewDatamodelWarning(
					"PostgreSQL datasource URL should start with `postgresql://` or `postgres://`.",
					diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
				))
			}
		}
	}
}

// ValidateScalarFieldUnknownDefaultFunctions validates scalar field default functions.
func (c *PostgresConnector) ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics) {
	// Use the base implementation
	for _, model := range db.WalkModels() {
		for _, field := range model.ScalarFields() {
			if defaultValue := field.DefaultValue(); defaultValue != nil {
				if funcCall, ok := defaultValue.Value().(ast.FunctionCall); ok {
					switch funcCall.Name.Name {
					case "now", "uuid", "cuid", "cuid2", "autoincrement", "dbgenerated", "env":
						// Known functions, do nothing
					default:
						diags.PushError(diagnostics.NewDefaultUnknownFunctionError(funcCall.Name.Name, funcCall.Span()))
					}
				}
			}
		}
	}
}
