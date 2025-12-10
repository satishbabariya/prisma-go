// Package pslcore provides datasource loading and validation functionality.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

const (
	previewFeaturesKey      = "previewFeatures"
	schemasKey              = "schemas"
	shadowDatabaseURLKey    = "shadowDatabaseUrl"
	urlKey                  = "url"
	directURLKey            = "directUrl"
	providerKey             = "provider"
	relationModeKey         = "relationMode"
	referentialIntegrityKey = "referentialIntegrity"
)

// LoadDatasourcesFromAST loads all datasources from the provided schema AST.
func LoadDatasourcesFromAST(
	astSchema *ast.SchemaAst,
	diags *diagnostics.Diagnostics,
	connectors *ConnectorRegistry,
) []Datasource {
	var sources []Datasource

	// Extract all source configs from AST
	for _, top := range astSchema.Tops {
		if source := top.AsSource(); source != nil {
			if ds := liftDatasource(source, diags, connectors); ds != nil {
				sources = append(sources, *ds)
			}
		}
	}

	// Validate that only one datasource is defined
	if len(sources) > 1 {
		for _, top := range astSchema.Tops {
			if source := top.AsSource(); source != nil {
				diags.PushError(diagnostics.NewSourceValidationError(
					"You defined more than one datasource. This is not allowed yet because support for multiple databases has not been implemented yet.",
					source.Name.Name,
					source.Span(),
				))
			}
		}
	}

	return sources
}

// liftDatasource lifts a datasource from the AST to a Datasource configuration.
func liftDatasource(
	astSource *ast.SourceConfig,
	diags *diagnostics.Diagnostics,
	connectors *ConnectorRegistry,
) *Datasource {
	sourceName := astSource.Name.Name

	// Extract properties into a map
	args := make(map[string]*ast.ConfigBlockProperty)
	for i := range astSource.Properties {
		prop := &astSource.Properties[i]
		if prop.Value != nil {
			args[prop.Name.Name] = prop
		} else {
			diags.PushError(diagnostics.NewSourceValidationError(
				fmt.Sprintf("Property \"%s\" is missing a value.", prop.Name.Name),
				sourceName,
				astSource.Span(),
			))
			return nil
		}
	}

	// Extract provider
	providerProp, hasProvider := args[providerKey]
	if !hasProvider {
		diags.PushError(diagnostics.NewSourceArgumentNotFoundError(
			providerKey,
			sourceName,
			astSource.Span(),
		))
		return nil
	}

	// Validate provider is not using env()
	// TODO: Check if expression is env() when expression types are available
	provider := ""
	if strLit, ok := providerProp.Value.(ast.StringLiteral); ok {
		provider = strLit.Value
	} else {
		diags.PushError(diagnostics.NewFunctionalEvaluationError(
			"A datasource must not use the env() function in the provider argument.",
			astSource.Span(),
		))
		return nil
	}

	if provider == "" {
		diags.PushError(diagnostics.NewSourceValidationError(
			"The provider argument in a datasource must not be empty",
			sourceName,
			providerProp.Span,
		))
		return nil
	}

	// Find connector
	connector := connectors.GetConnector(provider)
	if connector == nil {
		diags.PushError(diagnostics.NewDatasourceProviderNotKnownError(
			provider,
			providerProp.Span,
		))
		return nil
	}

	// Extract relation mode - use ExtendedConnector methods directly
	relationMode := getRelationMode(args, astSource, diags, connector)

	// Extract schemas
	schemas, schemasSpan := extractSchemas(args, astSource, diags, connector)

	// Validate unknown properties
	for key, prop := range args {
		// Skip known properties
		if key == providerKey || key == relationModeKey || key == referentialIntegrityKey || key == schemasKey {
			continue
		}

		// Check for deprecated/removed properties
		if key == urlKey {
			diags.PushError(diagnostics.NewDatasourceURLRemovedError(prop.Span))
			continue
		}
		if key == shadowDatabaseURLKey {
			diags.PushError(diagnostics.NewDatasourceShadowDatabaseURLRemovedError(prop.Span))
			continue
		}
		if key == directURLKey {
			diags.PushError(diagnostics.NewDatasourceDirectURLRemovedError(prop.Span))
			continue
		}
		if key == previewFeaturesKey {
			diags.PushError(diagnostics.NewDatamodelError(
				"Preview features are only supported in the generator block. Please move this field to the generator block.",
				prop.Span,
			))
			continue
		}

		diags.PushError(diagnostics.NewPropertyNotKnownError(key, prop.Span))
	}

	// Extract documentation
	// TODO: Extract documentation when available

	ds := &Datasource{
		Name:           sourceName,
		Provider:       provider,
		ActiveProvider: connector.ProviderName(),
		Schemas:        schemas,
		relationMode:   relationMode,
	}

	// Store schemas span if available
	if schemasSpan != nil {
		// TODO: Store schemas span when Datasource struct supports it
		_ = schemasSpan
	}

	return ds
}

// getRelationMode extracts and validates the relation mode from datasource arguments.
func getRelationMode(
	args map[string]*ast.ConfigBlockProperty,
	source *ast.SourceConfig,
	diags *diagnostics.Diagnostics,
	connector Connector,
) RelationMode {
	// Check for deprecated referentialIntegrity
	if _, hasReferentialIntegrity := args[referentialIntegrityKey]; hasReferentialIntegrity {
		prop := args[referentialIntegrityKey]
		diags.PushWarning(diagnostics.NewReferentialIntegrityAttrDeprecationWarning(prop.Span))
	}

	// Check for relationMode or referentialIntegrity
	relationModeProp, hasRelationMode := args[relationModeKey]
	referentialIntegrityProp, hasReferentialIntegrity := args[referentialIntegrityKey]

	if !hasRelationMode && !hasReferentialIntegrity {
		// Default relation mode from connector
		if connector != nil {
			return connector.DefaultRelationMode()
		}
		return RelationModePrisma
	}

	if hasRelationMode && hasReferentialIntegrity {
		// Both are present - error
		diags.PushError(diagnostics.NewReferentialIntegrityAndRelationModeCooccurError(
			referentialIntegrityProp.Span,
		))
		return RelationModePrisma
	}

	// Extract relation mode value
	var modeStr string
	if hasRelationMode {
		if strLit, ok := relationModeProp.Value.(ast.StringLiteral); ok {
			modeStr = strLit.Value
		} else {
			diags.PushError(diagnostics.NewSourceValidationError(
				"The relationMode argument must be a string literal",
				source.Name.Name,
				relationModeProp.Span,
			))
			return RelationModePrisma
		}
	} else if hasReferentialIntegrity {
		if strLit, ok := referentialIntegrityProp.Value.(ast.StringLiteral); ok {
			modeStr = strLit.Value
		} else {
			diags.PushError(diagnostics.NewSourceValidationError(
				"The referentialIntegrity argument must be a string literal",
				source.Name.Name,
				referentialIntegrityProp.Span,
			))
			return RelationModePrisma
		}
	}

	// Map string to RelationMode
	var relationMode RelationMode
	switch modeStr {
	case "prisma":
		relationMode = RelationModePrisma
	case "foreignKeys":
		relationMode = RelationModeForeignKeys
	default:
		diags.PushError(diagnostics.NewSourceValidationError(
			fmt.Sprintf("Invalid relation mode setting: \"%s\". Supported values: \"prisma\", \"foreignKeys\"", modeStr),
			"relationMode",
			source.Span(),
		))
		return RelationModePrisma
	}

	// Validate connector supports this relation mode
	allowedModes := connector.AllowedRelationModeSettings()
	supported := false
	for _, mode := range allowedModes {
		if mode == relationMode {
			supported = true
			break
		}
	}
	if !supported {
		supportedStr := ""
		for i, mode := range allowedModes {
			if i > 0 {
				supportedStr += ", "
			}
			supportedStr += fmt.Sprintf("\"%s\"", string(mode))
		}
		diags.PushError(diagnostics.NewSourceValidationError(
			fmt.Sprintf("Invalid relation mode setting: \"%s\". Supported values: %s", modeStr, supportedStr),
			"relationMode",
			relationModeProp.Span,
		))
		return RelationModePrisma
	}

	return relationMode
}

// extractSchemas extracts and validates the schemas property.
func extractSchemas(
	args map[string]*ast.ConfigBlockProperty,
	source *ast.SourceConfig,
	diags *diagnostics.Diagnostics,
	connector ExtendedConnector,
) ([]string, *diagnostics.Span) {
	schemasProp, hasSchemas := args[schemasKey]
	if !hasSchemas {
		return nil, nil
	}

	// Check connector capability
	if !connector.HasCapability(ConnectorCapabilityMultiSchema) {
		diags.PushError(diagnostics.NewDatamodelError(
			"The `schemas` property is not supported on the current connector.",
			schemasProp.Span,
		))
		return nil, nil
	}

	// Extract array of strings
	schemas := []string{}
	if arr, ok := schemasProp.Value.(ast.ArrayLiteral); ok {
		for _, elem := range arr.Elements {
			if strLit, ok := elem.(ast.StringLiteral); ok {
				schemas = append(schemas, strLit.Value)
			}
		}
	}

	schemasSpan := &schemasProp.Span

	if len(schemas) == 0 {
		diags.PushError(diagnostics.NewSchemasArrayEmptyError(schemasProp.Span))
		return nil, nil
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, schema := range schemas {
		if seen[schema] {
			diags.PushError(diagnostics.NewDatamodelError(
				"Duplicated schema names are not allowed",
				schemasProp.Span,
			))
			return nil, nil
		}
		seen[schema] = true
	}

	return schemas, schemasSpan
}
