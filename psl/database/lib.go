// Package parserdatabase provides schema parsing and validation functionality.
package database

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/core"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// ParserDatabase is a container for a Schema AST, together with information
// gathered during schema validation. Each validation step enriches the
// database with information that can be used to work with the schema.
type ParserDatabase struct {
	asts              Files
	interner          *StringInterner
	names             Names
	types             Types
	relations         Relations
	extensionMetadata ExtensionMetadata
}

// Files represents a collection of parsed schema files.
type Files struct {
	files []FileEntry
}

// NewFiles creates a new Files instance from multiple source files.
func NewFiles(files []core.SourceFile, diags *diagnostics.Diagnostics) Files {
	entries := make([]FileEntry, len(files))
	for i, source := range files {
		fileID := diagnostics.FileID(i)
		ast, parseDiags := parsing.ParseSchemaFromSourceFile(source)

		// Merge diagnostics
		for _, err := range parseDiags.Errors() {
			diags.PushError(err)
		}
		for _, warn := range parseDiags.Warnings() {
			diags.PushWarning(warn)
		}

		entries[i] = FileEntry{
			FileID: fileID,
			Name:   source.Path,
			Source: source,
			AST:    *ast,
		}
	}
	return Files{files: entries}
}

// Len returns the number of files.
func (f Files) Len() int {
	return len(f.files)
}

// Get returns the file entry for the given file ID.
func (f Files) Get(fileID diagnostics.FileID) *FileEntry {
	if int(fileID) < len(f.files) {
		return &f.files[fileID]
	}
	return nil
}

// Iter returns an iterator over all file entries.
func (f Files) Iter() []FileEntry {
	return f.files
}

// RenderDiagnostics renders the given diagnostics into a string.
// This method is multi-file aware.
func (f Files) RenderDiagnostics(diags *diagnostics.Diagnostics) string {
	var result strings.Builder

	for _, err := range diags.Errors() {
		fileEntry := f.Get(err.Span().FileID)
		if fileEntry != nil {
			result.WriteString(diags.ToPrettyString(fileEntry.Name, fileEntry.Source.Data))
		}
	}

	return result.String()
}

// FileEntry represents a single parsed file.
type FileEntry struct {
	FileID diagnostics.FileID
	Name   string
	Source core.SourceFile
	AST    ast.SchemaAst
}

// ExtensionTypes defines the interface for extension types.
type ExtensionTypes interface {
	Enumerate() []ExtensionTypeEntry
}

// ExtensionTypeEntry represents an extension type entry.
type ExtensionTypeEntry struct {
	ID              ExtensionTypeId
	PrismaName      string
	DbName          string
	DbTypeModifiers []string
}

// ExtensionTypeId represents an extension type identifier.
type ExtensionTypeId uint32

// NoExtensionTypes provides an empty implementation of ExtensionTypes.
type NoExtensionTypes struct{}

// Enumerate returns an empty slice.
func (n NoExtensionTypes) Enumerate() []ExtensionTypeEntry {
	return []ExtensionTypeEntry{}
}

// ExtensionMetadata holds metadata about extension types.
type ExtensionMetadata struct {
	idToPrismaName          map[ExtensionTypeId]StringId
	idToDbNameWithModifiers map[ExtensionTypeId]DbNameWithModifiers
}

// NewExtensionMetadata creates a new ExtensionMetadata from extension types.
func NewExtensionMetadata(extensionTypes ExtensionTypes, interner *StringInterner) ExtensionMetadata {
	idToPrismaName := make(map[ExtensionTypeId]StringId)
	idToDbNameWithModifiers := make(map[ExtensionTypeId]DbNameWithModifiers)

	for _, entry := range extensionTypes.Enumerate() {
		prismaNameID := interner.Intern(entry.PrismaName)
		idToPrismaName[entry.ID] = prismaNameID

		if len(entry.DbTypeModifiers) > 0 {
			dbNameID := interner.Intern(entry.DbName)
			idToDbNameWithModifiers[entry.ID] = DbNameWithModifiers{
				Name:      interner.Get(dbNameID),
				Modifiers: entry.DbTypeModifiers,
			}
		}
	}

	return ExtensionMetadata{
		idToPrismaName:          idToPrismaName,
		idToDbNameWithModifiers: idToDbNameWithModifiers,
	}
}

// GetExtensionTypePrismaName returns the Prisma name for an extension type ID.
func (pd *ParserDatabase) GetExtensionTypePrismaName(id ExtensionTypeId) string {
	stringID, ok := pd.extensionMetadata.idToPrismaName[id]
	if !ok {
		return ""
	}
	return pd.interner.Get(stringID)
}

// GetExtensionTypeDbNameWithModifiers returns the database name and modifiers for an extension type ID.
func (pd *ParserDatabase) GetExtensionTypeDbNameWithModifiers(id ExtensionTypeId) (string, []string) {
	dbNameWithModifiers, ok := pd.extensionMetadata.idToDbNameWithModifiers[id]
	if !ok {
		return "", nil
	}
	return dbNameWithModifiers.Name, dbNameWithModifiers.Modifiers
}

// DbNameWithModifiers holds a database name with modifiers.
type DbNameWithModifiers struct {
	Name      string
	Modifiers []string
}

// NewParserDatabase creates a new ParserDatabase from schema files.
func NewParserDatabase(
	schemas []core.SourceFile,
	diags *diagnostics.Diagnostics,
	extensionTypes ExtensionTypes,
) *ParserDatabase {
	files := NewFiles(schemas, diags)
	interner := NewStringInterner()
	extensionMetadata := NewExtensionMetadata(extensionTypes, interner)

	db := &ParserDatabase{
		asts:              files,
		interner:          interner,
		names:             NewNames(),
		types:             NewTypes(),
		relations:         NewRelations(),
		extensionMetadata: extensionMetadata,
	}

	// First pass: resolve names
	db.names = ResolveNames(db, diags)

	// Second pass: resolve types
	ctx := NewContext(
		&db.asts,
		db.interner,
		&db.names,
		&db.types,
		&db.relations,
		diags,
		extensionTypes,
	)
	ResolveTypes(ctx)

	// Third pass: resolve attributes
	ResolveAttributes(ctx)

	// Validate @id field arities (must be called after all attributes are resolved)
	for modelID, modelAttrs := range db.types.ModelAttributes {
		ValidateIdFieldArities(modelID, &modelAttrs, ctx)
		ValidateShardKeyFieldArities(modelID, &modelAttrs, ctx)
	}

	// Fourth pass: infer relations
	InferRelations(ctx)

	// Perform basic validation
	db.validateBasic(diags)

	return db
}

// validateBasic performs basic validation on the parsed schema.
func (pd *ParserDatabase) validateBasic(diags *diagnostics.Diagnostics) {
	// Check for duplicate top-level names
	nameCounts := make(map[string]int)

	for _, file := range pd.asts.files {
		for _, top := range file.AST.Tops {
			var name string
			var span diagnostics.Span

			switch {
			case top.AsModel() != nil:
				model := top.AsModel()
				name = model.Name.Name
				span = model.Name.Span()

				// Validate model fields
				pd.validateModel(model, diags)
			case top.AsEnum() != nil:
				enum := top.AsEnum()
				name = enum.Name.Name
				span = enum.Name.Span()

				// Validate enum values
				pd.validateEnum(enum, diags)
			case top.AsSource() != nil:
				name = top.AsSource().Name.Name
				span = top.AsSource().Name.Span()
			case top.AsGenerator() != nil:
				name = top.AsGenerator().Name.Name
				span = top.AsGenerator().Name.Span()
			}

			if name != "" {
				nameCounts[name]++
				if nameCounts[name] > 1 {
					diags.PushError(diagnostics.NewValidationError(
						fmt.Sprintf("Duplicate top-level name: %s", name),
						span,
					))
				}
			}
		}
	}
}

// validateModel performs validation on a model.
func (pd *ParserDatabase) validateModel(model *ast.Model, diags *diagnostics.Diagnostics) {
	fieldNames := make(map[string]bool)
	hasIdField := false

	for _, field := range model.Fields {
		// Check for duplicate field names
		if fieldNames[field.Name.Name] {
			diags.PushError(diagnostics.NewDuplicateFieldError(
				model.Name.Name,
				field.Name.Name,
				"model",
				field.Name.Span(),
			))
		}
		fieldNames[field.Name.Name] = true

		// Validate field type
		if field.FieldType.TypeName() == "" {
			diags.PushError(diagnostics.NewFieldValidationError(
				"Field type cannot be empty",
				"model",
				model.Name.Name,
				field.Name.Name,
				field.FieldType.Span(),
			))
		}

		// Check for @id attribute
		for _, attr := range field.Attributes {
			if attr.Name.Name == "id" {
				hasIdField = true
			}
		}

		// Models should have at least one @id field
		if !hasIdField && len(model.Fields) > 0 {
			diags.PushError(diagnostics.NewModelValidationError(
				"Model must have at least one field with @id attribute",
				"model",
				model.Name.Name,
				model.Span(),
			))
		}
	}
}

// validateEnum performs validation on an enum.
func (pd *ParserDatabase) validateEnum(enum *ast.Enum, diags *diagnostics.Diagnostics) {
	valueNames := make(map[string]bool)

	for _, value := range enum.Values {
		// Check for duplicate enum values
		if valueNames[value.Name.Name] {
			diags.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Duplicate enum value: %s", value.Name.Name),
				value.Name.Span(),
			))
		}
		valueNames[value.Name.Name] = true
	}
}

// NewSingleFile creates a ParserDatabase from a single schema file.
func NewSingleFile(
	file core.SourceFile,
	diagnostics *diagnostics.Diagnostics,
	extensionTypes ExtensionTypes,
) *ParserDatabase {
	return NewParserDatabase([]core.SourceFile{file}, diagnostics, extensionTypes)
}

// Ast returns the AST for the given file ID.
func (pd *ParserDatabase) Ast(fileID diagnostics.FileID) *ast.SchemaAst {
	for _, file := range pd.asts.files {
		if file.FileID == fileID {
			return &file.AST
		}
	}
	return nil
}

// AST returns the AST for the given file ID (exported version).
func (pd *ParserDatabase) AST(fileID diagnostics.FileID) *ast.SchemaAst {
	return pd.Ast(fileID)
}

// Source returns the source content for the given file ID.
func (pd *ParserDatabase) Source(fileID diagnostics.FileID) string {
	for _, file := range pd.asts.files {
		if file.FileID == fileID {
			return file.Source.Data
		}
	}
	return ""
}

// FileName returns the file name for the given file ID.
func (pd *ParserDatabase) FileName(fileID diagnostics.FileID) string {
	for _, file := range pd.asts.files {
		if file.FileID == fileID {
			return file.Name
		}
	}
	return ""
}

// FilesCount returns the total number of files.
func (pd *ParserDatabase) FilesCount() int {
	return len(pd.asts.files)
}

// ModelsCount returns the total number of models.
func (pd *ParserDatabase) ModelsCount() int {
	return len(pd.types.ModelAttributes)
}

// EnumsCount returns the total number of enums.
func (pd *ParserDatabase) EnumsCount() int {
	return len(pd.types.EnumAttributes)
}

// RenderDiagnostics renders the given diagnostics (warnings + errors) into a String.
// This method is multi-file aware.
func (pd *ParserDatabase) RenderDiagnostics(diags *diagnostics.Diagnostics) string {
	return pd.asts.RenderDiagnostics(diags)
}

// FileID returns the file ID for a given file name, or false if not found.
func (pd *ParserDatabase) FileID(fileName string) (diagnostics.FileID, bool) {
	for _, file := range pd.asts.files {
		if file.Name == fileName {
			return file.FileID, true
		}
	}
	return 0, false
}

// IterFileSources returns an iterator over all file sources and their paths.
func (pd *ParserDatabase) IterFileSources() []FileSource {
	result := make([]FileSource, len(pd.asts.files))
	for i, file := range pd.asts.files {
		result[i] = FileSource{
			Path:   file.Name,
			Source: file.Source,
		}
	}
	return result
}

// FileSource represents a file path and source file pair.
type FileSource struct {
	Path   string
	Source core.SourceFile
}

// getModelFromID is a helper method to get a model from its ID.
// This is used by walkers.
func (pd *ParserDatabase) getModelFromID(modelID ModelId) *ast.Model {
	file := pd.asts.Get(modelID.FileID)
	if file == nil {
		return nil
	}

	modelCount := 0
	for _, top := range file.AST.Tops {
		if model := top.AsModel(); model != nil {
			if uint32(modelCount) == modelID.ID {
				return model
			}
			modelCount++
		}
	}

	return nil
}

// GetString returns the string for the given StringId.
func (pd *ParserDatabase) GetString(id StringId) string {
	return pd.interner.Get(id)
}

// Datasources returns all datasource blocks from all ASTs in the schema.
func (pd *ParserDatabase) Datasources() []*ast.SourceConfig {
	var datasources []*ast.SourceConfig
	for _, file := range pd.asts.files {
		sources := file.AST.Sources()
		datasources = append(datasources, sources...)
	}
	return datasources
}

// Generators returns all generator blocks from all ASTs in the schema.
func (pd *ParserDatabase) Generators() []*ast.GeneratorConfig {
	var generators []*ast.GeneratorConfig
	for _, file := range pd.asts.files {
		gens := file.AST.Generators()
		generators = append(generators, gens...)
	}
	return generators
}

// IterASTs returns all parsed ASTs.
func (pd *ParserDatabase) IterASTs() []*ast.SchemaAst {
	asts := make([]*ast.SchemaAst, len(pd.asts.files))
	for i, file := range pd.asts.files {
		asts[i] = &file.AST
	}
	return asts
}

// IterSources returns all source file contents.
func (pd *ParserDatabase) IterSources() []string {
	sources := make([]string, len(pd.asts.files))
	for i, file := range pd.asts.files {
		sources[i] = file.Source.Data
	}
	return sources
}
