// Package parserdatabase provides relation inference functionality.
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// fieldsMatchUniqueCriteria checks if the given field IDs exactly match the fields in a unique criteria.
func fieldsMatchUniqueCriteria(fieldIDs []ScalarFieldId, criteriaFields []FieldWithArgs) bool {
	if len(fieldIDs) != len(criteriaFields) {
		return false
	}
	
	// Create a map of criteria field IDs for quick lookup
	criteriaFieldMap := make(map[ScalarFieldId]bool)
	for _, fieldWithArgs := range criteriaFields {
		criteriaFieldMap[fieldWithArgs.Field] = true
	}
	
	// Check if all field IDs are in the criteria
	for _, fieldID := range fieldIDs {
		if !criteriaFieldMap[fieldID] {
			return false
		}
	}
	
	return true
}

// InferRelations detects relation types and constructs relation objects.
// This is the fourth pass of validation after attribute resolution.
func InferRelations(ctx *Context) {
	// Create a new relations structure
	relations := NewRelations()

	// Iterate over all relation fields
	for _, entry := range ctx.types.IterRelationFields() {
		evidence := relationEvidence(entry.ID, entry.Field, ctx)
		if evidence != nil {
			ingestRelation(evidence, &relations, ctx)
		}
	}

	// Replace the relations in context
	*ctx.relations = relations
}

// RelationEvidence contains evidence about a relation field.
type RelationEvidence struct {
	AstModel                   *ast.Model
	AstField                   *ast.Field
	FieldID                    RelationFieldId
	IsSelfRelation             bool
	IsTwoWayEmbeddedManyToMany bool
	RelationField              *RelationField
	OppositeModel              *ast.Model
	OppositeRelationField      *OppositeRelationField
}

// OppositeRelationField represents an opposite relation field.
type OppositeRelationField struct {
	FieldID       RelationFieldId
	AstField      *ast.Field
	RelationField *RelationField
}

// relationEvidence collects evidence about a relation field.
func relationEvidence(rfid RelationFieldId, rf *RelationField, ctx *Context) *RelationEvidence {
	// Get the model containing this relation field
	astModel := getModelFromID(rf.ModelID, ctx)
	if astModel == nil {
		return nil
	}

	// Get the field from the model
	if int(rf.FieldID) >= len(astModel.Fields) {
		return nil
	}
	astField := &astModel.Fields[rf.FieldID]

	// Get the referenced model
	oppositeModel := getModelFromID(rf.ReferencedModel, ctx)
	if oppositeModel == nil {
		return nil
	}

	isSelfRelation := rf.ModelID == rf.ReferencedModel

	// Find the opposite relation field (if any)
	var oppositeRF *OppositeRelationField
	if !isSelfRelation {
		// Look for relation fields in the opposite model that point back to this model
		for _, oppEntry := range ctx.types.RangeModelRelationFields(rf.ReferencedModel) {
			oppRF := oppEntry.Field
			if oppRF.ReferencedModel == rf.ModelID {
				// Check if names match (for named relations)
				if rf.Name != nil && oppRF.Name != nil {
					if *rf.Name == *oppRF.Name {
						oppAstModel := getModelFromID(oppRF.ModelID, ctx)
						if oppAstModel != nil && int(oppRF.FieldID) < len(oppAstModel.Fields) {
							oppositeRF = &OppositeRelationField{
								FieldID:       oppEntry.ID,
								AstField:      &oppAstModel.Fields[oppRF.FieldID],
								RelationField: oppRF,
							}
							break
						}
					}
				} else if rf.Name == nil && oppRF.Name == nil {
					// Both unnamed - check if it's a valid match
					oppAstModel := getModelFromID(oppRF.ModelID, ctx)
					if oppAstModel != nil && int(oppRF.FieldID) < len(oppAstModel.Fields) {
						oppositeRF = &OppositeRelationField{
							FieldID:       oppEntry.ID,
							AstField:      &oppAstModel.Fields[oppRF.FieldID],
							RelationField: oppRF,
						}
						break
					}
				}
			}
		}
	}

	// Check if it's a two-way embedded many-to-many relation
	isTwoWayEmbeddedManyToMany := false
	if oppositeRF != nil {
		isTwoWayEmbeddedManyToMany = (rf.Fields != nil && len(*rf.Fields) > 0) ||
			(oppositeRF.RelationField.Fields != nil && len(*oppositeRF.RelationField.Fields) > 0)
	}

	return &RelationEvidence{
		AstModel:                   astModel,
		AstField:                   astField,
		FieldID:                    rfid,
		IsSelfRelation:             isSelfRelation,
		IsTwoWayEmbeddedManyToMany: isTwoWayEmbeddedManyToMany,
		RelationField:              rf,
		OppositeModel:              oppositeModel,
		OppositeRelationField:      oppositeRF,
	}
}

// ingestRelation creates a Relation object based on the evidence.
func ingestRelation(evidence *RelationEvidence, relations *Relations, ctx *Context) {
	// Determine relation type based on field arity and opposite field
	var relationAttrs RelationAttributes

	// Get field arity
	fieldArity := getFieldArity(evidence.AstField)

	if fieldArity == List && evidence.OppositeRelationField != nil {
		oppArity := getFieldArity(evidence.OppositeRelationField.AstField)
		if oppArity == List {
			// This is a many-to-many relation

			// We will meet the relation twice when we walk over all relation
			// fields, so we only instantiate it when the relation field is that
			// of model A, and the opposite is model B.
			modelAName := evidence.AstModel.Name.Name
			modelBName := evidence.OppositeModel.Name.Name

			if strings.Compare(modelAName, modelBName) > 0 {
				return // Skip - will be handled by the opposite field
			}

			// For self-relations, the ordering logic is different
			if evidence.IsSelfRelation {
				fieldAName := evidence.AstField.Name.Name
				fieldBName := evidence.OppositeRelationField.AstField.Name.Name
				if strings.Compare(fieldAName, fieldBName) > 0 {
					return // Skip - will be handled by the opposite field
				}
			}

			if evidence.IsTwoWayEmbeddedManyToMany {
				relationAttrs = RelationAttributes{
					Type:   RelationAttributeTypeTwoWayEmbeddedManyToMany,
					FieldA: &evidence.FieldID,
					FieldB: &evidence.OppositeRelationField.FieldID,
				}
			} else {
				relationAttrs = RelationAttributes{
					Type:   RelationAttributeTypeImplicitManyToMany,
					FieldA: &evidence.FieldID,
					FieldB: &evidence.OppositeRelationField.FieldID,
				}
			}
		} else if oppArity == Optional {
			// Required field pointing to optional field - 1:1 relation
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToOne,
				FieldA: &evidence.FieldID,
				FieldB: &evidence.OppositeRelationField.FieldID,
			}
		} else {
			// Required field pointing to required field - 1:1 relation
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToOne,
				FieldA: &evidence.FieldID,
				FieldB: &evidence.OppositeRelationField.FieldID,
			}
		}
	} else if fieldArity == Required && evidence.OppositeRelationField != nil {
		oppArity := getFieldArity(evidence.OppositeRelationField.AstField)
		if oppArity == Optional {
			// Required field pointing to optional field - 1:1 relation
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToOne,
				FieldA: &evidence.FieldID,
				FieldB: &evidence.OppositeRelationField.FieldID,
			}
		} else if oppArity == Required {
			// Required field pointing to required field - 1:1 relation
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToOne,
				FieldA: &evidence.FieldID,
				FieldB: &evidence.OppositeRelationField.FieldID,
			}
		} else {
			// Required field pointing to list - 1:m relation
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToMany,
				FieldA: &evidence.FieldID,
				FieldB: nil,
			}
		}
	} else if fieldArity == Optional && evidence.OppositeRelationField != nil {
		oppArity := getFieldArity(evidence.OppositeRelationField.AstField)
		if oppArity == List {
			// Optional field pointing to list - 1:m relation (back side)
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToMany,
				FieldA: nil,
				FieldB: &evidence.FieldID,
			}
		} else {
			// Optional field pointing to optional/required - 1:1 relation
			relationAttrs = RelationAttributes{
				Type:   RelationAttributeTypeOneToOne,
				FieldA: &evidence.FieldID,
				FieldB: &evidence.OppositeRelationField.FieldID,
			}
		}
	} else {
		// No opposite field - check if referencing fields are unique to determine 1:1 vs 1:m
		relationField := ctx.types.RelationFields[evidence.FieldID]
		
		// Get referencing fields
		var referencingFieldIDs []ScalarFieldId
		if relationField.Fields != nil {
			referencingFieldIDs = *relationField.Fields
		}
		
		if len(referencingFieldIDs) > 0 {
			// Check if referencing fields form a unique constraint
			modelAttrs, exists := ctx.types.ModelAttributes[relationField.ModelID]
			if exists {
				// Check primary key
				if pk := modelAttrs.PrimaryKey; pk != nil {
					if fieldsMatchUniqueCriteria(referencingFieldIDs, pk.Fields) {
						// Referencing fields match primary key - this is a 1:1 relation
						relationAttrs = RelationAttributes{
							Type:   RelationAttributeTypeOneToOne,
							FieldA: &evidence.FieldID,
							FieldB: nil,
						}
						goto relationCreated
					}
				}
				
				// Check unique indexes
				for _, indexEntry := range modelAttrs.AstIndexes {
					if indexEntry.Index.Type == IndexTypeUnique {
						if fieldsMatchUniqueCriteria(referencingFieldIDs, indexEntry.Index.Fields) {
							// Referencing fields match unique index - this is a 1:1 relation
							relationAttrs = RelationAttributes{
								Type:   RelationAttributeTypeOneToOne,
								FieldA: &evidence.FieldID,
								FieldB: nil,
							}
							goto relationCreated
						}
					}
				}
			}
		}
		
		// Referencing fields are not unique - 1:m relation (forward side)
		relationAttrs = RelationAttributes{
			Type:   RelationAttributeTypeOneToMany,
			FieldA: &evidence.FieldID,
			FieldB: nil,
		}
	}
	
relationCreated:

	// Determine model A and model B
	modelA := evidence.RelationField.ModelID
	modelB := evidence.RelationField.ReferencedModel

	// For back-only relations, swap models
	if relationAttrs.FieldA == nil && relationAttrs.FieldB != nil {
		modelA, modelB = modelB, modelA
	}

	// Create the relation
	relation := Relation{
		RelationName: evidence.RelationField.Name,
		Attributes:   relationAttrs,
		ModelA:       modelA,
		ModelB:       modelB,
	}

	// Add to relations
	relationID := relations.PushRelation(relation)

	// Map fields to relation
	relations.SetFieldRelation(evidence.FieldID, relationID)
	if evidence.OppositeRelationField != nil {
		relations.SetFieldRelation(evidence.OppositeRelationField.FieldID, relationID)
	}
}

// FieldArity represents the arity of a field.
type FieldArity int

const (
	Required FieldArity = iota
	Optional
	List
)

// getFieldArity determines the arity of a field from its AST representation.
func getFieldArity(field *ast.Field) FieldArity {
	// Convert from ast.FieldArity to parserdatabase.FieldArity
	switch field.Arity {
	case ast.Required:
		return Required
	case ast.Optional:
		return Optional
	case ast.List:
		return List
	default:
		return Required
	}
	// Default to required
	return Required
}

// getModelFromID retrieves a model from the AST by its ID.
func getModelFromID(modelID ModelId, ctx *Context) *ast.Model {
	for _, file := range ctx.asts.files {
		if file.FileID == modelID.FileID {
			// Find the model by index
			modelCount := 0
			for _, top := range file.AST.Tops {
				if top.AsModel() != nil {
					if uint32(modelCount) == modelID.ID {
						return top.AsModel()
					}
					modelCount++
				}
			}
		}
	}
	return nil
}
