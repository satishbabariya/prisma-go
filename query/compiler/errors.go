package compiler

import "errors"

var (
	ErrUnsupportedQuery  = errors.New("unsupported query type")
	ErrInvalidQuery      = errors.New("invalid query")
	ErrCompilationFailed = errors.New("query compilation failed")
)
