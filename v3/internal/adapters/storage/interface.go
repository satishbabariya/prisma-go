// Package storage provides storage adapter interfaces.
package storage

import (
	"context"
	"io"
)

// Storage defines the storage adapter interface.
type Storage interface {
	// Read reads contents from a path.
	Read(ctx context.Context, path string) ([]byte, error)

	// Write writes contents to a path.
	Write(ctx context.Context, path string, content []byte) error

	// Delete deletes a file at path.
	Delete(ctx context.Context, path string) error

	// Exists checks if a path exists.
	Exists(ctx context.Context, path string) (bool, error)

	// List lists all files in a directory.
	List(ctx context.Context, dir string) ([]string, error)

	// MkdirAll creates a directory and all parent directories.
	MkdirAll(ctx context.Context, path string) error

	// ReadStream reads contents as a stream.
	ReadStream(ctx context.Context, path string) (io.ReadCloser, error)

	// WriteStream writes from a stream.
	WriteStream(ctx context.Context, path string, reader io.Reader) error
}

// FileInfo represents file metadata.
type FileInfo struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime int64
}

// Config holds storage configuration.
type Config struct {
	// Type is the storage type (filesystem, memory).
	Type string

	// BasePath is the base path for filesystem storage.
	BasePath string
}
