package diagnostics

import (
	"fmt"
	"io"
	"strings"
)

// DatamodelWarning represents a non-fatal warning emitted by the schema parser.
type DatamodelWarning struct {
	message string
	span    Span
}

// NewDatamodelWarning creates a new DatamodelWarning with the given message and span.
func NewDatamodelWarning(message string, span Span) DatamodelWarning {
	return DatamodelWarning{
		message: message,
		span:    span,
	}
}

// NewPreviewFeatureDeprecatedWarning creates a warning for deprecated preview features.
func NewPreviewFeatureDeprecatedWarning(feature string, span Span) DatamodelWarning {
	message := fmt.Sprintf("Preview feature \"%s\" is deprecated. It will be removed in a future version of Prisma.", feature)
	return NewDatamodelWarning(message, span)
}

// NewPreviewFeatureIsStabilizedWarning creates a warning for stabilized preview features.
func NewPreviewFeatureIsStabilizedWarning(feature string, span Span) DatamodelWarning {
	message := fmt.Sprintf("Preview feature \"%s\" is deprecated. The functionality can be used without specifying it as a preview feature.", feature)
	return NewDatamodelWarning(message, span)
}

// NewPreviewFeatureRenamedWarning creates a warning for renamed preview features.
func NewPreviewFeatureRenamedWarning(deprecatedFeature, renamedFeature, prislyLinkEndpoint string, span Span) DatamodelWarning {
	message := fmt.Sprintf("Preview feature \"%s\" has been renamed to \"%s\". Learn more at https://pris.ly/d/%s.", deprecatedFeature, renamedFeature, prislyLinkEndpoint)
	return NewDatamodelWarning(message, span)
}

// NewPreviewFeatureRenamedForProviderWarning creates a warning for provider-specific renamed preview features.
func NewPreviewFeatureRenamedForProviderWarning(provider, deprecatedFeature, renamedFeature, prislyLinkEndpoint string, span Span) DatamodelWarning {
	message := fmt.Sprintf("On `provider = \"%s\"`, preview feature \"%s\" has been renamed to \"%s\". Learn more at https://pris.ly/d/%s.", provider, deprecatedFeature, renamedFeature, prislyLinkEndpoint)
	return NewDatamodelWarning(message, span)
}

// NewReferentialIntegrityAttrDeprecationWarning creates a warning for deprecated referentialIntegrity attribute.
func NewReferentialIntegrityAttrDeprecationWarning(span Span) DatamodelWarning {
	message := "The `referentialIntegrity` attribute is deprecated. Please use `relationMode` instead. Learn more at https://pris.ly/d/relation-mode"
	return NewDatamodelWarning(message, span)
}

// NewMissingIndexOnEmulatedRelationWarning creates a warning for missing indexes on emulated relations.
func NewMissingIndexOnEmulatedRelationWarning(span Span) DatamodelWarning {
	message := strings.TrimSpace(`
With relationMode = "prisma", no foreign keys are used, so relation fields will not benefit from the index usually created by the relational database under the hood.
This can lead to poor performance when querying these fields. We recommend adding an index manually.
Learn more at https://pris.ly/d/relation-mode-prisma-indexes
`)
	return NewDatamodelWarning(message, span)
}

// NewNamedEnvValWarning creates a warning for named env function arguments.
func NewNamedEnvValWarning(span Span) DatamodelWarning {
	message := "The env function doesn't expect named arguments"
	return NewDatamodelWarning(message, span)
}

// NewFieldValidationWarning creates a warning for field validation issues.
func NewFieldValidationWarning(message, model, field string, span Span) DatamodelWarning {
	msg := fmt.Sprintf("Warning validating field `%s` in model `%s`: %s", field, model, message)
	return NewDatamodelWarning(msg, span)
}

// Message returns the warning message.
func (w DatamodelWarning) Message() string {
	return w.message
}

// Span returns the span of the warning.
func (w DatamodelWarning) Span() Span {
	return w.span
}

// PrettyPrint writes a pretty-printed representation of the warning to the writer.
func (w DatamodelWarning) PrettyPrint(writer io.Writer, fileName, text string) error {
	return PrettyPrint(writer, fileName, text, w.span, w.message, WarningColorer{})
}

