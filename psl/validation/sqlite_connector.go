// Package pslcore provides SQLite connector implementation.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// SqliteConnector implements the ExtendedConnector interface for SQLite.
type SqliteConnector struct {
	*BaseExtendedConnector
}

// NewSqliteConnector creates a new SQLite connector.
func NewSqliteConnector() *SqliteConnector {
	return &SqliteConnector{
		BaseExtendedConnector: &BaseExtendedConnector{
			name:                "sqlite",
			providerName:        "sqlite",
			flavour:             FlavourSQLite,
			capabilities:        ConnectorCapabilities(ConnectorCapabilityEnums | ConnectorCapabilityJson | ConnectorCapabilityAutoIncrement | ConnectorCapabilityImplicitManyToManyRelation),
			maxIdentifierLength: 10000,
		},
	}
}

// Name returns the connector name.
func (c *SqliteConnector) Name() string {
	return "sqlite"
}

// HasCapability checks if the connector has a specific capability.
func (c *SqliteConnector) HasCapability(capability ConnectorCapability) bool {
	return c.Capabilities()&(1<<uint(capability)) != 0
}

// ReferentialActions returns the supported referential actions for the given relation mode.
func (c *SqliteConnector) ReferentialActions(relationMode RelationMode) []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// ForeignKeyReferentialActions returns the supported foreign key referential actions.
func (c *SqliteConnector) ForeignKeyReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionNoAction,
		ReferentialActionRestrict,
		ReferentialActionCascade,
		ReferentialActionSetNull,
		ReferentialActionSetDefault,
	}
}

// EmulatedReferentialActions returns the supported emulated referential actions.
func (c *SqliteConnector) EmulatedReferentialActions() []ReferentialAction {
	return []ReferentialAction{
		ReferentialActionRestrict,
		ReferentialActionSetNull,
		ReferentialActionCascade,
	}
}

// SupportsReferentialAction checks if the connector supports the given referential action.
func (c *SqliteConnector) SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool {
	actions := c.ReferentialActions(relationMode)
	for _, supportedAction := range actions {
		if supportedAction == action {
			return true
		}
	}
	return false
}

// ValidateURL validates a SQLite connection URL.
func (c *SqliteConnector) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "file:") && !strings.HasPrefix(url, "env:") {
		return fmt.Errorf("SQLite URL must start with `file:`")
	}
	return nil
}

// AvailableNativeTypeConstructors returns available native type constructors.
// SQLite doesn't support native types.
func (c *SqliteConnector) AvailableNativeTypeConstructors() []*NativeTypeConstructor {
	return []*NativeTypeConstructor{}
}

// ConstraintViolationScopes returns the constraint violation scopes.
func (c *SqliteConnector) ConstraintViolationScopes() []ConstraintScope {
	return []ConstraintScope{
		ConstraintScopeGlobalKeyIndex,
	}
}

// ConstraintViolationScopesExtended returns the extended constraint violation scopes.
func (c *SqliteConnector) ConstraintViolationScopesExtended() []ExtendedConstraintScope {
	return []ExtendedConstraintScope{
		ExtendedConstraintScopeGlobalKeyIndex,
	}
}

// SupportsIndexType checks if the connector supports the given index algorithm.
func (c *SqliteConnector) SupportsIndexType(algo IndexAlgorithm) bool {
	// SQLite only supports BTree indexes
	return database.IndexAlgorithm(algo) == database.IndexAlgorithmBTree
}

// ParseNativeType parses a native type from name and arguments.
// SQLite doesn't support native types.
func (c *SqliteConnector) ParseNativeType(name string, args []string, span Span, diags *Diagnostics) *NativeTypeInstance {
	diags.PushError(diagnostics.NewNativeTypesNotSupportedError(c.Name(), span))
	return nil
}

// ScalarTypeForNativeType returns the default scalar type for a native type.
// SQLite doesn't support native types.
func (c *SqliteConnector) ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType {
	return nil
}

// DefaultNativeTypeForScalarType returns the default native type for a scalar type.
// SQLite doesn't support native types.
func (c *SqliteConnector) DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance {
	return nil
}

// NativeTypeToString converts a native type instance to string representation.
// SQLite doesn't support native types.
func (c *SqliteConnector) NativeTypeToString(instance *NativeTypeInstance) string {
	return ""
}

// NativeTypeToParts returns the debug representation of a native type.
// SQLite doesn't support native types.
func (c *SqliteConnector) NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string) {
	return "", []string{}
}

// FindNativeTypeConstructor finds a native type constructor by name.
// SQLite doesn't support native types.
func (c *SqliteConnector) FindNativeTypeConstructor(name string) *NativeTypeConstructor {
	return nil
}

// NativeInstanceError creates a native type error factory.
// SQLite doesn't support native types.
func (c *SqliteConnector) NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory {
	return nil
}

// ValidateNativeTypeArguments validates native type attribute arguments.
// SQLite doesn't support native types.
func (c *SqliteConnector) ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics) {
	// SQLite doesn't support native types
}

// ValidateEnum performs enum-specific validation.
func (c *SqliteConnector) ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics) {
	// SQLite enums are validated by base validations
}

// ValidateModel performs model-specific validation.
func (c *SqliteConnector) ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics) {
	// SQLite-specific model validations
	// For now, base validations handle most cases
}

// ValidateView performs view-specific validation.
func (c *SqliteConnector) ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics) {
	// SQLite views are validated by base validations
}

// ValidateRelationField performs relation field validation.
func (c *SqliteConnector) ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics) {
	// SQLite relation fields are validated by base validations
}

// ValidateDatasource performs datasource-specific validation.
func (c *SqliteConnector) ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics) {
	if datasource.URL != nil {
		url := *datasource.URL
		if url != "" && !strings.HasPrefix(url, "file:") && !strings.HasPrefix(url, "env:") {
			diags.PushWarning(diagnostics.NewDatamodelWarning(
				"SQLite datasource URL should start with `file:`.",
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}
	}
}

// ValidateScalarFieldUnknownDefaultFunctions validates scalar field default functions.
func (c *SqliteConnector) ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics) {
	// Use base implementation
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
