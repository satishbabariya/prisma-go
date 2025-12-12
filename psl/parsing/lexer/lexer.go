// Package lexer provides lexical analysis for Prisma schema files.
package lexer

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
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

// Chunk represents a portion of input for parallel processing
type Chunk struct {
	Start     int
	End       int
	LineStart int
	ColStart  int
	ID        int
}

// ChunkResult represents the result of processing a chunk
type ChunkResult struct {
	Tokens  []Token
	Error   error
	ChunkID int
	Metrics ChunkMetrics
}

// ChunkMetrics provides performance metrics for chunk processing
type ChunkMetrics struct {
	Duration    time.Duration
	TokenCount  int
	CharsPerSec float64
}

// Lexer tokenizes Prisma schema input.
type Lexer struct {
	input           string
	pos             int
	line            int
	column          int
	tokens          []Token
	parallelEnabled bool
	workerCount     int
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	debug.Debug("Creating new lexer", "input_length", len(input))
	parallelEnabled := len(input) > 10000 // Auto-enable for large files
	workerCount := runtime.NumCPU()
	if workerCount > 8 {
		workerCount = 8 // Cap workers for efficiency
	}

	return &Lexer{
		input:           input,
		pos:             0,
		line:            1,
		column:          1,
		tokens:          make([]Token, 0),
		parallelEnabled: parallelEnabled,
		workerCount:     workerCount,
	}
}

// splitIntoChunks divides input into optimal chunks for parallel processing
func (l *Lexer) splitIntoChunks() []Chunk {
	inputSize := len(l.input)
	if inputSize <= 10000 {
		return nil // Small files don't need chunking
	}

	// Calculate optimal chunk size based on worker count
	optimalSize := inputSize / l.workerCount
	if optimalSize > 50000 {
		optimalSize = 50000 // Cap chunk size for efficiency
	}
	if optimalSize < 5000 {
		optimalSize = 5000 // Minimum viable chunk size
	}

	var chunks []Chunk
	for i := 0; i < inputSize; i += optimalSize {
		end := i + optimalSize
		if end > inputSize {
			end = inputSize
		}

		// Prefer line boundaries for better error reporting
		lineBreak := strings.LastIndexByte(l.input[i:end], '\n')
		if lineBreak != -1 && (lineBreak-i) < optimalSize/2 {
			end = i + lineBreak + 1
		}

		chunks = append(chunks, Chunk{
			Start:     i,
			End:       end,
			LineStart: l.lineNumberAt(i),
			ColStart:  l.columnNumberAt(i),
			ID:        len(chunks),
		})
	}

	return chunks
}

// lineNumberAt calculates line number for a given position
func (l *Lexer) lineNumberAt(pos int) int {
	line := 1
	for i := 0; i < pos; i++ {
		if l.input[i] == '\n' {
			line++
		}
	}
	return line
}

// columnNumberAt calculates column number for a given position
func (l *Lexer) columnNumberAt(pos int) int {
	col := 1
	lastNewline := -1
	for i := 0; i < pos; i++ {
		if l.input[i] == '\n' {
			lastNewline = i
			col = 1
		} else if lastNewline != -1 {
			col++
		}
	}
	return col
}

// Tokenize converts the input string into a slice of tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	debug.Debug("Starting tokenization", "input_length", len(l.input))

	if l.parallelEnabled {
		return l.tokenizeParallel()
	}

	return l.tokenizeSequential()
}

// tokenizeSequential processes input sequentially (original implementation)
func (l *Lexer) tokenizeSequential() ([]Token, error) {
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
		case char == '!':
			l.addToken(TokenExclamation, "!")
		default:
			return nil, fmt.Errorf("unexpected character '%c' at line %d, column %d", char, l.line, l.column)
		}
	}

	l.addToken(TokenEOF, "")
	return l.tokens, nil
}

// tokenizeParallel processes input using multiple workers
func (l *Lexer) tokenizeParallel() ([]Token, error) {
	chunks := l.splitIntoChunks()
	if chunks == nil {
		return l.tokenizeSequential()
	}

	debug.Debug("Processing chunks in parallel", "chunk_count", len(chunks), "workers", l.workerCount)

	// Create channels for work distribution
	jobChan := make(chan Chunk, len(chunks))
	resultChan := make(chan ChunkResult, len(chunks))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < l.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for chunk := range jobChan {
				result := l.processChunk(chunk)
				result.ChunkID = chunk.ID
				resultChan <- result
			}
		}(i)
	}

	// Distribute chunks
	go func() {
		defer close(jobChan)
		for _, chunk := range chunks {
			jobChan <- chunk
		}
	}()

	// Collect results
	results := make([]ChunkResult, len(chunks))
	completed := 0
	for result := range resultChan {
		results[result.ChunkID] = result
		completed++
		if completed == len(chunks) {
			break
		}
	}

	wg.Wait()

	// Merge tokens in order
	var allTokens []Token
	for _, result := range results {
		if result.Error != nil {
			return nil, result.Error
		}
		allTokens = append(allTokens, result.Tokens...)
	}

	debug.Debug("Parallel tokenization completed", "total_tokens", len(allTokens))
	l.addToken(TokenEOF, "")
	return allTokens, nil
}

// processChunk processes a single chunk of input
func (l *Lexer) processChunk(chunk Chunk) ChunkResult {
	startTime := time.Now()

	// Create chunk-specific lexer
	chunkLexer := &Lexer{
		input:  l.input[chunk.Start:chunk.End],
		pos:    0,
		line:   chunk.LineStart,
		column: chunk.ColStart,
		tokens: make([]Token, 0),
	}

	tokens, err := chunkLexer.tokenizeSequential()

	duration := time.Since(startTime)
	charsPerSec := float64(chunk.End-chunk.Start) / duration.Seconds()

	return ChunkResult{
		Tokens: tokens,
		Error:  err,
		Metrics: ChunkMetrics{
			Duration:    duration,
			TokenCount:  len(tokens),
			CharsPerSec: charsPerSec,
		},
	}
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
	case "datasource":
		tokenType = TokenDatasource
	case "model":
		tokenType = TokenModel
	case "enum":
		tokenType = TokenEnum
	case "type":
		tokenType = TokenTypeKeyword
	case "view":
		tokenType = TokenView
	case "true", "false":
		tokenType = TokenBoolean
	default:
		tokenType = TokenIdentifier
	}

	// Add token without advancing position (already advanced)
	l.tokens = append(l.tokens, Token{Type: tokenType, Value: value, Line: l.line, Column: l.column - len(value)})
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

	// Add token without advancing position (already advanced)
	l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: value, Line: l.line, Column: l.column - len(value)})
}
