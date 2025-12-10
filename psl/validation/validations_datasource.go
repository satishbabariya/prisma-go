// Package pslcore provides datasource validation functionality.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// validateDatasource performs datasource-specific validations.
func validateDatasource(ctx *ValidationContext) {
	if ctx.Datasource == nil {
		return
	}

	// Validate schemas property with connector support
	schemasPropertyWithNoConnectorSupport(ctx.Datasource, ctx)

	// Call connector-specific datasource validation
	if ctx.Connector != nil {
		ctx.Connector.ValidateDatasource(ctx.PreviewFeatures, ctx.Datasource, ctx.Diagnostics)
	}
}

// schemasPropertyWithNoConnectorSupport validates that the schemas property
// is only used when the connector supports MultiSchema capability.
func schemasPropertyWithNoConnectorSupport(datasource *Datasource, ctx *ValidationContext) {
	// If connector supports MultiSchema, schemas property is allowed
	if ctx.HasCapability(ConnectorCapabilityMultiSchema) {
		return
	}

	// Check if schemas property is defined
	// TODO: When Datasource struct has SchemasSpan field, use it
	// For now, check if schemas array is non-empty
	if len(datasource.Schemas) > 0 {
		// If schemas are defined but connector doesn't support MultiSchema, error
		// We need to find the span of the schemas property
		// For now, use a generic span
		ctx.PushError(diagnostics.NewDatamodelError(
			"The `schemas` property is not supported on the current connector.",
			diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		))
	}
}
