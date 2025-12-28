// Package repository implements repository interfaces for data access.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/satishbabariya/prisma-go/v3/internal/config"
)

// ConfigRepositoryImpl implements the ConfigRepository interface.
type ConfigRepositoryImpl struct {
	configPath string
}

// NewConfigRepository creates a new config repository.
func NewConfigRepository(configPath string) *ConfigRepositoryImpl {
	return &ConfigRepositoryImpl{
		configPath: configPath,
	}
}

// Load loads configuration from file.
func (r *ConfigRepositoryImpl) Load(ctx context.Context) (*config.Config, error) {
	// Check if config file exists
	if _, err := os.Stat(r.configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return r.getDefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(r.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save saves configuration to file.
func (r *ConfigRepositoryImpl) Save(ctx context.Context, cfg *config.Config) error {
	// Marshal to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(r.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfig returns the default configuration.
func (r *ConfigRepositoryImpl) getDefaultConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Provider:       "postgresql",
			URL:            os.Getenv("DATABASE_URL"),
			MaxConnections: 10,
			MaxIdleTime:    60,
			ConnectTimeout: 30,
		},
		Generator: config.GeneratorConfig{
			Output:  "./generated",
			Package: "db",
		},
	}
}

// Ensure ConfigRepositoryImpl implements ConfigRepository interface.
var _ ConfigRepository = (*ConfigRepositoryImpl)(nil)
