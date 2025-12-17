// Package storage provides in-memory storage implementation.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

// MemoryStorage implements Storage using in-memory storage.
type MemoryStorage struct {
	files map[string][]byte
	mu    sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage adapter.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		files: make(map[string][]byte),
	}
}

// Read reads contents from a path.
func (ms *MemoryStorage) Read(ctx context.Context, path string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	content, exists := ms.files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	// Return a copy to prevent modification
	result := make([]byte, len(content))
	copy(result, content)
	return result, nil
}

// Write writes contents to a path.
func (ms *MemoryStorage) Write(ctx context.Context, path string, content []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Store a copy to prevent external modification
	data := make([]byte, len(content))
	copy(data, content)
	ms.files[path] = data
	return nil
}

// Delete deletes a file at path.
func (ms *MemoryStorage) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.files, path)
	return nil
}

// Exists checks if a path exists.
func (ms *MemoryStorage) Exists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	_, exists := ms.files[path]
	return exists, nil
}

// List lists all files in a directory.
func (ms *MemoryStorage) List(ctx context.Context, dir string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Normalize directory path
	if !strings.HasSuffix(dir, "/") && dir != "" {
		dir += "/"
	}

	var files []string
	seen := make(map[string]bool)

	for path := range ms.files {
		if dir == "" || strings.HasPrefix(path, dir) {
			// Get relative path
			relPath := strings.TrimPrefix(path, dir)
			// Get first component (file or subdirectory)
			parts := strings.SplitN(relPath, "/", 2)
			if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
				files = append(files, parts[0])
				seen[parts[0]] = true
			}
		}
	}
	return files, nil
}

// MkdirAll creates a directory (no-op for memory storage).
func (ms *MemoryStorage) MkdirAll(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	// Directories are implicit in memory storage
	return nil
}

// ReadStream reads contents as a stream.
func (ms *MemoryStorage) ReadStream(ctx context.Context, path string) (io.ReadCloser, error) {
	content, err := ms.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

// WriteStream writes from a stream.
func (ms *MemoryStorage) WriteStream(ctx context.Context, path string, reader io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read stream: %w", err)
	}
	return ms.Write(ctx, path, content)
}

// Clear removes all files from memory storage.
func (ms *MemoryStorage) Clear() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.files = make(map[string][]byte)
}

// Size returns the number of files in storage.
func (ms *MemoryStorage) Size() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.files)
}

// Ensure MemoryStorage implements Storage interface.
var _ Storage = (*MemoryStorage)(nil)

// Unused import prevention
var _ = filepath.Join
