// Package parserdatabase provides @@index and @@unique attribute handling.
package database

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// HandleModelIndex handles @@index on a model.
func HandleModelIndex(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	indexAttr := IndexAttribute{
		Type:        IndexTypeNormal,
		Fields:      nil,
		SourceField: nil,
		Name:        nil,
		MappedName:  nil,
		Algorithm:   nil,
		Clustered:   nil,
	}

	commonIndexValidations(&indexAttr, modelID, true, ctx)

	name := getNameArgument(ctx)

	// Get mapped name from @map argument
	mappedName := getIndexMappedName(ctx)
	if mappedName == nil && name != nil {
		// Backwards compatibility: accept name arg on normal indexes and use it as map arg
		mappedName = name
		name = nil
	}

	if name != nil && mappedName != nil {
		ctx.PushAttributeValidationError(
			"The `@@index` attribute accepts the `name` argument as an alias for the `map` argument for legacy reasons. It does not accept both though. Please use the `map` argument to specify the database name of the index.",
		)
		mappedName = nil
	}

	indexAttr.MappedName = mappedName

	// Get algorithm (type argument)
	algo := getIndexAlgorithm(ctx)
	indexAttr.Algorithm = algo

	indexAttr.Clustered = validateClusteringSetting(ctx)

	// Store the index
	attrID := ctx.CurrentAttributeID()
	modelAttrs.AstIndexes = append(modelAttrs.AstIndexes, IndexAttributeEntry{
		AttributeID: uint32(attrID.Index), // Simplified
		Index:       indexAttr,
	})
}

// HandleModelUnique handles @@unique on a model.
func HandleModelUnique(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	indexAttr := IndexAttribute{
		Type:        IndexTypeUnique,
		Fields:      nil,
		SourceField: nil,
		Name:        nil,
		MappedName:  nil,
		Algorithm:   nil,
		Clustered:   nil,
	}

	commonIndexValidations(&indexAttr, modelID, true, ctx)

	attr := ctx.CurrentAttribute()
	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return
	}

	name := getNameArgument(ctx)
	if name != nil {
		pos := attr.Pos
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
		validateClientName(span, astModel.GetName(), *name, "@@unique", ctx)
	}

	mappedName := getIndexMappedName(ctx)

	indexAttr.Name = name
	indexAttr.MappedName = mappedName
	indexAttr.Clustered = validateClusteringSetting(ctx)

	// Store the index
	attrID := ctx.CurrentAttributeID()
	modelAttrs.AstIndexes = append(modelAttrs.AstIndexes, IndexAttributeEntry{
		AttributeID: uint32(attrID.Index), // Simplified
		Index:       indexAttr,
	})
}

// HandleModelFulltext handles @@fulltext on a model.
func HandleModelFulltext(
	modelID ModelId,
	modelAttrs *ModelAttributes,
	ctx *Context,
) {
	indexAttr := IndexAttribute{
		Type:        IndexTypeFulltext,
		Fields:      nil,
		SourceField: nil,
		Name:        nil,
		MappedName:  nil,
		Algorithm:   nil,
		Clustered:   nil,
	}

	commonIndexValidations(&indexAttr, modelID, true, ctx)

	mappedName := getIndexMappedName(ctx)
	indexAttr.MappedName = mappedName

	// Store the index
	attrID := ctx.CurrentAttributeID()
	modelAttrs.AstIndexes = append(modelAttrs.AstIndexes, IndexAttributeEntry{
		AttributeID: uint32(attrID.Index), // Simplified
		Index:       indexAttr,
	})
}

// commonIndexValidations performs common validation for index, unique, and fulltext attributes.
func commonIndexValidations(
	indexData *IndexAttribute,
	modelID ModelId,
	followComposites bool,
	ctx *Context,
) {
	attr := ctx.CurrentAttribute()
	fieldsExpr, _, err := ctx.VisitDefaultArg("fields")
	if err != nil {
		if diagErr, ok := err.(diagnostics.DatamodelError); ok {
			ctx.PushError(diagErr)
		} else {
			ctx.PushError(diagnostics.NewDatamodelError(err.Error(), diagnostics.EmptySpan()))
		}
		return
	}

	isUnique := indexData != nil && indexData.Type == IndexTypeUnique
	pos := attr.Pos
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
	resolvedFields, resolveErr := resolveFieldArrayWithArgs(fieldsExpr, span, modelID, followComposites, isUnique, ctx)
	if resolveErr == errFieldResolutionAlreadyDealtWith {
		return
	}
	if resolveErr != nil {
		// Errors already pushed
		return
	}

	indexData.Fields = resolvedFields
}

// resolveFieldArrayWithArgs resolves an array of field references with optional arguments.
// This is a simplified version that handles basic cases. Full implementation would support
// composite type paths like `compositeField.nestedField`.
func resolveFieldArrayWithArgs(
	values v2ast.Expression,
	attributeSpan diagnostics.Span,
	modelID ModelId,
	followComposites bool,
	isUnique bool,
	ctx *Context,
) ([]FieldWithArgs, error) {
	// Coerce to array
	arrayExpr, ok := values.(*v2ast.ArrayExpression)
	if !ok {
		ctx.PushError(diagnostics.NewValueParserError(
			"array",
			"expression",
			attributeSpan,
		))
		return nil, errFieldResolutionFailed
	}

	var resolvedFields []FieldWithArgs
	var unknownFields []string
	var relationFields []string

	astModel := getModelFromID(modelID, ctx)
	if astModel == nil {
		return nil, errFieldResolutionFailed
	}

	for _, elem := range arrayExpr.Elements {
		if elem == nil {
			continue
		}
		// Try to parse as function call first (field with arguments)
		fieldName, args, _, isFunc := CoerceFunction(elem, ctx.diagnostics)
		if !isFunc {
			// Try as constant
			var ok bool
			fieldName, ok = CoerceConstant(elem, ctx.diagnostics)
			if !ok {
				continue
			}
			args = nil
		}

		// Parse field arguments
		var sortOrder *SortOrder
		// TODO: Parse length and operator class arguments
		// For now, we'll parse simple function calls like field(sort: Desc)

		if args != nil {
			// TODO: Properly parse named arguments from function call
			// For now, we'll handle simple cases
			_ = args
		}

		// Check if field contains '.' (composite type path)
		if strings.Contains(fieldName, ".") {
			if !followComposites {
				unknownFields = append(unknownFields, fieldName)
				continue
			}
			// TODO: Implement composite type path resolution
			unknownFields = append(unknownFields, fieldName+" (composite type paths not yet supported)")
			continue
		}

		// Find the field
		fieldID := ctx.FindModelField(modelID, fieldName)
		if fieldID == nil {
			unknownFields = append(unknownFields, fieldName)
			continue
		}

		// Check if it's a scalar field
		sfid := ctx.types.FindModelScalarField(modelID, *fieldID)
		if sfid == nil {
			relationFields = append(relationFields, fieldName)
			continue
		}

		// Check for duplicates
		duplicate := false
		for _, existing := range resolvedFields {
			if existing.Field == *sfid {
				duplicate = true
				break
			}
		}
		if duplicate {
			ctx.PushError(diagnostics.NewModelValidationError(
				"The index definition refers to the field "+fieldName+" multiple times.",
				"model",
				astModel.GetName(),
				attributeSpan,
			))
			return nil, errFieldResolutionAlreadyDealtWith
		}

		resolvedFields = append(resolvedFields, FieldWithArgs{
			Field:         *sfid,
			Path:          nil, // TODO: Set path for composite type fields
			SortOrder:     sortOrder,
			OperatorClass: nil, // TODO: Parse operator class
		})
	}

	// Report errors for unknown and relation fields
	if len(unknownFields) > 0 {
		prefix := ""
		if isUnique {
			prefix = "unique "
		}
		ctx.PushError(diagnostics.NewModelValidationError(
			fmt.Sprintf("The %sindex definition refers to the unknown fields: %s.", prefix, formatFieldNames(unknownFields)),
			"model",
			astModel.GetName(),
			attributeSpan,
		))
	}

	if len(relationFields) > 0 {
		ctx.PushError(diagnostics.NewModelValidationError(
			"The index definition refers to the relation fields: "+formatFieldNames(relationFields)+". Index definitions must reference only scalar fields.",
			"model",
			astModel.GetName(),
			attributeSpan,
		))
	}

	if len(unknownFields) > 0 || len(relationFields) > 0 {
		return nil, errFieldResolutionFailed
	}

	return resolvedFields, nil
}

// getIndexMappedName extracts the mapped name from a @map argument on an index.
func getIndexMappedName(ctx *Context) *StringId {
	expr := ctx.VisitOptionalArg("map")
	if expr == nil {
		return nil
	}

	name, ok := CoerceString(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	if name == "" {
		ctx.PushAttributeValidationError("The `map` argument cannot be an empty string.")
		return nil
	}

	nameID := ctx.interner.Intern(name)
	return &nameID
}

// getIndexAlgorithm extracts and validates the algorithm (type argument) from an index attribute.
func getIndexAlgorithm(ctx *Context) *IndexAlgorithm {
	expr := ctx.VisitOptionalArg("type")
	if expr == nil {
		return nil
	}

	algoName, ok := CoerceConstant(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	switch algoName {
	case "BTree":
		btree := IndexAlgorithmBTree
		return &btree
	case "Hash":
		hash := IndexAlgorithmHash
		return &hash
	case "Gist":
		gist := IndexAlgorithmGist
		return &gist
	case "Gin":
		gin := IndexAlgorithmGin
		return &gin
	case "SpGist":
		spgist := IndexAlgorithmSpGist
		return &spgist
	case "Brin":
		brin := IndexAlgorithmBrin
		return &brin
	default:
		ctx.PushAttributeValidationError(fmt.Sprintf("Unknown index type: %s.", algoName))
		return nil
	}
}

// getArgumentName extracts the name from an argument expression.
// For now, this is a simplified version that handles function call arguments.
func getArgumentName(arg v2ast.Expression) (string, bool) {
	// In Prisma, arguments can be named or positional
	// For function calls like field(sort: Desc), we need to extract the argument name
	// This is a simplified implementation - the full version would need to handle
	// the AST structure more carefully
	return "", false
}
