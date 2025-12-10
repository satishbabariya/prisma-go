package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/migrate/converter"
	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
	psl "github.com/satishbabariya/prisma-go/psl"
)

func dbCommand(args []string) error {
	if len(args) == 0 {
		printDBHelp()
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "push":
		return dbPushCommand(args[1:])
	case "pull":
		return dbPullCommand(args[1:])
	case "seed":
		return dbSeedCommand(args[1:])
	case "execute":
		return dbExecuteCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown db subcommand: %s\n\n", subcommand)
		printDBHelp()
		os.Exit(1)
		return nil
	}
}

func printDBHelp() {
	help := `
USAGE:
    prisma-go db <subcommand> [options]

SUBCOMMANDS:
    push       Push schema changes to database without migrations
    pull       Pull schema from database (introspect to .prisma file)
    seed       Seed the database (auto-runs seed.go)
    execute    Execute raw SQL commands

EXAMPLES:
    prisma-go db push schema.prisma
    prisma-go db pull output.prisma
    prisma-go db seed
    prisma-go db execute "SELECT * FROM users"
    prisma-go db execute script.sql
`
	fmt.Println(help)
}

func dbPushCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: schema file required (use: prisma-go db push <schema-path>)")
		return fmt.Errorf("schema file required")
	}

	schemaPath := args[0]
	
	fmt.Println("🚀 Pushing schema changes to database...")
	
	// Read schema file
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read schema: %v\n", err)
		return err
	}

	// Parse schema
	sourceFile := psl.NewSourceFile(schemaPath, string(content))
	parsed, diags := psl.ParseSchemaFromFile(sourceFile)
	if diags.HasErrors() {
		fmt.Fprintf(os.Stderr, "❌ Error parsing schema:\n%s\n", diags.ToPrettyString(schemaPath, string(content)))
		return fmt.Errorf("schema parsing failed")
	}
	
	// Get connection info
	provider, connStr := extractConnectionInfo(parsed)
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ No connection string found in schema\n")
		return fmt.Errorf("no connection string")
	}
	
	driverProvider := normalizeProviderForDriver(provider)
	
	// Connect to database
	db, err := sql.Open(driverProvider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Introspect current database
	introspector, err := introspect.NewIntrospector(db, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create introspector: %v\n", err)
		return err
	}
	
	currentSchema, err := introspector.Introspect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to introspect database: %v\n", err)
		return err
	}
	
	fmt.Printf("📊 Current database has %d tables\n", len(currentSchema.Tables))
	
	// Convert schema AST to database schema
	targetSchema, err := converter.ConvertASTToDBSchema(parsed, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to convert schema: %v\n", err)
		return err
	}
	
	fmt.Printf("📋 Schema defines %d tables\n", len(targetSchema.Tables))
	
	// Compare schemas
	differ := diff.NewSimpleDiffer(provider)
	diffResult := differ.CompareSchemas(currentSchema, targetSchema)
	
	// Show differences
	fmt.Println("\n📝 Schema differences:")
	if len(diffResult.TablesToCreate) > 0 {
		fmt.Printf("  • %d table(s) to create\n", len(diffResult.TablesToCreate))
		for _, change := range diffResult.TablesToCreate {
			fmt.Printf("    - %s\n", change.Name)
		}
	}
	if len(diffResult.TablesToAlter) > 0 {
		fmt.Printf("  • %d table(s) to alter\n", len(diffResult.TablesToAlter))
		for _, change := range diffResult.TablesToAlter {
			fmt.Printf("    - %s (%d changes)\n", change.Name, len(change.Changes))
		}
	}
	if len(diffResult.TablesToDrop) > 0 {
		fmt.Printf("  • %d table(s) to drop\n", len(diffResult.TablesToDrop))
		for _, change := range diffResult.TablesToDrop {
			fmt.Printf("    - %s\n", change.Name)
		}
	}
	
	if len(diffResult.TablesToCreate) == 0 && len(diffResult.TablesToAlter) == 0 && len(diffResult.TablesToDrop) == 0 {
		fmt.Println("  ✓ No differences found - database is up to date!")
		return nil
	}
	
	// Generate SQL
	var sqlGenerator sqlgen.MigrationGenerator
	switch provider {
	case "postgresql", "postgres":
		sqlGenerator = sqlgen.NewPostgresMigrationGenerator()
	case "mysql":
		sqlGenerator = sqlgen.NewMySQLMigrationGenerator()
	case "sqlite":
		sqlGenerator = sqlgen.NewSQLiteMigrationGenerator()
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	
	sql, err := sqlGenerator.GenerateMigrationSQL(diffResult, targetSchema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to generate SQL: %v\n", err)
		return err
	}
	
	// Show preview
	fmt.Println("\n📄 Generated SQL:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println(sql)
	fmt.Println("─────────────────────────────────────────")
	
	// Prompt for confirmation
	fmt.Print("\n❓ Apply these changes to the database? (y/N): ")
	var confirmation string
	fmt.Scanln(&confirmation)
	
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))
	if confirmation != "y" && confirmation != "yes" {
		fmt.Println("✋ Aborted. No changes applied.")
		return nil
	}
	
	// Apply changes directly (prototype mode - no migration history)
	fmt.Println("\n🚀 Applying schema changes...")
	
	// Execute SQL statements
	statements := strings.Split(sql, ";")
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		
		_, err = db.ExecContext(ctx, stmt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to execute statement %d: %v\n", i+1, err)
			fmt.Fprintf(os.Stderr, "   SQL: %s\n", stmt)
			return fmt.Errorf("failed to apply schema changes")
		}
	}
	
	fmt.Println("\n✅ Schema changes pushed successfully!")
	fmt.Println("\n💡 Note: db push applies changes directly without migration history.")
	fmt.Println("   For production, use migrations instead:")
	fmt.Println("   prisma-go migrate diff schema.prisma --create-only")
	fmt.Println("   prisma-go migrate apply migrations/.../migration.sql")
	
	return nil
}

func dbPullCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: output file required (use: prisma-go db pull <output-schema-path>)")
		return fmt.Errorf("output file required")
	}

	outputPath := args[0]
	
	fmt.Println("🔍 Pulling schema from database...")
	
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		return fmt.Errorf("no connection string")
	}
	
	// Detect provider
	provider := detectProvider(connStr)
	driverProvider := normalizeProviderForDriver(provider)
	
	// Connect to database
	db, err := sql.Open(driverProvider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Introspect database
	introspector, err := introspect.NewIntrospector(db, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create introspector: %v\n", err)
		return err
	}
	
	schema, err := introspector.Introspect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to introspect database: %v\n", err)
		return err
	}
	
	fmt.Printf("✓ Found %d tables\n", len(schema.Tables))
	
	// Generate Prisma schema file
	schemaContent := generatePrismaSchemaFromDB(schema, provider)
	
	// Write to file
	err = os.WriteFile(outputPath, []byte(schemaContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write schema: %v\n", err)
		return err
	}
	
	fmt.Printf("\n✅ Schema written to %s\n", outputPath)
	fmt.Println("\n💡 Next steps:")
	fmt.Println("  1. Review the generated schema")
	fmt.Println("  2. Add relations if needed")
	fmt.Println("  3. Run 'prisma-go generate' to create the client")
	
	return nil
}

func dbSeedCommand(args []string) error {
	fmt.Println("🌱 Seeding database...")
	
	// Look for seed script in common locations
	seedPaths := []string{
		"seed.go",
		"prisma/seed.go",
		"scripts/seed.go",
		"db/seed.go",
	}
	
	var seedPath string
	for _, path := range seedPaths {
		if _, err := os.Stat(path); err == nil {
			seedPath = path
			break
		}
	}
	
	if seedPath == "" {
		fmt.Println("⚠️  No seed script found in common locations:")
		for _, path := range seedPaths {
			fmt.Printf("  • %s\n", path)
		}
		fmt.Println("\n💡 Create a seed script:")
		fmt.Println(`
  package main
  
  import (
      "context"
      "github.com/yourapp/generated"
  )
  
  func main() {
      ctx := context.Background()
      client, _ := generated.NewPrismaClient("postgresql://...")
      client.Connect(ctx)
      defer client.Disconnect(ctx)
      
      // Seed admin user
      _, _ = client.User.Create(ctx, User{
          Email: "admin@example.com",
          Name:  stringPtr("Admin"),
      })
      
      // Seed more data...
  }
  
  func stringPtr(s string) *string { return &s }
		`)
		fmt.Println("\n✅ Then run: prisma-go db seed")
		return nil
	}
	
	fmt.Printf("📝 Found seed script: %s\n", seedPath)
	fmt.Println("🚀 Running seed script...")
	
	// Execute seed script
	cmd := exec.Command("go", "run", seedPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Set working directory to script's directory
	scriptDir := filepath.Dir(seedPath)
	if scriptDir != "." {
		cmd.Dir = scriptDir
	}
	
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Seed script failed: %v\n", err)
		return fmt.Errorf("seed script execution failed: %w", err)
	}
	
	fmt.Println("\n✅ Database seeded successfully!")
	return nil
}

func dbExecuteCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: SQL command or file required (use: prisma-go db execute <sql-command|sql-file>)")
		return fmt.Errorf("SQL command or file required")
	}

	sqlInput := args[0]
	
	fmt.Println("🔧 Executing SQL command...")
	
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		return fmt.Errorf("no connection string")
	}
	
	provider := detectProvider(connStr)
	driverProvider := normalizeProviderForDriver(provider)
	
	// Connect to database
	db, err := sql.Open(driverProvider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Read SQL from file or use as-is
	var sql string
	if _, err := os.Stat(sqlInput); err == nil {
		// It's a file
		sqlBytes, err := os.ReadFile(sqlInput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to read SQL file: %v\n", err)
			return err
		}
		sql = string(sqlBytes)
		fmt.Printf("📄 Read SQL from file: %s\n", sqlInput)
	} else {
		// It's a direct SQL command
		sql = sqlInput
	}
	
	// Execute SQL
	rows, err := db.QueryContext(ctx, sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to execute SQL: %v\n", err)
		return err
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get columns: %v\n", err)
		return err
	}
	
	// Print header
	fmt.Println("\n📊 Results:")
	fmt.Println("─────────────────────────────────────────")
	for i, col := range columns {
		if i > 0 {
			fmt.Print(" | ")
		}
		fmt.Print(col)
	}
	fmt.Println()
	fmt.Println("─────────────────────────────────────────")
	
	// Print rows
	rowCount := 0
	for rows.Next() {
		// Create slice of pointers for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to scan row: %v\n", err)
			continue
		}
		
		// Print row
		for i, val := range values {
			if i > 0 {
				fmt.Print(" | ")
			}
			if val == nil {
				fmt.Print("NULL")
			} else {
				fmt.Print(val)
			}
		}
		fmt.Println()
		rowCount++
	}
	
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error iterating rows: %v\n", err)
		return err
	}
	
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("✅ Executed successfully (%d row(s))\n", rowCount)
	
	return nil
}

func detectProvider(connStr string) string {
	if strings.Contains(connStr, "mysql") {
		return "mysql"
	} else if strings.Contains(connStr, "sqlite") || strings.Contains(connStr, "file:") {
		return "sqlite"
	}
	return "postgresql"
}

// normalizeProviderForDriver normalizes provider name for sql.Open
// PostgreSQL driver uses "postgres", not "postgresql"
func normalizeProviderForDriver(provider string) string {
	if provider == "postgresql" || provider == "postgres" {
		return "postgres"
	}
	return provider
}

func generatePrismaSchemaFromDB(schema *introspect.DatabaseSchema, provider string) string {
	var result strings.Builder
	
	// Datasource
	result.WriteString("datasource db {\n")
	result.WriteString(fmt.Sprintf("  provider = \"%s\"\n", provider))
	result.WriteString("  url      = env(\"DATABASE_URL\")\n")
	result.WriteString("}\n\n")
	
	// Generator
	result.WriteString("generator client {\n")
	result.WriteString("  provider = \"prisma-client-go\"\n")
	result.WriteString("  output   = \"./generated\"\n")
	result.WriteString("}\n\n")
	
	// Models
	for _, table := range schema.Tables {
		result.WriteString(fmt.Sprintf("model %s {\n", toPascalCase(table.Name)))
		
		for _, col := range table.Columns {
			fieldType := mapDBTypeToPrisma(col.Type)
			nullable := ""
			if col.Nullable {
				nullable = "?"
			}
			
			attrs := ""
			// Check if primary key
			if table.PrimaryKey != nil && len(table.PrimaryKey.Columns) == 1 && table.PrimaryKey.Columns[0] == col.Name {
				attrs += " @id"
				if col.AutoIncrement {
					attrs += " @default(autoincrement())"
				}
			}
			
			// Check for unique indexes
			for _, idx := range table.Indexes {
				if idx.IsUnique && len(idx.Columns) == 1 && idx.Columns[0] == col.Name {
					attrs += " @unique"
					break
				}
			}
			
			result.WriteString(fmt.Sprintf("  %s %s%s%s\n", col.Name, fieldType, nullable, attrs))
		}
		
		result.WriteString("}\n\n")
	}
	
	return result.String()
}

func mapDBTypeToPrisma(dbType string) string {
	dbType = strings.ToUpper(dbType)
	switch {
	case strings.Contains(dbType, "INT"), strings.Contains(dbType, "SERIAL"):
		return "Int"
	case strings.Contains(dbType, "BOOL"):
		return "Boolean"
	case strings.Contains(dbType, "VARCHAR"), strings.Contains(dbType, "TEXT"), strings.Contains(dbType, "CHAR"):
		return "String"
	case strings.Contains(dbType, "TIMESTAMP"), strings.Contains(dbType, "DATE"):
		return "DateTime"
	case strings.Contains(dbType, "DECIMAL"), strings.Contains(dbType, "NUMERIC"), strings.Contains(dbType, "FLOAT"), strings.Contains(dbType, "DOUBLE"), strings.Contains(dbType, "REAL"):
		return "Float"
	case strings.Contains(dbType, "JSON"):
		return "Json"
	default:
		return "String"
	}
}

func toPascalCase(s string) string {
	words := strings.Split(s, "_")
	result := ""
	for _, word := range words {
		if len(word) > 0 {
			result += strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return result
}

// extractConnectionInfo extracts provider and connection string from schema
func extractConnectionInfo(schema *psl.SchemaAst) (string, string) {
	provider := "postgresql"
	connStr := ""
	
	for _, top := range schema.Tops {
		if source := top.AsSource(); source != nil {
			for _, prop := range source.Properties {
				if prop.Name.Name == "provider" {
					if strLit, _ := prop.Value.AsStringValue(); strLit != nil {
						provider = strLit.Value
					}
				}
				if prop.Name.Name == "url" {
					// Handle env("DATABASE_URL") function call
					if fnCall := prop.Value.AsFunction(); fnCall != nil && fnCall.Name.Name == "env" {
						if len(fnCall.Arguments) > 0 {
							if strLit, _ := fnCall.Arguments[0].AsStringValue(); strLit != nil {
								envVar := strLit.Value
								connStr = os.Getenv(envVar)
							}
						}
					} else if strLit, _ := prop.Value.AsStringValue(); strLit != nil {
						// Direct string literal
						connStr = strLit.Value
					}
				}
			}
		}
	}
	
	return provider, connStr
}

