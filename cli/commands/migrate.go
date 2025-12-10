package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/migrate/converter"
	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/executor"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
	psl "github.com/satishbabariya/prisma-go/psl"
)

func migrateCommand(args []string) error {
	if len(args) == 0 {
		printMigrateHelp()
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "dev":
		return migrateDevCommand(args[1:])
	case "deploy":
		return migrateDeployCommand(args[1:])
	case "diff":
		return migrateDiffCommand(args[1:])
	case "apply":
		return migrateApplyCommand(args[1:])
	case "status":
		return migrateStatusCommand(args[1:])
	case "resolve":
		return migrateResolveCommand(args[1:])
	case "reset":
		return migrateResetCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown migrate subcommand: %s\n\n", subcommand)
		printMigrateHelp()
		os.Exit(1)
		return nil
	}
}

func printMigrateHelp() {
	help := `
USAGE:
    prisma-go migrate <subcommand> [options]

SUBCOMMANDS:
    dev        Create and apply migrations in development
    deploy     Apply pending migrations to production
    diff       Compare schema to database
    apply      Apply a migration SQL file
    status     Check migration status
    resolve    Resolve migration conflicts
    reset      Reset the database

EXAMPLES:
    prisma-go migrate dev schema.prisma --name init
    prisma-go migrate deploy
    prisma-go migrate diff schema.prisma --create-only --name init
    prisma-go migrate apply migrations/20250110_init/migration.sql
    prisma-go migrate status
`
	fmt.Println(help)
}

func migrateDevCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: schema file required (use: prisma-go migrate dev <schema-path> [--name migration-name] [--apply])")
		return fmt.Errorf("schema file required")
	}

	schemaPath := args[0]
	migrationName := ""
	autoApply := false
	
	// Parse flags
	for i, arg := range args {
		if arg == "--name" && i+1 < len(args) {
			migrationName = args[i+1]
		}
		if arg == "--apply" {
			autoApply = true
		}
	}
	
	// Generate default migration name if not provided
	if migrationName == "" {
		migrationName = fmt.Sprintf("migration_%d", time.Now().Unix())
	}
	
	fmt.Println("🚀 Creating and applying migration in development...")
	fmt.Printf("📝 Migration name: %s\n", migrationName)
	
	// Step 1: Generate migration SQL using migrate diff logic
	fmt.Println("\n📋 Step 1: Analyzing schema differences...")
	
	// Read and parse schema
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read schema: %v\n", err)
		return err
	}

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
	
	// Convert schema AST to database schema
	targetSchema, err := converter.ConvertASTToDBSchema(parsed, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to convert schema: %v\n", err)
		return err
	}
	
	// Compare schemas
	differ := diff.NewSimpleDiffer(provider)
	diffResult := differ.CompareSchemas(currentSchema, targetSchema)
	
	if len(diffResult.TablesToCreate) == 0 && len(diffResult.TablesToAlter) == 0 && len(diffResult.TablesToDrop) == 0 {
		fmt.Println("✅ No differences found - schema is up to date!")
		fmt.Println("\n💡 Regenerating client...")
		// Regenerate client even if no migrations
		if err := generateCommand([]string{schemaPath}); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to regenerate client: %v\n", err)
		}
		return nil
	}
	
	// Step 2: Generate SQL
	fmt.Println("📝 Step 2: Generating migration SQL...")
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
	
	// Step 3: Save migration file
	fmt.Println("💾 Step 3: Saving migration file...")
	if err := os.MkdirAll("migrations", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create migrations directory: %v\n", err)
		return err
	}
	
	migrationDir := fmt.Sprintf("migrations/%s", migrationName)
	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create migration directory: %v\n", err)
		return err
	}
	
	sqlPath := fmt.Sprintf("%s/migration.sql", migrationDir)
	if err := os.WriteFile(sqlPath, []byte(sql), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write migration file: %v\n", err)
		return err
	}
	
	absPath, _ := filepath.Abs(sqlPath)
	fmt.Printf("✅ Migration SQL saved: %s\n", absPath)
	
	// Step 4: Apply migration (if --apply flag)
	if autoApply {
		fmt.Println("\n🚀 Step 4: Applying migration...")
		if err := migrateApplyCommand([]string{sqlPath, "--name", migrationName}); err != nil {
			return err
		}
	} else {
		fmt.Println("\n💡 Migration generated but not applied.")
		fmt.Printf("   Apply it with: prisma-go migrate apply %s\n", sqlPath)
		fmt.Println("   Or use --apply flag to auto-apply: prisma-go migrate dev <schema> --apply")
	}
	
	// Step 5: Regenerate client
	fmt.Println("\n🔄 Step 5: Regenerating client...")
	if err := generateCommand([]string{schemaPath}); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to regenerate client: %v\n", err)
		// Don't fail the whole command if client generation fails
	} else {
		fmt.Println("✅ Client regenerated successfully!")
	}
	
	fmt.Println("\n🎉 Migration dev workflow completed!")
	return nil
}

func migrateDeployCommand(args []string) error {
	fmt.Println("🚀 Deploying pending migrations...")
	
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
	
	// Build map of applied migration names
	appliedMap := make(map[string]bool)
	for _, m := range applied {
		appliedMap[m.Name] = true
	}
	
	fmt.Printf("✓ Found %d applied migrations\n", len(applied))
	
	// Scan migrations directory for pending migrations
	migrationsDir := "migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("\n✅ No migrations directory found - nothing to deploy")
			return nil
		}
		fmt.Fprintf(os.Stderr, "❌ Failed to read migrations directory: %v\n", err)
		return err
	}
	
	var pendingMigrations []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		migrationName := entry.Name()
		if appliedMap[migrationName] {
			continue // Already applied
		}
		
		// Check if migration.sql exists
		sqlPath := filepath.Join(migrationsDir, migrationName, "migration.sql")
		if _, err := os.Stat(sqlPath); err == nil {
			pendingMigrations = append(pendingMigrations, sqlPath)
		}
	}
	
	if len(pendingMigrations) == 0 {
		fmt.Println("\n✅ No pending migrations found - database is up to date!")
		return nil
	}
	
	fmt.Printf("\n📋 Found %d pending migration(s):\n", len(pendingMigrations))
	for _, path := range pendingMigrations {
		migrationName := filepath.Base(filepath.Dir(path))
		fmt.Printf("  • %s\n", migrationName)
	}
	
	fmt.Print("\n❓ Apply these migrations? (y/N): ")
	var confirmation string
	fmt.Scanln(&confirmation)
	
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))
	if confirmation != "y" && confirmation != "yes" {
		fmt.Println("✋ Aborted. No migrations applied.")
		return nil
	}
	
	// Apply pending migrations
	fmt.Println("\n🚀 Applying pending migrations...")
	successCount := 0
	failCount := 0
	
	for _, sqlPath := range pendingMigrations {
		migrationName := filepath.Base(filepath.Dir(sqlPath))
		fmt.Printf("\n📝 Applying: %s\n", migrationName)
		
		// Read SQL file
		sqlContent, err := os.ReadFile(sqlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ❌ Failed to read migration file: %v\n", err)
			failCount++
			continue
		}
		
		// Apply migration
		err = migrationExecutor.ExecuteMigration(ctx, string(sqlContent), migrationName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ❌ Failed to apply migration: %v\n", err)
			failCount++
			continue
		}
		
		fmt.Printf("  ✅ Applied successfully\n")
		successCount++
	}
	
	fmt.Printf("\n📊 Deployment Summary:\n")
	fmt.Printf("  ✅ Applied: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("  ❌ Failed: %d\n", failCount)
	}
	
	if failCount == 0 {
		fmt.Println("\n🎉 All migrations deployed successfully!")
	} else {
		fmt.Printf("\n⚠️  Some migrations failed. Please review errors above.\n")
		return fmt.Errorf("%d migration(s) failed", failCount)
	}
	
	return nil
}

func migrateDiffCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: schema file required (use: prisma-go migrate diff <schema-path> [--create-only])")
		return fmt.Errorf("schema file required")
	}

	schemaPath := args[0]
	createOnly := false
	migrationName := ""
	
	// Parse flags
	for i, arg := range args {
		if arg == "--create-only" || arg == "--create" {
			createOnly = true
		}
		if arg == "--name" && i+1 < len(args) {
			migrationName = args[i+1]
		}
	}
	
	fmt.Println("🔍 Analyzing schema differences...")
	
	// Read and parse schema
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read schema: %v\n", err)
		return err
	}

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
	
	fmt.Printf("\n📊 Current database has %d tables\n", len(currentSchema.Tables))
	
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
	fmt.Println("\n📝 Differences found:")
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
		fmt.Println("  ✓ No differences found - schema is up to date!")
		return nil
	}
	
	// Generate SQL if --create-only flag is set
	if createOnly {
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
		
		// Create migrations directory
		if err := os.MkdirAll("migrations", 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create migrations directory: %v\n", err)
			return err
		}
		
		// Generate migration name
		if migrationName == "" {
			migrationName = fmt.Sprintf("migration_%d", time.Now().Unix())
		}
		migrationDir := fmt.Sprintf("migrations/%s", migrationName)
		if err := os.MkdirAll(migrationDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create migration directory: %v\n", err)
			return err
		}
		
		// Write SQL file
		sqlPath := fmt.Sprintf("%s/migration.sql", migrationDir)
		if err := os.WriteFile(sqlPath, []byte(sql), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to write migration file: %v\n", err)
			return err
		}
		
		absPath, _ := filepath.Abs(sqlPath)
		fmt.Printf("\n✅ Migration SQL generated: %s\n", absPath)
		fmt.Println("\n💡 Next steps:")
		fmt.Printf("  1. Review the migration SQL\n")
		fmt.Printf("  2. Apply it: prisma-go migrate apply %s\n", sqlPath)
	} else {
		fmt.Println("\n💡 To generate migration SQL file:")
		fmt.Println("  prisma-go migrate diff <schema> --create-only [--name migration-name]")
	}
	
	return nil
}

func migrateApplyCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: migration file required (use: prisma-go migrate apply <migration-file.sql> [--name migration-name])")
		return fmt.Errorf("migration file required")
	}

	migrationPath := args[0]
	migrationName := ""
	
	// Parse flags
	for i, arg := range args {
		if arg == "--name" && i+1 < len(args) {
			migrationName = args[i+1]
		}
	}
	
	// Extract migration name from path if not provided
	if migrationName == "" {
		// Extract from path like migrations/20250110_init/migration.sql
		dir := filepath.Dir(migrationPath)
		migrationName = filepath.Base(dir)
		if migrationName == "." || migrationName == "/" {
			migrationName = filepath.Base(migrationPath)
			// Remove .sql extension
			if ext := filepath.Ext(migrationName); ext == ".sql" {
				migrationName = migrationName[:len(migrationName)-len(ext)]
			}
		}
	}
	
	fmt.Printf("🚀 Applying migration: %s\n", migrationName)
	
	// Read migration SQL file
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read migration file: %v\n", err)
		return err
	}
	
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
	
	// Setup migration executor
	migrationExecutor := executor.NewMigrationExecutor(db, provider)
	
	// Ensure migration table exists
	err = migrationExecutor.EnsureMigrationTable(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to setup migration table: %v\n", err)
		return err
	}
	
	// Execute migration
	fmt.Println("📝 Executing migration SQL...")
	err = migrationExecutor.ExecuteMigration(ctx, string(sqlContent), migrationName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to apply migration: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ Migration '%s' applied successfully!\n", migrationName)
	
	// Show updated status
	applied, err := migrationExecutor.GetAppliedMigrations(ctx)
	if err == nil {
		fmt.Printf("\n📊 Total applied migrations: %d\n", len(applied))
	}
	
	return nil
}

func migrateStatusCommand(args []string) error {
	fmt.Println("📊 Migration Status")
	
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

func migrateResolveCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: migration name required (use: prisma-go migrate resolve <migration-name> [--applied|--rolled-back])")
		return fmt.Errorf("migration name required")
	}

	migrationName := args[0]
	action := "applied"
	
	// Parse flags
	for i, arg := range args {
		if arg == "--applied" {
			action = "applied"
		}
		if arg == "--rolled-back" {
			action = "rolled-back"
		}
		if arg == "--action" && i+1 < len(args) {
			action = args[i+1]
		}
	}
	
	fmt.Printf("🔧 Resolving migration: %s\n", migrationName)
	fmt.Printf("📝 Action: %s\n", action)
	
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
	
	// Setup migration executor
	migrationExecutor := executor.NewMigrationExecutor(db, provider)
	
	// Ensure migration table exists
	err = migrationExecutor.EnsureMigrationTable(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to setup migration table: %v\n", err)
		return err
	}
	
	// Check if migration already exists
	applied, err := migrationExecutor.GetAppliedMigrations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get migration history: %v\n", err)
		return err
	}
	
	// Check if migration is already applied
	for _, m := range applied {
		if m.Name == migrationName {
			if action == "applied" {
				fmt.Printf("✅ Migration '%s' is already marked as applied\n", migrationName)
				return nil
			}
			// Mark as rolled-back (remove from history)
			_, err = db.ExecContext(ctx, "DELETE FROM _prisma_migrations WHERE migration_name = $1", migrationName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to mark migration as rolled-back: %v\n", err)
				return err
			}
			fmt.Printf("✅ Migration '%s' marked as rolled-back\n", migrationName)
			return nil
		}
	}
	
	// Migration not found - mark as applied
	if action == "applied" {
		// Record migration without executing SQL
		checksum := fmt.Sprintf("%x", time.Now().UnixNano())
		var insertSQL string
		switch provider {
		case "postgresql", "postgres":
			insertSQL = "INSERT INTO _prisma_migrations (migration_name, applied_at, checksum) VALUES ($1, $2, $3)"
		case "mysql":
			insertSQL = "INSERT INTO _prisma_migrations (migration_name, applied_at, checksum) VALUES (?, ?, ?)"
		case "sqlite":
			insertSQL = "INSERT INTO _prisma_migrations (migration_name, applied_at, checksum) VALUES (?, ?, ?)"
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}
		
		_, err = db.ExecContext(ctx, insertSQL, migrationName, time.Now(), checksum)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to mark migration as applied: %v\n", err)
			return err
		}
		fmt.Printf("✅ Migration '%s' marked as applied (without executing SQL)\n", migrationName)
	} else {
		fmt.Printf("✅ Migration '%s' marked as %s\n", migrationName, action)
	}
	
	return nil
}

func migrateResetCommand(args []string) error {
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

