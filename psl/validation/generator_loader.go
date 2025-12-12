// Package pslcore provides generator loading and validation functionality.
package validation

import (
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"

	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

const (
	generatorProviderKey        = "provider"
	generatorOutputKey          = "output"
	generatorBinaryTargetsKey   = "binaryTargets"
	generatorPreviewFeaturesKey = "previewFeatures"
	generatorEngineTypeKey      = "engineType"
)

// LoadGeneratorsFromAST loads all generators from the provided schema AST.
// featureMapWithProvider can be nil if preview features are not yet implemented.
func LoadGeneratorsFromAST(
	astSchema *ast.SchemaAst,
	diags *diagnostics.Diagnostics,
	featureMapWithProvider *FeatureMapWithProvider,
) []Generator {
	var generators []Generator

	// Extract all generator configs from AST
	for _, top := range astSchema.Tops {
		if generator := top.AsGenerator(); generator != nil {
			if gen := liftGenerator(generator, diags, featureMapWithProvider); gen != nil {
				generators = append(generators, *gen)
			}
		}
	}

	return generators
}

// liftGenerator lifts a generator from the AST to a Generator configuration.
func liftGenerator(
	astGenerator *ast.GeneratorConfig,
	diags *diagnostics.Diagnostics,
	featureMapWithProvider *FeatureMapWithProvider,
) *Generator {
	generatorName := astGenerator.Name.Name

	// Extract properties into a map
	args := make(map[string]*ast.ConfigBlockProperty)
	hasErrors := false
	for i := range astGenerator.Properties {
		prop := &astGenerator.Properties[i]
		key := prop.Name.Name
		if prop.Value != nil {
			args[key] = prop
		} else {
			diags.PushError(diagnostics.NewConfigPropertyMissingValueError(
				key,
				generatorName,
				"generator",
				astGenerator.Span(),
			))
			hasErrors = true
		}
	}

	// Return early if there are missing values
	if hasErrors {
		return nil
	}

	// Validate engineType is a string if present
	if engineTypeProp, hasEngineType := args[generatorEngineTypeKey]; hasEngineType {
		if _, ok := engineTypeProp.Value.(ast.StringLiteral); !ok {
			diags.PushError(diagnostics.NewTypeMismatchError(
				"String",
				"unknown",
				"",
				engineTypeProp.Span(),
			))
		}
	}

	// Extract provider (required)
	providerProp, hasProvider := args[generatorProviderKey]
	if !hasProvider {
		diags.PushError(diagnostics.NewGeneratorArgumentNotFoundError(
			generatorProviderKey,
			generatorName,
			astGenerator.Span(),
		))
		return nil
	}

	provider, err := coerceStringFromEnvVar(providerProp.Value, diags)
	if err != nil {
		return nil
	}

	// Extract output (optional)
	var output *string
	if outputProp, hasOutput := args[generatorOutputKey]; hasOutput {
		if str, err := coerceStringFromEnvVar(outputProp.Value, diags); err == nil {
			output = &str
		}
	}

	// Extract binaryTargets (optional)
	binaryTargets := []string{}
	if binaryTargetsProp, hasBinaryTargets := args[generatorBinaryTargetsKey]; hasBinaryTargets {
		if arr, ok := binaryTargetsProp.Value.(ast.ArrayLiteral); ok {
			for _, elem := range arr.Elements {
				if str, err := coerceStringFromEnvVar(elem, diags); err == nil {
					binaryTargets = append(binaryTargets, str)
				}
			}
		}
	}

	// Extract previewFeatures (optional)
	var previewFeatures *PreviewFeatures
	if previewFeaturesProp, hasPreviewFeatures := args[generatorPreviewFeaturesKey]; hasPreviewFeatures {
		if arr, ok := previewFeaturesProp.Value.(ast.ArrayLiteral); ok {
			features := []string{}
			for _, elem := range arr.Elements {
				if strLit, ok := elem.(ast.StringLiteral); ok {
					features = append(features, strLit.Value)
				}
			}
			// Parse and validate preview features
			if featureMapWithProvider != nil {
				previewFeatures = parseAndValidatePreviewFeatures(
					features,
					featureMapWithProvider,
					previewFeaturesProp.Span(),
					diags,
				)
			} else {
				// If feature map is not available, create empty preview features
				previewFeatures = &PreviewFeatures{}
			}
		}
	}

	// Extract remaining properties as config
	config := make(map[string]interface{})
	for key, prop := range args {
		// Skip known properties
		if key == generatorProviderKey || key == generatorOutputKey ||
			key == generatorBinaryTargetsKey || key == generatorPreviewFeaturesKey ||
			key == generatorEngineTypeKey {
			continue
		}

		// Extract config value
		configValue := extractGeneratorConfigValue(prop.Value, diags)
		if configValue != nil {
			config[key] = configValue
		}
	}

	// Extract documentation
	// TODO: Extract documentation when available

	return &Generator{
		Name:            generatorName,
		Provider:        provider,
		Output:          output,
		BinaryTargets:   binaryTargets,
		PreviewFeatures: previewFeatures,
		Config:          config,
	}
}

// coerceStringFromEnvVar coerces an expression to a string, supporting env() function.
func coerceStringFromEnvVar(expr ast.Expression, diags *diagnostics.Diagnostics) (string, error) {
	// Handle string literal
	if strLit, ok := expr.(ast.StringLiteral); ok {
		return strLit.Value, nil
	}

	// Handle env() function
	if funcCall, ok := expr.(ast.FunctionCall); ok {
		if funcCall.Name.Name == "env" {
			if len(funcCall.Arguments) > 0 {
				if strLit, ok := funcCall.Arguments[0].(ast.StringLiteral); ok {
					// Return as env:VAR_NAME format
					return fmt.Sprintf("env:%s", strLit.Value), nil
				}
			}
			diags.PushError(diagnostics.NewNamedEnvValError(expr.Span()))
			return "", fmt.Errorf("invalid env() function")
		}
	}

	diags.PushError(diagnostics.NewTypeMismatchError(
		"String",
		"unknown",
		"",
		expr.Span(),
	))
	return "", fmt.Errorf("not a string")
}

// extractGeneratorConfigValue extracts a value from an AST expression for generator config.
func extractGeneratorConfigValue(value ast.Expression, diags *diagnostics.Diagnostics) interface{} {
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
			result = append(result, extractGeneratorConfigValue(elem, diags))
		}
		return result
	case ast.FunctionCall:
		if v.Name.Name == "env" && len(v.Arguments) > 0 {
			if strLit, ok := v.Arguments[0].(ast.StringLiteral); ok {
				return fmt.Sprintf("env:%s", strLit.Value)
			}
		}
		return nil
	default:
		return nil
	}
}

// parseAndValidatePreviewFeatures parses and validates preview features.
func parseAndValidatePreviewFeatures(
	previewFeatures []string,
	featureMapWithProvider *FeatureMapWithProvider,
	span diagnostics.Span,
	diags *diagnostics.Diagnostics,
) *PreviewFeatures {
	result := EmptyPreviewFeatures()

	for _, featureStr := range previewFeatures {
		// Parse the feature string
		var pf PreviewFeature
		featureOpt := pf.ParseOpt(featureStr)

		if featureOpt == nil {
			// Unknown feature - get list of valid features for error message
			activeFeatures := featureMapWithProvider.ActiveFeatures()
			var validFeatures []string
			for i := uint(0); i < 64; i++ {
				feature := PreviewFeature(1 << i)
				if activeFeatures.Contains(feature) {
					validFeatures = append(validFeatures, feature.String())
				}
			}
			expectedFeatures := ""
			if len(validFeatures) > 0 {
				expectedFeatures = strings.Join(validFeatures, ", ")
			}
			diags.PushError(diagnostics.NewPreviewFeatureNotKnownError(featureStr, expectedFeatures, span))
			continue
		}

		feature := *featureOpt

		// Check if deprecated
		if featureMapWithProvider.IsDeprecated(feature) {
			result.Add(feature)
			diags.PushWarning(diagnostics.NewPreviewFeatureDeprecatedWarning(featureStr, span))
			continue
		}

		// Check if stabilized
		if featureMapWithProvider.IsStabilized(feature) {
			// Check if renamed
			renamed := featureMapWithProvider.IsRenamed(feature)
			if renamed != nil {
				result.Add(renamed.Value.To)
				if renamed.IsForProvider {
					diags.PushWarning(diagnostics.NewPreviewFeatureRenamedForProviderWarning(
						renamed.Provider,
						featureStr,
						renamed.Value.To.String(),
						renamed.Value.PrislyLinkEndpoint,
						span,
					))
				} else {
					diags.PushWarning(diagnostics.NewPreviewFeatureRenamedWarning(
						featureStr,
						renamed.Value.To.String(),
						renamed.Value.PrislyLinkEndpoint,
						span,
					))
				}
			} else {
				result.Add(feature)
				diags.PushWarning(diagnostics.NewPreviewFeatureIsStabilizedWarning(featureStr, span))
			}
			continue
		}

		// Check if valid
		if !featureMapWithProvider.IsValid(feature) {
			// Get list of valid features for error message
			activeFeatures := featureMapWithProvider.ActiveFeatures()
			var validFeatures []string
			for i := uint(0); i < 64; i++ {
				f := PreviewFeature(1 << i)
				if activeFeatures.Contains(f) {
					validFeatures = append(validFeatures, f.String())
				}
			}
			expectedFeatures := ""
			if len(validFeatures) > 0 {
				expectedFeatures = strings.Join(validFeatures, ", ")
			}
			diags.PushError(diagnostics.NewPreviewFeatureNotKnownError(featureStr, expectedFeatures, span))
			continue
		}

		// Valid feature
		result.Add(feature)
	}

	return &result
}
