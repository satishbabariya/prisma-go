package e2e

import (
	"fmt"
	"os"
)

// TestDatabaseConfig holds configuration for test databases
type TestDatabaseConfig struct {
	PostgreSQL TestDBConfig
	MySQL      TestDBConfig
	SQLite     TestDBConfig
}

// TestDBConfig holds configuration for a single database
type TestDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// GetTestConfig returns test database configuration from environment variables or defaults
func GetTestConfig() *TestDatabaseConfig {
	return &TestDatabaseConfig{
		PostgreSQL: TestDBConfig{
			Host:     getEnv("TEST_PG_HOST", "localhost"),
			Port:     getEnvInt("TEST_PG_PORT", 5432),
			User:     getEnv("TEST_PG_USER", "postgres"),
			Password: getEnv("TEST_PG_PASSWORD", "password"),
			Database: getEnv("TEST_PG_DATABASE", "prisma_test"),
			SSLMode:  getEnv("TEST_PG_SSLMODE", "disable"),
		},
		MySQL: TestDBConfig{
			Host:     getEnv("TEST_MYSQL_HOST", "localhost"),
			Port:     getEnvInt("TEST_MYSQL_PORT", 3306),
			User:     getEnv("TEST_MYSQL_USER", "root"),
			Password: getEnv("TEST_MYSQL_PASSWORD", "password"),
			Database: getEnv("TEST_MYSQL_DATABASE", "prisma_test"),
		},
		SQLite: TestDBConfig{
			Host:     getEnv("TEST_SQLITE_PATH", "./test.db"),
			Port:     0,  // Not used for SQLite
			User:     "", // Not used for SQLite
			Password: "", // Not used for SQLite
			Database: getEnv("TEST_SQLITE_PATH", "./test.db"),
		},
	}
}

// GetPostgresConnectionString returns PostgreSQL connection string
func (c *TestDBConfig) GetPostgresConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User,
		c.Password, c.Database, c.SSLMode)
}

// GetMySQLConnectionString returns MySQL connection string
func (c *TestDBConfig) GetMySQLConnectionString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		c.User, c.Password, c.Host, c.Port, c.Database)
}

// GetSQLiteConnectionString returns SQLite connection string
func (c *TestDBConfig) GetSQLiteConnectionString() string {
	return c.Database
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue := parseInt(value); intValue != 0 {
			return intValue
		}
	}
	return defaultValue
}

func parseInt(s string) int {
	var result int
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return 0
		}
	}
	return result
}
