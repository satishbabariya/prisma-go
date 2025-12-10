// Package lexer provides lexical analysis for Prisma schema files.
package lexer

import (
	"fmt"
	"strings"
	"unicode"
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
	for l.pos < len(l.input) {
		char := rune(l.input[l.pos])

		switch {
		case unicode.IsSpace(char):
			l.advance()
		case char == '/' && l.peek() == '/':
			l.tokenizeComment()
		case char == '/' && l.peek() == '*':
			l.tokenizeBlockComment()
		case char == '"':
			l.tokenizeString()
		case unicode.IsLetter(char) || char == '_':
			l.tokenizeIdentifier()
		case unicode.IsDigit(char):
			l.tokenizeNumber()
		case char == '{':
			l.addToken(TokenLBrace, "{")
		case char == '}':
			l.addToken(TokenRBrace, "}")
		case char == '[':
			l.addToken(TokenLBracket, "[")
		case char == ']':
			l.addToken(TokenRBracket, "]")
		case char == '(':
			l.addToken(TokenLParen, "(")
		case char == ')':
			l.addToken(TokenRParen, ")")
		case char == '=':
			l.addToken(TokenEquals, "=")
		case char == ',':
			l.addToken(TokenComma, ",")
		case char == '?':
			l.addToken(TokenQuestion, "?")
		case char == '@':
			l.addToken(TokenAt, "@")
		case char == ':':
			l.addToken(TokenColon, ":")
		case char == '.':
			l.addToken(TokenDot, ".")
		case char == '|':
			l.addToken(TokenPipe, "|")
		default:
			return nil, fmt.Errorf("unexpected character '%c' at line %d, column %d", char, l.line, l.column)
		}
	}

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
	l.tokens = append(l.tokens, Token{
		Type:   tokenType,
		Value:  value,
		Line:   l.line,
		Column: l.column,
	})
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
	l.tokens = append(l.tokens, Token{
		Type:   TokenComment,
		Value:  value,
		Line:   l.line,
		Column: l.column - len(value) - 2, // Position before //
	})
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
			l.tokens = append(l.tokens, Token{
				Type:   TokenComment,
				Value:  value,
				Line:   startLine,
				Column: startColumn,
			})
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
		panic("unterminated string")
	}

	value := l.input[start:l.pos]
	// addToken will advance past the closing quote, so we don't need to advance again
	l.addToken(TokenString, value)
}

func (l *Lexer) tokenizeIdentifier() {
	start := l.pos

	for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_') {
		l.advance()
	}

	value := l.input[start:l.pos]

	// Check if it's a keyword
	switch strings.ToLower(value) {
	case "generator":
		l.tokens = append(l.tokens, Token{Type: TokenGenerator, Value: value, Line: l.line, Column: l.column})
	case "datasource":
		l.tokens = append(l.tokens, Token{Type: TokenDatasource, Value: value, Line: l.line, Column: l.column})
	case "model":
		l.tokens = append(l.tokens, Token{Type: TokenModel, Value: value, Line: l.line, Column: l.column})
	case "enum":
		l.tokens = append(l.tokens, Token{Type: TokenEnum, Value: value, Line: l.line, Column: l.column})
	case "type":
		l.tokens = append(l.tokens, Token{Type: TokenTypeKeyword, Value: value, Line: l.line, Column: l.column})
	case "true", "false":
		l.tokens = append(l.tokens, Token{Type: TokenBoolean, Value: value, Line: l.line, Column: l.column})
	default:
		l.tokens = append(l.tokens, Token{Type: TokenIdentifier, Value: value, Line: l.line, Column: l.column})
	}
}

func (l *Lexer) tokenizeNumber() {
	start := l.pos

	for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
		l.advance()
	}

	value := l.input[start:l.pos]
	l.addToken(TokenNumber, value)
}
