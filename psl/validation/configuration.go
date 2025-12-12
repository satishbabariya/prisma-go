// Package pslcore provides configuration parsing functionality.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// extractConfiguration extracts datasource and generator configuration from the AST.
func extractConfiguration(ast *ast.SchemaAst, diags *diagnostics.Diagnostics) Configuration {
	config := Configuration{
		Datasources: []Datasource{},
		Generators:  []Generator{},
		Warnings:    []diagnostics.DatamodelWarning{},
	}

	// First extract datasources to determine provider for feature map
	for _, top := range ast.Tops {
		if top == nil {
			continue
		}

		if source := top.AsSource(); source != nil {
			ds := extractDatasource(source, diags)
			if ds != nil {
				config.Datasources = append(config.Datasources, *ds)
			}
		}
	}

	// Determine provider for feature map (use first datasource provider if available)
	var provider *string
	if len(config.Datasources) > 0 && config.Datasources[0].Provider != "" {
		provider = &config.Datasources[0].Provider
	}

	// Create feature map with provider
	featureMap := NewFeatureMapWithProvider(provider)

	// Extract generators using LoadGeneratorsFromAST (which handles preview features properly)
	config.Generators = LoadGeneratorsFromAST(ast, diags, featureMap)

	return config
}

// extractDatasource extracts datasource configuration from a source AST node.
func extractDatasource(source *ast.SourceConfig, diags *diagnostics.Diagnostics) *Datasource {
	ds := &Datasource{
		Name:           source.Name.Name,
		ActiveProvider: source.Name.Name,
		Schemas:        []string{},
		Span:           source.Span(),
	}

	// Extract properties
	for _, prop := range source.Properties {
		switch prop.Name.Name {
		case "provider":
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				ds.Provider = strLit.Value
				ds.ActiveProvider = strLit.Value
				ds.ProviderSpan = prop.Span()
			} else {
				diags.PushError(diagnostics.NewValidationError(
					"Provider must be a string literal.",
					prop.Name.Span(),
				))
				ds.ProviderSpan = prop.Span()
			}
		case "url":
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				url := strLit.Value
				ds.URL = &url
			} else if envVar, ok := prop.Value.(ast.FunctionCall); ok {
				if envVar.Name.Name == "env" && len(envVar.Arguments) > 0 {
					// Handle env("VAR_NAME")
					if strLit, ok := envVar.Arguments[0].(ast.StringLiteral); ok {
						url := fmt.Sprintf("env:%s", strLit.Value)
						ds.URL = &url
					}
				}
			}
		case "directUrl":
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				url := strLit.Value
				ds.DirectURL = &url
			} else if envVar, ok := prop.Value.(ast.FunctionCall); ok {
				if envVar.Name.Name == "env" && len(envVar.Arguments) > 0 {
					if strLit, ok := envVar.Arguments[0].(ast.StringLiteral); ok {
						url := fmt.Sprintf("env:%s", strLit.Value)
						ds.DirectURL = &url
					}
				}
			}
		case "shadowDatabaseUrl":
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				url := strLit.Value
				ds.ShadowDatabase = &url
			} else if envVar, ok := prop.Value.(ast.FunctionCall); ok {
				if envVar.Name.Name == "env" && len(envVar.Arguments) > 0 {
					if strLit, ok := envVar.Arguments[0].(ast.StringLiteral); ok {
						url := fmt.Sprintf("env:%s", strLit.Value)
						ds.ShadowDatabase = &url
					}
				}
			}
		case "schemas":
			schemasSpan := prop.Span()
			ds.SchemasSpan = &schemasSpan
			if arr, ok := prop.Value.(ast.ArrayLiteral); ok {
				for _, elem := range arr.Elements {
					if strLit, ok := elem.(ast.StringLiteral); ok {
						ds.Schemas = append(ds.Schemas, strLit.Value)
					}
				}
			}
		case "relationMode":
			// Extract relation mode if present
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				ds.relationMode = RelationMode(strLit.Value)
			} else if ident, ok := prop.Value.(ast.Identifier); ok {
				ds.relationMode = RelationMode(ident.Name)
			}
		}
	}

	// Validate required fields
	if ds.Provider == "" {
		diags.PushError(diagnostics.NewValidationError(
			"Datasource must have a provider.",
			source.Name.Span(),
		))
		return nil
	}

	return ds
}

// extractGenerator extracts generator configuration from a generator AST node.
func extractGenerator(generator *ast.GeneratorConfig, diags *diagnostics.Diagnostics) *Generator {
	gen := &Generator{
		Name:   generator.Name.Name,
		Config: make(map[string]interface{}),
		Span:   generator.Span(),
	}

	// Extract properties
	for _, prop := range generator.Properties {
		switch prop.Name.Name {
		case "provider":
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				gen.Provider = strLit.Value
			} else {
				diags.PushError(diagnostics.NewValidationError(
					"Provider must be a string literal.",
					prop.Name.Span(),
				))
			}
		case "output":
			if strLit, ok := prop.Value.(ast.StringLiteral); ok {
				output := strLit.Value
				gen.Output = &output
			} else if envVar, ok := prop.Value.(ast.FunctionCall); ok {
				if envVar.Name.Name == "env" && len(envVar.Arguments) > 0 {
					if strLit, ok := envVar.Arguments[0].(ast.StringLiteral); ok {
						output := fmt.Sprintf("env:%s", strLit.Value)
						gen.Output = &output
					}
				}
			}
		case "binaryTargets":
			// Handle binary targets array
			if arr, ok := prop.Value.(ast.ArrayLiteral); ok {
				targets := []string{}
				for _, elem := range arr.Elements {
					if strLit, ok := elem.(ast.StringLiteral); ok {
						targets = append(targets, strLit.Value)
					}
				}
				gen.Config["binaryTargets"] = targets
			}
		default:
			// Store other properties in Config map
			gen.Config[prop.Name.Name] = extractConfigValue(prop.Value)
		}
	}

	// Validate required fields
	if gen.Provider == "" {
		diags.PushError(diagnostics.NewValidationError(
			"Generator must have a provider.",
			generator.Name.Span(),
		))
		return nil
	}

	return gen
}

// extractConfigValue extracts a value from an AST expression for config storage.
func extractConfigValue(value ast.Expression) interface{} {
	switch v := value.(type) {
	case ast.StringLiteral:
		return v.Value
	case ast.BooleanLiteral:
		return v.Value
	case ast.IntLiteral:
		return v.Value
	case ast.FloatLiteral:
		return v.Value
	case ast.ArrayLiteral:
		result := []interface{}{}
		for _, elem := range v.Elements {
			result = append(result, extractConfigValue(elem))
		}
		return result
	case ast.FunctionCall:
		if v.Name.Name == "env" && len(v.Arguments) > 0 {
			if strLit, ok := v.Arguments[0].(ast.StringLiteral); ok {
				return fmt.Sprintf("env:%s", strLit.Value)
			}
		}
		return v.Name.Name // Fallback to function name
	default:
		return nil
	}
}

// ExtractPreviewFeatures extracts and merges preview features from all generators.
func ExtractPreviewFeatures(config Configuration) PreviewFeatures {
	result := EmptyPreviewFeatures()

	// Merge preview features from all generators
	for _, gen := range config.Generators {
		if gen.PreviewFeatures != nil {
			result = result.Union(*gen.PreviewFeatures)
		}
	}

	return result
}

// ValidateDatasources validates datasource configuration.
func ValidateDatasources(config Configuration, diags *diagnostics.Diagnostics) {
	if len(config.Datasources) == 0 {
		diags.PushError(diagnostics.NewValidationError(
			"A datasource must be defined.",
			diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		))
		return
	}

	if len(config.Datasources) > 1 {
		diags.PushError(diagnostics.NewValidationError(
			"You defined more than one datasource. This is not allowed yet because support for multiple databases has not been implemented yet.",
			diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		))
	}

	// Validate each datasource
	for _, ds := range config.Datasources {
		if ds.Provider == "" {
			diags.PushError(diagnostics.NewValidationError(
				fmt.Sprintf("Datasource '%s' must have a provider.", ds.Name),
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}

		// Validate provider is known
		knownProviders := []string{"postgresql", "mysql", "sqlite", "sqlserver", "cockroachdb", "mongodb"}
		isKnown := false
		for _, known := range knownProviders {
			if strings.ToLower(ds.Provider) == known {
				isKnown = true
				break
			}
		}
		if !isKnown {
			diags.PushWarning(diagnostics.NewDatamodelWarning(
				fmt.Sprintf("Unknown datasource provider '%s'.", ds.Provider),
				diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
			))
		}
	}
}
