// Package commands implements CLI commands for db push.
package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	pslparser "github.com/satishbabariya/prisma-go/psl/parsing/v2"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database"
	"github.com/satishbabariya/prisma-go/v3/internal/adapters/database/postgres"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/executor"
	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/introspector"
	"github.com/spf13/cobra"
)

// NewDBPushCommand creates the db push command.
func NewDBPushCommand() *cobra.Command {
	var acceptDataLoss bool
	var forceReset bool
	var skipGenerate bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push the Prisma schema state to the database without migrations",
		Long: `Push the Prisma schema state directly to the database without
creating migration files. This is ideal for prototyping and development.

Note: This command will modify your database schema. Data loss may occur
if you make destructive changes (like removing columns).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			schemaPath := "prisma/schema.prisma"
			if len(args) > 0 {
				schemaPath = args[0]
			}

			return runDBPush(schemaPath, acceptDataLoss, forceReset, skipGenerate)
		},
	}

	cmd.Flags().BoolVar(&acceptDataLoss, "accept-data-loss", false, "Accept data loss to apply changes")
	cmd.Flags().BoolVar(&forceReset, "force-reset", false, "Force reset the database (destructive)")
	cmd.Flags().BoolVar(&skipGenerate, "skip-generate", false, "Skip running prisma generate")

	return cmd
}

// NewDBPullCommand creates the db pull command.
func NewDBPullCommand() *cobra.Command {
	var force bool
	var print bool

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull the database schema and update the Prisma schema",
		Long: `Pull the current database schema and update your schema.prisma file.
This is useful for introspecting an existing database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			schemaPath := "prisma/schema.prisma"
			if len(args) > 0 {
				schemaPath = args[0]
			}

			return runDBPull(schemaPath, force, print)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Override the schema.prisma file")
	cmd.Flags().BoolVar(&print, "print", false, "Print the introspected schema instead of saving")

	return cmd
}

// NewDBCommand creates the parent db command.
func NewDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Manage your database",
		Long:  "Commands to manage your database schema and data.",
	}

	cmd.AddCommand(NewDBPushCommand())
	cmd.AddCommand(NewDBPullCommand())
	cmd.AddCommand(NewDBSeedCommand())
	return cmd
}

func runDBPush(schemaPath string, acceptDataLoss, forceReset, skipGenerate bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("🔄 Pushing schema from %s...\n", schemaPath)

	// Read and parse schema
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	pslAst, err := pslparser.ParseSchema("schema.prisma", strings.NewReader(string(content)))
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Get database URL from datasource using Sources() method
	var dbURL string
	for _, ds := range pslAst.Sources() {
		if urlProp := ds.GetProperty("url"); urlProp != nil {
			if strVal, ok := urlProp.Value.AsStringValue(); ok {
				dbURL = strVal.GetValue()
			} else if funcVal, ok := urlProp.Value.AsFunction(); ok {
				if funcVal.Name == "env" && funcVal.Arguments != nil && len(funcVal.Arguments.Arguments) > 0 {
					if envName, ok := funcVal.Arguments.Arguments[0].Value.AsStringValue(); ok {
						dbURL = os.Getenv(envName.GetValue())
					}
				}
			}
		}
	}

	if dbURL == "" {
		return fmt.Errorf("no database URL found in schema")
	}

	// Create database adapter
	dbConfig := database.Config{
		URL:            dbURL,
		MaxConnections: 5,
		ConnectTimeout: 30,
	}

	adapter, err := postgres.NewPostgresAdapter(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to create database adapter: %w", err)
	}

	if err := adapter.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer adapter.Disconnect(ctx)

	fmt.Println("✅ Connected to database")

	// Introspect current database state
	introspectorInstance := introspector.NewDatabaseIntrospector(adapter)
	currentState, err := introspectorInstance.IntrospectDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect database: %w", err)
	}

	fmt.Printf("📊 Found %d existing tables\n", len(currentState.Tables))

	// Convert schema to desired state
	desiredState := schemaToState(pslAst)

	fmt.Printf("📝 Schema defines %d tables\n", len(desiredState.Tables))

	// Calculate changes needed
	changes := calculateChanges(currentState, desiredState)

	if len(changes) == 0 {
		fmt.Println("✅ Database is already in sync with schema")
		return nil
	}

	// Check for destructive changes
	hasDestructive := false
	for _, change := range changes {
		if change.IsDestructive() {
			hasDestructive = true
			fmt.Printf("⚠️  Destructive change: %s\n", change.Description())
		} else {
			fmt.Printf("➕ %s\n", change.Description())
		}
	}

	if hasDestructive && !acceptDataLoss {
		return fmt.Errorf("destructive changes detected. Use --accept-data-loss to apply")
	}

	// Generate SQL and apply
	var sqlStatements []string
	dialect := domain.PostgreSQL // Default

	for _, change := range changes {
		stmts, err := change.ToSQL(dialect)
		if err != nil {
			return fmt.Errorf("failed to generate SQL: %w", err)
		}
		sqlStatements = append(sqlStatements, stmts...)
	}

	fmt.Printf("\n🚀 Applying %d SQL statements...\n", len(sqlStatements))

	// Execute
	exec := executor.NewMigrationExecutor(adapter)
	if err := exec.ExecuteSQL(ctx, sqlStatements); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	fmt.Println("✅ Database schema updated successfully!")

	if !skipGenerate {
		fmt.Println("\n💡 Run 'prisma-go generate' to update your client")
	}

	return nil
}

func runDBPull(schemaPath string, force, print bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("🔄 Introspecting database...")

	// Read existing schema to get connection info
	content, err := os.ReadFile(schemaPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	var dbURL string
	if len(content) > 0 {
		pslAst, err := pslparser.ParseSchema("schema.prisma", strings.NewReader(string(content)))
		if err != nil {
			return fmt.Errorf("failed to parse existing schema: %w", err)
		}

		for _, ds := range pslAst.Sources() {
			if urlProp := ds.GetProperty("url"); urlProp != nil {
				if strVal, ok := urlProp.Value.AsStringValue(); ok {
					dbURL = strVal.GetValue()
				} else if funcVal, ok := urlProp.Value.AsFunction(); ok {
					if funcVal.Name == "env" && funcVal.Arguments != nil && len(funcVal.Arguments.Arguments) > 0 {
						if envName, ok := funcVal.Arguments.Arguments[0].Value.AsStringValue(); ok {
							dbURL = os.Getenv(envName.GetValue())
						}
					}
				}
			}
		}
	}

	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}

	if dbURL == "" {
		return fmt.Errorf("no database URL found. Set DATABASE_URL or provide a schema with datasource")
	}

	// Create database adapter
	dbConfig := database.Config{
		URL:            dbURL,
		MaxConnections: 5,
		ConnectTimeout: 30,
	}

	adapter, err := postgres.NewPostgresAdapter(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to create database adapter: %w", err)
	}

	if err := adapter.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer adapter.Disconnect(ctx)

	fmt.Println("✅ Connected to database")

	// Introspect database
	introspectorInstance := introspector.NewDatabaseIntrospector(adapter)
	state, err := introspectorInstance.IntrospectDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to introspect database: %w", err)
	}

	fmt.Printf("📊 Found %d tables\n", len(state.Tables))

	// Generate Prisma schema from introspected state
	schema := generatePrismaSchema(state, dbURL)

	if print {
		fmt.Println("\n// Generated Prisma Schema")
		fmt.Println(schema)
		return nil
	}

	// Check if file exists and force not set
	if _, err := os.Stat(schemaPath); err == nil && !force {
		return fmt.Errorf("schema file exists. Use --force to override")
	}

	// Write schema to file
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	fmt.Printf("✅ Schema written to %s\n", schemaPath)
	return nil
}

// generatePrismaSchema generates a Prisma schema from database state.
func generatePrismaSchema(state *domain.DatabaseState, dbURL string) string {
	var sb strings.Builder

	// Datasource
	sb.WriteString("datasource db {\n")
	sb.WriteString("  provider = \"postgresql\"\n")
	sb.WriteString("  url      = env(\"DATABASE_URL\")\n")
	sb.WriteString("}\n\n")

	// Generator
	sb.WriteString("generator client {\n")
	sb.WriteString("  provider = \"go\"\n")
	sb.WriteString("}\n\n")

	// Models
	for _, table := range state.Tables {
		sb.WriteString(fmt.Sprintf("model %s {\n", toPascalCase(table.Name)))
		for _, col := range table.Columns {
			prismaType := sqlTypeToPrismaType(col.Type)
			optional := ""
			if col.IsNullable {
				optional = "?"
			}
			attrs := ""
			if col.IsPrimaryKey {
				attrs += " @id"
			}
			if col.IsUnique {
				attrs += " @unique"
			}
			if col.DefaultValue != "" {
				attrs += fmt.Sprintf(" @default(%s)", col.DefaultValue)
			}
			sb.WriteString(fmt.Sprintf("  %s %s%s%s\n", col.Name, prismaType, optional, attrs))
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

// toPascalCase converts a snake_case string to PascalCase.
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(string(p[0])) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, "")
}

// sqlTypeToPrismaType converts SQL types to Prisma types.
func sqlTypeToPrismaType(sqlType string) string {
	lower := strings.ToLower(sqlType)
	switch {
	case strings.Contains(lower, "int"):
		return "Int"
	case strings.Contains(lower, "varchar"), strings.Contains(lower, "text"), strings.Contains(lower, "char"):
		return "String"
	case strings.Contains(lower, "bool"):
		return "Boolean"
	case strings.Contains(lower, "timestamp"), strings.Contains(lower, "date"):
		return "DateTime"
	case strings.Contains(lower, "float"), strings.Contains(lower, "double"), strings.Contains(lower, "real"):
		return "Float"
	case strings.Contains(lower, "decimal"), strings.Contains(lower, "numeric"):
		return "Decimal"
	case strings.Contains(lower, "json"):
		return "Json"
	case strings.Contains(lower, "bytea"), strings.Contains(lower, "blob"):
		return "Bytes"
	default:
		return "String"
	}
}

// schemaToState converts a parsed schema to a database state.
func schemaToState(pslAst interface{}) *domain.DatabaseState {
	state := &domain.DatabaseState{
		Tables: []domain.Table{},
	}

	// Use type assertion to access Models() method
	type modelsProvider interface {
		Models() interface{}
	}

	if mp, ok := pslAst.(modelsProvider); ok {
		models := mp.Models()
		// Handle different model types based on the AST structure
		if modelSlice, ok := models.([]interface{}); ok {
			for _, m := range modelSlice {
				if model, ok := m.(interface{ GetName() string }); ok {
					table := domain.Table{
						Name:    model.GetName(),
						Columns: []domain.Column{},
					}
					state.Tables = append(state.Tables, table)
				}
			}
		}
	}

	return state
}

// calculateChanges calculates the changes needed between current and desired state.
func calculateChanges(current, desired *domain.DatabaseState) []domain.Change {
	var changes []domain.Change

	// Build maps for efficient lookup
	currentTables := make(map[string]*domain.Table)
	for i := range current.Tables {
		currentTables[current.Tables[i].Name] = &current.Tables[i]
	}

	desiredTables := make(map[string]*domain.Table)
	for i := range desired.Tables {
		desiredTables[desired.Tables[i].Name] = &desired.Tables[i]
	}

	// Find tables to create (in desired but not in current)
	for name, table := range desiredTables {
		if _, exists := currentTables[name]; !exists {
			changes = append(changes, &createTableChange{Table: *table})
		}
	}

	// Find tables to drop (in current but not in desired)
	for name := range currentTables {
		if _, exists := desiredTables[name]; !exists {
			changes = append(changes, &dropTableChange{TableName: name})
		}
	}

	// Find tables that exist in both and check for column changes
	for name, desiredTable := range desiredTables {
		if currentTable, exists := currentTables[name]; exists {
			// Column-level diffing
			columnChanges := calculateColumnChanges(currentTable, desiredTable)
			changes = append(changes, columnChanges...)
		}
	}

	return changes
}

// calculateColumnChanges finds differences in columns between current and desired table.
func calculateColumnChanges(current, desired *domain.Table) []domain.Change {
	var changes []domain.Change

	// Build column maps
	currentCols := make(map[string]*domain.Column)
	for i := range current.Columns {
		currentCols[current.Columns[i].Name] = &current.Columns[i]
	}

	desiredCols := make(map[string]*domain.Column)
	for i := range desired.Columns {
		desiredCols[desired.Columns[i].Name] = &desired.Columns[i]
	}

	// Find columns to add
	for name, col := range desiredCols {
		if _, exists := currentCols[name]; !exists {
			changes = append(changes, &addColumnChange{
				TableName: current.Name,
				Column:    *col,
			})
		}
	}

	// Find columns to drop
	for name := range currentCols {
		if _, exists := desiredCols[name]; !exists {
			changes = append(changes, &dropColumnChange{
				TableName:  current.Name,
				ColumnName: name,
			})
		}
	}

	// Find columns that changed type
	for name, desiredCol := range desiredCols {
		if currentCol, exists := currentCols[name]; exists {
			if currentCol.Type != desiredCol.Type || currentCol.IsNullable != desiredCol.IsNullable {
				changes = append(changes, &alterColumnChange{
					TableName: current.Name,
					OldColumn: *currentCol,
					NewColumn: *desiredCol,
				})
			}
		}
	}

	return changes
}

// createTableChange implements domain.Change for CREATE TABLE.
type createTableChange struct {
	Table domain.Table
}

func (c *createTableChange) Type() domain.ChangeType { return domain.CreateTable }
func (c *createTableChange) Description() string {
	return fmt.Sprintf("Create table %s", c.Table.Name)
}
func (c *createTableChange) IsDestructive() bool { return false }
func (c *createTableChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY)", c.Table.Name)
	return []string{sql}, nil
}

// dropTableChange implements domain.Change for DROP TABLE.
type dropTableChange struct {
	TableName string
}

func (c *dropTableChange) Type() domain.ChangeType { return domain.DropTable }
func (c *dropTableChange) Description() string {
	return fmt.Sprintf("Drop table %s", c.TableName)
}
func (c *dropTableChange) IsDestructive() bool { return true }
func (c *dropTableChange) ToSQL(dialect domain.SQLDialect) ([]string, error) {
	return []string{fmt.Sprintf("DROP TABLE %s", c.TableName)}, nil
}
