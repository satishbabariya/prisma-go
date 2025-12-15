// Package parserdatabase provides @relation attribute handling.
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// HandleRelation handles @relation on a relation field.
func HandleRelation(
	modelID ModelId,
	rfid RelationFieldId,
	ctx *Context,
) {
	attr := ctx.CurrentAttribute()

	// Store the relation attribute ID
	if int(rfid) < len(ctx.types.RelationFields) {
		attrID := ctx.CurrentAttributeID()
		attrIDUint := uint32(attrID.Index) // Simplified
		ctx.types.RelationFields[rfid].RelationAttribute = &attrIDUint
	}

	// Handle "fields" argument
	if fieldsExpr := ctx.VisitOptionalArg("fields"); fieldsExpr != nil {
		pos := attr.Pos
		span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
		fields, err := resolveFieldArrayWithoutArgs(fieldsExpr, span, modelID, ctx)
		if err == errFieldResolutionAlreadyDealtWith {
			// Error already handled
		} else if err != nil {
			// Errors already pushed
		} else {
			if int(rfid) < len(ctx.types.RelationFields) {
				ctx.types.RelationFields[rfid].Fields = &fields
			}
		}
	}

	// Handle "references" argument
	if int(rfid) < len(ctx.types.RelationFields) {
		rf := &ctx.types.RelationFields[rfid]
		if referencesExpr := ctx.VisitOptionalArg("references"); referencesExpr != nil {
			pos := attr.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attr.GetName()), diagnostics.FileIDZero)
			references, err := resolveFieldArrayWithoutArgs(referencesExpr, span, rf.ReferencedModel, ctx)
			if err == errFieldResolutionAlreadyDealtWith {
				// Error already handled
			} else if err != nil {
				// Errors already pushed
			} else {
				rf.References = &references
			}
		}
	}

	// Handle "name" argument (optional, but if present must be a string)
	if nameExpr := ctx.VisitOptionalArg("name"); nameExpr != nil {
		name, ok := CoerceString(nameExpr, ctx.diagnostics)
		if ok {
			if name == "" {
				ctx.PushAttributeValidationError("A relation cannot have an empty name.")
			} else {
				nameID := ctx.interner.Intern(name)
				if int(rfid) < len(ctx.types.RelationFields) {
					ctx.types.RelationFields[rfid].Name = &nameID
				}
			}
		}
	}

	// Handle "map" argument
	if mapExpr := ctx.VisitOptionalArg("map"); mapExpr != nil {
		mappedName, ok := CoerceString(mapExpr, ctx.diagnostics)
		if ok {
			if mappedName == "" {
				ctx.PushAttributeValidationError("The `map` argument cannot be an empty string.")
			} else {
				nameID := ctx.interner.Intern(mappedName)
				if int(rfid) < len(ctx.types.RelationFields) {
					ctx.types.RelationFields[rfid].MappedName = &nameID
				}
			}
		}
	}

	// Handle "onDelete" argument
	if onDeleteExpr := ctx.VisitOptionalArg("onDelete"); onDeleteExpr != nil {
		action := parseReferentialAction(onDeleteExpr, ctx)
		if action != nil {
			if int(rfid) < len(ctx.types.RelationFields) {
				ctx.types.RelationFields[rfid].OnDelete = &ReferentialActionInfo{
					Action: *action,
					Span:   getExpressionSpan(onDeleteExpr),
				}
			}
		}
	}

	// Handle "onUpdate" argument
	if onUpdateExpr := ctx.VisitOptionalArg("onUpdate"); onUpdateExpr != nil {
		action := parseReferentialAction(onUpdateExpr, ctx)
		if action != nil {
			if int(rfid) < len(ctx.types.RelationFields) {
				ctx.types.RelationFields[rfid].OnUpdate = &ReferentialActionInfo{
					Action: *action,
					Span:   getExpressionSpan(onUpdateExpr),
				}
			}
		}
	}
}

// resolveFieldArrayWithoutArgs resolves an array of field names to ScalarFieldIds.
// This is simpler than resolveFieldArrayWithArgs as it doesn't handle field arguments.
func resolveFieldArrayWithoutArgs(
	values v2ast.Expression,
	attributeSpan diagnostics.Span,
	modelID ModelId,
	ctx *Context,
) ([]ScalarFieldId, error) {
	// Coerce to array of constants
	arrayExpr, ok := values.(*v2ast.ArrayExpression)
	if !ok {
		ctx.PushError(diagnostics.NewValueParserError(
			"array",
			"expression",
			attributeSpan,
		))
		return nil, errFieldResolutionFailed
	}

	var fieldIDs []ScalarFieldId
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
		fieldName, ok := CoerceConstant(elem, ctx.diagnostics)
		if !ok {
			continue
		}

		// Check if field contains '.' (composite type path - not supported in @relation fields/references)
		if strings.Contains(fieldName, ".") {
			unknownFields = append(unknownFields, fieldName)
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
		for _, existing := range fieldIDs {
			if existing == *sfid {
				duplicate = true
				break
			}
		}
		if duplicate {
			ctx.PushError(diagnostics.NewModelValidationError(
				"The relation definition refers to the field "+fieldName+" multiple times.",
				"model",
				astModel.GetName(),
				attributeSpan,
			))
			return nil, errFieldResolutionAlreadyDealtWith
		}

		fieldIDs = append(fieldIDs, *sfid)
	}

	// Report errors
	if len(unknownFields) > 0 {
		fieldsStr := strings.Join(unknownFields, ", ")
		ctx.PushError(diagnostics.NewValidationError(
			"The argument fields must refer only to existing fields. The following fields do not exist in this model: "+fieldsStr,
			attributeSpan,
		))
	}

	if len(relationFields) > 0 {
		fieldsStr := strings.Join(relationFields, ", ")
		ctx.PushError(diagnostics.NewValidationError(
			"The argument fields must refer only to scalar fields. But it is referencing the following relation fields: "+fieldsStr,
			attributeSpan,
		))
	}

	if len(unknownFields) > 0 || len(relationFields) > 0 {
		return nil, errFieldResolutionFailed
	}

	return fieldIDs, nil
}

// parseReferentialAction parses a referential action from an expression.
func parseReferentialAction(expr v2ast.Expression, ctx *Context) *ReferentialAction {
	actionName, ok := CoerceConstant(expr, ctx.diagnostics)
	if !ok {
		return nil
	}

	switch actionName {
	case "Cascade":
		action := ReferentialActionCascade
		return &action
	case "Restrict":
		action := ReferentialActionRestrict
		return &action
	case "NoAction":
		action := ReferentialActionNoAction
		return &action
	case "SetNull":
		action := ReferentialActionSetNull
		return &action
	case "SetDefault":
		action := ReferentialActionSetDefault
		return &action
	default:
		ctx.PushAttributeValidationError("Invalid referential action: " + actionName + ". Valid actions are: Cascade, Restrict, NoAction, SetNull, SetDefault.")
		return nil
	}
}

// getExpressionSpan extracts the span from an expression.
func getExpressionSpan(expr v2ast.Expression) diagnostics.Span {
	pos := expr.Span()
	return diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
}
