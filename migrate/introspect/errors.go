package introspect

import "errors"

var (
	ErrUnsupportedProvider = errors.New("unsupported database provider")
	ErrConnectionFailed    = errors.New("failed to connect to database")
	ErrIntrospectionFailed = errors.New("database introspection failed")
)
