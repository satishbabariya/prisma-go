// Package pslcore provides core schema validation and configuration functionality.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/core"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"

	"github.com/satishbabariya/prisma-go/psl/formatting"
	"github.com/satishbabariya/prisma-go/psl/parsing"

	"github.com/satishbabariya/prisma-go/psl/database"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// ValidatedSchema represents a fully validated Prisma schema.
type ValidatedSchema struct {
	Configuration Configuration
	Db            *database.ParserDatabase
	Connector     Connector
	Diagnostics   diagnostics.Diagnostics
	RelationMode  RelationMode
}

// RenderOwnDiagnostics renders the diagnostics for this validated schema.
func (vs ValidatedSchema) RenderOwnDiagnostics() string {
	return vs.Db.RenderDiagnostics(&vs.Diagnostics)
}

// Configuration represents the parsed configuration from datasources and generators.
type Configuration struct {
	Datasources []Datasource
	Generators  []Generator
	Warnings    []diagnostics.DatamodelWarning
}

// Datasource represents a datasource configuration.
type Datasource struct {
	Name           string
	Provider       string
	URL            *string
	DirectURL      *string
	ShadowDatabase *string
	Schemas        []string
	ActiveProvider string
	relationMode   RelationMode
	Span           diagnostics.Span
	ProviderSpan   diagnostics.Span
	SchemasSpan    *diagnostics.Span
	Documentation  *string
}

// RelationMode returns the relation mode for this datasource.
func (d *Datasource) RelationMode() RelationMode {
	return d.relationMode
}

// Generator represents a generator configuration.
type Generator struct {
	Name            string
	Provider        string
	Output          *string
	BinaryTargets   []string
	PreviewFeatures *PreviewFeatures
	Config          map[string]interface{}
	Span            diagnostics.Span
}

// Connector represents a database connector interface.
type Connector interface {
	Name() string
	HasCapability(capability ConnectorCapability) bool
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
	ValidateEnum(enumWalker *EnumWalker, diags *diagnostics.Diagnostics)
	ValidateModel(modelWalker *ModelWalker, relationMode RelationMode, diags *diagnostics.Diagnostics)
	ValidateView(viewWalker *ModelWalker, diags *diagnostics.Diagnostics)
	ValidateRelationField(fieldWalker *RelationFieldWalker, diags *diagnostics.Diagnostics)
	ValidateDatasource(previewFeatures PreviewFeatures, datasource *Datasource, diags *diagnostics.Diagnostics)
	ValidateScalarFieldUnknownDefaultFunctions(db *database.ParserDatabase, diagnostics *diagnostics.Diagnostics)
	ConstraintViolationScopes() []ConstraintScope // Note: ExtendedConnector uses []ExtendedConstraintScope
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
	// DatamodelCompletions provides IDE/LSP autocomplete support for schema positions.
	DatamodelCompletions(db *database.ParserDatabase, position SchemaPosition, completions *CompletionList)
	// DatasourceCompletions provides IDE/LSP autocomplete support for datasource configuration.
	DatasourceCompletions(config Configuration, completions *CompletionList)
}

// RelationMode represents the relation mode.
type RelationMode string

const (
	RelationModePrisma      RelationMode = "prisma"
	RelationModeForeignKeys RelationMode = "foreignKeys"
)

// StringFilter represents available filters for a String scalar field.
type StringFilter int

const (
	// StringFilterContains represents a filter that checks if a string contains another string.
	StringFilterContains StringFilter = iota
	// StringFilterStartsWith represents a filter that checks if a string starts with another string.
	StringFilterStartsWith
	// StringFilterEndsWith represents a filter that checks if a string ends with another string.
	StringFilterEndsWith
)

// Name returns the property name of the filter in the client API (camelCase).
func (f StringFilter) Name() string {
	switch f {
	case StringFilterContains:
		return "contains"
	case StringFilterStartsWith:
		return "startsWith"
	case StringFilterEndsWith:
		return "endsWith"
	default:
		return ""
	}
}

// Validate validates a Prisma schema and returns a ValidatedSchema.
func Validate(
	file core.SourceFile,
	connectors []Connector,
	extensionTypes database.ExtensionTypes,
) ValidatedSchema {
	diags := diagnostics.NewDiagnostics()
	db := database.NewSingleFile(file, &diags, extensionTypes)

	// Parse the schema
	ast, parseDiags := parsing.ParseSchemaFromSourceFileV2(file)
	// Merge diagnostics
	for _, err := range parseDiags.Errors() {
		diags.PushError(err)
	}
	for _, warn := range parseDiags.Warnings() {
		diags.PushWarning(warn)
	}

	// Basic validation: check for required datasource
	hasDatasource := false
	for _, top := range ast.Tops {
		if top != nil {
			if _, ok := top.(*v2ast.SourceConfig); ok {
				hasDatasource = true
				break
			}
		}
	}

	if !hasDatasource {
		diags.PushError(diagnostics.NewValidationError("A datasource must be defined.", diagnostics.NewSpan(0, 0, diagnostics.FileIDZero)))
	}

	// Extract configuration from AST
	configuration := extractConfiguration(ast, &diags)

	// Validate configuration
	ValidateDatasources(configuration, &diags)

	// Run validation pipeline
	previewFeatures := ExtractPreviewFeatures(configuration)
	validateOutput := ValidateSchema(db, configuration.Datasources, previewFeatures, diags, extensionTypes)

	// Populate warnings from diagnostics
	configuration.Warnings = validateOutput.Diagnostics.Warnings()

	return ValidatedSchema{
		Configuration: configuration,
		Db:            validateOutput.Db,
		Connector:     validateOutput.Connector,
		Diagnostics:   validateOutput.Diagnostics,
		RelationMode:  validateOutput.RelationMode,
	}
}

// ParseSchemaWithoutValidation parses a schema without full validation.
func ParseSchemaWithoutValidation(
	file core.SourceFile,
	connectors []Connector,
	extensionTypes database.ExtensionTypes,
) ValidatedSchema {
	diags := diagnostics.NewDiagnostics()
	db := database.NewSingleFile(file, &diags, extensionTypes)

	// Parse the schema
	ast, parseDiags := parsing.ParseSchemaFromSourceFileV2(file)
	// Merge diagnostics
	for _, err := range parseDiags.Errors() {
		diags.PushError(err)
	}
	for _, warn := range parseDiags.Warnings() {
		diags.PushWarning(warn)
	}

	// Extract configuration from AST
	configuration := extractConfiguration(ast, &diags)

	// Parse without validation - just determine connector and relation mode
	var connector Connector
	var relationMode RelationMode = RelationModePrisma

	if len(configuration.Datasources) > 0 {
		ds := configuration.Datasources[0]
		// Find connector from registry
		for _, c := range connectors {
			if c != nil && c.IsProvider(ds.Provider) {
				connector = c
				break
			}
		}
		relationMode = ds.RelationMode()
	}

	return ValidatedSchema{
		Configuration: configuration,
		Db:            db,
		Connector:     connector,
		Diagnostics:   diags,
		RelationMode:  relationMode,
	}
}

// ValidateMultiFile validates multiple Prisma schema files.
func ValidateMultiFile(
	files []core.SourceFile,
	connectors []Connector,
	extensionTypes database.ExtensionTypes,
) ValidatedSchema {
	diags := diagnostics.NewDiagnostics()
	db := database.NewParserDatabase(files, &diags, extensionTypes)

	// Extract configuration from all files
	configuration := Configuration{
		Datasources: []Datasource{},
		Generators:  []Generator{},
		Warnings:    []diagnostics.DatamodelWarning{},
	}

	// Extract configuration from all ASTs
	for _, file := range files {
		ast, parseDiags := parsing.ParseSchemaFromSourceFileV2(file)
		// Merge diagnostics
		for _, err := range parseDiags.Errors() {
			diags.PushError(err)
		}
		for _, warn := range parseDiags.Warnings() {
			diags.PushWarning(warn)
		}
		fileConfig := extractConfiguration(ast, &diags)
		configuration.Datasources = append(configuration.Datasources, fileConfig.Datasources...)
		configuration.Generators = append(configuration.Generators, fileConfig.Generators...)
		configuration.Warnings = append(configuration.Warnings, fileConfig.Warnings...)
	}

	// Run validation pipeline
	previewFeatures := ExtractPreviewFeatures(configuration)
	validateOutput := ValidateSchema(db, configuration.Datasources, previewFeatures, diags, extensionTypes)

	// Populate warnings from diagnostics
	configuration.Warnings = append(configuration.Warnings, validateOutput.Diagnostics.Warnings()...)

	return ValidatedSchema{
		Configuration: configuration,
		Db:            validateOutput.Db,
		Connector:     validateOutput.Connector,
		Diagnostics:   validateOutput.Diagnostics,
		RelationMode:  validateOutput.RelationMode,
	}
}

// ParseConfiguration parses configuration blocks from a schema string.
func ParseConfiguration(
	schema string,
	connectors []Connector,
) (Configuration, error) {
	file := core.NewSourceFile("schema.prisma", schema)
	validated := Validate(file, connectors, database.NoExtensionTypes{})

	if validated.Diagnostics.HasErrors() {
		return Configuration{}, validated.Diagnostics.ToResult()
	}

	return validated.Configuration, nil
}

// ParseConfigurationMultiFile parses configuration blocks from multiple schema files.
func ParseConfigurationMultiFile(
	files []core.SourceFile,
	connectors []Connector,
) (database.Files, Configuration, error) {
	diags := diagnostics.NewDiagnostics()
	parsedFiles := database.NewFiles(files, &diags)

	configuration := Configuration{
		Datasources: []Datasource{},
		Generators:  []Generator{},
	}

	// Extract configuration from all ASTs
	for _, fileEntry := range parsedFiles.Iter() {
		fileConfig := extractConfiguration(&fileEntry.AST, &diags)
		configuration.Datasources = append(configuration.Datasources, fileConfig.Datasources...)
		configuration.Generators = append(configuration.Generators, fileConfig.Generators...)
	}

	// Validate datasources
	ValidateDatasources(configuration, &diags)

	if diags.HasErrors() {
		return parsedFiles, Configuration{}, diags.ToResult()
	}

	return parsedFiles, configuration, nil
}

// ErrorTolerantParseConfiguration parses configuration blocks from multiple files,
// collecting all errors instead of failing fast.
func ErrorTolerantParseConfiguration(
	files []core.SourceFile,
	connectors []Connector,
) (database.Files, Configuration, diagnostics.Diagnostics) {
	diags := diagnostics.NewDiagnostics()
	parsedFiles := database.NewFiles(files, &diags)

	configuration := Configuration{
		Datasources: []Datasource{},
		Generators:  []Generator{},
		Warnings:    []diagnostics.DatamodelWarning{},
	}

	// Extract configuration from all ASTs
	for _, fileEntry := range parsedFiles.Iter() {
		fileConfig := extractConfiguration(&fileEntry.AST, &diags)
		configuration.Datasources = append(configuration.Datasources, fileConfig.Datasources...)
		configuration.Generators = append(configuration.Generators, fileConfig.Generators...)
		configuration.Warnings = append(configuration.Warnings, fileConfig.Warnings...)
	}

	// Validate datasources (but don't fail - collect errors)
	ValidateDatasources(configuration, &diags)

	// Add warnings from diagnostics
	configuration.Warnings = append(configuration.Warnings, diags.Warnings()...)

	return parsedFiles, configuration, diags
}

// FileResult represents a reformatted file result.
type FileResult struct {
	FileName string
	Content  string
}

// Reformat reformats a Prisma schema string.
// indentWidth specifies the number of spaces for indentation (defaults to 2 if 0).
// This is a basic reformatting that preserves the schema structure without adding missing fields.
func Reformat(source string, indentWidth int) (string, error) {
	return formatting.Reformat(source, indentWidth)
}

// ReformatMultiple reformats multiple Prisma schema files.
// indentWidth specifies the number of spaces for indentation (defaults to 2 if 0).
// Returns a slice of FileResult containing the reformatted content for each file.
// This is a basic reformatting that preserves the schema structure without adding missing fields.
func ReformatMultiple(sources []core.SourceFile, indentWidth int) ([]FileResult, error) {
	var results []FileResult
	for _, source := range sources {
		formatted, err := formatting.Reformat(source.Data, indentWidth)
		if err != nil {
			return nil, err
		}
		results = append(results, FileResult{
			FileName: source.Path,
			Content:  formatted,
		})
	}
	return results, nil
}
