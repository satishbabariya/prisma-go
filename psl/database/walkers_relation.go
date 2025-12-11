// Package parserdatabase provides RelationWalker for accessing relation data.
package database

// RelationWalker provides access to a relation between models.
type RelationWalker struct {
	db *ParserDatabase
	id RelationId
}

// WalkRelation creates a RelationWalker for the given RelationId.
func (pd *ParserDatabase) WalkRelation(id RelationId) *RelationWalker {
	return &RelationWalker{
		db: pd,
		id: id,
	}
}

// Models returns the two models at each end of the relation.
func (w *RelationWalker) Models() [2]ModelId {
	rel := w.astRelation()
	return [2]ModelId{rel.ModelA, rel.ModelB}
}

// RelationFields returns the relation fields that define the relation.
func (w *RelationWalker) RelationFields() []*RelationFieldWalker {
	rel := w.astRelation()
	if rel == nil {
		return nil
	}

	var result []*RelationFieldWalker

	// Get fields from relation attributes based on type
	if rel.Attributes.FieldA != nil {
		result = append(result, w.db.WalkRelationField(*rel.Attributes.FieldA))
	}
	if rel.Attributes.FieldB != nil {
		result = append(result, w.db.WalkRelationField(*rel.Attributes.FieldB))
	}

	return result
}

// IsIgnored returns whether any field part of the relation is ignored.
func (w *RelationWalker) IsIgnored() bool {
	fields := w.RelationFields()
	for _, field := range fields {
		if field.IsIgnored() {
			return true
		}
		// TODO: Check referencing fields for ignored/unsupported
	}
	return false
}

// IsSelfRelation returns whether both ends of the relation are the same model.
func (w *RelationWalker) IsSelfRelation() bool {
	rel := w.astRelation()
	return rel.ModelA == rel.ModelB
}

// RelationKind returns a string describing the relation kind.
func (w *RelationWalker) RelationKind() string {
	rel := w.astRelation()
	if rel == nil {
		return "unknown"
	}

	switch rel.Attributes.Type {
	case RelationAttributeTypeImplicitManyToMany:
		return "implicit many-to-many"
	case RelationAttributeTypeTwoWayEmbeddedManyToMany:
		return "implicit many-to-many"
	case RelationAttributeTypeOneToOne:
		return "one-to-one"
	case RelationAttributeTypeOneToMany:
		return "one-to-many"
	default:
		return "unknown"
	}
}

// Refine converts the walker to a refined relation walker.
func (w *RelationWalker) Refine() *RefinedRelationWalker {
	rel := w.astRelation()
	if rel.IsImplicitManyToMany() {
		return &RefinedRelationWalker{
			IsImplicitManyToMany: true,
			ImplicitManyToMany: &ImplicitManyToManyRelationWalker{
				RelationWalker: w,
			},
		}
	} else if rel.IsTwoWayEmbeddedManyToMany() {
		return &RefinedRelationWalker{
			IsTwoWayEmbeddedManyToMany: true,
			TwoWayEmbeddedManyToMany: &TwoWayEmbeddedManyToManyRelationWalker{
				RelationWalker: w,
			},
		}
	} else {
		return &RefinedRelationWalker{
			IsInline: true,
			Inline: &InlineRelationWalker{
				db: w.db,
				id: w.id,
			},
		}
	}
}

// ExplicitRelationName returns the explicit relation name if present.
func (w *RelationWalker) ExplicitRelationName() *string {
	rel := w.astRelation()
	if rel.RelationName == nil {
		return nil
	}
	name := w.db.interner.Get(*rel.RelationName)
	return &name
}

// RelationName returns the relation name, explicit or inferred.
func (w *RelationWalker) RelationName() string {
	rel := w.astRelation()
	if rel.RelationName != nil {
		return w.db.interner.Get(*rel.RelationName)
	}
	// Generate name from model names
	modelA := w.db.WalkModel(rel.ModelA)
	modelB := w.db.WalkModel(rel.ModelB)
	return modelA.Name() + "To" + modelB.Name()
}

// astRelation returns the relation attributes.
func (w *RelationWalker) astRelation() *Relation {
	return w.db.relations.GetRelation(w.id)
}

// RefinedRelationWalker represents a relation that has been refined to a specific type.
type RefinedRelationWalker struct {
	IsInline                   bool
	Inline                     *InlineRelationWalker
	IsImplicitManyToMany       bool
	ImplicitManyToMany         *ImplicitManyToManyRelationWalker
	IsTwoWayEmbeddedManyToMany bool
	TwoWayEmbeddedManyToMany   *TwoWayEmbeddedManyToManyRelationWalker
}

// AsInline returns the inline relation walker if this is an inline relation.
func (r *RefinedRelationWalker) AsInline() *InlineRelationWalker {
	if r.IsInline {
		return r.Inline
	}
	return nil
}

// AsManyToMany returns the implicit many-to-many relation walker if this is a many-to-many relation.
func (r *RefinedRelationWalker) AsManyToMany() *ImplicitManyToManyRelationWalker {
	if r.IsImplicitManyToMany {
		return r.ImplicitManyToMany
	}
	return nil
}

// AsImplicitManyToMany returns an ImplicitManyToManyRelationWalker if this is an implicit many-to-many relation.
func (r *RefinedRelationWalker) AsImplicitManyToMany() *ImplicitManyToManyRelationWalker {
	return r.AsManyToMany()
}

// AsTwoWayEmbeddedManyToMany returns a TwoWayEmbeddedManyToManyRelationWalker if this is a two-way embedded many-to-many relation.
func (r *RefinedRelationWalker) AsTwoWayEmbeddedManyToMany() *TwoWayEmbeddedManyToManyRelationWalker {
	if r.IsTwoWayEmbeddedManyToMany {
		return r.TwoWayEmbeddedManyToMany
	}
	return nil
}

// InlineRelationWalker provides access to an explicitly defined 1:1 or 1:n relation.
type InlineRelationWalker struct {
	db *ParserDatabase
	id RelationId
}

// IsOneToOne returns whether this is a one-to-one relation.
func (w *InlineRelationWalker) IsOneToOne() bool {
	rel := w.get()
	if rel == nil {
		return false
	}
	return rel.Attributes.Type == RelationAttributeTypeOneToOne
}

// IsOneToMany returns whether this is a one-to-many relation.
func (w *InlineRelationWalker) IsOneToMany() bool {
	rel := w.get()
	if rel == nil {
		return false
	}
	return rel.Attributes.Type == RelationAttributeTypeOneToMany
}

// ReferencingModel returns the model which holds the relation arguments.
func (w *InlineRelationWalker) ReferencingModel() *ModelWalker {
	rel := w.get()
	return w.db.WalkModel(rel.ModelA)
}

// ReferencedModel returns the model referenced and which holds the back-relation field.
func (w *InlineRelationWalker) ReferencedModel() *ModelWalker {
	rel := w.get()
	return w.db.WalkModel(rel.ModelB)
}

// ForwardRelationField returns the forward relation field (on model A, the referencing model).
func (w *InlineRelationWalker) ForwardRelationField() *RelationFieldWalker {
	rel := w.get()
	if rel == nil {
		return nil
	}

	// For one-to-one and one-to-many, FieldA is the forward field
	if rel.Attributes.FieldA != nil {
		return w.db.WalkRelationField(*rel.Attributes.FieldA)
	}
	return nil
}

// BackRelationField returns the back relation field (on model B, the referenced model).
func (w *InlineRelationWalker) BackRelationField() *RelationFieldWalker {
	rel := w.get()
	if rel == nil {
		return nil
	}

	// For one-to-one and one-to-many, FieldB is the back field
	if rel.Attributes.FieldB != nil {
		return w.db.WalkRelationField(*rel.Attributes.FieldB)
	}
	return nil
}

// AsComplete converts to a complete inline relation walker if both sides are defined.
func (w *InlineRelationWalker) AsComplete() *CompleteInlineRelationWalker {
	forward := w.ForwardRelationField()
	back := w.BackRelationField()
	if forward != nil && back != nil {
		return &CompleteInlineRelationWalker{
			SideA: forward.id,
			SideB: back.id,
			db:    w.db,
		}
	}
	return nil
}

// MappedName returns the mapped name from the @relation attribute.
func (w *InlineRelationWalker) MappedName() *string {
	field := w.ForwardRelationField()
	if field == nil {
		return nil
	}
	rf := field.attributes()
	if rf == nil || rf.MappedName == nil {
		return nil
	}
	name := w.db.interner.Get(*rf.MappedName)
	return &name
}

// ConstraintName returns the constraint name (foreign key name) from the @map attribute on the relation field.
func (w *InlineRelationWalker) ConstraintName() *string {
	return w.MappedName()
}

// get returns the relation attributes.
func (w *InlineRelationWalker) get() *Relation {
	return w.astRelation()
}

// astRelation returns the relation attributes.
func (w *InlineRelationWalker) astRelation() *Relation {
	return w.db.relations.GetRelation(w.id)
}

// CompleteInlineRelationWalker provides access to a complete inline relation (both sides defined).
type CompleteInlineRelationWalker struct {
	SideA RelationFieldId
	SideB RelationFieldId
	db    *ParserDatabase
}

// SideAField returns the relation field on side A.
func (w *CompleteInlineRelationWalker) SideAField() *RelationFieldWalker {
	return w.db.WalkRelationField(w.SideA)
}

// SideBField returns the relation field on side B.
func (w *CompleteInlineRelationWalker) SideBField() *RelationFieldWalker {
	return w.db.WalkRelationField(w.SideB)
}

// ImplicitManyToManyRelationWalker provides access to an implicit many-to-many relation.
type ImplicitManyToManyRelationWalker struct {
	*RelationWalker
}

// FieldA returns the relation field on model A.
func (w *ImplicitManyToManyRelationWalker) FieldA() *RelationFieldWalker {
	rel := w.astRelation()
	if rel == nil || rel.Attributes.FieldA == nil {
		return nil
	}
	return w.db.WalkRelationField(*rel.Attributes.FieldA)
}

// FieldB returns the relation field on model B.
func (w *ImplicitManyToManyRelationWalker) FieldB() *RelationFieldWalker {
	rel := w.astRelation()
	if rel == nil || rel.Attributes.FieldB == nil {
		return nil
	}
	return w.db.WalkRelationField(*rel.Attributes.FieldB)
}

// TwoWayEmbeddedManyToManyRelationWalker provides access to a two-way embedded many-to-many relation.
type TwoWayEmbeddedManyToManyRelationWalker struct {
	*RelationWalker
}

// FieldA returns the relation field on model A.
func (w *TwoWayEmbeddedManyToManyRelationWalker) FieldA() *RelationFieldWalker {
	rel := w.astRelation()
	if rel == nil || rel.Attributes.FieldA == nil {
		return nil
	}
	return w.db.WalkRelationField(*rel.Attributes.FieldA)
}

// FieldB returns the relation field on model B.
func (w *TwoWayEmbeddedManyToManyRelationWalker) FieldB() *RelationFieldWalker {
	rel := w.astRelation()
	if rel == nil || rel.Attributes.FieldB == nil {
		return nil
	}
	return w.db.WalkRelationField(*rel.Attributes.FieldB)
}

// WalkRelations returns all relations in the schema.
func (pd *ParserDatabase) WalkRelations() []*RelationWalker {
	var result []*RelationWalker
	// Iterate through relations storage
	for i := uint32(0); i < uint32(len(pd.relations.relationsStorage)); i++ {
		result = append(result, pd.WalkRelation(RelationId(i)))
	}
	return result
}
