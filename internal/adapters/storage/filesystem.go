// Package storage provides filesystem storage implementation.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FilesystemStorage implements Storage using the local filesystem.
type FilesystemStorage struct {
	basePath string
}

// NewFilesystemStorage creates a new filesystem storage adapter.
func NewFilesystemStorage(basePath string) *FilesystemStorage {
	return &FilesystemStorage{
		basePath: basePath,
	}
}

// resolvePath resolves a path relative to the base path.
func (fs *FilesystemStorage) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(fs.basePath, path)
}

// Read reads contents from a path.
func (fs *FilesystemStorage) Read(ctx context.Context, path string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return content, nil
}

// Write writes contents to a path.
func (fs *FilesystemStorage) Write(ctx context.Context, path string, content []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// Delete deletes a file at path.
func (fs *FilesystemStorage) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists checks if a path exists.
func (fs *FilesystemStorage) Exists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file: %w", err)
	}
	return true, nil
}

// List lists all files in a directory.
func (fs *FilesystemStorage) List(ctx context.Context, dir string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(dir)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	return files, nil
}

// MkdirAll creates a directory and all parent directories.
func (fs *FilesystemStorage) MkdirAll(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// ReadStream reads contents as a stream.
func (fs *FilesystemStorage) ReadStream(ctx context.Context, path string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// WriteStream writes from a stream.
func (fs *FilesystemStorage) WriteStream(ctx context.Context, path string, reader io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fullPath := fs.resolvePath(path)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write stream: %w", err)
	}
	return nil
}

// Ensure FilesystemStorage implements Storage interface.
var _ Storage = (*FilesystemStorage)(nil)
