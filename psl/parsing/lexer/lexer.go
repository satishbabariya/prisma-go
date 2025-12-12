// Package lexer provides lexical analysis for Prisma schema files.
package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/satishbabariya/prisma-go/internal/debug"
)

// TokenType represents the type of a token.
type TokenType int

const (
	// Keywords
	TokenGenerator TokenType = iota
	TokenDatasource
	TokenModel
	TokenEnum
	TokenTypeKeyword
	TokenView

	// Literals
	TokenIdentifier
	TokenString
	TokenNumber
	TokenBoolean

	// Symbols
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenLParen
	TokenRParen
	TokenEquals
	TokenComma
	TokenQuestion
	TokenAt
	TokenColon
	TokenDot
	TokenPipe
	TokenExclamation

	// Special
	TokenComment
	TokenEOF
)

// Token represents a lexical token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// Lexer tokenizes Prisma schema input.
type Lexer struct {
	input  string
	pos    int
	line   int
	column int
	tokens []Token
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	debug.Debug("Creating new lexer", "input_length", len(input))
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
		tokens: make([]Token, 0),
	}
}

// Tokenize converts the input string into a slice of tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	debug.Debug("Starting tokenization", "input_length", len(l.input))

	for l.pos < len(l.input) {
		char := rune(l.input[l.pos])

		switch {
		case unicode.IsSpace(char):
			debug.Debug("Tokenizing whitespace", "char", string(char), "line", l.line, "column", l.column)
			l.advance()
		case char == '/' && l.peek() == '/':
			debug.Debug("Tokenizing line comment", "line", l.line, "column", l.column)
			l.tokenizeComment()
		case char == '/' && l.peek() == '*':
			debug.Debug("Tokenizing block comment", "line", l.line, "column", l.column)
			l.tokenizeBlockComment()
		case char == '"':
			debug.Debug("Tokenizing string", "line", l.line, "column", l.column)
			l.tokenizeString()
		case unicode.IsLetter(char) || char == '_':
			debug.Debug("Tokenizing identifier", "char", string(char), "line", l.line, "column", l.column)
			l.tokenizeIdentifier()
		case unicode.IsDigit(char):
			debug.Debug("Tokenizing number", "char", string(char), "line", l.line, "column", l.column)
			l.tokenizeNumber()
		case char == '{':
			debug.Debug("Tokenizing left brace", "line", l.line, "column", l.column)
			l.addToken(TokenLBrace, "{")
		case char == '}':
			debug.Debug("Tokenizing right brace", "line", l.line, "column", l.column)
			l.addToken(TokenRBrace, "}")
		case char == '[':
			debug.Debug("Tokenizing left bracket", "line", l.line, "column", l.column)
			l.addToken(TokenLBracket, "[")
		case char == ']':
			debug.Debug("Tokenizing right bracket", "line", l.line, "column", l.column)
			l.addToken(TokenRBracket, "]")
		case char == '(':
			debug.Debug("Tokenizing left paren", "line", l.line, "column", l.column)
			l.addToken(TokenLParen, "(")
		case char == ')':
			debug.Debug("Tokenizing right paren", "line", l.line, "column", l.column)
			l.addToken(TokenRParen, ")")
		case char == '=':
			debug.Debug("Tokenizing equals", "line", l.line, "column", l.column)
			l.addToken(TokenEquals, "=")
		case char == ',':
			debug.Debug("Tokenizing comma", "line", l.line, "column", l.column)
			l.addToken(TokenComma, ",")
		case char == '?':
			debug.Debug("Tokenizing question mark", "line", l.line, "column", l.column)
			l.addToken(TokenQuestion, "?")
		case char == '@':
			debug.Debug("Tokenizing at symbol", "line", l.line, "column", l.column)
			l.addToken(TokenAt, "@")
		case char == ':':
			debug.Debug("Tokenizing colon", "line", l.line, "column", l.column)
			l.addToken(TokenColon, ":")
		case char == '.':
			debug.Debug("Tokenizing dot", "line", l.line, "column", l.column)
			l.addToken(TokenDot, ".")
		case char == '|':
			debug.Debug("Tokenizing pipe", "line", l.line, "column", l.column)
			l.addToken(TokenPipe, "|")
		case char == '!':
			debug.Debug("Tokenizing exclamation", "line", l.line, "column", l.column)
			l.addToken(TokenExclamation, "!")
		default:
			debug.Error("Unexpected character during tokenization", "char", string(char), "line", l.line, "column", l.column)
			return nil, fmt.Errorf("unexpected character '%c' at line %d, column %d", char, l.line, l.column)
		}
	}

	debug.Debug("Tokenization completed", "token_count", len(l.tokens))
	l.addToken(TokenEOF, "")
	return l.tokens, nil
}

func (l *Lexer) advance() {
	if l.pos >= len(l.input) {
		return
	}
	if l.input[l.pos] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.pos++
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos+1])
}

func (l *Lexer) addToken(tokenType TokenType, value string) {
	token := Token{
		Type:   tokenType,
		Value:  value,
		Line:   l.line,
		Column: l.column,
	}
	debug.Debug("Adding token", "type", tokenType, "value", value, "line", l.line, "column", l.column)
	l.tokens = append(l.tokens, token)
	l.advance()
}

func (l *Lexer) tokenizeComment() {
	// Skip //
	l.advance()
	l.advance()

	start := l.pos

	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.advance()
	}

	value := l.input[start:l.pos]
	commentToken := Token{
		Type:   TokenComment,
		Value:  value,
		Line:   l.line,
		Column: l.column - len(value) - 2, // Position before //
	}
	debug.Debug("Tokenized line comment", "value", value, "line", commentToken.Line, "column", commentToken.Column)
	l.tokens = append(l.tokens, commentToken)
}

func (l *Lexer) tokenizeBlockComment() {
	// Skip /*
	l.advance()
	l.advance()

	start := l.pos
	startLine := l.line
	startColumn := l.column - 2

	for l.pos < len(l.input)-1 {
		if l.input[l.pos] == '*' && l.input[l.pos+1] == '/' {
			value := l.input[start:l.pos]
			commentToken := Token{
				Type:   TokenComment,
				Value:  value,
				Line:   startLine,
				Column: startColumn,
			}
			debug.Debug("Tokenized block comment", "value", value, "line", commentToken.Line, "column", commentToken.Column)
			l.tokens = append(l.tokens, commentToken)
			l.advance()
			l.advance()
			break
		}
		l.advance()
	}
}

func (l *Lexer) tokenizeString() {
	l.advance() // Skip opening quote
	start := l.pos

	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' {
			l.advance() // Skip escape character
		}
		l.advance()
	}

	if l.pos >= len(l.input) {
		debug.Error("Unterminated string literal", "line", l.line, "column", l.column)
		panic("unterminated string")
	}

	value := l.input[start:l.pos]
	debug.Debug("Tokenized string literal", "value", value, "line", l.line, "column", l.column)
	// addToken will advance past the closing quote, so we don't need to advance again
	l.addToken(TokenString, value)
}

func (l *Lexer) tokenizeIdentifier() {
	start := l.pos

	// First character must be a letter or underscore
	if l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || l.input[l.pos] == '_') {
		l.advance()
	} else {
		// Invalid identifier start
		return
	}

	// Subsequent characters can be letters, digits, underscore, or hyphen
	for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_' || l.input[l.pos] == '-') {
		l.advance()
	}

	value := l.input[start:l.pos]

	// Check if it's a keyword
	var tokenType TokenType
	switch strings.ToLower(value) {
	case "generator":
		tokenType = TokenGenerator
		debug.Debug("Tokenized keyword", "type", "generator", "value", value, "line", l.line, "column", l.column)
	case "datasource":
		tokenType = TokenDatasource
		debug.Debug("Tokenized keyword", "type", "datasource", "value", value, "line", l.line, "column", l.column)
	case "model":
		tokenType = TokenModel
		debug.Debug("Tokenized keyword", "type", "model", "value", value, "line", l.line, "column", l.column)
	case "enum":
		tokenType = TokenEnum
		debug.Debug("Tokenized keyword", "type", "enum", "value", value, "line", l.line, "column", l.column)
	case "type":
		tokenType = TokenTypeKeyword
		debug.Debug("Tokenized keyword", "type", "type", "value", value, "line", l.line, "column", l.column)
	case "view":
		tokenType = TokenView
		debug.Debug("Tokenized keyword", "type", "view", "value", value, "line", l.line, "column", l.column)
	case "true", "false":
		tokenType = TokenBoolean
		debug.Debug("Tokenized boolean literal", "value", value, "line", l.line, "column", l.column)
	default:
		tokenType = TokenIdentifier
		debug.Debug("Tokenized identifier", "value", value, "line", l.line, "column", l.column)
	}

	l.tokens = append(l.tokens, Token{Type: tokenType, Value: value, Line: l.line, Column: l.column})
}

func (l *Lexer) tokenizeNumber() {
	start := l.pos
	hasDecimal := false

	// Check for negative sign
	if l.pos < len(l.input) && l.input[l.pos] == '-' {
		l.advance()
	}

	// Parse digits before decimal point
	for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
		l.advance()
	}

	// Check for decimal point
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		hasDecimal = true
		l.advance()
		// Parse digits after decimal point
		for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
			l.advance()
		}
	}

	value := l.input[start:l.pos]
	debug.Debug("Tokenized number", "value", value, "line", l.line, "column", l.column, "has_decimal", hasDecimal)
	l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: value, Line: l.line, Column: l.column})
}
