// Package parserdatabase provides attribute visiting functionality for Context.
package database

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	v2ast "github.com/satishbabariya/prisma-go/psl/parsing/v2/ast"
)

// VisitAttributes starts visiting an attribute container.
// This must be called before visiting individual attributes.
func (ctx *Context) VisitAttributes(container AttributeContainer) {
	if ctx.attributes.attributes != nil || len(ctx.attributes.unusedAttributes) > 0 {
		panic(fmt.Sprintf("visit_attributes called with %v while Context is still validating previous attribute set", container))
	}

	ctx.setAttributes(container)
}

// VisitOptionalSingleAttr visits an optional single attribute (like @@id).
// Returns true if the attribute was found and is valid.
func (ctx *Context) VisitOptionalSingleAttr(name string) bool {
	attrs := ctx.iterAttributes()
	var foundAttr *AttributeEntry

	// Find first matching attribute
	for _, entry := range attrs {
		if entry.Attr != nil && entry.Attr.GetName() == name {
			if foundAttr == nil {
				foundAttr = &entry
			} else {
				// Duplicate found - report error for all of them
				pos := entry.Attr.Pos
				span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(name), diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewDuplicateAttributeError(
					name,
					span,
				))
				// Remove from unused
				delete(ctx.attributes.unusedAttributes, entry.ID)
			}
		}
	}

	if foundAttr == nil {
		return false
	}

	// Check for duplicates
	hasDuplicates := false
	for _, entry := range attrs {
		if entry.Attr != nil && entry.Attr.GetName() == name && entry.ID != foundAttr.ID {
			hasDuplicates = true
			pos := entry.Attr.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(name), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewDuplicateAttributeError(
				name,
				span,
			))
			delete(ctx.attributes.unusedAttributes, entry.ID)
		}
	}

	if hasDuplicates {
		return false
	}

	// Remove from unused and set as current
	delete(ctx.attributes.unusedAttributes, foundAttr.ID)
	return ctx.setAttribute(foundAttr.ID, foundAttr.Attr)
}

// VisitRepeatedAttr visits a repeated attribute (like @@index).
// Returns true if an attribute was found and is valid.
func (ctx *Context) VisitRepeatedAttr(name string) bool {
	hasValidAttribute := false

	for !hasValidAttribute {
		var foundAttr *AttributeEntry

		// Find next unused attribute with this name
		for _, entry := range ctx.iterAttributes() {
			if entry.Attr != nil && entry.Attr.GetName() == name {
				if _, unused := ctx.attributes.unusedAttributes[entry.ID]; unused {
					foundAttr = &entry
					break
				}
			}
		}

		if foundAttr == nil {
			break
		}

		delete(ctx.attributes.unusedAttributes, foundAttr.ID)
		hasValidAttribute = ctx.setAttribute(foundAttr.ID, foundAttr.Attr)
	}

	return hasValidAttribute
}

// VisitDefaultArg visits a default argument (named or unnamed).
// Returns the expression and its index, or an error.
func (ctx *Context) VisitDefaultArg(name string) (v2ast.Expression, int, error) {
	nameID := ctx.interner.Intern(name)

	// Try named argument first
	namedIdx, hasNamed := ctx.attributes.args[&nameID]
	// Try unnamed argument
	unnamedIdx, hasUnnamed := ctx.attributes.args[nil]

	if hasNamed && !hasUnnamed {
		// Only named argument
		delete(ctx.attributes.args, &nameID)
		arg := ctx.argAt(namedIdx)
		if arg == nil {
			return nil, 0, fmt.Errorf("invalid argument index")
		}
		return arg.Value, namedIdx, nil
	} else if !hasNamed && hasUnnamed {
		// Only unnamed argument
		delete(ctx.attributes.args, nil)
		arg := ctx.argAt(unnamedIdx)
		if arg == nil {
			return nil, 0, fmt.Errorf("invalid argument index")
		}
		return arg.Value, unnamedIdx, nil
	} else if hasNamed && hasUnnamed {
		// Both present - error
		arg := ctx.argAt(namedIdx)
		if arg != nil && arg.Name != nil {
			pos := arg.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(name), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewDuplicateDefaultArgumentError(
				name,
				span,
			))
		}
		return nil, 0, fmt.Errorf("duplicate default argument")
	} else {
		// Neither present - error
		if attr := ctx.currentAttribute(); attr != nil {
			pos := attr.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(name), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewArgumentNotFoundError(
				name,
				span,
			))
		}
		return nil, 0, fmt.Errorf("argument %s not found", name)
	}
}

// VisitOptionalArg visits an optional argument.
// Returns the expression if found, nil otherwise.
func (ctx *Context) VisitOptionalArg(name string) v2ast.Expression {
	nameID := ctx.interner.Intern(name)
	idx, ok := ctx.attributes.args[&nameID]
	if !ok {
		return nil
	}
	delete(ctx.attributes.args, &nameID)
	arg := ctx.argAt(idx)
	return arg.Value
}

// VisitDefaultArgWithIdx visits a default argument (named or unnamed) and returns both the expression and its index.
// This is similar to VisitDefaultArg but also returns the argument index.
func (ctx *Context) VisitDefaultArgWithIdx(name string) (v2ast.Expression, int, error) {
	return ctx.VisitDefaultArg(name)
}

// ValidateVisitedArguments validates that all arguments were used.
// Must be called after validating an attribute's arguments.
func (ctx *Context) ValidateVisitedArguments() {
	if ctx.attributes.attribute == nil {
		panic("State error: missing attribute in validate_visited_arguments")
	}

	attr := ctx.currentAttribute()
	if attr != nil && attr.Arguments != nil {
		for _, argIdx := range ctx.attributes.args {
			if argIdx < len(attr.Arguments.Arguments) {
				arg := attr.Arguments.Arguments[argIdx]
				if arg != nil {
					pos := arg.Pos
					span := diagnostics.NewSpan(pos.Offset, pos.Offset+10, diagnostics.FileIDZero)
					ctx.PushError(diagnostics.NewUnusedArgumentError(span))
				}
			}
		}
	}

	ctx.discardArguments()
}

// ValidateVisitedAttributes validates that all attributes were used.
// Must be called after validating an attribute set.
func (ctx *Context) ValidateVisitedAttributes() {
	if len(ctx.attributes.args) > 0 || ctx.attributes.attribute != nil {
		panic("State error: validate_visited_attributes when an attribute is still under validation")
	}

	for attrID := range ctx.attributes.unusedAttributes {
		if attr := ctx.getAttribute(attrID); attr != nil {
			attrName := attr.GetName()
			pos := attr.Pos
			span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attrName), diagnostics.FileIDZero)
			ctx.PushError(diagnostics.NewAttributeNotKnownError(
				attrName,
				span,
			))
		}
	}

	ctx.attributes.attributes = nil
	ctx.attributes.unusedAttributes = make(map[AttributeId]bool)
}

// DiscardArguments discards arguments without validation.
func (ctx *Context) DiscardArguments() {
	ctx.discardArguments()
}

// CurrentAttributeID returns the current attribute ID being validated.
func (ctx *Context) CurrentAttributeID() AttributeId {
	if ctx.attributes.attribute == nil {
		panic("State error: no current attribute")
	}
	return *ctx.attributes.attribute
}

// CurrentAttribute returns the current attribute being validated.
func (ctx *Context) CurrentAttribute() *v2ast.Attribute {
	return ctx.currentAttribute()
}

// PushAttributeValidationError pushes an attribute validation error.
func (ctx *Context) PushAttributeValidationError(message string) {
	attr := ctx.currentAttribute()
	attrName := attr.GetName()
	pos := attr.Pos
	span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(attrName), diagnostics.FileIDZero)
	ctx.PushError(diagnostics.NewAttributeValidationError(
		message,
		"@"+attrName,
		span,
	))
}

// Private helper methods

// AttributeEntry represents an attribute with its ID.
type AttributeEntry struct {
	ID   AttributeId
	Attr *v2ast.Attribute
}

// setAttributes initializes the attribute validation state.
func (ctx *Context) setAttributes(container AttributeContainer) {
	ctx.attributes.attributes = &container
	ctx.attributes.unusedAttributes = make(map[AttributeId]bool)

	// Get attributes from the container
	attrs := ctx.getAttributesFromContainer(container)
	for i := range attrs {
		attrID := AttributeId{
			FileID:    container.FileID,
			Container: container,
			Index:     uint32(i),
		}
		ctx.attributes.unusedAttributes[attrID] = true
	}
}

// getAttributesFromContainer gets attributes from an attribute container.
// This is a simplified version - proper implementation would need to track
// container types (model, field, enum, etc.) in the AttributeContainer.
func (ctx *Context) getAttributesFromContainer(container AttributeContainer) []*v2ast.Attribute {
	// Find the file
	var file *FileEntry
	for i := range ctx.asts.files {
		if ctx.asts.files[i].FileID == container.FileID {
			file = &ctx.asts.files[i]
			break
		}
	}
	if file == nil {
		return nil
	}

	// Get attributes based on container ID
	// For now, we'll iterate through tops to find the right container
	// TODO: Implement proper container type tracking

	// Simplified: try to find model by index
	modelCount := 0
	for _, top := range file.AST.Tops {
		if model, ok := top.(*v2ast.Model); ok {
			if uint32(modelCount) == container.ID {
				// Convert BlockAttribute to Attribute
				result := make([]*v2ast.Attribute, len(model.BlockAttributes))
				for i, blockAttr := range model.BlockAttributes {
					if blockAttr != nil {
						// Create Attribute from BlockAttribute (they have the same structure)
						result[i] = &v2ast.Attribute{
							Pos:       blockAttr.Pos,
							Name:      blockAttr.Name,
							Arguments: blockAttr.Arguments,
						}
					}
				}
				return result
			}
			modelCount++
		}
	}

	return nil
}

// iterAttributes iterates over all attributes in the current container.
func (ctx *Context) iterAttributes() []AttributeEntry {
	if ctx.attributes.attributes == nil {
		return nil
	}

	container := *ctx.attributes.attributes
	attrs := ctx.getAttributesFromContainer(container)

	var result []AttributeEntry
	for i, attr := range attrs {
		if attr == nil {
			continue
		}
		attrID := AttributeId{
			FileID:    container.FileID,
			Container: container,
			Index:     uint32(i),
		}
		result = append(result, AttributeEntry{
			ID:   attrID,
			Attr: attr,
		})
	}
	return result
}

// setAttribute sets the current attribute being validated.
// Returns true if the attribute is valid enough to be usable.
func (ctx *Context) setAttribute(attrID AttributeId, attr *v2ast.Attribute) bool {
	if ctx.attributes.attribute != nil || len(ctx.attributes.args) > 0 {
		panic("State error: cannot start validating new arguments before validate_visited_arguments or discard_arguments has been called")
	}

	isReasonablyValid := true

	// Validate arguments
	ctx.attributes.attribute = &attrID
	ctx.attributes.args = make(map[*StringId]int)

	// Process arguments
	var unnamedArguments []string
	if attr.Arguments != nil {
		for i, arg := range attr.Arguments.Arguments {
			if arg == nil {
				continue
			}
		var argName *StringId
		if arg.Name != nil {
			nameID := ctx.interner.Intern(arg.Name.Name)
			argName = &nameID
		}

		// Check for duplicates
		if existingIdx, exists := ctx.attributes.args[argName]; exists {
			if argName == nil {
				// Unnamed argument duplicate
					if len(unnamedArguments) == 0 && existingIdx < len(attr.Arguments.Arguments) {
						existingArg := attr.Arguments.Arguments[existingIdx]
						if existingArg != nil {
					unnamedArguments = append(unnamedArguments, exprToString(existingArg.Value))
						}
				}
				unnamedArguments = append(unnamedArguments, exprToString(arg.Value))
			} else {
				// Named argument duplicate
					pos := arg.Pos
					span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(arg.Name.Name), diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewDuplicateArgumentError(
					arg.Name.Name,
						span,
				))
			}
		} else {
			ctx.attributes.args[argName] = i
			}
		}
	}

	if len(unnamedArguments) > 0 {
		ctx.PushAttributeValidationError(
			fmt.Sprintf("You provided multiple unnamed arguments. This is not possible. Did you forget the brackets? Did you mean `[%s]`?", strings.Join(unnamedArguments, ", ")),
		)
		isReasonablyValid = false
	}

	return isReasonablyValid
}

// discardArguments discards the current attribute's arguments.
func (ctx *Context) discardArguments() {
	ctx.attributes.attribute = nil
	ctx.attributes.args = make(map[*StringId]int)
}

// currentAttribute returns the current attribute being validated.
func (ctx *Context) currentAttribute() *v2ast.Attribute {
	if ctx.attributes.attribute == nil {
		panic("State error: no current attribute")
	}

	attrID := *ctx.attributes.attribute
	return ctx.getAttribute(attrID)
}

// getAttribute gets an attribute by its ID.
func (ctx *Context) getAttribute(attrID AttributeId) *v2ast.Attribute {
	attrs := ctx.getAttributesFromContainer(attrID.Container)
	if attrs != nil && int(attrID.Index) < len(attrs) {
		return attrs[attrID.Index]
	}
	return nil
}

// argAt returns an argument at the given index in the current attribute.
func (ctx *Context) argAt(idx int) *v2ast.Argument {
	attr := ctx.currentAttribute()
	if attr == nil || attr.Arguments == nil {
		return nil
	}
	if idx < len(attr.Arguments.Arguments) {
		return attr.Arguments.Arguments[idx]
	}
	return nil
}

// exprToString converts an expression to a string (simplified).
func exprToString(expr v2ast.Expression) string {
	// This is a simplified version - in reality we'd want proper formatting
	switch e := expr.(type) {
	case *v2ast.StringValue:
		return e.GetValue()
	case *v2ast.NumericValue:
		return e.Value
	case *v2ast.ConstantValue:
		return e.Value
	default:
		return "<expression>"
	}
}
