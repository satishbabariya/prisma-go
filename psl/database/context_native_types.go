// Package parserdatabase provides datasource-scoped attribute visiting for native types.
package database

import (
	"strings"

	"github.com/satishbabariya/prisma-go/psl/diagnostics"
)

// VisitDatasourceScoped looks for an optional attribute with a name of the form
// "<datasource_name>.<attribute_name>" (e.g., "db.Text"), returns the scope name,
// attribute name, and the attribute ID.
//
// Native type arguments are treated differently from arguments to other attributes:
// everywhere else, attributes are named with a default that can be first, but with
// native types, arguments are purely positional.
func (ctx *Context) VisitDatasourceScoped() (StringId, StringId, AttributeId, bool) {
	if ctx.attributes.attributes == nil {
		return 0, 0, AttributeId{}, false
	}

	var nativeTypeAttr *AttributeEntry
	var foundAttrID AttributeId

	// Find attributes with names containing '.'
	for _, entry := range ctx.iterAttributes() {
		if strings.Contains(entry.Attr.Name.Name, ".") {
			// Check if it's unused
			if _, unused := ctx.attributes.unusedAttributes[entry.ID]; !unused {
				continue
			}

			// Split on '.'
			parts := strings.SplitN(entry.Attr.Name.Name, ".", 2)
			if len(parts) != 2 {
				continue
			}

			datasourceName := parts[0]
			attrName := parts[1]

			dsID := ctx.interner.Intern(datasourceName)
			attrNameID := ctx.interner.Intern(attrName)

			// Check for duplicates
			if nativeTypeAttr != nil {
				// Convert lexer.Position to diagnostics.Span
				pos := entry.Attr.Pos
				span := diagnostics.NewSpan(pos.Offset, pos.Offset+len(datasourceName), diagnostics.FileIDZero)
				ctx.PushError(diagnostics.NewDuplicateAttributeError(
					datasourceName,
					span,
				))
				continue
			}

			nativeTypeAttr = &entry
			foundAttrID = entry.ID

			// Remove from unused
			delete(ctx.attributes.unusedAttributes, entry.ID)

			// Return the first matching attribute
			return dsID, attrNameID, foundAttrID, true
		}
	}

	return 0, 0, AttributeId{}, false
}
