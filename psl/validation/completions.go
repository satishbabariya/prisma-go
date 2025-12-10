// Package pslcore provides completion utilities for IDE/LSP support.
package validation

import (
	"fmt"
	"strings"
)

// FormatCompletionDocs formats the documentation for a completion.
// example: How the completion is expected to be used.
// description: Description of what the completion does.
// params: Optional map of parameter labels to their documentation.
//
// Example:
//
//	doc := FormatCompletionDocs(
//		`relationMode = "foreignKeys" | "prisma"`,
//		"Sets the global relation mode for relations.",
//		nil,
//	)
func FormatCompletionDocs(example, description string, params map[string]string) string {
	var paramDocs strings.Builder

	if params != nil && len(params) > 0 {
		for paramLabel, paramDoc := range params {
			fmt.Fprintf(&paramDocs, "_@param_ %s %s\n", paramLabel, paramDoc)
		}
	}

	paramDocsStr := paramDocs.String()
	if paramDocsStr != "" {
		return fmt.Sprintf("```prisma\n%s\n```\n___\n%s\n\n%s", example, description, paramDocsStr)
	}
	return fmt.Sprintf("```prisma\n%s\n```\n___\n%s\n\n", example, description)
}

// CompletionItem represents a single completion item for IDE/LSP support.
type CompletionItem struct {
	Label            string
	Kind             CompletionItemKind
	Detail           *string
	Documentation    *string
	InsertText       *string
	InsertTextFormat *InsertTextFormat
}

// CompletionItemKind represents the kind of completion item.
type CompletionItemKind int

const (
	CompletionItemKindText CompletionItemKind = iota + 1
	CompletionItemKindMethod
	CompletionItemKindFunction
	CompletionItemKindConstructor
	CompletionItemKindField
	CompletionItemKindVariable
	CompletionItemKindClass
	CompletionItemKindInterface
	CompletionItemKindModule
	CompletionItemKindProperty
	CompletionItemKindUnit
	CompletionItemKindValue
	CompletionItemKindEnum
	CompletionItemKindKeyword
	CompletionItemKindSnippet
	CompletionItemKindColor
	CompletionItemKindFile
	CompletionItemKindReference
	CompletionItemKindFolder
	CompletionItemKindEnumMember
	CompletionItemKindConstant
	CompletionItemKindStruct
	CompletionItemKindEvent
	CompletionItemKindOperator
	CompletionItemKindTypeParameter
)

// InsertTextFormat represents the format of the insert text.
type InsertTextFormat int

const (
	InsertTextFormatPlainText InsertTextFormat = iota + 1
	InsertTextFormatSnippet
)

// CompletionList represents a list of completion items.
type CompletionList struct {
	Items []CompletionItem
}

// AddItem adds a completion item to the list.
func (cl *CompletionList) AddItem(item CompletionItem) {
	cl.Items = append(cl.Items, item)
}

// SchemaPosition represents a position in the schema for completions.
// This is a simplified version - full implementation would include more context.
type SchemaPosition struct {
	// Position details would be added here based on actual usage
	// For now, this is a placeholder structure
}
