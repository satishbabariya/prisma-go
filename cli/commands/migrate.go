package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/satishbabariya/prisma-go/migrate/converter"
	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/executor"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/shadow"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
	psl "github.com/satishbabariya/prisma-go/psl"
	"github.com/satishbabariya/prisma-go/telemetry"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `Manage database migrations for your Prisma schema.

This command provides subcommands for:
- Creating and applying migrations in development
- Deploying migrations to production
- Comparing schema differences
- Checking migration status`,
}

var (
	migrateDevCmd      *cobra.Command
	migrateDeployCmd   *cobra.Command
	migrateDiffCmd     *cobra.Command
	migrateApplyCmd    *cobra.Command
	migrateStatusCmd   *cobra.Command
	migrateResetCmd    *cobra.Command
	migrateResolveCmd  *cobra.Command
	migrateRollbackCmd *cobra.Command
)

func init() {
	initMigrateCommands()
	initMigrateFlags()

	// Add migrate subcommands
	migrateCmd.AddCommand(migrateDevCmd)
	migrateCmd.AddCommand(migrateDeployCmd)
	migrateCmd.AddCommand(migrateDiffCmd)
	migrateCmd.AddCommand(migrateApplyCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateResetCmd)
	migrateCmd.AddCommand(migrateResolveCmd)
	migrateCmd.AddCommand(migrateRollbackCmd)

	rootCmd.AddCommand(migrateCmd)
}

func initMigrateCommands() {
	migrateDevCmd = &cobra.Command{
		Use:   "dev [schema-path]",
		Short: "Create and apply migrations in development",
		Long:  "Create a new migration and optionally apply it to your development database",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			schemaPath := "schema.prisma"
			if len(args) > 0 {
				schemaPath = args[0]
			}
			migrationName, _ := cmd.Flags().GetString("name")
			autoApply, _ := cmd.Flags().GetBool("apply")
			argsList := []string{schemaPath}
			if migrationName != "" {
				argsList = append(argsList, "--name", migrationName)
			}
			if autoApply {
				argsList = append(argsList, "--apply")
			}
			return migrateDevCommand(argsList)
		},
	}

	migrateDeployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Apply pending migrations to production",
		Long:  "Apply all pending migrations to your production database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateDeployCommand([]string{})
		},
	}

	migrateDiffCmd = &cobra.Command{
		Use:   "diff [schema-path]",
		Short: "Compare schema to database",
		Long:  "Compare your Prisma schema to the current database state",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			schemaPath := "schema.prisma"
			if len(args) > 0 {
				schemaPath = args[0]
			}
			createOnly, _ := cmd.Flags().GetBool("create-only")
			migrationName, _ := cmd.Flags().GetString("name")
			argsList := []string{schemaPath}
			if createOnly {
				argsList = append(argsList, "--create-only")
			}
			if migrationName != "" {
				argsList = append(argsList, "--name", migrationName)
			}
			return migrateDiffCommand(argsList)
		},
	}

	migrateApplyCmd = &cobra.Command{
		Use:   "apply [migration-file]",
		Short: "Apply a migration SQL file",
		Long:  "Apply a specific migration SQL file to your database",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationName, _ := cmd.Flags().GetString("name")
			argsList := args
			if migrationName != "" {
				argsList = append(argsList, "--name", migrationName)
			}
			return migrateApplyCommand(argsList)
		},
	}

	migrateStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check migration status",
		Long:  "Check the status of applied migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateStatusCommand([]string{})
		},
	}

	migrateResetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset the database",
		Long:  "Reset the database (WARNING: This will delete all data)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateResetCommand([]string{})
		},
	}

	migrateResolveCmd = &cobra.Command{
		Use:   "resolve [migration-name]",
		Short: "Resolve failed migrations",
		Long:  "Mark a failed migration as applied or rolled-back",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action, _ := cmd.Flags().GetString("action")
			argsList := args
			if action != "" {
				argsList = append(argsList, "--action", action)
			}
			return migrateResolveCommand(argsList)
		},
	}

	migrateRollbackCmd = &cobra.Command{
		Use:   "rollback [migration-name]",
		Short: "Rollback migrations",
		Long:  "Rollback one or more migrations by executing their rollback SQL",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			steps, _ := cmd.Flags().GetInt("steps")
			argsList := args
			if steps > 0 {
				argsList = append(argsList, "--steps", fmt.Sprintf("%d", steps))
			}
			return migrateRollbackCommand(argsList)
		},
	}
}

func initMigrateFlags() {
	migrateDevCmd.Flags().StringP("name", "n", "", "Migration name")
	migrateDevCmd.Flags().Bool("apply", false, "Automatically apply migration after creation")

	migrateDiffCmd.Flags().Bool("create-only", false, "Only create migration file, don't apply")
	migrateDiffCmd.Flags().Bool("skip-shadow-db", false, "Skip using shadow database for diffing")
	migrateDiffCmd.Flags().StringP("name", "n", "", "Migration name")

	migrateApplyCmd.Flags().StringP("name", "n", "", "Migration name")

	migrateResolveCmd.Flags().StringP("action", "a", "applied", "Action: applied or rolled-back")
	migrateRollbackCmd.Flags().IntP("steps", "s", 1, "Number of migrations to rollback")
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
    rollback   Rollback one or more migrations
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
	startTime := time.Now()
	var err error
	var provider string
	defer func() {
		telemetry.RecordCommand("migrate dev", provider, time.Since(startTime), err)
	}()

	schemaPath := "schema.prisma"
	migrationName := ""
	autoApply := false

	// Parse arguments - first non-flag arg is schema path, rest are flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			// It's a flag
			if arg == "--name" && i+1 < len(args) {
				migrationName = args[i+1]
				i++ // Skip next arg as it's the flag value
			} else if arg == "--apply" {
				autoApply = true
			}
		} else if schemaPath == "schema.prisma" {
			// First non-flag argument is schema path
			schemaPath = arg
		}
	}

	// Use standardized schema path resolution
	schemaPath = getSchemaPath("", []string{schemaPath})

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
	differ, err := diff.NewDiffer(provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create differ: %v\n", err)
		return err
	}
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

	// Generate rollback SQL
	rollbackSQL, err := sqlGenerator.GenerateRollbackSQL(diffResult, targetSchema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to generate rollback SQL: %v\n", err)
		// Don't fail, just warn
		rollbackSQL = "-- Rollback SQL generation failed\n"
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

	rollbackPath := fmt.Sprintf("%s/rollback.sql", migrationDir)
	if err := os.WriteFile(rollbackPath, []byte(rollbackSQL), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to write rollback file: %v\n", err)
		// Don't fail, just warn
	}

	absPath, _ := filepath.Abs(sqlPath)
	fmt.Printf("✅ Migration SQL saved: %s\n", absPath)
	rollbackAbsPath, _ := filepath.Abs(rollbackPath)
	fmt.Printf("✅ Rollback SQL saved: %s\n", rollbackAbsPath)

	// Step 4: Apply migration (if --apply flag)
	if autoApply {
		fmt.Println("\n🚀 Step 4: Applying migration...")
		// Apply migration directly using the same connection
		migrationExecutor := executor.NewMigrationExecutor(db, provider)

		// Ensure migration table exists
		err = migrationExecutor.EnsureMigrationTable(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to setup migration table: %v\n", err)
			return err
		}

		// Read migration SQL
		migrationSQL, err := os.ReadFile(sqlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to read migration file: %v\n", err)
			return err
		}

		// Execute migration
		fmt.Println("📝 Executing migration SQL...")
		err = migrationExecutor.ExecuteMigration(ctx, string(migrationSQL), migrationName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to apply migration: %v\n", err)
			return err
		}

		fmt.Printf("✅ Migration '%s' applied successfully!\n", migrationName)
	} else {
		fmt.Println("\n💡 Migration generated but not applied.")
		fmt.Printf("   Apply it with: prisma-go migrate apply %s\n", sqlPath)
		fmt.Println("   Or use --apply flag to auto-apply: prisma-go migrate dev <schema> --apply")
	}

	// Step 5: Regenerate client
	fmt.Println("\n🔄 Step 5: Regenerating client...")
	if err := runGenerate(nil, []string{schemaPath}); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to regenerate client: %v\n", err)
		// Don't fail the whole command if client generation fails
	} else {
		fmt.Println("✅ Client regenerated successfully!")
	}

	fmt.Println("\n🎉 Migration dev workflow completed!")
	err = nil
	return nil
}

func migrateDeployCommand(args []string) error {
	fmt.Println("🚀 Deploying pending migrations...")

	// Get connection string from environment or .env files
	connStr := getDatabaseURLFromEnv()
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL not found in environment or .env files\n")
		fmt.Fprintf(os.Stderr, "💡 Set DATABASE_URL environment variable or add it to .env file\n")
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
	schemaPath := "schema.prisma"
	createOnly := false
	migrationName := ""
	skipShadow := false

	// Parse arguments - first non-flag arg is schema path, rest are flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			// It's a flag
			if arg == "--create-only" || arg == "--create" {
				createOnly = true
			} else if arg == "--skip-shadow-db" {
				skipShadow = true
			} else if arg == "--name" && i+1 < len(args) {
				migrationName = args[i+1]
				i++ // Skip next arg as it's the flag value
			}
		} else if schemaPath == "schema.prisma" {
			// First non-flag argument is schema path
			schemaPath = arg
		}
	}

	// Use standardized schema path resolution
	schemaPath = getSchemaPath("", []string{schemaPath})

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

	// Get connection info (including shadow database URL)
	provider, connStr, shadowConnStr := extractConnectionInfoWithShadow(parsed)
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ No connection string found in schema\n")
		return fmt.Errorf("no connection string")
	}

	ctx := context.Background()
	var currentSchema *introspect.DatabaseSchema

	// Use shadow database if not skipped
	if !skipShadow {
		fmt.Println("🌑 Setting up shadow database...")
		shadowDB := shadow.NewShadowDB(provider, connStr, shadowConnStr, skipShadow)

		// Create shadow database
		if err := shadowDB.Create(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to create shadow database: %v\n", err)
			fmt.Fprintf(os.Stderr, "💡 Falling back to main database. Use --skip-shadow-db to skip shadow database.\n")
			skipShadow = true
		} else {
			defer func() {
				if err := shadowDB.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Failed to close shadow database: %v\n", err)
				}
			}()

			// Apply existing migrations to shadow database
			fmt.Println("📦 Applying migrations to shadow database...")
			migrationsDir := "migrations"
			if entries, err := os.ReadDir(migrationsDir); err == nil {
				var migrationSQLs []string
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					migrationName := entry.Name()
					sqlPath := filepath.Join(migrationsDir, migrationName, "migration.sql")
					if sqlContent, err := os.ReadFile(sqlPath); err == nil {
						migrationSQLs = append(migrationSQLs, string(sqlContent))
					}
				}

				if err := shadowDB.ApplyMigrations(ctx, migrationSQLs); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Failed to apply migrations to shadow database: %v\n", err)
					fmt.Fprintf(os.Stderr, "💡 Falling back to main database. Use --skip-shadow-db to skip shadow database.\n")
					skipShadow = true
				} else {
					fmt.Printf("✅ Applied %d migrations to shadow database\n", len(migrationSQLs))
				}
			}

			// Introspect shadow database
			if !skipShadow {
				currentSchema, err = shadowDB.Introspect(ctx)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Failed to introspect shadow database: %v\n", err)
					fmt.Fprintf(os.Stderr, "💡 Falling back to main database. Use --skip-shadow-db to skip shadow database.\n")
					skipShadow = true
				} else {
					fmt.Printf("📊 Shadow database has %d tables\n", len(currentSchema.Tables))
				}
			}
		}
	}

	// Fallback to main database if shadow database failed or was skipped
	if skipShadow || currentSchema == nil {
		fmt.Println("📊 Using main database for comparison...")
		driverProvider := normalizeProviderForDriver(provider)

		// Connect to database
		db, err := sql.Open(driverProvider, connStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to connect: %v\n", err)
			return err
		}
		defer db.Close()

		// Introspect current database
		introspector, err := introspect.NewIntrospector(db, provider)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create introspector: %v\n", err)
			return err
		}

		currentSchema, err = introspector.Introspect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to introspect database: %v\n", err)
			return err
		}

		fmt.Printf("\n📊 Current database has %d tables\n", len(currentSchema.Tables))
	}

	// Convert schema AST to database schema
	targetSchema, err := converter.ConvertASTToDBSchema(parsed, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to convert schema: %v\n", err)
		return err
	}

	fmt.Printf("📋 Schema defines %d tables\n", len(targetSchema.Tables))

	// Compare schemas
	differ, err := diff.NewDiffer(provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create differ: %v\n", err)
		return err
	}
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

	// Get connection string from environment or .env files
	connStr := getDatabaseURLFromEnv()
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL not found in environment or .env files\n")
		fmt.Fprintf(os.Stderr, "💡 Set DATABASE_URL environment variable or add it to .env file\n")
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

	// Get connection string from environment or .env files
	connStr := getDatabaseURLFromEnv()
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL not found in environment or .env files\n")
		fmt.Fprintf(os.Stderr, "💡 Set DATABASE_URL environment variable or add it to .env file\n")
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

	// Get connection string from environment or .env files
	connStr := getDatabaseURLFromEnv()
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL not found in environment or .env files\n")
		fmt.Fprintf(os.Stderr, "💡 Set DATABASE_URL environment variable or add it to .env file\n")
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
			var deleteSQL string
			switch provider {
			case "postgresql", "postgres":
				deleteSQL = "DELETE FROM _prisma_migrations WHERE migration_name = $1"
			case "mysql", "sqlite":
				deleteSQL = "DELETE FROM _prisma_migrations WHERE migration_name = ?"
			default:
				return fmt.Errorf("unsupported provider: %s", provider)
			}
			_, err = db.ExecContext(ctx, deleteSQL, migrationName)
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

func migrateRollbackCommand(args []string) error {
	steps := 1
	migrationName := ""

	// Parse arguments
	for i, arg := range args {
		if arg == "--steps" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &steps)
			i++ // Skip next arg
		} else if migrationName == "" && !strings.HasPrefix(arg, "--") {
			migrationName = arg
		}
	}

	fmt.Println("⏪ Rolling back migrations...")

	// Get connection string from environment or .env files
	connStr := getDatabaseURLFromEnv()
	if connStr == "" {
		fmt.Fprintf(os.Stderr, "❌ DATABASE_URL not found in environment or .env files\n")
		fmt.Fprintf(os.Stderr, "💡 Set DATABASE_URL environment variable or add it to .env file\n")
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

	if len(applied) == 0 {
		fmt.Println("✅ No migrations to rollback")
		return nil
	}

	// Filter out already rolled back migrations
	// Note: We'll check rollback status when attempting rollback
	var migrationsToRollback []executor.Migration
	for _, m := range applied {
		migrationsToRollback = append(migrationsToRollback, m)
	}

	// If specific migration name provided, rollback only that one
	if migrationName != "" {
		found := false
		for _, m := range applied {
			if m.Name == migrationName {
				migrationsToRollback = []executor.Migration{m}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("migration '%s' not found in applied migrations", migrationName)
		}
	} else {
		// Rollback last N migrations
		if steps > len(applied) {
			steps = len(applied)
		}
		// Get last N migrations (most recent first)
		migrationsToRollback = applied[len(applied)-steps:]
	}

	if len(migrationsToRollback) == 0 {
		fmt.Println("✅ No migrations to rollback")
		return nil
	}

	// Show what will be rolled back
	fmt.Printf("\n📋 Migrations to rollback (%d):\n", len(migrationsToRollback))
	for i := len(migrationsToRollback) - 1; i >= 0; i-- {
		m := migrationsToRollback[i]
		fmt.Printf("  • %s (applied at %s)\n", m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
	}

	// Safety check: warn about production
	fmt.Print("\n⚠️  WARNING: Rolling back migrations can cause data loss!\n")
	fmt.Print("❓ Continue? (y/N): ")
	var confirmation string
	fmt.Scanln(&confirmation)
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))
	if confirmation != "y" && confirmation != "yes" {
		fmt.Println("✋ Aborted. No migrations rolled back.")
		return nil
	}

	// Rollback migrations in reverse order (most recent first)
	fmt.Println("\n⏪ Rolling back migrations...")
	successCount := 0
	failCount := 0

	for i := len(migrationsToRollback) - 1; i >= 0; i-- {
		m := migrationsToRollback[i]
		fmt.Printf("\n📝 Rolling back: %s\n", m.Name)

		// Read rollback SQL file
		rollbackPath := filepath.Join("migrations", m.Name, "rollback.sql")
		rollbackSQL, err := os.ReadFile(rollbackPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "  ⚠️  Rollback SQL file not found: %s\n", rollbackPath)
				fmt.Fprintf(os.Stderr, "  💡 Skipping rollback for this migration\n")
				failCount++
				continue
			}
			fmt.Fprintf(os.Stderr, "  ❌ Failed to read rollback file: %v\n", err)
			failCount++
			continue
		}

		// Execute rollback
		err = migrationExecutor.RollbackMigration(ctx, string(rollbackSQL), m.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ❌ Failed to rollback migration: %v\n", err)
			failCount++
			continue
		}

		fmt.Printf("  ✅ Rolled back successfully\n")
		successCount++
	}

	fmt.Printf("\n📊 Rollback Summary:\n")
	fmt.Printf("  ✅ Rolled back: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("  ❌ Failed: %d\n", failCount)
	}

	if failCount == 0 {
		fmt.Println("\n🎉 All migrations rolled back successfully!")
	} else {
		fmt.Printf("\n⚠️  Some rollbacks failed. Please review errors above.\n")
		return fmt.Errorf("%d rollback(s) failed", failCount)
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
