// Package storage provides tests for the storage adapters.
package storage

import (
	"context"
	"testing"
)

func TestMemoryStorage_ReadWrite(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Write
	content := []byte("hello world")
	err := storage.Write(ctx, "test.txt", content)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	result, err := storage.Read(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(result) != string(content) {
		t.Errorf("Expected %q, got %q", content, result)
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Write
	err := storage.Write(ctx, "delete-me.txt", []byte("temp"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Delete
	err = storage.Delete(ctx, "delete-me.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	exists, _ := storage.Exists(ctx, "delete-me.txt")
	if exists {
		t.Error("File should not exist after delete")
	}
}

func TestMemoryStorage_Exists(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Non-existent
	exists, err := storage.Exists(ctx, "nonexistent.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Non-existent file should return false")
	}

	// Create and check
	storage.Write(ctx, "exists.txt", []byte("content"))
	exists, err = storage.Exists(ctx, "exists.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Existing file should return true")
	}
}

func TestMemoryStorage_List(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Create files
	storage.Write(ctx, "dir/file1.txt", []byte("1"))
	storage.Write(ctx, "dir/file2.txt", []byte("2"))
	storage.Write(ctx, "dir/subdir/file3.txt", []byte("3"))

	// List
	files, err := storage.List(ctx, "dir")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 entries, got %d: %v", len(files), files)
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	storage.Write(ctx, "a.txt", []byte("a"))
	storage.Write(ctx, "b.txt", []byte("b"))

	if storage.Size() != 2 {
		t.Errorf("Expected size 2, got %d", storage.Size())
	}

	storage.Clear()

	if storage.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", storage.Size())
	}
}

func TestFilesystemStorage_ResolvePath(t *testing.T) {
	fs := NewFilesystemStorage("/base/path")

	// Relative path
	path := fs.resolvePath("relative.txt")
	expected := "/base/path/relative.txt"
	if path != expected {
		t.Errorf("Expected %q, got %q", expected, path)
	}

	// Absolute path
	path = fs.resolvePath("/absolute/path.txt")
	expected = "/absolute/path.txt"
	if path != expected {
		t.Errorf("Expected %q, got %q", expected, path)
	}
}
