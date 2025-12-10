package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var AppFs = afero.NewOsFs()

// Config holds the application configuration
type Config struct {
	SchemaPath   string
	OutputPath   string
	DatabaseURL  string
	Provider     string
	SkipEnvCheck bool
}

// LoadConfig loads configuration from various sources
func LoadConfig() (*Config, error) {
	// Find home directory
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	// Set config file paths
	viper.SetConfigName(".prisma-go")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(home)
	viper.AddConfigPath(filepath.Join(home, ".config", "prisma-go"))

	// Set environment variable prefix
	viper.SetEnvPrefix("PRISMA_GO")
	viper.AutomaticEnv()

	// Bind DATABASE_URL explicitly (without prefix) for consistency
	viper.BindEnv("database_url", "DATABASE_URL")

	// Set defaults
	viper.SetDefault("schema_path", "schema.prisma")
	viper.SetDefault("output_path", "./generated")
	viper.SetDefault("skip_env_check", false)

	// Try to read config file (ignore if not found)
	_ = viper.ReadInConfig()

	// Load .env file if it exists (using afero for filesystem abstraction)
	if data, err := afero.ReadFile(AppFs, ".env"); err == nil {
		envMap, err := godotenv.Unmarshal(string(data))
		if err == nil {
			for k, v := range envMap {
				os.Setenv(k, v)
			}
		}
	}

	// Load .env.local if it exists (higher priority)
	if data, err := afero.ReadFile(AppFs, ".env.local"); err == nil {
		envMap, err := godotenv.Unmarshal(string(data))
		if err == nil {
			for k, v := range envMap {
				os.Setenv(k, v)
			}
		}
	}

	cfg := &Config{
		SchemaPath:   viper.GetString("schema_path"),
		OutputPath:   viper.GetString("output_path"),
		DatabaseURL:  viper.GetString("database_url"),
		Provider:     viper.GetString("provider"),
		SkipEnvCheck: viper.GetBool("skip_env_check"),
	}

	return cfg, nil
}

// SaveConfig saves configuration to file
func SaveConfig(cfg *Config) error {
	viper.Set("schema_path", cfg.SchemaPath)
	viper.Set("output_path", cfg.OutputPath)
	viper.Set("provider", cfg.Provider)
	viper.Set("skip_env_check", cfg.SkipEnvCheck)

	home, err := homedir.Dir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".config", "prisma-go")
	if err := AppFs.MkdirAll(configPath, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configPath, ".prisma-go.yaml")
	return viper.WriteConfigAs(configFile)
}

