// Package parserdatabase provides TopWalker for accessing top-level declarations.
package database

import (
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/ast"
)

// TopWalker provides access to any top-level declaration in the Prisma schema.
type TopWalker struct {
	db *ParserDatabase
	id TopId
}

// WalkTop creates a TopWalker for the given TopId.
func (pd *ParserDatabase) WalkTop(id TopId) *TopWalker {
	return &TopWalker{
		db: pd,
		id: id,
	}
}

// Name returns the name of the top-level declaration.
func (w *TopWalker) Name() string {
	top := w.AstTop()
	if top == nil {
		return ""
	}

	// Get name based on type
	// Top is a pointer to interface, so we need to dereference it
	topVal := *top
	if model := topVal.AsModel(); model != nil {
		return model.Name.Name
	}
	if enum := topVal.AsEnum(); enum != nil {
		return enum.Name.Name
	}
	if ct := topVal.AsCompositeType(); ct != nil {
		return ct.Name.Name
	}

	return ""
}

// FileID returns the ID of the file containing the top-level declaration.
func (w *TopWalker) FileID() diagnostics.FileID {
	return w.id.FileID
}

// AstTop returns the AST node for the top-level declaration.
func (w *TopWalker) AstTop() *ast.Top {
	file := w.db.asts.Get(w.id.FileID)
	if file == nil || int(w.id.ID) >= len(file.AST.Tops) {
		return nil
	}
	return &file.AST.Tops[w.id.ID]
}

// IsDefinedInFile returns whether the top-level declaration is defined in the given file.
func (w *TopWalker) IsDefinedInFile(fileID diagnostics.FileID) bool {
	return w.id.FileID == fileID
}

// WalkTops returns all top-level declarations in the schema.
func (pd *ParserDatabase) WalkTops() []*TopWalker {
	var result []*TopWalker
	for _, file := range pd.asts.files {
		for i := range file.AST.Tops {
			topID := TopId{
				FileID: file.FileID,
				ID:     uint32(i),
			}
			result = append(result, pd.WalkTop(topID))
		}
	}
	return result
}
