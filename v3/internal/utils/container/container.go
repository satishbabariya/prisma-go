// Package container provides dependency injection.
package container

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database/mysql"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database/postgres"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database/sqlite"
	"github.com/satishbabariya/prisma-go/v3/internal/config"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/differ"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/executor"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/introspector"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/planner"
	querycompiler "github.com/satishbabariya/prisma-go/v3/internal/core/query/compiler"
	querydomain "github.com/satishbabariya/prisma-go/v3/internal/core/query/domain"
	queryexecutor "github.com/satishbabariya/prisma-go/v3/internal/core/query/executor"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/formatter"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/parser"
	"github.com/satishbabariya/prisma-go/v3/internal/core/schema/validator"
	"github.com/satishbabariya/prisma-go/v3/internal/repository"
	"github.com/satishbabariya/prisma-go/v3/internal/service"
)

// Container holds all application dependencies.
type Container struct {
	// Configuration
	config *config.Config

	// Adapters
	dbAdapter database.Adapter

	// Repositories
	schemaRepo    repository.SchemaRepository
	migrationRepo repository.MigrationRepository
	historyRepo   repository.HistoryRepository
	configRepo    repository.ConfigRepository

	// Services
	schemaService     *service.SchemaService
	generateService   *service.GenerateService
	migrationService  *service.MigrationService
	introspectService *service.IntrospectService
	queryService      *service.QueryService
}

// NewContainer creates a new dependency injection container.
func NewContainer(cfg *config.Config) (*Container, error) {
	c := &Container{
		config: cfg,
	}

	// Initialize adapters
	var err error
	c.dbAdapter, err = createDatabaseAdapter(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create database adapter: %w", err)
	}

	// Initialize repositories
	schemaParser := parser.NewParser()
	c.schemaRepo = repository.NewSchemaRepository(schemaParser)
	c.migrationRepo = repository.NewMigrationRepository("./prisma/migrations")
	c.historyRepo = repository.NewHistoryRepository(c.dbAdapter)
	c.configRepo = repository.NewConfigRepository("./prisma/config.json")

	// Initialize migration domain components
	introspector := introspector.NewDatabaseIntrospector(c.dbAdapter)
	differ := differ.NewDatabaseDiffer()

	// Convert database dialect to domain dialect
	var dialect domain.SQLDialect
	if c.dbAdapter != nil {
		dialect = domain.SQLDialect(c.dbAdapter.GetDialect())
	} else {
		dialect = domain.SQLite // Default dialect
	}

	planner := planner.NewMigrationPlanner(dialect)
	executor := executor.NewMigrationExecutor(c.dbAdapter)

	// Initialize schema service
	schemaValidator := validator.NewValidator()
	schemaFormatter := formatter.NewFormatter()
	c.schemaService = service.NewSchemaService(
		c.schemaRepo,
		schemaValidator,
		schemaFormatter,
	)

	// Initialize migration service with actual components
	c.migrationService = service.NewMigrationService(
		c.schemaRepo,
		c.migrationRepo,
		c.historyRepo,
		introspector,
		differ,
		planner,
		executor,
	)

	// Initialize introspect service
	c.introspectService = service.NewIntrospectService(
		c.schemaRepo,
		introspector,
	)

	// Initialize query service
	// Convert database dialect to query dialect
	queryDialect := querydomain.SQLDialect(c.dbAdapter.GetDialect())
	queryComp := querycompiler.NewSQLCompiler(queryDialect)
	queryExec := queryexecutor.NewQueryExecutor(c.dbAdapter)
	c.queryService = service.NewQueryService(queryComp, queryExec)

	// Initialize generate service (would add analyzer, engine, writer)
	c.generateService = service.NewGenerateService(
		c.schemaRepo,
		nil, // analyzer
		nil, // engine
		nil, // writer
	)

	return c, nil
}

// SchemaService returns the schema service.
func (c *Container) SchemaService() *service.SchemaService {
	return c.schemaService
}

// GenerateService returns the generate service.
func (c *Container) GenerateService() *service.GenerateService {
	return c.generateService
}

// MigrationService returns the migration service.
func (c *Container) MigrationService() *service.MigrationService {
	return c.migrationService
}

// IntrospectService returns the introspect service.
func (c *Container) IntrospectService() *service.IntrospectService {
	return c.introspectService
}

// QueryService returns the query service.
func (c *Container) QueryService() *service.QueryService {
	return c.queryService
}

// Close cleans up resources.
func (c *Container) Close(ctx context.Context) error {
	if c.dbAdapter != nil {
		return c.dbAdapter.Disconnect(ctx)
	}
	return nil
}

// createDatabaseAdapter creates the appropriate database adapter based on provider.
func createDatabaseAdapter(cfg config.DatabaseConfig) (database.Adapter, error) {
	dbConfig := database.Config{
		Provider:       cfg.Provider,
		URL:            cfg.URL,
		MaxConnections: cfg.MaxConnections,
		MaxIdleTime:    cfg.MaxIdleTime,
		ConnectTimeout: cfg.ConnectTimeout,
	}

	var adapter database.Adapter
	var err error

	switch cfg.Provider {
	case "postgresql", "postgres":
		adapter, err = postgres.NewPostgresAdapter(dbConfig)
	case "mysql":
		adapter, err = mysql.NewMySQLAdapter(dbConfig)
	case "sqlite":
		adapter, err = sqlite.NewSQLiteAdapter(dbConfig)
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	return adapter, nil
}
