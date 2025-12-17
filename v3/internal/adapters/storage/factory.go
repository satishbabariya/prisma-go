// Package storage provides a factory for creating storage adapters.
package storage

import (
	"fmt"
)

// StorageType represents the type of storage.
type StorageType string

const (
	// TypeFilesystem is the filesystem storage type.
	TypeFilesystem StorageType = "filesystem"

	// TypeMemory is the in-memory storage type.
	TypeMemory StorageType = "memory"
)

// NewStorage creates a new storage adapter based on configuration.
func NewStorage(config *Config) (Storage, error) {
	if config == nil {
		config = &Config{Type: string(TypeFilesystem)}
	}

	switch StorageType(config.Type) {
	case TypeFilesystem:
		if config.BasePath == "" {
			config.BasePath = "."
		}
		return NewFilesystemStorage(config.BasePath), nil

	case TypeMemory:
		return NewMemoryStorage(), nil

	default:
		return nil, fmt.Errorf("unknown storage type: %s", config.Type)
	}
}
