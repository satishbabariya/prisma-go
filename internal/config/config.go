// Package config provides configuration management.
package config

// Config represents application configuration.
type Config struct {
	Database  DatabaseConfig
	Generator GeneratorConfig
}

// DatabaseConfig represents database configuration.
type DatabaseConfig struct {
	Provider       string
	URL            string
	MaxConnections int
	MaxIdleTime    int
	ConnectTimeout int
}

// GeneratorConfig represents generator configuration.
type GeneratorConfig struct {
	Output  string
	Package string
}
