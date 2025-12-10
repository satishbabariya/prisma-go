// Package parserdatabase provides relation inference functionality.
package database

import (
	"sort"
)

// RelationId represents an identifier for a relation.
type RelationId uint32

// ManyToManyRelationId represents an identifier for an implicit many-to-many relation.
type ManyToManyRelationId struct {
	RelationId RelationId
}

// Relations stores all relations in a schema.
//
// A relation is always between two models. One model is assigned the role
// of "model A", and the other is "model B". The meaning of "model A" and
// "model B" depends on the type of relation.
//
// - In implicit many-to-many relations, model A and model B are ordered
//   lexicographically, by model name, and failing that by relation field
//   name. This order must be stable in order for the columns in the
//   implicit many-to-many relation table columns and the data in them to
//   keep their meaning.
// - In one-to-one and one-to-many relations, model A is the one carrying
//   the referencing information and possible constraint. For example, on a
//   SQL database, model A would correspond to the table with the foreign
//   key constraint, while model B would correspond to the table referenced
//   by the foreign key.
type Relations struct {
	// Storage. Private. Do not use directly.
	relationsStorage []Relation

	// Which field belongs to which relation. Secondary index for optimization.
	fields map[RelationFieldId]RelationId

	// Indexes for efficient querying.
	//
	// Why sorted slices?
	//
	// - We can't use a map because there can be more than one relation
	//   between two models.
	// - We use sorted slices because we want range queries. Meaning that with a
	//   sorted slice, we can efficiently ask:
	//   - Give me all the relations on other models that point to this model
	//   - Give me all the relations on this model that point to other models
	//
	// Where "on this model" doesn't mean "the relation field is on the model"
	// but "the foreign key is on this model" (= this model is model a)
	//
	// (model_a, model_b, relation_idx)
	//
	// This can be interpreted as the relations _from_ a model.
	forward []RelationIndexEntry
	// (model_b, model_a, relation_idx)
	//
	// This can be interpreted as the relations _to_ a model.
	back []RelationIndexEntry
}

// RelationIndexEntry represents an entry in the relation index.
type RelationIndexEntry struct {
	ModelA     ModelId
	ModelB     ModelId
	RelationID RelationId
}

// Relation represents a relation between two models.
type Relation struct {
	// The `name` argument in @relation
	RelationName *StringId
	// The attributes of the relation
	Attributes RelationAttributes
	// The two models involved in the relation
	ModelA ModelId
	ModelB ModelId
}

// RelationAttributes represents the attributes of a relation.
type RelationAttributes struct {
	// Type discriminator
	Type RelationAttributeType
	// Fields for the relation (varies by type)
	FieldA *RelationFieldId
	FieldB *RelationFieldId
}

// RelationAttributeType represents the type of relation attributes.
type RelationAttributeType int

const (
	// RelationAttributeTypeImplicitManyToMany represents an implicit many-to-many relation
	RelationAttributeTypeImplicitManyToMany RelationAttributeType = iota
	// RelationAttributeTypeTwoWayEmbeddedManyToMany represents a two-way embedded many-to-many relation
	RelationAttributeTypeTwoWayEmbeddedManyToMany
	// RelationAttributeTypeOneToOne represents a one-to-one relation
	RelationAttributeTypeOneToOne
	// RelationAttributeTypeOneToMany represents a one-to-many relation
	RelationAttributeTypeOneToMany
)

// OneToManyRelationFields represents the fields in a one-to-many relation.
type OneToManyRelationFields struct {
	Type    OneToManyRelationFieldsType
	Forward *RelationFieldId
	Back    *RelationFieldId
}

// OneToManyRelationFieldsType represents the type of one-to-many relation fields.
type OneToManyRelationFieldsType int

const (
	OneToManyRelationFieldsTypeForward OneToManyRelationFieldsType = iota
	OneToManyRelationFieldsTypeBack
	OneToManyRelationFieldsTypeBoth
)

// OneToOneRelationFields represents the fields in a one-to-one relation.
type OneToOneRelationFields struct {
	Type    OneToOneRelationFieldsType
	Forward *RelationFieldId
}

// OneToOneRelationFieldsType represents the type of one-to-one relation fields.
type OneToOneRelationFieldsType int

const (
	OneToOneRelationFieldsTypeForward OneToOneRelationFieldsType = iota
	OneToOneRelationFieldsTypeBoth
)

// Fields returns the fields for the relation attributes.
func (ra RelationAttributes) Fields() (*RelationFieldId, *RelationFieldId) {
	return ra.FieldA, ra.FieldB
}

// IsImplicitManyToMany returns true if this is an implicit many-to-many relation.
func (r Relation) IsImplicitManyToMany() bool {
	return r.Attributes.Type == RelationAttributeTypeImplicitManyToMany
}

// IsTwoWayEmbeddedManyToMany returns true if this is a two-way embedded many-to-many relation.
func (r Relation) IsTwoWayEmbeddedManyToMany() bool {
	return r.Attributes.Type == RelationAttributeTypeTwoWayEmbeddedManyToMany
}

// AsCompleteFields returns both fields if the relation has complete fields.
func (r Relation) AsCompleteFields() (*RelationFieldId, *RelationFieldId) {
	if r.Attributes.FieldA != nil && r.Attributes.FieldB != nil {
		return r.Attributes.FieldA, r.Attributes.FieldB
	}
	return nil, nil
}

// NewRelations creates a new empty Relations structure.
func NewRelations() Relations {
	return Relations{
		relationsStorage: make([]Relation, 0),
		fields:           make(map[RelationFieldId]RelationId),
		forward:           make([]RelationIndexEntry, 0),
		back:              make([]RelationIndexEntry, 0),
	}
}

// GetRelation returns the relation for the given RelationId.
func (r *Relations) GetRelation(id RelationId) *Relation {
	if int(id) < len(r.relationsStorage) {
		return &r.relationsStorage[id]
	}
	return nil
}

// GetRelationID returns the RelationId for the given RelationFieldId.
func (r *Relations) GetRelationID(fieldID RelationFieldId) (RelationId, bool) {
	id, ok := r.fields[fieldID]
	return id, ok
}

// PushRelation adds a new relation and returns its RelationId.
func (r *Relations) PushRelation(relation Relation) RelationId {
	id := RelationId(len(r.relationsStorage))
	r.relationsStorage = append(r.relationsStorage, relation)

	// Add to forward index
	r.forward = append(r.forward, RelationIndexEntry{
		ModelA:     relation.ModelA,
		ModelB:     relation.ModelB,
		RelationID: id,
	})

	// Add to back index
	r.back = append(r.back, RelationIndexEntry{
		ModelA:     relation.ModelB,
		ModelB:     relation.ModelA,
		RelationID: id,
	})

	// Sort indexes to maintain order
	r.sortIndexes()

	return id
}

// SetFieldRelation maps a RelationFieldId to a RelationId.
func (r *Relations) SetFieldRelation(fieldID RelationFieldId, relationID RelationId) {
	r.fields[fieldID] = relationID
}

// sortIndexes sorts the forward and back indexes.
func (r *Relations) sortIndexes() {
	sort.Slice(r.forward, func(i, j int) bool {
		a, b := r.forward[i], r.forward[j]
		if a.ModelA.FileID != b.ModelA.FileID {
			return a.ModelA.FileID < b.ModelA.FileID
		}
		if a.ModelA.ID != b.ModelA.ID {
			return a.ModelA.ID < b.ModelA.ID
		}
		if a.ModelB.FileID != b.ModelB.FileID {
			return a.ModelB.FileID < b.ModelB.FileID
		}
		if a.ModelB.ID != b.ModelB.ID {
			return a.ModelB.ID < b.ModelB.ID
		}
		return a.RelationID < b.RelationID
	})

	sort.Slice(r.back, func(i, j int) bool {
		a, b := r.back[i], r.back[j]
		if a.ModelA.FileID != b.ModelA.FileID {
			return a.ModelA.FileID < b.ModelA.FileID
		}
		if a.ModelA.ID != b.ModelA.ID {
			return a.ModelA.ID < b.ModelA.ID
		}
		if a.ModelB.FileID != b.ModelB.FileID {
			return a.ModelB.FileID < b.ModelB.FileID
		}
		if a.ModelB.ID != b.ModelB.ID {
			return a.ModelB.ID < b.ModelB.ID
		}
		return a.RelationID < b.RelationID
	})
}

// FindRelationsFromModel finds all relations where the given model is model A (forward relations).
func (r *Relations) FindRelationsFromModel(modelID ModelId) []RelationId {
	var result []RelationId
	for _, entry := range r.forward {
		if entry.ModelA == modelID {
			result = append(result, entry.RelationID)
		}
	}
	return result
}

// FindRelationsToModel finds all relations where the given model is model B (backward relations).
func (r *Relations) FindRelationsToModel(modelID ModelId) []RelationId {
	var result []RelationId
	for _, entry := range r.back {
		if entry.ModelA == modelID {
			result = append(result, entry.RelationID)
		}
	}
	return result
}

