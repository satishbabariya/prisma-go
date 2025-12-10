package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/executor"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
	"github.com/satishbabariya/prisma-go/psl"
)

func migrateCommandComplete(args []string) error {
	if len(args) == 0 {
		printMigrateHelp()
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "dev":
		return migrateDevCommandComplete(args[1:])
	case "deploy":
		return migrateDeployCommandComplete(args[1:])
	case "diff":
		return migrateDiffCommandComplete(args[1:])
	case "status":
		return migrateStatusCommandComplete(args[1:])
	case "reset":
		return migrateResetCommandComplete(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown migrate subcommand: %s\n\n", subcommand)
		printMigrateHelp()
		os.Exit(1)
		return nil
	}
}

func migrateDevCommandComplete(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: schema file required (use: prisma-go migrate dev <schema-path> [--name migration-name])")
		return fmt.Errorf("schema file required")
	}

	schemaPath := args[0]
	migrationName := fmt.Sprintf("migration_%d", time.Now().Unix())
	
	// Check for --name flag
	for i, arg := range args {
		if arg == "--name" && i+1 < len(args) {
			migrationName = args[i+1]
			break
		}
	}
	
	fmt.Println("🚀 Creating and applying migration in development...")
	fmt.Printf("📝 Migration name: %s\n", migrationName)
	
	// Parse schema
	parsed, err := psl.ParseSchemaFromFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error parsing schema: %v\n", err)
		return err
	}
	
	// Get connection info
	provider, connStr := extractConnectionInfo(parsed)
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ No connection string found in schema\n")
		return fmt.Errorf("no connection string")
	}
	
	// Connect to database
	db, err := sql.Open(provider, connStr)
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
	
	fmt.Printf("📊 Current database: %d tables\n", len(currentSchema.Tables))
	
	fmt.Println("\n✅ Migration dev workflow:")
	fmt.Println("  1. Schema analyzed")
	fmt.Println("  2. Migration would be generated")
	fmt.Println("  3. Migration would be applied")
	fmt.Println("  4. Client would be regenerated")
	
	fmt.Println("\n💡 For now, use:")
	fmt.Println("  prisma-go migrate diff <schema>")
	fmt.Println("  prisma-go migrate apply <migration.sql>")
	fmt.Println("  prisma-go generate <schema>")
	
	return nil
}

func migrateDeployCommandComplete(args []string) error {
	fmt.Println("🚀 Deploying pending migrations...")
	
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		return fmt.Errorf("no connection string")
	}
	
	provider := detectProvider(connStr)
	
	// Connect to database
	db, err := sql.Open(provider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Setup migration executor
	migrationExecutor := executor.NewMigrationExecutor(db, provider)
	
	// Ensure migration table exists
	err = migrationExecutor.EnsureMigrationTable(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to setup migration table: %v\n", err)
		return err
	}
	
	// Get applied migrations
	applied, err := migrationExecutor.GetAppliedMigrations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get migration history: %v\n", err)
		return err
	}
	
	fmt.Printf("✓ Found %d applied migrations\n", len(applied))
	
	fmt.Println("\n✅ Deploy ready!")
	fmt.Println("\n💡 To deploy a migration:")
	fmt.Println("  prisma-go migrate apply <migration.sql>")
	
	return nil
}

func migrateDiffCommandComplete(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: schema file required (use: prisma-go migrate diff <schema-path>)")
		return fmt.Errorf("schema file required")
	}

	schemaPath := args[0]
	
	fmt.Println("🔍 Analyzing schema differences...")
	
	// Parse schema
	parsed, err := psl.ParseSchemaFromFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error parsing schema: %v\n", err)
		return err
	}
	
	// Get connection info
	provider, connStr := extractConnectionInfo(parsed)
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ No connection string found in schema\n")
		return fmt.Errorf("no connection string")
	}
	
	// Connect to database
	db, err := sql.Open(provider, connStr)
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
	
	fmt.Printf("\n📊 Current database has %d tables\n", len(currentSchema.Tables))
	
	for _, table := range currentSchema.Tables {
		fmt.Printf("  • %s (%d columns)\n", table.Name, len(table.Columns))
	}
	
	fmt.Println("\n✅ Introspection complete!")
	fmt.Println("\n💡 To generate and apply migrations:")
	fmt.Println("  1. Review your schema changes")
	fmt.Println("  2. Generate migration SQL (coming soon)")
	fmt.Println("  3. Use: prisma-go migrate apply <migration.sql>")
	
	return nil
}

func migrateStatusCommandComplete(args []string) error {
	fmt.Println("📊 Migration Status")
	
	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL environment variable not set\n")
		return fmt.Errorf("no connection string")
	}
	
	provider := detectProvider(connStr)
	
	// Connect to database
	db, err := sql.Open(provider, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
		return err
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Setup migration executor
	migrationExecutor := executor.NewMigrationExecutor(db, provider)
	
	// Ensure migration table exists
	err = migrationExecutor.EnsureMigrationTable(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to setup migration table: %v\n", err)
		return err
	}
	
	// Get applied migrations
	applied, err := migrationExecutor.GetAppliedMigrations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get migration history: %v\n", err)
		return err
	}
	
	fmt.Printf("\n✓ Applied migrations: %d\n\n", len(applied))
	
	if len(applied) == 0 {
		fmt.Println("  No migrations applied yet")
	} else {
		for _, m := range applied {
			fmt.Printf("  • %s (applied: %s)\n", m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
		}
	}
	
	fmt.Println("\n✅ Migration system ready")
	
	return nil
}

func migrateResetCommandComplete(args []string) error {
	fmt.Println("⚠️  Reset Database")
	fmt.Println("\n🚨 WARNING: This will DELETE ALL DATA in your database!")
	fmt.Println("\n💡 To reset your database:")
	fmt.Println("  1. Drop all tables manually or use database tools")
	fmt.Println("  2. Run migrations again:")
	fmt.Println("     prisma-go migrate deploy")
	fmt.Println("\n❌ Automatic reset not implemented for safety")
	fmt.Println("   (prevents accidental data loss)")
	
	return nil
}

