package ast

import (
	"fmt"
	"strconv"

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
	return &Parser{
		tokens:      tokens,
		pos:         0,
		diagnostics: diags,
	}
}

// Parse parses the tokens into a SchemaAst.
func (p *Parser) Parse() *SchemaAst {
	schema := &SchemaAst{Tops: []Top{}}

	for !p.isAtEnd() {
		// Skip comments
		for p.check(lexer.TokenComment) {
			p.advance()
		}

		if p.isAtEnd() {
			break
		}

		if top := p.parseTopLevel(); top != nil {
			schema.Tops = append(schema.Tops, top)
		}
	}

	return schema
}

func (p *Parser) parseTopLevel() Top {
	// Skip comments
	for p.check(lexer.TokenComment) {
		p.advance()
	}

	if p.isAtEnd() {
		return nil
	}

	token := p.current()

	switch token.Type {
	case lexer.TokenGenerator:
		return p.parseGenerator()
	case lexer.TokenDatasource:
		return p.parseDatasource()
	case lexer.TokenModel:
		return p.parseModel()
	case lexer.TokenEnum:
		return p.parseEnum()
	case lexer.TokenTypeKeyword:
		return p.parseCompositeType()
	case lexer.TokenEOF:
		return nil
	default:
		p.error(fmt.Sprintf("Unexpected token '%s' at top level. Expected: model, enum, generator, datasource, or type", token.Value))
		p.advance()
		return nil
	}
}

func (p *Parser) parseGenerator() Top {
	p.expect(lexer.TokenGenerator)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	properties := []ConfigBlockProperty{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Skip comments
		for p.check(lexer.TokenComment) {
			p.advance()
		}

		if p.check(lexer.TokenRBrace) || p.isAtEnd() {
			break
		}

		if prop := p.parseConfigProperty(); prop != nil {
			properties = append(properties, *prop)
		} else {
			// If parsing failed, advance to avoid infinite loop
			p.advance()
		}
	}

	p.expect(lexer.TokenRBrace)

	return &GeneratorConfig{
		Name:       Identifier{Name: name.Value, span: p.spanForToken(name)},
		Properties: properties,
		span:       p.spanFrom(name),
	}
}

func (p *Parser) parseDatasource() Top {
	p.expect(lexer.TokenDatasource)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	properties := []ConfigBlockProperty{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		if prop := p.parseConfigProperty(); prop != nil {
			properties = append(properties, *prop)
		} else {
			// If parsing failed, advance to avoid infinite loop
			p.advance()
		}
	}

	p.expect(lexer.TokenRBrace)

	return &SourceConfig{
		Name:       Identifier{Name: name.Value, span: p.spanForToken(name)},
		Properties: properties,
		span:       p.spanFrom(name),
	}
}

func (p *Parser) parseModel() Top {
	p.expect(lexer.TokenModel)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	fields := []Field{}
	attributes := []Attribute{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Skip comments
		for p.check(lexer.TokenComment) {
			p.advance()
		}

		if p.check(lexer.TokenRBrace) {
			break
		}

		// Check for model-level attributes (@@)
		if p.check(lexer.TokenAt) && p.peek().Type == lexer.TokenAt {
			// This is a model-level attribute (e.g., @@id, @@index)
			// Consume both @ tokens - we've already verified the next token is also @
			firstAt := p.advance()  // consume first @
			secondAt := p.advance() // consume second @
			_ = firstAt
			_ = secondAt
			// Skip any comments/whitespace (handled by lexer, but check for comments)
			for p.check(lexer.TokenComment) {
				p.advance()
			}
			// Now parse as a regular attribute (but it's model-level)
			// After consuming both @ tokens, the next token should be the attribute name
			if attr := p.parseAttributeAfterAt(); attr != nil {
				attributes = append(attributes, *attr)
			} else {
				// If parsing failed, advance to avoid infinite loop
				if !p.isAtEnd() && !p.check(lexer.TokenRBrace) {
					p.advance()
				}
			}
		} else if field := p.parseField(); field != nil {
			// Try to parse as a field (field-level @ attributes are handled in parseField)
			fields = append(fields, *field)
		} else {
			// If parsing failed, advance to avoid infinite loop
			p.advance()
		}
	}

	p.expect(lexer.TokenRBrace)

	return &Model{
		Name:       Identifier{Name: name.Value, span: p.spanForToken(name)},
		Fields:     fields,
		Attributes: attributes,
		span:       p.spanFrom(name),
	}
}

func (p *Parser) parseEnum() Top {
	p.expect(lexer.TokenEnum)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	values := []EnumValue{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		if value := p.parseEnumValue(); value != nil {
			values = append(values, *value)
		}
	}

	p.expect(lexer.TokenRBrace)

	return &Enum{
		Name:   Identifier{Name: name.Value, span: p.spanForToken(name)},
		Values: values,
		span:   p.spanFrom(name),
	}
}

func (p *Parser) parseCompositeType() Top {
	p.expect(lexer.TokenTypeKeyword)
	name := p.expect(lexer.TokenIdentifier)

	p.expect(lexer.TokenLBrace)

	fields := []Field{}
	attributes := []Attribute{}

	for !p.check(lexer.TokenRBrace) && !p.isAtEnd() {
		// Check for composite type-level attributes (@@)
		if p.check(lexer.TokenAt) && p.peek().Type == lexer.TokenAt {
			// This is a composite type-level attribute
			// Consume both @ tokens
			p.advance() // consume first @
			p.advance() // consume second @
			// Now parse as a regular attribute
			if attr := p.parseAttributeAfterAt(); attr != nil {
				attributes = append(attributes, *attr)
			}
		} else if field := p.parseField(); field != nil {
			// Try to parse as a field (field-level @ attributes are handled in parseField)
			fields = append(fields, *field)
		} else {
			// If parsing failed, advance to avoid infinite loop
			p.advance()
		}
	}

	p.expect(lexer.TokenRBrace)

	return &CompositeType{
		Name:       Identifier{Name: name.Value, span: p.spanForToken(name)},
		Fields:     fields,
		Attributes: attributes,
		span:       p.spanFrom(name),
	}
}

func (p *Parser) parseField() *Field {
	name := p.expect(lexer.TokenIdentifier)
	fieldType, arity := p.parseFieldType()

	attributes := []Attribute{}
	// Parse field-level attributes (@), but stop if we see model-level attributes (@@)
	for p.check(lexer.TokenAt) && p.peek().Type != lexer.TokenAt {
		if attr := p.parseAttribute(); attr != nil {
			attributes = append(attributes, *attr)
		} else {
			break
		}
	}

	return &Field{
		Name:       Identifier{Name: name.Value, span: p.spanForToken(name)},
		FieldType:  fieldType,
		Arity:      arity,
		Attributes: attributes,
		span:       p.spanFrom(name),
	}
}

func (p *Parser) parseFieldType() (FieldType, FieldArity) {
	// Check if we have a valid type name
	if p.check(lexer.TokenRBrace) || p.check(lexer.TokenAt) || p.isAtEnd() {
		// Missing field type
		p.error("Expected field type. Field types can be: String, Int, Float, Boolean, DateTime, Json, Bytes, or a model/enum name")
		return FieldType{
			Type: UnsupportedFieldType{TypeName: ""},
			span: diagnostics.NewSpan(0, 0, diagnostics.FileIDZero),
		}, Required
	}

	// Parse the base type name
	typeNameToken := p.expect(lexer.TokenIdentifier)
	typeName := Identifier{Name: typeNameToken.Value, span: p.spanForToken(typeNameToken)}

	// Check for array type
	isArray := p.check(lexer.TokenLBracket)
	if isArray {
		p.expect(lexer.TokenLBracket)
		p.expect(lexer.TokenRBracket)
	}

	// Check for optional
	isOptional := p.check(lexer.TokenQuestion)
	if isOptional {
		p.expect(lexer.TokenQuestion)
	}

	// Determine arity
	var arity FieldArity
	if isArray {
		arity = List
	} else if isOptional {
		arity = Optional
	} else {
		arity = Required
	}

	return FieldType{
		Type: SupportedFieldType{Identifier: typeName},
		span: p.spanForToken(typeNameToken),
	}, arity
}

func (p *Parser) parseEnumValue() *EnumValue {
	name := p.expect(lexer.TokenIdentifier)

	attributes := []Attribute{}
	for p.check(lexer.TokenAt) {
		if attr := p.parseAttribute(); attr != nil {
			attributes = append(attributes, *attr)
		}
	}

	return &EnumValue{
		Name:       Identifier{Name: name.Value, span: p.spanForToken(name)},
		Attributes: attributes,
		span:       p.spanForToken(name),
	}
}

func (p *Parser) parseAttribute() *Attribute {
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
	name := p.expect(lexer.TokenIdentifier)

	argsList := ArgumentsList{
		Arguments:      []Argument{},
		EmptyArguments: []EmptyArgument{},
		TrailingComma:  nil,
	}

	if p.check(lexer.TokenLParen) {
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
							Name: Identifier{Name: nameToken.Value, span: p.spanForToken(nameToken)},
						})

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
					argsList.Arguments = append(argsList.Arguments, Argument{
						Name:  &Identifier{Name: nameToken.Value, span: p.spanForToken(nameToken)},
						Value: value,
						Span:  p.spanFrom(nameToken),
					})
				} else {
					// Regular unnamed argument or expression
					if arg := p.parseArgument(); arg != nil {
						argsList.Arguments = append(argsList.Arguments, *arg)
					} else {
						// If parsing failed, advance to avoid infinite loop
						p.advance()
					}
				}
			} else if arg := p.parseArgument(); arg != nil {
				argsList.Arguments = append(argsList.Arguments, *arg)
			} else {
				// If parsing failed, advance to avoid infinite loop
				p.advance()
			}

			if !p.check(lexer.TokenRParen) {
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
		}

		p.expect(lexer.TokenRParen)
	}

	return &Attribute{
		Name:      Identifier{Name: name.Value, span: p.spanForToken(name)},
		Arguments: argsList,
		Span:      p.spanFrom(name),
	}
}

func (p *Parser) parseArgument() *Argument {
	var name *Identifier
	var value Expression

	// Check for named argument (name: value)
	if p.check(lexer.TokenIdentifier) && p.peek().Type == lexer.TokenColon {
		nameToken := p.expect(lexer.TokenIdentifier)
		name = &Identifier{Name: nameToken.Value, span: p.spanForToken(nameToken)}
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

	switch token.Type {
	case lexer.TokenString:
		p.advance()
		return StringLiteral{Value: token.Value, span: p.spanForToken(token)}
	case lexer.TokenNumber:
		p.advance()
		if intVal, err := strconv.Atoi(token.Value); err == nil {
			return IntLiteral{Value: intVal, span: p.spanForToken(token)}
		}
		if floatVal, err := strconv.ParseFloat(token.Value, 64); err == nil {
			return FloatLiteral{Value: floatVal, span: p.spanForToken(token)}
		}
		// Default to int if parsing fails
		return IntLiteral{Value: 0, span: p.spanForToken(token)}
	case lexer.TokenBoolean:
		p.advance()
		boolVal := token.Value == "true"
		return BooleanLiteral{Value: boolVal, span: p.spanForToken(token)}
	case lexer.TokenLBracket:
		return p.parseArrayLiteral()
	case lexer.TokenIdentifier:
		if p.peek().Type == lexer.TokenLParen {
			return p.parseFunctionCall()
		}
		p.advance()
		return Identifier{Name: token.Value, span: p.spanForToken(token)}
	case lexer.TokenRBrace, lexer.TokenRParen, lexer.TokenRBracket, lexer.TokenComma, lexer.TokenEOF:
		// End of expression - return error and don't advance
		p.error("Expected expression (value, function call, or array) but found end of construct")
		return Identifier{Name: "error", span: p.spanForToken(token)}
	default:
		// For unexpected tokens, create an error expression and advance to avoid infinite loop
		p.error(fmt.Sprintf("Unexpected token '%s' in expression. Expected: string, number, boolean, identifier, function call, or array", token.Value))
		p.advance()
		return Identifier{Name: "error", span: p.spanForToken(token)}
	}
}

func (p *Parser) parseArrayLiteral() Expression {
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

	return ArrayLiteral{
		Elements: elements,
		span:     p.spanFrom(p.previous()),
	}
}

func (p *Parser) parseFunctionCall() Expression {
	name := p.expect(lexer.TokenIdentifier)
	p.expect(lexer.TokenLParen)

	args := []Expression{}

	for !p.check(lexer.TokenRParen) && !p.isAtEnd() {
		arg := p.parseExpression()
		args = append(args, arg)

		if !p.check(lexer.TokenRParen) {
			if !p.check(lexer.TokenComma) && !p.isAtEnd() {
				p.error(fmt.Sprintf("Expected ',' or ')' after argument in function call '%s()'", name.Value))
				break
			}
			p.expect(lexer.TokenComma)
		}
	}

	if !p.isAtEnd() {
		p.expect(lexer.TokenRParen)
	}

	return FunctionCall{
		Name:      Identifier{Name: name.Value, span: p.spanForToken(name)},
		Arguments: args,
		span:      p.spanFrom(name),
	}
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

	// Check for end of block after equals
	if p.check(lexer.TokenRBrace) || p.isAtEnd() {
		p.error(fmt.Sprintf("Property '%s' is missing a value after '='. Expected: %s = <value>", name.Value, name.Value))
		return nil
	}

	value := p.parseExpression()

	return &ConfigBlockProperty{
		Name:  Identifier{Name: name.Value, span: p.spanForToken(name)},
		Value: value,
		Span:  p.spanFrom(name),
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
	if !p.isAtEnd() {
		token := p.current()
		span = diagnostics.NewSpan(token.Column, token.Column+len(token.Value), diagnostics.FileIDZero)
	}
	p.diagnostics.PushError(diagnostics.NewValidationError(message, span))
}

func (p *Parser) spanForToken(token lexer.Token) diagnostics.Span {
	return diagnostics.NewSpan(token.Column, token.Column+len(token.Value), diagnostics.FileIDZero)
}

func (p *Parser) spanFrom(token lexer.Token) diagnostics.Span {
	return p.spanForToken(token)
}

// Expression types are defined in ast.go
