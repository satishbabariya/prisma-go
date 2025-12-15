// Package pslcore provides configuration parsing functionality.
package validation

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"

	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// extractConfiguration extracts datasource and generator configuration from the AST.
func extractConfiguration(astArg *v2ast.SchemaAst, diags *diagnostics.Diagnostics) Configuration {
	config := Configuration{
		Datasources: []Datasource{},
		Generators:  []Generator{},
		Warnings:    []diagnostics.DatamodelWarning{},
	}

	// First extract datasources to determine provider for feature map
	for _, top := range astArg.Tops {
		if top == nil {
			continue
		}

		if source, ok := top.(*v2ast.SourceConfig); ok {
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
	config.Generators = LoadGeneratorsFromAST(astArg, diags, featureMap)

	return config
}

// extractDatasource extracts datasource configuration from a source AST node.
func extractDatasource(source *v2ast.SourceConfig, diags *diagnostics.Diagnostics) *Datasource {
	ds := &Datasource{
		Name:           source.GetName(),
		ActiveProvider: source.GetName(),
		Schemas:        []string{},
	}

	// Convert Position to Span
	pos := source.Pos
	ds.Span = diagnostics.NewSpan(pos.Offset, pos.Offset+len(source.GetName()), diagnostics.FileIDZero)

	// Extract properties
	for _, prop := range source.Properties {
		propName := prop.GetName()
		switch propName {
		case "provider":
			if strVal, ok := prop.Value.AsStringValue(); ok {
				ds.Provider = strVal.GetValue()
				ds.ActiveProvider = strVal.GetValue()
				propPos := prop.Pos
				ds.ProviderSpan = diagnostics.NewSpan(propPos.Offset, propPos.Offset+len(propName), diagnostics.FileIDZero)
			} else {
				propPos := prop.Name.Pos
				span := diagnostics.NewSpan(propPos.Offset, propPos.Offset+len(propName), diagnostics.FileIDZero)
				diags.PushError(diagnostics.NewValidationError(
					"Provider must be a string literal.",
					span,
				))
				propPos = prop.Pos
				ds.ProviderSpan = diagnostics.NewSpan(propPos.Offset, propPos.Offset+10, diagnostics.FileIDZero)
			}
		case "url":
			if strVal, ok := prop.Value.AsStringValue(); ok {
				url := strVal.GetValue()
				ds.URL = &url
			} else if funcCall, ok := prop.Value.AsFunction(); ok {
				if funcCall.Name == "env" && funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
					// Handle env("VAR_NAME")
					if arg := funcCall.Arguments.Arguments[0]; arg != nil {
						if strVal, ok := arg.Value.AsStringValue(); ok {
							url := fmt.Sprintf("env:%s", strVal.GetValue())
							ds.URL = &url
						}
					}
				}
			}
		case "directUrl":
			if strVal, ok := prop.Value.AsStringValue(); ok {
				url := strVal.GetValue()
				ds.DirectURL = &url
			} else if funcCall, ok := prop.Value.AsFunction(); ok {
				if funcCall.Name == "env" && funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
					if arg := funcCall.Arguments.Arguments[0]; arg != nil {
						if strVal, ok := arg.Value.AsStringValue(); ok {
							url := fmt.Sprintf("env:%s", strVal.GetValue())
							ds.DirectURL = &url
						}
					}
				}
			}
		case "shadowDatabaseUrl":
			if strVal, ok := prop.Value.AsStringValue(); ok {
				url := strVal.GetValue()
				ds.ShadowDatabase = &url
			} else if funcCall, ok := prop.Value.AsFunction(); ok {
				if funcCall.Name == "env" && funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
					if arg := funcCall.Arguments.Arguments[0]; arg != nil {
						if strVal, ok := arg.Value.AsStringValue(); ok {
							url := fmt.Sprintf("env:%s", strVal.GetValue())
							ds.ShadowDatabase = &url
						}
					}
				}
			}
		case "schemas":
			propPos := prop.Pos
			schemasSpan := diagnostics.NewSpan(propPos.Offset, propPos.Offset+len(propName), diagnostics.FileIDZero)
			ds.SchemasSpan = &schemasSpan
			if arrExpr, ok := prop.Value.AsArray(); ok {
				for _, elem := range arrExpr.Elements {
					if strVal, ok := elem.AsStringValue(); ok {
						ds.Schemas = append(ds.Schemas, strVal.GetValue())
					}
				}
			}
		case "relationMode":
			// Extract relation mode if present
			if strVal, ok := prop.Value.AsStringValue(); ok {
				ds.relationMode = RelationMode(strVal.GetValue())
			} else if constVal, ok := prop.Value.AsConstantValue(); ok {
				ds.relationMode = RelationMode(constVal.Value)
			}
		}
	}

	// Validate required fields
	if ds.Provider == "" {
		namePos := source.Name.Pos
		nameSpan := diagnostics.NewSpan(namePos.Offset, namePos.Offset+len(source.GetName()), diagnostics.FileIDZero)
		diags.PushError(diagnostics.NewValidationError(
			"Datasource must have a provider.",
			nameSpan,
		))
		return nil
	}

	return ds
}

// extractGenerator extracts generator configuration from a generator AST node.
func extractGenerator(generator *v2ast.GeneratorConfig, diags *diagnostics.Diagnostics) *Generator {
	gen := &Generator{
		Name:   generator.GetName(),
		Config: make(map[string]interface{}),
	}
	// Convert Position to Span
	pos := generator.Pos
	gen.Span = diagnostics.NewSpan(pos.Offset, pos.Offset+len(generator.GetName()), diagnostics.FileIDZero)

	// Extract properties
	for _, prop := range generator.Properties {
		propName := prop.GetName()
		switch propName {
		case "provider":
			if strVal, ok := prop.Value.AsStringValue(); ok {
				gen.Provider = strVal.GetValue()
			} else {
				propNamePos := prop.Name.Pos
				span := diagnostics.NewSpan(propNamePos.Offset, propNamePos.Offset+len(propName), diagnostics.FileIDZero)
				diags.PushError(diagnostics.NewValidationError(
					"Provider must be a string literal.",
					span,
				))
			}
		case "output":
			if strVal, ok := prop.Value.AsStringValue(); ok {
				output := strVal.GetValue()
				gen.Output = &output
			} else if funcCall, ok := prop.Value.AsFunction(); ok {
				if funcCall.Name == "env" && funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
					if arg := funcCall.Arguments.Arguments[0]; arg != nil {
						if strVal, ok := arg.Value.AsStringValue(); ok {
							output := fmt.Sprintf("env:%s", strVal.GetValue())
							gen.Output = &output
						}
					}
				}
			}
		case "binaryTargets":
			// Handle binary targets array
			if arrExpr, ok := prop.Value.AsArray(); ok {
				targets := []string{}
				for _, elem := range arrExpr.Elements {
					if strVal, ok := elem.AsStringValue(); ok {
						targets = append(targets, strVal.GetValue())
					}
				}
				gen.Config["binaryTargets"] = targets
			}
		default:
			// Store other properties in Config map
			gen.Config[propName] = extractConfigValue(prop.Value)
		}
	}

	// Validate required fields
	if gen.Provider == "" {
		genNamePos := generator.Name.Pos
		genNameSpan := diagnostics.NewSpan(genNamePos.Offset, genNamePos.Offset+len(generator.GetName()), diagnostics.FileIDZero)
		diags.PushError(diagnostics.NewValidationError(
			"Generator must have a provider.",
			genNameSpan,
		))
		return nil
	}

	return gen
}

// extractConfigValue extracts a value from an AST expression for config storage.
func extractConfigValue(value v2ast.Expression) interface{} {
	if strVal, ok := value.AsStringValue(); ok {
		return strVal.GetValue()
	}
	if numVal, ok := value.AsNumericValue(); ok {
		return numVal.Value
	}
	if constVal, ok := value.AsConstantValue(); ok {
		// Handle boolean constants
		if boolVal, ok := constVal.AsBooleanValue(); ok {
			return boolVal
		}
		return constVal.Value
	}
	if arrExpr, ok := value.AsArray(); ok {
		result := []interface{}{}
		for _, elem := range arrExpr.Elements {
			result = append(result, extractConfigValue(elem))
		}
		return result
	}
	if funcCall, ok := value.AsFunction(); ok {
		if funcCall.Name == "env" && funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
			if arg := funcCall.Arguments.Arguments[0]; arg != nil {
				if strVal, ok := arg.Value.AsStringValue(); ok {
					return fmt.Sprintf("env:%s", strVal.GetValue())
				}
			}
		}
		return funcCall.Name // Fallback to function name
	}
	return nil
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
