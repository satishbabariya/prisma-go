package ast

import (
	"fmt"
	"strings"

	"github.com/satishbabariya/prisma-go/internal/debug"
	"github.com/satishbabariya/prisma-go/psl/diagnostics"
	"github.com/satishbabariya/prisma-go/psl/parsing/lexer"
)

// Parser parses Prisma schema tokens into an AST.
type Parser struct {
	tokens      []lexer.Token
	pos         int
	diagnostics *diagnostics.Diagnostics
}

// NewParser creates a new parser for the given tokens.
func NewParser(tokens []lexer.Token, diags *diagnostics.Diagnostics) *Parser {
	debug.Debug("Creating new parser", "token_count", len(tokens))
	return &Parser{
		tokens:      tokens,
		pos:         0,
		diagnostics: diags,
	}
}

// Parse parses the tokens into a SchemaAst.
func (p *Parser) Parse() *SchemaAst {
	debug.Debug("Starting parsing process")
	schema := &SchemaAst{Tops: []Top{}}
	var pendingBlockComment *Comment

	for !p.isAtEnd() {
		// Collect block comments before top-level declarations
		if p.check(lexer.TokenComment) {
			comment := p.parseCommentBlock()
			if comment != nil {
				// Check if next token is a top-level declaration
				peekPos := p.pos
				for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
					peekPos++
				}
				if peekPos < len(p.tokens) {
					nextToken := p.tokens[peekPos]
					if nextToken.Type == lexer.TokenModel || nextToken.Type == lexer.TokenEnum ||
						nextToken.Type == lexer.TokenTypeKeyword || nextToken.Type == lexer.TokenView ||
						nextToken.Type == lexer.TokenDatasource || nextToken.Type == lexer.TokenGenerator {
						pendingBlockComment = comment
						continue
					}
				}
			}
			// Skip standalone comments
			p.advance()
			continue
		}

		if p.isAtEnd() {
			break
		}

		if top := p.parseTopLevelWithComment(pendingBlockComment); top != nil {
			schema.Tops = append(schema.Tops, top)
			pendingBlockComment = nil // Reset after using
			debug.Debug("Parsed top-level item", "type", top.GetType(), "name", top.TopName())
		}
	}

	debug.Debug("Parsing completed", "top_level_items", len(schema.Tops))
	return schema
}

func (p *Parser) parseTopLevelWithComment(docComment *Comment) Top {
	if p.isAtEnd() {
		return nil
	}

	token := p.current()
	debug.Debug("Parsing top-level item", "token_type", token.Type, "token_value", token.Value)

	switch token.Type {
	case lexer.TokenGenerator:
		return p.parseGeneratorWithComment(docComment)
	case lexer.TokenDatasource:
		return p.parseDatasourceWithComment(docComment)
	case lexer.TokenModel:
		return p.parseModelWithComment(docComment)
	case lexer.TokenEnum:
		return p.parseEnumWithComment(docComment)
	case lexer.TokenTypeKeyword:
		return p.parseCompositeTypeWithComment(docComment)
	case lexer.TokenView:
		return p.parseViewWithComment(docComment)
	case lexer.TokenEOF:
		return nil
	default:
		// Handle invalid top-level constructs
		// Check if this might be a type alias (type Name = ...) which is invalid
		if token.Type == lexer.TokenTypeKeyword {
			peekPos := p.pos + 1
			for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
				peekPos++
			}
			if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenIdentifier {
				peekPos2 := peekPos + 1
				for peekPos2 < len(p.tokens) && p.tokens[peekPos2].Type == lexer.TokenComment {
					peekPos2++
				}
				if peekPos2 < len(p.tokens) && p.tokens[peekPos2].Type == lexer.TokenEquals {
					// This is a type alias, which is invalid
					p.error("Invalid type definition. Please check the documentation in https://pris.ly/d/composite-types")
					// Skip the type alias
					p.advance() // skip 'type'
					if p.check(lexer.TokenIdentifier) {
						p.advance() // skip identifier
					}
					if p.check(lexer.TokenEquals) {
						p.advance() // skip '='
					}
					// Skip the rest until we find something recognizable
					for !p.isAtEnd() && !p.check(lexer.TokenModel) && !p.check(lexer.TokenEnum) &&
						!p.check(lexer.TokenTypeKeyword) && !p.check(lexer.TokenView) &&
						!p.check(lexer.TokenDatasource) && !p.check(lexer.TokenGenerator) &&
						!p.check(lexer.TokenRBrace) {
						p.advance()
					}
					return nil
				}
			}
		}

		// Check if this might be an arbitrary block (identifier { ... })
		if token.Type == lexer.TokenIdentifier {
			peekPos := p.pos + 1
			for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
				peekPos++
			}
			if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenLBrace {
				// This is an arbitrary block without a keyword
				p.error("This block is invalid. It does not start with any known Prisma schema keyword. Valid keywords include 'model', 'enum', 'type', 'datasource' and 'generator'.")
				// Skip the block
				p.advance() // skip identifier
				p.expect(lexer.TokenLBrace)
				braceDepth := 1
				for braceDepth > 0 && !p.isAtEnd() {
					if p.check(lexer.TokenLBrace) {
						braceDepth++
						p.advance()
					} else if p.check(lexer.TokenRBrace) {
						braceDepth--
						p.advance()
					} else {
						p.advance()
					}
				}
				return nil
			}
		}

		// Generic catch-all for invalid lines
		p.error("This line is invalid. It does not start with any known Prisma schema keyword.")
		// Advance past the invalid token to continue parsing
		p.advance()
		return nil
	}
}

func (p *Parser) parseTopLevel() Top {
	return p.parseTopLevelWithComment(nil)
}

func (p *Parser) parseGenerator() Top {
	return p.parseGeneratorWithComment(nil)
}

func (p *Parser) parseGeneratorWithComment(docComment *Comment) Top {
	debug.Debug("Parsing generator block")
	p.expect(lexer.TokenGenerator)
	name := p.expect(lexer.TokenIdentifier)
	debug.Debug("Generator name", "name", name.Value)

	p.expect(lexer.TokenLBrace)

	properties := []ConfigBlockProperty{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Skip comments
		if p.check(lexer.TokenComment) {
			p.advance()
			continue
		}

		if p.check(lexer.TokenRBrace) || p.isAtEnd() {
			break
		}

		if prop := p.parseConfigProperty(); prop != nil {
			properties = append(properties, *prop)
			debug.Debug("Parsed generator property", "name", prop.Name.Name)
		} else {
			// BLOCK_LEVEL_CATCH_ALL - invalid line in config block
			if !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
				msg := fmt.Sprintf("This line is not a valid definition within a %s.", "generator")
				p.error(msg)
				// Advance past the invalid token to continue parsing
				p.advance()
			}
		}
	}

	p.expect(lexer.TokenRBrace)

	generator := &GeneratorConfig{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Properties:    properties,
		Documentation: docComment,
		ASTSpan:       p.spanFrom(name),
	}
	debug.Debug("Completed parsing generator", "property_count", len(properties))
	return generator
}

func (p *Parser) parseDatasource() Top {
	return p.parseDatasourceWithComment(nil)
}

func (p *Parser) parseDatasourceWithComment(docComment *Comment) Top {
	debug.Debug("Parsing datasource block")
	p.expect(lexer.TokenDatasource)
	name := p.expect(lexer.TokenIdentifier)
	debug.Debug("Datasource name", "name", name.Value)

	p.expect(lexer.TokenLBrace)
	innerSpanStartToken := p.current()

	properties := []ConfigBlockProperty{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Skip comments
		if p.check(lexer.TokenComment) {
			p.advance()
			continue
		}

		if prop := p.parseConfigProperty(); prop != nil {
			properties = append(properties, *prop)
			debug.Debug("Parsed datasource property", "name", prop.Name.Name)
		} else {
			// BLOCK_LEVEL_CATCH_ALL - invalid line in config block
			if !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
				msg := fmt.Sprintf("This line is not a valid definition within a %s.", "datasource")
				p.error(msg)
				// Advance past the invalid token to continue parsing
				p.advance()
			}
		}
	}

	p.expect(lexer.TokenRBrace)
	innerSpanEndToken := p.previous()

	// Calculate inner span
	innerSpan := diagnostics.NewSpan(
		innerSpanStartToken.Column,
		innerSpanEndToken.Column+len(innerSpanEndToken.Value),
		diagnostics.FileIDZero,
	)

	datasource := &SourceConfig{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Properties:    properties,
		Documentation: docComment,
		InnerSpan:     innerSpan,
		ASTSpan:       p.spanFrom(name),
	}
	debug.Debug("Completed parsing datasource", "property_count", len(properties))
	return datasource
}

func (p *Parser) parseModel() Top {
	return p.parseModelWithComment(nil)
}

func (p *Parser) parseModelWithComment(docComment *Comment) Top {
	debug.Debug("Parsing model block")
	p.expect(lexer.TokenModel)
	name := p.expect(lexer.TokenIdentifier)
	debug.Debug("Model name", "name", name.Value)

	p.expect(lexer.TokenLBrace)

	fields := []Field{}
	attributes := []Attribute{}
	var pendingFieldComment *Comment

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Check for block comment before field
		if p.check(lexer.TokenComment) {
			comment := p.parseCommentBlock()
			if comment != nil {
				// Check if next token is a field or attribute
				peekPos := p.pos + 1
				for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
					peekPos++
				}
				if peekPos < len(p.tokens) {
					nextToken := p.tokens[peekPos]
					if nextToken.Type == lexer.TokenIdentifier || nextToken.Type == lexer.TokenAt {
						pendingFieldComment = comment
						p.advance()
						continue
					}
				}
			}
			p.advance()
			continue
		}

		if p.check(lexer.TokenRBrace) {
			break
		}

		// Check for model-level attributes (@@)
		if p.check(lexer.TokenAt) && p.peek().Type == lexer.TokenAt {
			// This is a model-level attribute (e.g., @@id, @@index)
			// Consume both @ tokens - we've already verified the next token is also @
			p.advance() // consume first @
			p.advance() // consume second @
			debug.Debug("Parsing model-level attribute")
			// Skip any comments/whitespace (handled by lexer, but check for comments)
			for p.check(lexer.TokenComment) {
				p.advance()
			}
			// Now parse as a regular attribute (but it's model-level)
			// After consuming both @ tokens, the next token should be the attribute name
			if attr := p.parseAttributeAfterAt(); attr != nil {
				// Check for trailing comment on block attribute (matching Rust grammar)
				var trailingComment *Comment
				if p.check(lexer.TokenComment) {
					trailingComment = p.parseTrailingComment()
					if trailingComment != nil {
						p.advance()
					}
				}
				// Note: Rust doesn't store trailing comments on attributes, but we parse them for completeness
				attributes = append(attributes, *attr)
				debug.Debug("Parsed model attribute", "name", attr.Name.Name)
			} else {
				// If parsing failed, advance to avoid infinite loop
				if !p.isAtEnd() && !p.check(lexer.TokenRBrace) {
					p.advance()
				}
			}
		} else if field := p.parseFieldWithComment(pendingFieldComment); field != nil {
			// Try to parse as a field (field-level @ attributes are handled in parseField)
			fields = append(fields, *field)
			pendingFieldComment = nil // Reset after using
			debug.Debug("Parsed model field", "name", field.Name.Name, "type", field.FieldType.Name())
		} else {
			// If parsing failed, advance to avoid infinite loop
			p.advance()
		}
	}

	p.expect(lexer.TokenRBrace)

	model := &Model{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Fields:        fields,
		Attributes:    attributes,
		Documentation: docComment,
		ASTSpan:       p.spanFrom(name),
	}
	debug.Debug("Completed parsing model", "field_count", len(fields), "attribute_count", len(attributes))
	return model
}

func (p *Parser) parseView() Top {
	return p.parseViewWithComment(nil)
}

func (p *Parser) parseViewWithComment(docComment *Comment) Top {
	debug.Debug("Parsing view block")
	p.expect(lexer.TokenView)
	name := p.expect(lexer.TokenIdentifier)
	debug.Debug("View name", "name", name.Value)

	p.expect(lexer.TokenLBrace)

	fields := []Field{}
	attributes := []Attribute{}
	var pendingFieldComment *Comment

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Check for block comment before field
		if p.check(lexer.TokenComment) {
			comment := p.parseCommentBlock()
			if comment != nil {
				peekPos := p.pos + 1
				for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
					peekPos++
				}
				if peekPos < len(p.tokens) {
					nextToken := p.tokens[peekPos]
					if nextToken.Type == lexer.TokenIdentifier || nextToken.Type == lexer.TokenAt {
						pendingFieldComment = comment
						p.advance()
						continue
					}
				}
			}
			p.advance()
			continue
		}

		if p.check(lexer.TokenRBrace) {
			break
		}

		// Check for view-level attributes (@@)
		if p.check(lexer.TokenAt) && p.peek().Type == lexer.TokenAt {
			// This is a view-level attribute
			// Consume both @ tokens
			p.advance() // consume first @
			p.advance() // consume second @
			debug.Debug("Parsing view-level attribute")
			// Skip any comments/whitespace
			for p.check(lexer.TokenComment) {
				p.advance()
			}
			// Now parse as a regular attribute
			if attr := p.parseAttributeAfterAt(); attr != nil {
				attributes = append(attributes, *attr)
				debug.Debug("Parsed view attribute", "name", attr.Name.Name)
			} else {
				// If parsing failed, advance to avoid infinite loop
				if !p.isAtEnd() && !p.check(lexer.TokenRBrace) {
					p.advance()
				}
			}
		} else if field := p.parseFieldWithComment(pendingFieldComment); field != nil {
			// Try to parse as a field
			fields = append(fields, *field)
			pendingFieldComment = nil // Reset after using
			debug.Debug("Parsed view field", "name", field.Name.Name, "type", field.FieldType.Name())
		} else {
			// If parsing failed, advance to avoid infinite loop
			p.advance()
		}
	}

	p.expect(lexer.TokenRBrace)

	view := &Model{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Fields:        fields,
		Attributes:    attributes,
		IsView:        true,
		Documentation: docComment,
		ASTSpan:       p.spanFrom(name),
	}
	debug.Debug("Completed parsing view", "field_count", len(fields), "attribute_count", len(attributes))
	return view
}

func (p *Parser) parseEnum() Top {
	return p.parseEnumWithComment(nil)
}

func (p *Parser) parseEnumWithComment(docComment *Comment) Top {
	p.expect(lexer.TokenEnum)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	values := []EnumValue{}
	attributes := []Attribute{}
	var pendingValueComment *Comment
	innerSpanStartToken := p.current()

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Check for block comment before enum value
		if p.check(lexer.TokenComment) {
			comment := p.parseCommentBlock()
			if comment != nil {
				peekPos := p.pos + 1
				for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
					peekPos++
				}
				if peekPos < len(p.tokens) {
					nextToken := p.tokens[peekPos]
					if nextToken.Type == lexer.TokenIdentifier || nextToken.Type == lexer.TokenAt {
						pendingValueComment = comment
						p.advance()
						continue
					}
				}
			}
			p.advance()
			continue
		}

		if p.check(lexer.TokenRBrace) {
			break
		}

		// Check for enum-level attributes (@@)
		if p.check(lexer.TokenAt) && p.peek().Type == lexer.TokenAt {
			p.advance() // consume first @
			p.advance() // consume second @
			for p.check(lexer.TokenComment) {
				p.advance()
			}
			if attr := p.parseAttributeAfterAt(); attr != nil {
				attributes = append(attributes, *attr)
			}
		} else if value := p.parseEnumValueWithComment(pendingValueComment); value != nil {
			values = append(values, *value)
			pendingValueComment = nil // Reset after using
		} else {
			// BLOCK_LEVEL_CATCH_ALL - invalid line in enum block
			if !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
				p.error("This line is not an enum value definition.")
				// Advance past the invalid token to continue parsing
				p.advance()
			}
		}
	}

	p.expect(lexer.TokenRBrace)
	innerSpanEndToken := p.previous()

	// Calculate inner span (from opening brace to closing brace)
	innerSpan := diagnostics.NewSpan(
		innerSpanStartToken.Column,
		innerSpanEndToken.Column+len(innerSpanEndToken.Value),
		diagnostics.FileIDZero,
	)

	return &Enum{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Values:        values,
		Attributes:    attributes,
		Documentation: docComment,
		InnerSpan:     innerSpan,
		ASTSpan:       p.spanFrom(name),
	}
}

func (p *Parser) parseCompositeType() Top {
	return p.parseCompositeTypeWithComment(nil)
}

func (p *Parser) parseCompositeTypeWithComment(docComment *Comment) Top {
	p.expect(lexer.TokenTypeKeyword)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	fields := []Field{}
	attributes := []Attribute{}
	var pendingFieldComment *Comment
	innerSpanStartToken := p.current() // Token after opening brace

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Check for block comment before field
		if p.check(lexer.TokenComment) {
			comment := p.parseCommentBlock()
			if comment != nil {
				peekPos := p.pos + 1
				for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
					peekPos++
				}
				if peekPos < len(p.tokens) {
					nextToken := p.tokens[peekPos]
					if nextToken.Type == lexer.TokenIdentifier || nextToken.Type == lexer.TokenAt {
						pendingFieldComment = comment
						p.advance()
						continue
					}
				}
			}
			p.advance()
			continue
		}

		if p.check(lexer.TokenRBrace) {
			break
		}

		// Check for composite type-level attributes (@@)
		if p.check(lexer.TokenAt) && p.peek().Type == lexer.TokenAt {
			// This is a composite type-level attribute
			// Consume both @ tokens
			p.advance() // consume first @
			p.advance() // consume second @
			// Validate that composite types can't have certain attributes
			peekToken := p.current()
			if peekToken.Type == lexer.TokenIdentifier {
				attrName := peekToken.Value
				switch attrName {
				case "map", "unique", "index", "fulltext", "id":
					p.error(fmt.Sprintf("A composite type cannot have block-level attribute '@@%s'", attrName))
				}
			}
			// Now parse as a regular attribute
			if attr := p.parseAttributeAfterAt(); attr != nil {
				// Additional validation for composite type attributes
				if attr.Name.Name == "map" || attr.Name.Name == "unique" || attr.Name.Name == "index" ||
					attr.Name.Name == "fulltext" || attr.Name.Name == "id" {
					p.error(fmt.Sprintf("A composite type cannot have block-level attribute '@@%s'", attr.Name.Name))
				}
				// Check for trailing comment on block attribute (matching Rust grammar)
				var trailingComment *Comment
				if p.check(lexer.TokenComment) {
					trailingComment = p.parseTrailingComment()
					if trailingComment != nil {
						p.advance()
					}
				}
				// Note: Rust doesn't store trailing comments on attributes, but we parse them for completeness
				attributes = append(attributes, *attr)
			}
		} else if field := p.parseFieldWithComment(pendingFieldComment); field != nil {
			// Validate field attributes for composite types
			for _, attr := range field.Attributes {
				switch attr.Name.Name {
				case "relation", "unique", "id":
					p.error(fmt.Sprintf("Defining `@%s` attribute for a field in a composite type is not allowed.", attr.Name.Name))
				}
			}
			fields = append(fields, *field)
			pendingFieldComment = nil // Reset after using
		} else {
			// BLOCK_LEVEL_CATCH_ALL - invalid line in composite type block
			if !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
				p.error("This line is not a valid field or attribute definition.")
				// Advance past the invalid token to continue parsing
				p.advance()
			}
		}
	}

	p.expect(lexer.TokenRBrace)
	innerSpanEndToken := p.previous()

	// Calculate inner span (from opening brace to closing brace)
	innerSpan := diagnostics.NewSpan(
		innerSpanStartToken.Column,
		innerSpanEndToken.Column+len(innerSpanEndToken.Value),
		diagnostics.FileIDZero,
	)

	return &CompositeType{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Fields:        fields,
		Attributes:    attributes,
		Documentation: docComment,
		InnerSpan:     innerSpan,
		ASTSpan:       p.spanFrom(name),
	}
}

func (p *Parser) parseField() *Field {
	return p.parseFieldWithComment(nil)
}

func (p *Parser) parseFieldWithComment(docComment *Comment) *Field {
	name := p.expect(lexer.TokenIdentifier)
	debug.Debug("Parsing field", "name", name.Value)

	// Check for legacy colon (:)
	if p.check(lexer.TokenColon) {
		p.error("Field declarations don't require a `:`.")
		p.advance() // consume the colon
	}

	// Field type is optional (for autocompletion), but we'll try to parse it
	var fieldType FieldType
	var arity FieldArity
	if !p.check(lexer.TokenAt) && !p.check(lexer.TokenComment) && !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		fieldType, arity = p.parseFieldType()
	} else {
		// Missing field type - this is invalid but we parse it for better error messages
		p.error("This field declaration is invalid. It is either missing a name or a type.")
		fieldType = FieldType{
			Type:    UnsupportedFieldType{TypeName: ""},
			ASTSpan: diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		}
		arity = Required
	}

	attributes := []Attribute{}
	// Parse field-level attributes (@), but stop if we see model-level attributes (@@)
	for p.check(lexer.TokenAt) && p.peek().Type != lexer.TokenAt {
		if attr := p.parseAttribute(); attr != nil {
			attributes = append(attributes, *attr)
			debug.Debug("Parsed field attribute", "name", attr.Name.Name)
		} else {
			break
		}
	}

	// Check for trailing comment
	var trailingComment *Comment
	if p.check(lexer.TokenComment) {
		trailingComment = p.parseTrailingComment()
		if trailingComment != nil {
			p.advance()
		}
	}

	// Merge block comment and trailing comment if both exist
	var finalComment *Comment
	if docComment != nil && trailingComment != nil {
		finalComment = &Comment{
			Text: docComment.Text + "\n" + trailingComment.Text,
			Span: docComment.Span,
		}
	} else if docComment != nil {
		finalComment = docComment
	} else if trailingComment != nil {
		finalComment = trailingComment
	}

	field := &Field{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		FieldType:     fieldType,
		Arity:         arity,
		Attributes:    attributes,
		Documentation: finalComment,
		ASTSpan:       p.spanFrom(name),
	}
	debug.Debug("Completed parsing field", "arity", arity, "attribute_count", len(attributes))
	return field
}

func (p *Parser) parseFieldType() (FieldType, FieldArity) {
	// Parse field type following Rust order:
	// unsupported_optional_list_type | list_type | optional_type | legacy_required_type | legacy_list_type | base_type

	// Check for unsupported_optional_list_type (Type[]?)
	// This must be checked first before checking list_type
	if p.check(lexer.TokenIdentifier) {
		peekPos := p.pos + 1
		// Skip comments
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenLBracket {
			peekPos++
			for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
				peekPos++
			}
			if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenRBracket {
				peekPos++
				for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
					peekPos++
				}
				if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenQuestion {
					// This is unsupported_optional_list_type
					p.error("Optional lists are not supported. Use either `Type[]` or `Type?`.")
					// Parse it anyway for error recovery
					typeNameToken := p.expect(lexer.TokenIdentifier)
					p.expect(lexer.TokenLBracket)
					p.expect(lexer.TokenRBracket)
					p.expect(lexer.TokenQuestion)
					return FieldType{
						Type:    SupportedFieldType{Identifier: Identifier{Name: typeNameToken.Value, ASTSpan: p.spanForToken(typeNameToken)}},
						ASTSpan: p.spanForToken(typeNameToken),
					}, List
				}
			}
		}
	}

	// Check for list_type (Type[])
	if p.check(lexer.TokenIdentifier) {
		peekPos := p.pos + 1
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenLBracket {
			peekPos++
			for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
				peekPos++
			}
			if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenRBracket {
				// This is list_type
				typeNameToken := p.expect(lexer.TokenIdentifier)
				p.expect(lexer.TokenLBracket)
				p.expect(lexer.TokenRBracket)
				return FieldType{
					Type:    SupportedFieldType{Identifier: Identifier{Name: typeNameToken.Value, ASTSpan: p.spanForToken(typeNameToken)}},
					ASTSpan: p.spanForToken(typeNameToken),
				}, List
			}
		}
	}

	// Check for optional_type (Type?)
	if p.check(lexer.TokenIdentifier) {
		peekPos := p.pos + 1
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenQuestion {
			// This is optional_type
			typeNameToken := p.expect(lexer.TokenIdentifier)
			p.expect(lexer.TokenQuestion)
			return FieldType{
				Type:    SupportedFieldType{Identifier: Identifier{Name: typeNameToken.Value, ASTSpan: p.spanForToken(typeNameToken)}},
				ASTSpan: p.spanForToken(typeNameToken),
			}, Optional
		}
	}

	// Check for legacy_required_type (Type!)
	if p.check(lexer.TokenIdentifier) {
		peekPos := p.pos + 1
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenExclamation {
			// This is legacy_required_type
			p.error("Fields are required by default, `!` is no longer required.")
			typeNameToken := p.expect(lexer.TokenIdentifier)
			p.expect(lexer.TokenExclamation)
			return FieldType{
				Type:    SupportedFieldType{Identifier: Identifier{Name: typeNameToken.Value, ASTSpan: p.spanForToken(typeNameToken)}},
				ASTSpan: p.spanForToken(typeNameToken),
			}, Required
		}
	}

	// Check for legacy_list_type ([Type])
	if p.check(lexer.TokenLBracket) {
		peekPos := p.pos + 1
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenIdentifier {
			peekPos2 := peekPos + 1
			for peekPos2 < len(p.tokens) && p.tokens[peekPos2].Type == lexer.TokenComment {
				peekPos2++
			}
			if peekPos2 < len(p.tokens) && p.tokens[peekPos2].Type == lexer.TokenRBracket {
				// Check if this is followed by [] (new syntax) or not (legacy)
				peekPos3 := peekPos2 + 1
				for peekPos3 < len(p.tokens) && p.tokens[peekPos3].Type == lexer.TokenComment {
					peekPos3++
				}
				if peekPos3 >= len(p.tokens) || p.tokens[peekPos3].Type != lexer.TokenLBracket {
					// This is legacy_list_type
					p.error("To specify a list, please use `Type[]` instead of `[Type]`.")
					p.expect(lexer.TokenLBracket)
					typeNameToken := p.expect(lexer.TokenIdentifier)
					p.expect(lexer.TokenRBracket)
					return FieldType{
						Type:    SupportedFieldType{Identifier: Identifier{Name: typeNameToken.Value, ASTSpan: p.spanForToken(typeNameToken)}},
						ASTSpan: p.spanForToken(typeNameToken),
					}, List
				}
			}
		}
	}

	// Parse base_type (unsupported_type | identifier)
	// Check for unsupported_type: Unsupported("...")
	if p.check(lexer.TokenIdentifier) && strings.ToLower(p.current().Value) == "unsupported" {
		unsupportedToken := p.current()
		peekPos := p.pos + 1
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenLParen {
			// This is Unsupported("...")
			p.advance() // consume "Unsupported"
			p.expect(lexer.TokenLParen)
			if p.check(lexer.TokenString) {
				stringToken := p.expect(lexer.TokenString)
				p.expect(lexer.TokenRParen)
				// Parse the string literal
				parsedValue := p.parseStringLiteral(stringToken.Value)
				return FieldType{
					Type:    UnsupportedFieldType{TypeName: parsedValue},
					ASTSpan: p.spanForToken(unsupportedToken),
				}, Required
			} else {
				p.error("Expected string literal in Unsupported() type")
				p.expect(lexer.TokenRParen)
				return FieldType{
					Type:    UnsupportedFieldType{TypeName: ""},
					ASTSpan: p.spanForToken(unsupportedToken),
				}, Required
			}
		}
	}

	// Parse regular identifier (base type)
	typeNameToken := p.expect(lexer.TokenIdentifier)
	typeName := Identifier{Name: typeNameToken.Value, ASTSpan: p.spanForToken(typeNameToken)}

	fieldType := FieldType{
		Type:    SupportedFieldType{Identifier: typeName},
		ASTSpan: p.spanForToken(typeNameToken),
	}
	debug.Debug("Parsed field type", "type_name", typeName.Name, "arity", Required)
	return fieldType, Required
}

func (p *Parser) parseEnumValue() *EnumValue {
	return p.parseEnumValueWithComment(nil)
}

func (p *Parser) parseEnumValueWithComment(docComment *Comment) *EnumValue {
	name := p.expect(lexer.TokenIdentifier)

	attributes := []Attribute{}
	for p.check(lexer.TokenAt) && p.peek().Type != lexer.TokenAt {
		if attr := p.parseAttribute(); attr != nil {
			attributes = append(attributes, *attr)
		} else {
			break
		}
	}

	// Check for trailing comment
	var trailingComment *Comment
	if p.check(lexer.TokenComment) {
		trailingComment = p.parseTrailingComment()
		if trailingComment != nil {
			p.advance()
		}
	}

	// Merge block comment and trailing comment if both exist
	var finalComment *Comment
	if docComment != nil && trailingComment != nil {
		finalComment = &Comment{
			Text: docComment.Text + "\n" + trailingComment.Text,
			Span: docComment.Span,
		}
	} else if docComment != nil {
		finalComment = docComment
	} else if trailingComment != nil {
		finalComment = trailingComment
	}

	return &EnumValue{
		Name:          Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Attributes:    attributes,
		Documentation: finalComment,
		ASTSpan:       p.spanForToken(name),
	}
}

func (p *Parser) parseAttribute() *Attribute {
	debug.Debug("Parsing field-level attribute")
	p.expect(lexer.TokenAt)
	return p.parseAttributeAfterAt()
}

// parseAttributeAfterAt parses an attribute after the @ token has been consumed.
// This is used for both field-level (@attr) and model-level (@@attr) attributes.
func (p *Parser) parseAttributeAfterAt() *Attribute {
	// Skip comments before identifier
	for p.check(lexer.TokenComment) {
		p.advance()
	}

	if !p.check(lexer.TokenIdentifier) {
		currentToken := p.current()
		p.error(fmt.Sprintf("Expected attribute name after '@', but got token type %d (%s)", currentToken.Type, currentToken.Value))
		return nil
	}
	nameToken := p.expect(lexer.TokenIdentifier)
	attributeName := nameToken.Value
	debug.Debug("Attribute name", "name", attributeName)

	argsList := ArgumentsList{
		Arguments:      []Argument{},
		EmptyArguments: []EmptyArgument{},
		TrailingComma:  nil,
	}

	// Handle dotted attribute names like @db.VarChar or @db.Timestamp(6)
	// Path can be recursive: identifier ~ ("." ~ path?)*
	// The database layer expects the full dotted name (e.g., "db.VarChar") as the attribute name
	for p.check(lexer.TokenDot) {
		p.advance() // consume the dot

		// Parse the identifier after the dot and append it to the attribute name
		if p.check(lexer.TokenIdentifier) {
			typeNameToken := p.expect(lexer.TokenIdentifier)
			attributeName = attributeName + "." + typeNameToken.Value
			debug.Debug("Parsed dotted attribute name", "full_name", attributeName)
		} else {
			p.error("Expected identifier after '.' in attribute")
			return nil
		}
	}

	// Create the name identifier with the full name (potentially dotted)
	name := Identifier{
		Name:    attributeName,
		ASTSpan: p.spanForToken(nameToken),
	}

	if p.check(lexer.TokenLParen) {
		debug.Debug("Parsing attribute arguments")
		p.expect(lexer.TokenLParen)

		for !p.check(lexer.TokenRParen) && !p.isAtEnd() {
			// Skip comments in arguments
			for p.check(lexer.TokenComment) {
				p.advance()
			}

			if p.check(lexer.TokenRParen) {
				break
			}

			// Check for empty argument (name: without value)
			// This happens when we have "name:" followed by comma or closing paren
			if p.check(lexer.TokenIdentifier) {
				nameToken := p.current()
				peekToken := p.peek()

				if peekToken.Type == lexer.TokenColon {
					// We have "name:", check what comes after
					p.advance() // consume identifier
					p.advance() // consume colon

					// Skip whitespace/comments
					for p.check(lexer.TokenComment) {
						p.advance()
					}

					// Check if there's a value or if it's empty
					if p.check(lexer.TokenRParen) || p.check(lexer.TokenComma) {
						// This is an empty argument (name: without value)
						argsList.EmptyArguments = append(argsList.EmptyArguments, EmptyArgument{
							Name: Identifier{Name: nameToken.Value, ASTSpan: p.spanForToken(nameToken)},
						})
						debug.Debug("Parsed empty argument", "name", nameToken.Value)

						// If there's a comma, check if it's trailing
						if p.check(lexer.TokenComma) {
							commaToken := p.current()
							p.advance()

							// Check if this is a trailing comma
							if p.check(lexer.TokenRParen) {
								span := p.spanForToken(commaToken)
								argsList.TrailingComma = &span
							}
						}
						continue
					}

					// There's a value after the colon, parse as normal named argument
					value := p.parseExpression()
					arg := Argument{
						Name:  &Identifier{Name: nameToken.Value, ASTSpan: p.spanForToken(nameToken)},
						Value: value,
						Span:  p.spanFrom(nameToken),
					}
					argsList.Arguments = append(argsList.Arguments, arg)
					debug.Debug("Parsed named argument", "name", nameToken.Value)
				} else {
					// Regular unnamed argument or expression
					if arg := p.parseArgument(); arg != nil {
						argsList.Arguments = append(argsList.Arguments, *arg)
						debug.Debug("Parsed unnamed argument")
						// Check if we've reached the closing parenthesis after parsing
						if p.check(lexer.TokenRParen) {
							break
						}
					} else {
						// If parsing failed, advance to avoid infinite loop
						p.advance()
					}
				}
			} else if arg := p.parseArgument(); arg != nil {
				argsList.Arguments = append(argsList.Arguments, *arg)
				currentToken := p.current()
				debug.Debug("Parsed argument", "next_token_type", currentToken.Type, "next_token_value", currentToken.Value, "is_rparen", p.check(lexer.TokenRParen))
				// Check if we've reached the closing parenthesis after parsing
				if p.check(lexer.TokenRParen) {
					debug.Debug("Breaking argument loop - found closing parenthesis")
					break
				}
			} else {
				// If parsing failed, advance to avoid infinite loop
				p.advance()
			}

			// Check if we've reached the closing parenthesis
			if p.check(lexer.TokenRParen) {
				debug.Debug("Breaking argument loop - found closing parenthesis (second check)")
				break
			}

			// Skip comments before comma
			for p.check(lexer.TokenComment) {
				p.advance()
			}

			if p.check(lexer.TokenComma) {
				commaToken := p.current()
				p.advance()

				// Check if this is a trailing comma (followed by closing paren)
				if p.check(lexer.TokenRParen) {
					span := p.spanForToken(commaToken)
					argsList.TrailingComma = &span
				}
			} else if !p.check(lexer.TokenRParen) {
				// Expected comma but didn't find one
				p.error("Expected comma or closing parenthesis")
			}
		}

		p.expect(lexer.TokenRParen)
		debug.Debug("Completed parsing attribute arguments", "argument_count", len(argsList.Arguments))
	}

	attribute := &Attribute{
		Name:      name,
		Arguments: argsList,
		Span:      p.spanForToken(nameToken),
	}
	debug.Debug("Completed parsing attribute", "name", name.Name, "argument_count", len(argsList.Arguments))
	return attribute
}

func (p *Parser) parseArgument() *Argument {
	var name *Identifier
	var value Expression

	// Check for named argument (name: value)
	if p.check(lexer.TokenIdentifier) && p.peek().Type == lexer.TokenColon {
		nameToken := p.expect(lexer.TokenIdentifier)
		name = &Identifier{Name: nameToken.Value, ASTSpan: p.spanForToken(nameToken)}
		p.expect(lexer.TokenColon)
	}

	value = p.parseExpression()

	return &Argument{
		Name:  name,
		Value: value,
		Span:  p.spanFrom(p.previous()),
	}
}

func (p *Parser) parseExpression() Expression {
	token := p.current()
	debug.Debug("Parsing expression", "token_type", token.Type, "token_value", token.Value)

	// Expression parsing order matches Rust: function_call | array_expression | numeric_literal | string_literal | path
	// Check function_call first (identifier followed by '(')
	if token.Type == lexer.TokenIdentifier {
		peekPos := p.pos + 1
		// Skip comments between identifier and potential '('
		for peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenComment {
			peekPos++
		}
		if peekPos < len(p.tokens) && p.tokens[peekPos].Type == lexer.TokenLParen {
			return p.parseFunctionCall()
		}
		// If not a function call, it's a path (constant value)
		p.advance()
		// Handle dotted paths (e.g., db.VarChar)
		for p.check(lexer.TokenDot) {
			p.advance() // consume dot
			if p.check(lexer.TokenIdentifier) {
				nextPart := p.expect(lexer.TokenIdentifier)
				token.Value = token.Value + "." + nextPart.Value
			} else {
				p.error("Expected identifier after '.' in path")
				break
			}
		}
		// Identifiers that aren't function calls are ConstantValue (for enums, etc.), matching Rust
		expr := ConstantValue{Value: token.Value, ASTSpan: p.spanForToken(token)}
		debug.Debug("Parsed constant value expression", "value", token.Value)
		return expr
	}

	// Check array_expression second
	if token.Type == lexer.TokenLBracket {
		return p.parseArrayLiteral()
	}

	// Check numeric_literal third
	if token.Type == lexer.TokenNumber {
		p.advance()
		// Store numeric value as string to preserve precision, matching Rust implementation
		expr := NumericValue{Value: token.Value, ASTSpan: p.spanForToken(token)}
		debug.Debug("Parsed numeric literal expression", "value", token.Value)
		return expr
	}

	// Check string_literal fourth
	if token.Type == lexer.TokenString {
		p.advance()
		// Parse string literal with proper escape sequence handling
		parsedValue := p.parseStringLiteral(token.Value)
		expr := StringLiteral{Value: parsedValue, ASTSpan: p.spanForToken(token)}
		debug.Debug("Parsed string literal expression", "value", parsedValue)
		return expr
	}

	// Check boolean (not in Rust grammar but we handle it)
	if token.Type == lexer.TokenBoolean {
		p.advance()
		// Boolean values are stored as ConstantValue, matching Rust implementation
		expr := ConstantValue{Value: token.Value, ASTSpan: p.spanForToken(token)}
		debug.Debug("Parsed boolean literal expression", "value", token.Value)
		return expr
	}

	// Error cases
	if token.Type == lexer.TokenRParen || token.Type == lexer.TokenRBracket || token.Type == lexer.TokenComma || token.Type == lexer.TokenEOF {
		// End of expression - return error and don't advance
		// Note: TokenRParen is a valid end marker for expressions in argument lists
		p.error("Expected expression (value, function call, or array) but found end of construct")
		return Identifier{Name: "error", ASTSpan: p.spanForToken(token)}
	}

	if token.Type == lexer.TokenRBrace {
		// TokenRBrace should not appear in the middle of an expression
		// This usually means we've gone too far (e.g., past a closing paren)
		p.error("Unexpected '}' - did you forget a closing ')'?")
		return Identifier{Name: "error", ASTSpan: p.spanForToken(token)}
	}

	// For unexpected tokens, create an error expression and advance to avoid infinite loop
	p.error(fmt.Sprintf("Unexpected token '%s' in expression. Expected: string, number, boolean, identifier, function call, or array", token.Value))
	p.advance()
	return Identifier{Name: "error", ASTSpan: p.spanForToken(token)}
}

func (p *Parser) parseArrayLiteral() Expression {
	debug.Debug("Parsing array literal")
	p.expect(lexer.TokenLBracket)

	elements := []Expression{}

	for !p.check(lexer.TokenRBracket) && !p.isAtEnd() {
		element := p.parseExpression()
		elements = append(elements, element)

		if !p.check(lexer.TokenRBracket) {
			if !p.check(lexer.TokenComma) && !p.isAtEnd() {
				p.error("Expected ',' or ']' to separate array elements")
				break
			}
			p.expect(lexer.TokenComma)
		}
	}

	if !p.isAtEnd() {
		p.expect(lexer.TokenRBracket)
	}

	array := ArrayLiteral{
		Elements: elements,
		ASTSpan:  p.spanFrom(p.previous()),
	}
	debug.Debug("Completed parsing array literal", "element_count", len(elements))
	return array
}

func (p *Parser) parseFunctionCall() Expression {
	// Parse path (can be dotted identifier like "db.VarChar")
	nameToken := p.expect(lexer.TokenIdentifier)
	functionName := nameToken.Value

	// Handle dotted paths (e.g., db.VarChar)
	for p.check(lexer.TokenDot) {
		p.advance() // consume dot
		if p.check(lexer.TokenIdentifier) {
			nextPart := p.expect(lexer.TokenIdentifier)
			functionName = functionName + "." + nextPart.Value
		} else {
			p.error("Expected identifier after '.' in function name")
			break
		}
	}

	debug.Debug("Parsing function call", "name", functionName)
	p.expect(lexer.TokenLParen)

	args := []Expression{}

	for !p.check(lexer.TokenRParen) && !p.isAtEnd() {
		arg := p.parseExpression()
		args = append(args, arg)

		if !p.check(lexer.TokenRParen) {
			if !p.check(lexer.TokenComma) && !p.isAtEnd() {
				p.error(fmt.Sprintf("Expected ',' or ')' after argument in function call '%s()'", functionName))
				break
			}
			p.expect(lexer.TokenComma)
		}
	}

	if !p.isAtEnd() {
		p.expect(lexer.TokenRParen)
	}

	functionCall := FunctionCall{
		Name:      Identifier{Name: functionName, ASTSpan: p.spanForToken(nameToken)},
		Arguments: args,
		ASTSpan:   p.spanFrom(nameToken),
	}
	debug.Debug("Completed parsing function call", "name", functionName, "argument_count", len(args))
	return functionCall
}

func (p *Parser) parseConfigProperty() *ConfigBlockProperty {
	name := p.expect(lexer.TokenIdentifier)

	// Check for end of block instead of always expecting equals
	if p.check(lexer.TokenRBrace) || p.isAtEnd() {
		// Property without value - this is an error
		p.error(fmt.Sprintf("Property '%s' must have a value. Expected format: %s = <value>", name.Value, name.Value))
		return nil
	}

	p.expect(lexer.TokenEquals)

	// Expression is optional in config blocks (for autocompletion)
	var value Expression
	if !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Skip comments before expression
		for p.check(lexer.TokenComment) {
			p.advance()
		}
		if !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
			expr := p.parseExpression()
			value = expr
		}
	}

	return &ConfigBlockProperty{
		Name:    Identifier{Name: name.Value, ASTSpan: p.spanForToken(name)},
		Value:   value,
		ASTSpan: p.spanFrom(name),
	}
}

// Helper methods

func (p *Parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() lexer.Token {
	if p.pos+1 >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.tokens[p.pos-1]
}

func (p *Parser) previous() lexer.Token {
	return p.tokens[p.pos-1]
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.current().Type == lexer.TokenEOF
}

func (p *Parser) check(tokenType lexer.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.current().Type == tokenType
}

func (p *Parser) expect(tokenType lexer.TokenType) lexer.Token {
	if p.check(tokenType) {
		return p.advance()
	}

	p.error(fmt.Sprintf("Expected %v, got %v", tokenType, p.current().Type))
	return lexer.Token{Type: lexer.TokenEOF}
}

func (p *Parser) error(message string) {
	// Add error to diagnostics
	span := diagnostics.NewSpan(0, 0, diagnostics.FileIDZero) // Default span
	var line, column int
	if !p.isAtEnd() {
		token := p.current()
		span = diagnostics.NewSpan(token.Column, token.Column+len(token.Value), diagnostics.FileIDZero)
		line = token.Line
		column = token.Column
	}
	debug.Error("Parser error", "message", message, "line", line, "column", column)
	p.diagnostics.PushError(diagnostics.NewValidationError(message, span))
}

func (p *Parser) spanForToken(token lexer.Token) diagnostics.Span {
	return diagnostics.NewSpan(token.Column, token.Column+len(token.Value), diagnostics.FileIDZero)
}

func (p *Parser) spanFrom(token lexer.Token) diagnostics.Span {
	return p.spanForToken(token)
}

// parseCommentBlock parses a block comment token into a Comment.
func (p *Parser) parseCommentBlock() *Comment {
	if !p.check(lexer.TokenComment) {
		return nil
	}

	token := p.current()
	commentText := token.Value

	// Trim leading whitespace and normalize
	lines := []string{}
	for _, line := range splitLines(commentText) {
		trimmed := trimCommentLine(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	if len(lines) == 0 {
		return nil
	}

	return &Comment{
		Text: joinLines(lines),
		Span: p.spanForToken(token),
	}
}

// parseTrailingComment parses a trailing comment.
func (p *Parser) parseTrailingComment() *Comment {
	return p.parseCommentBlock()
}

// splitLines splits a string into lines, preserving empty lines.
func splitLines(s string) []string {
	var lines []string
	var current strings.Builder

	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current.String())
			current.Reset()
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}

	return lines
}

// trimCommentLine trims comment markers and whitespace from a comment line.
func trimCommentLine(line string) string {
	// Remove // or /* */ markers
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "//") {
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)
	} else if strings.HasPrefix(line, "/*") {
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)
	}
	return line
}

// joinLines joins lines with newlines.
func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// parseStringLiteral parses a string literal with proper escape sequence handling.
// This handles unicode escape sequences according to RFC 8259.
func (p *Parser) parseStringLiteral(value string) string {
	// The value from lexer doesn't include quotes, but may include escape sequences
	var result strings.Builder
	result.Grow(len(value)) // Pre-allocate capacity

	i := 0
	for i < len(value) {
		if value[i] == '\\' && i+1 < len(value) {
			// Handle escape sequences
			i++ // Skip backslash
			switch value[i] {
			case '"':
				result.WriteByte('"')
				i++
			case '\\':
				result.WriteByte('\\')
				i++
			case '/':
				result.WriteByte('/')
				i++
			case 'b':
				result.WriteByte('\b')
				i++
			case 'f':
				result.WriteByte('\f')
				i++
			case 'n':
				result.WriteByte('\n')
				i++
			case 'r':
				result.WriteByte('\r')
				i++
			case 't':
				result.WriteByte('\t')
				i++
			case 'u':
				// Unicode escape sequence \uXXXX or \uXXXX\uYYYY (surrogate pair)
				if i+4 < len(value) {
					// Try to parse first codepoint
					codepoint1, consumed1 := p.parseUnicodeCodepoint(value[i+1:])
					if codepoint1 != nil {
						char := rune(*codepoint1)
						// Check if it's a valid UTF-8 character
						if char <= 0xD7FF || (char >= 0xE000 && char <= 0xFFFF) {
							// Valid single character
							result.WriteRune(char)
							i += consumed1 + 1
						} else if char >= 0xD800 && char <= 0xDBFF {
							// High surrogate - try to parse low surrogate
							if i+consumed1+1+6 < len(value) && value[i+consumed1+1] == '\\' && value[i+consumed1+2] == 'u' {
								codepoint2, consumed2 := p.parseUnicodeCodepoint(value[i+consumed1+3:])
								if codepoint2 != nil {
									lowSurrogate := rune(*codepoint2)
									if lowSurrogate >= 0xDC00 && lowSurrogate <= 0xDFFF {
										// Valid surrogate pair
										decoded := utf16DecodeSurrogatePair(uint16(char), uint16(lowSurrogate))
										if decoded != nil {
											result.WriteRune(*decoded)
											i += consumed1 + consumed2 + 3
											continue
										}
									}
								}
							}
							// Invalid surrogate pair
							p.error(fmt.Sprintf("Invalid unicode escape sequence at position %d", i))
							i += consumed1 + 1
						} else {
							// Invalid codepoint
							p.error(fmt.Sprintf("Invalid unicode escape sequence at position %d", i))
							i += consumed1 + 1
						}
					} else {
						p.error(fmt.Sprintf("Invalid unicode escape sequence at position %d", i))
						i += 5 // Skip \uXXXX
					}
				} else {
					p.error("Incomplete unicode escape sequence")
					i = len(value) // Skip to end
				}
			default:
				// Unknown escape sequence
				p.error(fmt.Sprintf("Unknown escape sequence '\\%c'. If the value is a windows-style path, `\\` must be escaped as `\\\\`.", value[i]))
				result.WriteByte('\\')
				result.WriteByte(value[i])
				i++
			}
		} else {
			result.WriteByte(value[i])
			i++
		}
	}

	return result.String()
}

// parseUnicodeCodepoint parses a 4-digit hexadecimal unicode codepoint.
// Returns the codepoint value and the number of characters consumed.
func (p *Parser) parseUnicodeCodepoint(s string) (*uint16, int) {
	if len(s) < 4 {
		return nil, 0
	}

	var codepoint uint16
	for i := 0; i < 4; i++ {
		char := s[i]
		var nibble uint16
		if char >= '0' && char <= '9' {
			nibble = uint16(char - '0')
		} else if char >= 'a' && char <= 'f' {
			nibble = uint16(char - 'a' + 10)
		} else if char >= 'A' && char <= 'F' {
			nibble = uint16(char - 'A' + 10)
		} else {
			return nil, i
		}
		codepoint = codepoint<<4 | nibble
	}

	return &codepoint, 4
}

// utf16DecodeSurrogatePair decodes a UTF-16 surrogate pair into a Unicode character.
func utf16DecodeSurrogatePair(high, low uint16) *rune {
	// Decode UTF-16 surrogate pair
	codePoint := uint32(high-0xD800)<<10 + uint32(low-0xDC00) + 0x10000
	char := rune(codePoint)
	if char >= 0x10000 && char <= 0x10FFFF {
		return &char
	}
	return nil
}

// Expression types are defined in ast.go
