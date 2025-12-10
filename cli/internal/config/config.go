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

	// Set defaults
	viper.SetDefault("schema_path", "schema.prisma")
	viper.SetDefault("output_path", "./generated")
	viper.SetDefault("skip_env_check", false)

	// Try to read config file (ignore if not found)
	_ = viper.ReadInConfig()

	// Load .env file if it exists
	if _, err := AppFs.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			// Don't fail if .env can't be loaded
		}
	}

	// Load .env.local if it exists (higher priority)
	if _, err := AppFs.Stat(".env.local"); err == nil {
		if err := godotenv.Overload(".env.local"); err != nil {
			// Don't fail if .env.local can't be loaded
		}
	}

	cfg := &Config{
		SchemaPath:   viper.GetString("schema_path"),
		OutputPath:   viper.GetString("output_path"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
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

