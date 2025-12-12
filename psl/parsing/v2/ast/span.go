// Package ast defines the Abstract Syntax Tree types for Prisma Schema Language.
package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Span represents a source location span with start and end positions.
type Span struct {
	Start Position
	End   Position
}

// Position represents a position in a source file.
type Position struct {
	Filename string
	Offset   int
	Line     int
	Column   int
}

// FromLexerPosition converts a participle lexer.Position to our Position type.
func FromLexerPosition(pos lexer.Position) Position {
	return Position{
		Filename: pos.Filename,
		Offset:   pos.Offset,
		Line:     pos.Line,
		Column:   pos.Column,
	}
}

// SpanFromPositions creates a Span from start and end lexer positions.
func SpanFromPositions(start, end lexer.Position) Span {
	return Span{
		Start: FromLexerPosition(start),
		End:   FromLexerPosition(end),
	}
}
