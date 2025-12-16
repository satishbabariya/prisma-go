// Package domain contains the core business entities and interfaces for the Generator domain.
package domain

import "context"

// GeneratedModel represents a model to be generated.
type GeneratedModel struct {
	Name      string
	Fields    []GeneratedField
	Methods   []GeneratedMethod
	Relations []GeneratedRelation
}

// GeneratedField represents a field in the generated model.
type GeneratedField struct {
	Name      string
	Type      string
	JSONTag   string
	DBTag     string
	IsPointer bool
	IsSlice   bool
}

// GeneratedMethod represents a method in the generated client.
type GeneratedMethod struct {
	Name     string
	Receiver string
	Params   []Param
	Returns  []Return
	Body     string
}

// Param represents a method parameter.
type Param struct {
	Name string
	Type string
}

// Return represents a method return value.
type Return struct {
	Type string
}

// GeneratedRelation represents a relation in the generated code.
type GeneratedRelation struct {
	Name  string
	Type  RelationType
	Model string
}

// RelationType represents the type of relation.
type RelationType string

const (
	// OneToOne represents a one-to-one relation.
	OneToOne RelationType = "OneToOne"
	// OneToMany represents a one-to-many relation.
	OneToMany RelationType = "OneToMany"
	// ManyToMany represents a many-to-many relation.
	ManyToMany RelationType = "ManyToMany"
)

// GeneratedFile represents a generated file.
type GeneratedFile struct {
	Path    string
	Content string
}

// AnalyzedSchema represents analyzed schema data.
type AnalyzedSchema struct {
	Models    []GeneratedModel
	Enums     []GeneratedEnum
	Relations []GeneratedRelation
	Config    GeneratorConfig
}

// GeneratedEnum represents a generated enum.
type GeneratedEnum struct {
	Name   string
	Values []string
}

// GeneratorConfig represents generator configuration.
type GeneratorConfig struct {
	Output  string
	Package string
}

// Analyzer defines the interface for schema analysis.
type Analyzer interface {
	// Analyze analyzes a schema and prepares it for code generation.
	Analyze(ctx context.Context, schema interface{}) (*AnalyzedSchema, error)

	// AnalyzeModel analyzes a single model.
	AnalyzeModel(ctx context.Context, model interface{}) (*GeneratedModel, error)
}

// TemplateEngine defines the interface for template rendering.
type TemplateEngine interface {
	// Render renders a template with data.
	Render(ctx context.Context, template string, data interface{}) (string, error)

	// RenderFile renders a template file with data.
	RenderFile(ctx context.Context, templatePath string, data interface{}) (string, error)
}

// CodeWriter defines the interface for code writing.
type CodeWriter interface {
	// Write writes multiple generated files.
	Write(ctx context.Context, files []GeneratedFile) error

	// WriteFile writes a single generated file.
	WriteFile(ctx context.Context, file GeneratedFile) error

	// Format formats Go code.
	Format(ctx context.Context, code string) (string, error)
}
