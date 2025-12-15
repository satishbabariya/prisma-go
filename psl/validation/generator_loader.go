// Package pslcore provides generator loading and validation functionality.
package validation

import (
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"

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
	astSchema *v2ast.SchemaAst,
	diags *diagnostics.Diagnostics,
	featureMapWithProvider *FeatureMapWithProvider,
) []Generator {
	var generators []Generator

	// Extract all generator configs from AST
	for _, top := range astSchema.Tops {
		if generator, ok := top.(*v2ast.GeneratorConfig); ok {
			if gen := liftGenerator(generator, diags, featureMapWithProvider); gen != nil {
				generators = append(generators, *gen)
			}
		}
	}

	return generators
}

// liftGenerator lifts a generator from the AST to a Generator configuration.
func liftGenerator(
	astGenerator *v2ast.GeneratorConfig,
	diags *diagnostics.Diagnostics,
	featureMapWithProvider *FeatureMapWithProvider,
) *Generator {
	generatorName := astGenerator.GetName()

	// Extract properties into a map
	args := make(map[string]*v2ast.ConfigBlockProperty)
	hasErrors := false
	for i := range astGenerator.Properties {
		prop := astGenerator.Properties[i]
		key := prop.GetName()
		if prop.Value != nil {
			args[key] = prop
		} else {
			genPos := astGenerator.Pos
			genSpan := diagnostics.NewSpan(genPos.Offset, genPos.Offset+len(generatorName), diagnostics.FileIDZero)
			diags.PushError(diagnostics.NewConfigPropertyMissingValueError(
				key,
				generatorName,
				"generator",
				genSpan,
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
		if _, ok := engineTypeProp.Value.AsStringValue(); !ok {
			propPos := engineTypeProp.Pos
			propSpan := diagnostics.NewSpan(propPos.Offset, propPos.Offset+len(generatorEngineTypeKey), diagnostics.FileIDZero)
			diags.PushError(diagnostics.NewTypeMismatchError(
				"String",
				"unknown",
				"",
				propSpan,
			))
		}
	}

	// Extract provider (required)
	providerProp, hasProvider := args[generatorProviderKey]
	if !hasProvider {
		genPos := astGenerator.Pos
		genSpan := diagnostics.NewSpan(genPos.Offset, genPos.Offset+len(generatorName), diagnostics.FileIDZero)
		diags.PushError(diagnostics.NewGeneratorArgumentNotFoundError(
			generatorProviderKey,
			generatorName,
			genSpan,
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
		if arr, ok := binaryTargetsProp.Value.AsArray(); ok {
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
		if arr, ok := previewFeaturesProp.Value.AsArray(); ok {
			features := []string{}
			for _, elem := range arr.Elements {
				if strVal, ok := elem.AsStringValue(); ok {
					features = append(features, strVal.GetValue())
				}
			}
			// Parse and validate preview features
			if featureMapWithProvider != nil {
				propPos := previewFeaturesProp.Pos
				propSpan := diagnostics.NewSpan(propPos.Offset, propPos.Offset+len(generatorPreviewFeaturesKey), diagnostics.FileIDZero)
				previewFeatures = parseAndValidatePreviewFeatures(
					features,
					featureMapWithProvider,
					propSpan,
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
func coerceStringFromEnvVar(expr v2ast.Expression, diags *diagnostics.Diagnostics) (string, error) {
	// Handle string literal
	if strVal, ok := expr.AsStringValue(); ok {
		return strVal.GetValue(), nil
	}

	// Handle env() function
	if funcCall, ok := expr.AsFunction(); ok {
		if funcCall.Name == "env" {
			if funcCall.Arguments != nil && len(funcCall.Arguments.Arguments) > 0 {
				if arg := funcCall.Arguments.Arguments[0]; arg != nil {
					if strVal, ok := arg.Value.AsStringValue(); ok {
						// Return as env:VAR_NAME format
						return fmt.Sprintf("env:%s", strVal.GetValue()), nil
					}
				}
			}
			pos := expr.Span()
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
			diags.PushError(diagnostics.NewNamedEnvValError(span))
			return "", fmt.Errorf("invalid env() function")
		}
	}

	pos := expr.Span()
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
	diags.PushError(diagnostics.NewTypeMismatchError(
		"String",
		"unknown",
		"",
		span,
	))
	return "", fmt.Errorf("not a string")
}

// extractGeneratorConfigValue extracts a value from an AST expression for generator config.
func extractGeneratorConfigValue(value v2ast.Expression, diags *diagnostics.Diagnostics) interface{} {
	if strVal, ok := value.AsStringValue(); ok {
		return strVal.GetValue()
	}
	if numVal, ok := value.AsNumericValue(); ok {
		return numVal.Value
	}
	if constVal, ok := value.AsConstantValue(); ok {
		if boolVal, ok := constVal.AsBooleanValue(); ok {
			return boolVal
		}
		return constVal.Value
	}
	if arrExpr, ok := value.AsArray(); ok {
		result := []interface{}{}
		for _, elem := range arrExpr.Elements {
			result = append(result, extractGeneratorConfigValue(elem, diags))
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
		return nil
	}
	return nil
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
