package formatting

import (
	"fmt"
	"strings"

	parser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
)

// Reformat reformats a Prisma schema string.
// indentWidth specifies the number of spaces for indentation (defaults to 2 if 0).
func Reformat(input string, indentWidth int) (string, error) {
	// Parse the input first
	ast, err := parser.ParseSchema("schema.prisma", strings.NewReader(input))
	if err != nil {
		return "", fmt.Errorf("cannot reformat invalid schema: %w", err)
	}

	// Render the AST back to formatted string
	renderer := NewRenderer()
	return renderer.Render(ast), nil
}
