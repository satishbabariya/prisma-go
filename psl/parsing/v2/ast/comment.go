package ast

import (
	"github.com/alecthomas/participle/v2/lexer"
)

// Comment represents a documentation comment (///).
type Comment struct {
	Pos  lexer.Position
	Text string `@DocComment`
}

// CommentBlock represents multiple consecutive doc comments.
type CommentBlock struct {
	Comments []*Comment `@@*`
}

// GetText returns the combined text of all comments.
func (c *CommentBlock) GetText() string {
	if c == nil || len(c.Comments) == 0 {
		return ""
	}
	result := ""
	for i, comment := range c.Comments {
		// Strip the leading "///" from each comment
		text := comment.Text
		if len(text) >= 3 {
			text = text[3:] // Remove "///"
		}
		if i > 0 {
			result += "\n"
		}
		result += text
	}
	return result
}
