package core

import "github.com/satishbabariya/prisma-go/psl/diagnostics"

// RelationMode represents the relation mode.
type RelationMode string

const (
	RelationModePrisma      RelationMode = "prisma"
	RelationModeForeignKeys RelationMode = "foreignKeys"
)

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

// SetRelationMode sets the relation mode for this datasource.
func (d *Datasource) SetRelationMode(mode RelationMode) {
	d.relationMode = mode
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

// PreviewFeatures represents the preview features configured in a generator.
type PreviewFeatures struct {
	features map[string]bool
}

// NewPreviewFeatures creates a new PreviewFeatures instance.
func NewPreviewFeatures() *PreviewFeatures {
	return &PreviewFeatures{
		features: make(map[string]bool),
	}
}

// Has checks if a preview feature is enabled.
func (pf *PreviewFeatures) Has(feature string) bool {
	if pf == nil {
		return false
	}
	return pf.features[feature]
}

// Set enables a preview feature.
func (pf *PreviewFeatures) Set(feature string) {
	if pf != nil {
		pf.features[feature] = true
	}
}
