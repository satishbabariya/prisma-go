// Package pslcore provides extended connector interface functionality.
package validation

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// DatabaseFlavour represents the database flavour.
type DatabaseFlavour int

const (
	FlavourCockroach DatabaseFlavour = iota
	FlavourMongo
	FlavourSQLServer
	FlavourMySQL
	FlavourPostgres
	FlavourSQLite
)

func (f DatabaseFlavour) String() string {
	switch f {
	case FlavourCockroach:
		return "Cockroach"
	case FlavourMongo:
		return "Mongo"
	case FlavourSQLServer:
		return "SQLServer"
	case FlavourMySQL:
		return "MySQL"
	case FlavourPostgres:
		return "Postgres"
	case FlavourSQLite:
		return "SQLite"
	default:
		return "Unknown"
	}
}

// IsSQL returns whether the flavour is SQL-based.
func (f DatabaseFlavour) IsSQL() bool {
	return !f.IsMongo()
}

// IsMongo returns whether the flavour is MongoDB.
func (f DatabaseFlavour) IsMongo() bool {
	return f == FlavourMongo
}

// ExtendedConnector extends the base Connector interface with additional methods.
type ExtendedConnector interface {
	Connector
	ProviderName() string
	IsProvider(name string) bool
	Flavour() DatabaseFlavour
	MaxIdentifierLength() int
	AllowedRelationModeSettings() []RelationMode
	DefaultRelationMode() RelationMode
	ReferentialActions(relationMode RelationMode) []ReferentialAction
	ForeignKeyReferentialActions() []ReferentialAction
	EmulatedReferentialActions() []ReferentialAction
	AllowsSetNullReferentialActionOnNonNullableFields(relationMode RelationMode) bool
	SupportsReferentialAction(relationMode RelationMode, action ReferentialAction) bool
	ScalarFilterName(scalarTypeName string, nativeTypeName *string) string
	StringFilters(inputObjectName string) []StringFilter
	ValidateNativeTypeArguments(nativeType *NativeTypeInstance, scalarType *ScalarType, span diagnostics.Span, diags *diagnostics.Diagnostics)
	ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diags *diagnostics.Diagnostics)
	ConstraintViolationScopesExtended() []ExtendedConstraintScope
	AvailableNativeTypeConstructors() []*NativeTypeConstructor
	ScalarTypeForNativeType(nativeType *NativeTypeInstance, extensionTypes ExtensionTypes) *ScalarFieldType
	DefaultNativeTypeForScalarType(scalarType *ScalarFieldType, schema *ValidatedSchema) *NativeTypeInstance
	NativeTypeToParts(nativeType *NativeTypeInstance) (string, []string)
	FindNativeTypeConstructor(name string) *NativeTypeConstructor
	ParseNativeType(name string, args []string, span Span, diagnostics *Diagnostics) *NativeTypeInstance
	NativeTypeSupportsCompacting(nativeType *NativeTypeInstance) bool
	StaticJoinStrategySupport() bool
	RuntimeJoinStrategySupport() JoinStrategySupport
	SupportedIndexTypes() []IndexAlgorithm
	SupportsIndexType(algo IndexAlgorithm) bool
	ShouldSuggestMissingReferencingFieldsIndexes() bool
	NativeTypeToString(instance *NativeTypeInstance) string
	NativeInstanceError(instance *NativeTypeInstance) *NativeTypeErrorFactory
	ValidateURL(url string) error
	IsSQL() bool
	IsMongo() bool
	SupportsShardKeys() bool
	DoesManageUDTs() bool
}

// BaseExtendedConnector provides a base implementation that can be embedded in specific connectors.
type BaseExtendedConnector struct {
	name                string
	providerName        string
	flavour             DatabaseFlavour
	capabilities        ConnectorCapabilities
	maxIdentifierLength int
}

// ProviderName returns the provider name.
func (c *BaseExtendedConnector) ProviderName() string {
	return c.providerName
}

// Capabilities returns the connector capabilities.
func (c *BaseExtendedConnector) Capabilities() ConnectorCapabilities {
	return c.capabilities
}

// IsProvider checks if the given name matches this connector.
func (c *BaseExtendedConnector) IsProvider(name string) bool {
	return name == c.providerName
}

// Flavour returns the database flavour.
func (c *BaseExtendedConnector) Flavour() DatabaseFlavour {
	return c.flavour
}

// MaxIdentifierLength returns the maximum identifier length.
func (c *BaseExtendedConnector) MaxIdentifierLength() int {
	return c.maxIdentifierLength
}

// IsSQL returns whether this is an SQL connector.
func (c *BaseExtendedConnector) IsSQL() bool {
	return c.flavour.IsSQL()
}

// IsMongo returns whether this is a MongoDB connector.
func (c *BaseExtendedConnector) IsMongo() bool {
	return c.flavour.IsMongo()
}

// Default implementations for methods that can be overridden

// AllowedRelationModeSettings returns the default allowed relation modes.
func (c *BaseExtendedConnector) AllowedRelationModeSettings() []RelationMode {
	return []RelationMode{RelationMode("foreignKeys"), RelationMode("prisma")}
}

// DefaultRelationMode returns the default relation mode.
func (c *BaseExtendedConnector) DefaultRelationMode() RelationMode {
	return RelationMode("foreignKeys")
}

// AllowsSetNullReferentialActionOnNonNullableFields returns whether SET NULL is allowed on non-nullable fields.
func (c *BaseExtendedConnector) AllowsSetNullReferentialActionOnNonNullableFields(relationMode RelationMode) bool {
	return false
}

// ScalarFilterName returns the scalar filter name (default implementation).
func (c *BaseExtendedConnector) ScalarFilterName(scalarTypeName string, nativeTypeName *string) string {
	return scalarTypeName
}

// StringFilters returns string filters for the given input object name.
func (c *BaseExtendedConnector) StringFilters(inputObjectName string) []StringFilter {
	if inputObjectName == "String" {
		// Return all filters - empty slice means all filters available by default
		return []StringFilter{
			StringFilterContains,
			StringFilterStartsWith,
			StringFilterEndsWith,
		}
	}
	panic("Unexpected scalar input object name for string filters: " + inputObjectName)
}

// DatamodelCompletions provides IDE/LSP autocomplete support for schema positions.
func (c *BaseExtendedConnector) DatamodelCompletions(db *database.ParserDatabase, position SchemaPosition, completions *CompletionList) {
	// Default implementation - connectors can override for specific completions
}

// DatasourceCompletions provides IDE/LSP autocomplete support for datasource configuration.
func (c *BaseExtendedConnector) DatasourceCompletions(config Configuration, completions *CompletionList) {
	// Default implementation - connectors can override for specific completions
}

// NativeTypeSupportsCompacting returns whether compacting is supported.
func (c *BaseExtendedConnector) NativeTypeSupportsCompacting(nativeType *NativeTypeInstance) bool {
	return true
}

// StaticJoinStrategySupport returns static join strategy support.
func (c *BaseExtendedConnector) StaticJoinStrategySupport() bool {
	return c.capabilities&(1<<uint(ConnectorCapabilityLateralJoin)) != 0 ||
		c.capabilities&(1<<uint(ConnectorCapabilityCorrelatedSubqueries)) != 0
}

// RuntimeJoinStrategySupport returns runtime join strategy support.
func (c *BaseExtendedConnector) RuntimeJoinStrategySupport() JoinStrategySupport {
	if c.StaticJoinStrategySupport() {
		return JoinStrategySupportYes
	}
	return JoinStrategySupportNo
}

// SupportedIndexTypes returns the supported index types.
func (c *BaseExtendedConnector) SupportedIndexTypes() []IndexAlgorithm {
	return []IndexAlgorithm{IndexAlgorithmBTree}
}

// ShouldSuggestMissingReferencingFieldsIndexes returns whether to suggest missing indexes.
func (c *BaseExtendedConnector) ShouldSuggestMissingReferencingFieldsIndexes() bool {
	return true
}

// SupportsShardKeys returns whether shard keys are supported.
func (c *BaseExtendedConnector) SupportsShardKeys() bool {
	return false
}

// DoesManageUDTs returns whether UDTs are managed.
func (c *BaseExtendedConnector) DoesManageUDTs() bool {
	return false
}

// Helper types and constants

// JoinStrategySupport describes whether a connector supports relation join strategy.
type JoinStrategySupport int

const (
	JoinStrategySupportYes JoinStrategySupport = iota
	JoinStrategySupportUnsupportedDbVersion
	JoinStrategySupportNo
	JoinStrategySupportUnknownYet
)

// ConstraintType represents the type of constraint.
type ConstraintType int

const (
	ConstraintTypePrimary ConstraintType = iota
	ConstraintTypeForeign
	ConstraintTypeKeyOrIndex
	ConstraintTypeDefault
)

// ExtendedConstraintScope represents extended constraint scopes (to avoid conflicts).
type ExtendedConstraintScope int

const (
	ExtendedConstraintScopeGlobalKeyIndex ExtendedConstraintScope = iota
	ExtendedConstraintScopeGlobalForeignKey
	ExtendedConstraintScopeGlobalPrimaryKeyKeyIndex
	ExtendedConstraintScopeGlobalPrimaryKeyForeignKeyDefault
	ExtendedConstraintScopeModelKeyIndex
	ExtendedConstraintScopeModelPrimaryKeyKeyIndex
	ExtendedConstraintScopeModelPrimaryKeyKeyIndexForeignKey
)

// Description returns a human-readable description of the constraint scope.
func (s ExtendedConstraintScope) Description(modelName string) string {
	switch s {
	case ExtendedConstraintScopeGlobalKeyIndex:
		return "global for indexes and unique constraints"
	case ExtendedConstraintScopeGlobalForeignKey:
		return "global for foreign keys"
	case ExtendedConstraintScopeGlobalPrimaryKeyKeyIndex:
		return "global for primary key, indexes and unique constraints"
	case ExtendedConstraintScopeGlobalPrimaryKeyForeignKeyDefault:
		return "global for primary keys, foreign keys and default constraints"
	case ExtendedConstraintScopeModelKeyIndex:
		return fmt.Sprintf("on model `%s` for indexes and unique constraints", modelName)
	case ExtendedConstraintScopeModelPrimaryKeyKeyIndex:
		return fmt.Sprintf("on model `%s` for primary key, indexes and unique constraints", modelName)
	case ExtendedConstraintScopeModelPrimaryKeyKeyIndexForeignKey:
		return fmt.Sprintf("on model `%s` for primary key, indexes, unique constraints and foreign keys", modelName)
	default:
		return "unknown"
	}
}

// Type aliases for missing types
type ConnectorCapabilities int
type ReferentialAction int
type IndexAlgorithm int

// Additional capability constants
const (
	ConnectorCapabilityLateralJoin ConnectorCapability = 1 << iota
	ConnectorCapabilityCorrelatedSubqueries
)

// Additional index algorithm constants
const (
	IndexAlgorithmBTree IndexAlgorithm = iota
	IndexAlgorithmHash
	IndexAlgorithmGist
	IndexAlgorithmGin
	IndexAlgorithmSpGist
	IndexAlgorithmBrin
)

// Additional referential action constants
const (
	ReferentialActionNoAction ReferentialAction = iota
	ReferentialActionRestrict
	ReferentialActionCascade
	ReferentialActionSetNull
	ReferentialActionSetDefault
)

// String returns the string representation of a ReferentialAction.
func (ra ReferentialAction) String() string {
	switch ra {
	case ReferentialActionCascade:
		return "Cascade"
	case ReferentialActionRestrict:
		return "Restrict"
	case ReferentialActionNoAction:
		return "NoAction"
	case ReferentialActionSetNull:
		return "SetNull"
	case ReferentialActionSetDefault:
		return "SetDefault"
	default:
		return "NoAction"
	}
}

// Additional string filter constants would be defined here as needed

// Forward declarations for types that need to be defined elsewhere
type (
	// NativeTypeInstance wraps connector-specific native type instances
	NativeTypeInstance struct {
		// Connector-specific native type data
		// For PostgreSQL, this would be *PostgresNativeTypeInstance
		// For other connectors, it would be their specific types
		Data interface{}
	}
	ScalarType     int
	ExtensionTypes interface{}
)

// Missing type definitions
type (
	// PreviewFeatureFlags represents a set of preview features
	PreviewFeatureFlags int

	// RelationModeFlags represents a set of relation modes
	RelationModeFlags int

	// ReferentialActionFlags represents a set of referential actions
	ReferentialActionFlags int

	// StringFilterFlags represents a set of string filters (deprecated - use []StringFilter)
	StringFilterFlags int

	// IndexAlgorithmFlags represents a set of index algorithms
	IndexAlgorithmFlags int

	// DatasourceConnectorData represents datasource connector-specific data
	DatasourceConnectorData struct{}

	// Note: StringFilter is defined in lib.go
	// Note: CompletionList and SchemaPosition are defined in completions.go
)

// Constants
const (
	// Referential action flags
	ReferentialActionDefaultSet ReferentialActionFlags = 1 << iota
	ReferentialActionNoActionFlag
	ReferentialActionRestrictFlag
	ReferentialActionCascadeFlag
	ReferentialActionSetNullFlag
	ReferentialActionSetDefaultFlag

	// String filter flags
	StringFilterAll StringFilterFlags = 0xFFFFFFFF

	// Index algorithm flags
	IndexAlgorithmBTreeFlag IndexAlgorithmFlags = 1 << iota
	IndexAlgorithmHashFlag
	IndexAlgorithmGistFlag
	IndexAlgorithmGinFlag
	IndexAlgorithmBrinFlag
)

// Helper functions for flag operations
func (flags RelationModeFlags) Contains(mode RelationMode) bool {
	// This is a simplified implementation
	return flags != 0
}

func (flags ReferentialActionFlags) Contains(action ReferentialAction) bool {
	// This is a simplified implementation
	return flags != 0
}

func (flags ConnectorCapabilities) Contains(capability ConnectorCapability) bool {
	return flags&(1<<uint(capability)) != 0
}

func (flags IndexAlgorithmFlags) Contains(algo IndexAlgorithm) bool {
	return flags&(1<<uint(algo)) != 0
}

// RelationModeAllowedEmulatedReferentialActionsDefault returns default emulated referential actions.
func RelationModeAllowedEmulatedReferentialActionsDefault() ReferentialActionFlags {
	return ReferentialActionDefaultSet | ReferentialActionNoActionFlag | ReferentialActionRestrictFlag | ReferentialActionCascadeFlag | ReferentialActionSetNullFlag | ReferentialActionSetDefaultFlag
}

// Type aliases for connector interface compatibility
type ScalarFieldType = database.ScalarFieldType
type Span = diagnostics.Span
type Diagnostics = diagnostics.Diagnostics
type EnumWalker = database.EnumWalker
type ModelWalker = database.ModelWalker
type RelationFieldWalker = database.RelationFieldWalker

// NativeTypeConstructor represents a native type constructor with its metadata.
type NativeTypeConstructor struct {
	Name                 string
	NumberOfArgs         int
	NumberOfOptionalArgs int
	AllowedTypes         []AllowedType
}

// AllowedType represents a scalar type that a native type is compatible with.
type AllowedType struct {
	FieldType         *ScalarFieldType
	ExpectedArguments []string
}

type NativeTypeErrorFactory = diagnostics.NativeTypeErrorFactory
