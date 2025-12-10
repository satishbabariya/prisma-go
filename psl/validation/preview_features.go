// Package pslcore provides preview feature definitions and handling.
package validation

import (
	"fmt"
	"strings"
)

// PreviewFeature represents a preview feature flag.
type PreviewFeature uint64

// Preview features (alphabetically sorted, matching Rust implementation)
const (
	PreviewFeatureAggregateApi PreviewFeature = 1 << iota
	PreviewFeatureAtomicNumberOperations
	PreviewFeatureClientExtensions
	PreviewFeatureCockroachdb
	PreviewFeatureConnectOrCreate
	PreviewFeatureCreateMany
	PreviewFeatureDataProxy
	PreviewFeatureDeno
	PreviewFeatureDistinct
	PreviewFeatureDriverAdapters
	PreviewFeatureExtendedIndexes
	PreviewFeatureExtendedWhereUnique
	PreviewFeatureFieldReference
	PreviewFeatureFilterJson
	PreviewFeatureFilteredRelationCount
	PreviewFeatureFullTextIndex
	PreviewFeatureFullTextSearch
	PreviewFeatureFullTextSearchPostgres
	PreviewFeatureGroupBy
	PreviewFeatureImprovedQueryRaw
	PreviewFeatureInteractiveTransactions
	PreviewFeatureJsonProtocol
	PreviewFeatureMicrosoftSqlServer
	PreviewFeatureMiddlewares
	PreviewFeatureMongoDb
	PreviewFeatureMultiSchema
	PreviewFeatureNApi
	PreviewFeatureNamedConstraints
	PreviewFeatureNativeDistinct
	PreviewFeatureNativeTypes
	PreviewFeatureOmitApi
	PreviewFeatureOrderByAggregateGroup
	PreviewFeatureOrderByNulls
	PreviewFeatureOrderByRelation
	PreviewFeaturePostgresqlExtensions
	PreviewFeaturePrismaSchemaFolder
	PreviewFeatureQueryCompiler
	PreviewFeatureReactNative
	PreviewFeatureReferentialActions
	PreviewFeatureReferentialIntegrity
	PreviewFeatureRelationJoins
	PreviewFeatureSchemaEngineDriverAdapters
	PreviewFeatureSelectRelationCount
	PreviewFeatureShardKeys
	PreviewFeatureStrictUndefinedChecks
	PreviewFeatureTracing
	PreviewFeatureTransactionApi
	PreviewFeatureTypedSql
	PreviewFeatureUncheckedScalarInputs
	PreviewFeatureViews
)

// ParseOpt parses a preview feature string and returns the corresponding PreviewFeature.
// Returns nil if the string doesn't match any known feature.
func (pf PreviewFeature) ParseOpt(s string) *PreviewFeature {
	// Normalize to lowercase for case-insensitive matching
	s = strings.ToLower(s)

	// Map of lowercase feature names to their enum values
	featureMap := map[string]PreviewFeature{
		"aggregateapi":               PreviewFeatureAggregateApi,
		"atomicnumberoperations":     PreviewFeatureAtomicNumberOperations,
		"clientextensions":           PreviewFeatureClientExtensions,
		"cockroachdb":                PreviewFeatureCockroachdb,
		"connectorcreate":            PreviewFeatureConnectOrCreate,
		"createmany":                 PreviewFeatureCreateMany,
		"dataproxy":                  PreviewFeatureDataProxy,
		"deno":                       PreviewFeatureDeno,
		"distinct":                   PreviewFeatureDistinct,
		"driveradapters":             PreviewFeatureDriverAdapters,
		"extendedindexes":            PreviewFeatureExtendedIndexes,
		"extendedwhereunique":        PreviewFeatureExtendedWhereUnique,
		"fieldreference":             PreviewFeatureFieldReference,
		"filterjson":                 PreviewFeatureFilterJson,
		"filteredrelationcount":      PreviewFeatureFilteredRelationCount,
		"fulltextindex":              PreviewFeatureFullTextIndex,
		"fulltextsearch":             PreviewFeatureFullTextSearch,
		"fulltextsearchpostgres":     PreviewFeatureFullTextSearchPostgres,
		"groupby":                    PreviewFeatureGroupBy,
		"improvedqueryraw":           PreviewFeatureImprovedQueryRaw,
		"interactivetransactions":    PreviewFeatureInteractiveTransactions,
		"jsonprotocol":               PreviewFeatureJsonProtocol,
		"microsoftsqlserver":         PreviewFeatureMicrosoftSqlServer,
		"middlewares":                PreviewFeatureMiddlewares,
		"mongodb":                    PreviewFeatureMongoDb,
		"multischema":                PreviewFeatureMultiSchema,
		"napi":                       PreviewFeatureNApi,
		"namedconstraints":           PreviewFeatureNamedConstraints,
		"nativedistinct":             PreviewFeatureNativeDistinct,
		"nativetypes":                PreviewFeatureNativeTypes,
		"omitapi":                    PreviewFeatureOmitApi,
		"orderbyaggregategroup":      PreviewFeatureOrderByAggregateGroup,
		"orderbynulls":               PreviewFeatureOrderByNulls,
		"orderbyrelation":            PreviewFeatureOrderByRelation,
		"postgresqlextensions":       PreviewFeaturePostgresqlExtensions,
		"prismaschemafolder":         PreviewFeaturePrismaSchemaFolder,
		"querycompiler":              PreviewFeatureQueryCompiler,
		"reactnative":                PreviewFeatureReactNative,
		"referentialactions":         PreviewFeatureReferentialActions,
		"referentialintegrity":       PreviewFeatureReferentialIntegrity,
		"relationjoins":              PreviewFeatureRelationJoins,
		"schemaenginedriveradapters": PreviewFeatureSchemaEngineDriverAdapters,
		"selectrelationcount":        PreviewFeatureSelectRelationCount,
		"shardkeys":                  PreviewFeatureShardKeys,
		"strictundefinedchecks":      PreviewFeatureStrictUndefinedChecks,
		"tracing":                    PreviewFeatureTracing,
		"transactionapi":             PreviewFeatureTransactionApi,
		"typedsql":                   PreviewFeatureTypedSql,
		"uncheckedscalarinputs":      PreviewFeatureUncheckedScalarInputs,
		"views":                      PreviewFeatureViews,
	}

	if feature, ok := featureMap[s]; ok {
		return &feature
	}
	return nil
}

// String returns the string representation of the preview feature.
// Format: first letter lowercase, rest as-is (e.g., "aggregateApi")
func (pf PreviewFeature) String() string {
	switch pf {
	case PreviewFeatureAggregateApi:
		return "aggregateApi"
	case PreviewFeatureAtomicNumberOperations:
		return "atomicNumberOperations"
	case PreviewFeatureClientExtensions:
		return "clientExtensions"
	case PreviewFeatureCockroachdb:
		return "cockroachdb"
	case PreviewFeatureConnectOrCreate:
		return "connectOrCreate"
	case PreviewFeatureCreateMany:
		return "createMany"
	case PreviewFeatureDataProxy:
		return "dataProxy"
	case PreviewFeatureDeno:
		return "deno"
	case PreviewFeatureDistinct:
		return "distinct"
	case PreviewFeatureDriverAdapters:
		return "driverAdapters"
	case PreviewFeatureExtendedIndexes:
		return "extendedIndexes"
	case PreviewFeatureExtendedWhereUnique:
		return "extendedWhereUnique"
	case PreviewFeatureFieldReference:
		return "fieldReference"
	case PreviewFeatureFilterJson:
		return "filterJson"
	case PreviewFeatureFilteredRelationCount:
		return "filteredRelationCount"
	case PreviewFeatureFullTextIndex:
		return "fullTextIndex"
	case PreviewFeatureFullTextSearch:
		return "fullTextSearch"
	case PreviewFeatureFullTextSearchPostgres:
		return "fullTextSearchPostgres"
	case PreviewFeatureGroupBy:
		return "groupBy"
	case PreviewFeatureImprovedQueryRaw:
		return "improvedQueryRaw"
	case PreviewFeatureInteractiveTransactions:
		return "interactiveTransactions"
	case PreviewFeatureJsonProtocol:
		return "jsonProtocol"
	case PreviewFeatureMicrosoftSqlServer:
		return "microsoftSqlServer"
	case PreviewFeatureMiddlewares:
		return "middlewares"
	case PreviewFeatureMongoDb:
		return "mongoDb"
	case PreviewFeatureMultiSchema:
		return "multiSchema"
	case PreviewFeatureNApi:
		return "nApi"
	case PreviewFeatureNamedConstraints:
		return "namedConstraints"
	case PreviewFeatureNativeDistinct:
		return "nativeDistinct"
	case PreviewFeatureNativeTypes:
		return "nativeTypes"
	case PreviewFeatureOmitApi:
		return "omitApi"
	case PreviewFeatureOrderByAggregateGroup:
		return "orderByAggregateGroup"
	case PreviewFeatureOrderByNulls:
		return "orderByNulls"
	case PreviewFeatureOrderByRelation:
		return "orderByRelation"
	case PreviewFeaturePostgresqlExtensions:
		return "postgresqlExtensions"
	case PreviewFeaturePrismaSchemaFolder:
		return "prismaSchemaFolder"
	case PreviewFeatureQueryCompiler:
		return "queryCompiler"
	case PreviewFeatureReactNative:
		return "reactNative"
	case PreviewFeatureReferentialActions:
		return "referentialActions"
	case PreviewFeatureReferentialIntegrity:
		return "referentialIntegrity"
	case PreviewFeatureRelationJoins:
		return "relationJoins"
	case PreviewFeatureSchemaEngineDriverAdapters:
		return "schemaEngineDriverAdapters"
	case PreviewFeatureSelectRelationCount:
		return "selectRelationCount"
	case PreviewFeatureShardKeys:
		return "shardKeys"
	case PreviewFeatureStrictUndefinedChecks:
		return "strictUndefinedChecks"
	case PreviewFeatureTracing:
		return "tracing"
	case PreviewFeatureTransactionApi:
		return "transactionApi"
	case PreviewFeatureTypedSql:
		return "typedSql"
	case PreviewFeatureUncheckedScalarInputs:
		return "uncheckedScalarInputs"
	case PreviewFeatureViews:
		return "views"
	default:
		return "unknown"
	}
}

// ParsePreviewFeature parses a string to a PreviewFeature.
// This is a convenience function that wraps ParseOpt.
func ParsePreviewFeature(s string) *PreviewFeature {
	var pf PreviewFeature
	return pf.ParseOpt(s)
}

// PreviewFeatures represents a set of preview features using bitflags.
type PreviewFeatures struct {
	flags uint64
}

// Empty returns an empty PreviewFeatures set.
func EmptyPreviewFeatures() PreviewFeatures {
	return PreviewFeatures{flags: 0}
}

// NewPreviewFeatures creates a PreviewFeatures set from individual features.
func NewPreviewFeatures(features ...PreviewFeature) PreviewFeatures {
	var flags uint64
	for _, feature := range features {
		flags |= uint64(feature)
	}
	return PreviewFeatures{flags: flags}
}

// Contains checks if the set contains the given feature.
func (pf PreviewFeatures) Contains(feature PreviewFeature) bool {
	return (pf.flags & uint64(feature)) != 0
}

// Union returns the union of two PreviewFeatures sets.
func (pf PreviewFeatures) Union(other PreviewFeatures) PreviewFeatures {
	return PreviewFeatures{flags: pf.flags | other.flags}
}

// Intersection returns the intersection of two PreviewFeatures sets.
func (pf PreviewFeatures) Intersection(other PreviewFeatures) PreviewFeatures {
	return PreviewFeatures{flags: pf.flags & other.flags}
}

// IsEmpty checks if the set is empty.
func (pf PreviewFeatures) IsEmpty() bool {
	return pf.flags == 0
}

// Add adds a feature to the set.
func (pf *PreviewFeatures) Add(feature PreviewFeature) {
	pf.flags |= uint64(feature)
}

// Remove removes a feature from the set.
func (pf *PreviewFeatures) Remove(feature PreviewFeature) {
	pf.flags &^= uint64(feature)
}

// String returns a string representation of the preview features.
func (pf PreviewFeatures) String() string {
	if pf.IsEmpty() {
		return "[]"
	}

	var features []string
	for i := uint(0); i < 64; i++ {
		feature := PreviewFeature(1 << i)
		if pf.Contains(feature) {
			features = append(features, feature.String())
		}
	}

	return fmt.Sprintf("[%s]", strings.Join(features, ", "))
}

// FeatureMapWithProvider represents a feature map with provider information.
type FeatureMapWithProvider struct {
	provider   *string
	active     PreviewFeatures
	native     map[string]PreviewFeatures
	stabilized PreviewFeatures
	deprecated PreviewFeatures
	renamed    map[RenamedFeatureKey]RenamedFeatureValue
	hidden     PreviewFeatures
}

// RenamedFeatureKey represents a key for renamed features.
type RenamedFeatureKey struct {
	From     PreviewFeature
	Provider *string
}

// RenamedFeatureValue represents a renamed feature value.
type RenamedFeatureValue struct {
	To                 PreviewFeature
	PrislyLinkEndpoint string
}

// RenamedFeature represents a renamed feature (provider-specific or all providers).
type RenamedFeature struct {
	IsForProvider bool
	Provider      string
	Value         RenamedFeatureValue
}

// NewFeatureMapWithProvider creates a new FeatureMapWithProvider.
func NewFeatureMapWithProvider(connectorProvider *string) *FeatureMapWithProvider {
	// Active features (currently active preview features)
	active := NewPreviewFeatures(
		PreviewFeatureNativeDistinct,
		PreviewFeaturePostgresqlExtensions,
		PreviewFeatureRelationJoins,
		PreviewFeatureSchemaEngineDriverAdapters,
		PreviewFeatureShardKeys,
		PreviewFeatureStrictUndefinedChecks,
		PreviewFeatureViews,
	)

	// Native features (provider-specific)
	native := make(map[string]PreviewFeatures)
	native["postgresql"] = NewPreviewFeatures(PreviewFeatureFullTextSearchPostgres)

	// Stabilized features (no longer preview, but kept for compatibility)
	stabilized := NewPreviewFeatures(
		PreviewFeatureAggregateApi,
		PreviewFeatureAtomicNumberOperations,
		PreviewFeatureClientExtensions,
		PreviewFeatureCockroachdb,
		PreviewFeatureConnectOrCreate,
		PreviewFeatureCreateMany,
		PreviewFeatureDataProxy,
		PreviewFeatureDeno,
		PreviewFeatureDistinct,
		PreviewFeatureDriverAdapters,
		PreviewFeatureExtendedIndexes,
		PreviewFeatureExtendedWhereUnique,
		PreviewFeatureFieldReference,
		PreviewFeatureFilteredRelationCount,
		PreviewFeatureFilterJson,
		PreviewFeatureFullTextIndex,
		PreviewFeatureFullTextSearch,
		PreviewFeatureGroupBy,
		PreviewFeatureImprovedQueryRaw,
		PreviewFeatureInteractiveTransactions,
		PreviewFeatureJsonProtocol,
		PreviewFeatureMicrosoftSqlServer,
		PreviewFeatureMiddlewares,
		PreviewFeatureMongoDb,
		PreviewFeatureMultiSchema,
		PreviewFeatureNamedConstraints,
		PreviewFeatureNApi,
		PreviewFeatureNativeTypes,
		PreviewFeatureOmitApi,
		PreviewFeatureOrderByAggregateGroup,
		PreviewFeatureOrderByNulls,
		PreviewFeatureOrderByRelation,
		PreviewFeaturePrismaSchemaFolder,
		PreviewFeatureQueryCompiler,
		PreviewFeatureReferentialActions,
		PreviewFeatureReferentialIntegrity,
		PreviewFeatureSelectRelationCount,
		PreviewFeatureTracing,
		PreviewFeatureTransactionApi,
		PreviewFeatureUncheckedScalarInputs,
	)

	// Deprecated features (currently none)
	deprecated := EmptyPreviewFeatures()

	// Renamed features
	renamed := make(map[RenamedFeatureKey]RenamedFeatureValue)
	postgresProvider := "postgresql"
	renamed[RenamedFeatureKey{
		From:     PreviewFeatureFullTextSearch,
		Provider: &postgresProvider,
	}] = RenamedFeatureValue{
		To:                 PreviewFeatureFullTextSearchPostgres,
		PrislyLinkEndpoint: "fts-postgres",
	}

	// Hidden features (valid but not shown in tooling)
	hidden := NewPreviewFeatures(
		PreviewFeatureReactNative,
		PreviewFeatureTypedSql,
	)

	return &FeatureMapWithProvider{
		provider:   connectorProvider,
		active:     active,
		native:     native,
		stabilized: stabilized,
		deprecated: deprecated,
		renamed:    renamed,
		hidden:     hidden,
	}
}

// NativeFeatures returns provider-specific native features.
func (fmp *FeatureMapWithProvider) NativeFeatures() PreviewFeatures {
	if fmp.provider == nil {
		return EmptyPreviewFeatures()
	}
	if features, ok := fmp.native[*fmp.provider]; ok {
		return features
	}
	return EmptyPreviewFeatures()
}

// ActiveFeatures returns all active features (including native).
func (fmp *FeatureMapWithProvider) ActiveFeatures() PreviewFeatures {
	return fmp.active.Union(fmp.NativeFeatures())
}

// HiddenFeatures returns hidden features.
func (fmp *FeatureMapWithProvider) HiddenFeatures() PreviewFeatures {
	return fmp.hidden
}

// IsValid checks if a feature is valid (active or hidden).
func (fmp *FeatureMapWithProvider) IsValid(flag PreviewFeature) bool {
	allValid := fmp.ActiveFeatures().Union(fmp.hidden)
	return allValid.Contains(flag)
}

// IsStabilized checks if a feature is stabilized.
func (fmp *FeatureMapWithProvider) IsStabilized(flag PreviewFeature) bool {
	return fmp.stabilized.Contains(flag)
}

// IsDeprecated checks if a feature is deprecated.
func (fmp *FeatureMapWithProvider) IsDeprecated(flag PreviewFeature) bool {
	return fmp.deprecated.Contains(flag)
}

// IsRenamed checks if a feature was renamed and returns the renamed feature info.
func (fmp *FeatureMapWithProvider) IsRenamed(flag PreviewFeature) *RenamedFeature {
	// Check for provider-specific rename first
	if fmp.provider != nil {
		key := RenamedFeatureKey{
			From:     flag,
			Provider: fmp.provider,
		}
		if value, ok := fmp.renamed[key]; ok {
			return &RenamedFeature{
				IsForProvider: true,
				Provider:      *fmp.provider,
				Value:         value,
			}
		}
	}

	// Check for all-providers rename
	key := RenamedFeatureKey{
		From:     flag,
		Provider: nil,
	}
	if value, ok := fmp.renamed[key]; ok {
		return &RenamedFeature{
			IsForProvider: false,
			Value:         value,
		}
	}

	return nil
}
