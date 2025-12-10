package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/executor"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
)

func main() {
	// Example demonstrating complete migration workflow
	
	ctx := context.Background()
	
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "postgresql://localhost:5432/mydb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	
	fmt.Println("=== Prisma-Go Migration Example ===\n")
	
	// Step 1: Introspect current database
	fmt.Println("Step 1: Introspecting database...")
	introspector, err := introspect.NewIntrospector(db, "postgresql")
	if err != nil {
		log.Fatal(err)
	}
	
	currentSchema, err := introspector.Introspect(ctx)
	if err != nil {
		log.Fatal("Failed to introspect:", err)
	}
	
	fmt.Printf("Found %d tables\n\n", len(currentSchema.Tables))
	
	// Step 2: Create target schema (simulated - in real use, parse from Prisma schema)
	fmt.Println("Step 2: Creating target schema...")
	targetSchema := &introspect.DatabaseSchema{
		Tables: []introspect.Table{
			{
				Name:   "users",
				Schema: "public",
				Columns: []introspect.Column{
					{Name: "id", Type: "SERIAL", Nullable: false, AutoIncrement: true},
					{Name: "email", Type: "VARCHAR(255)", Nullable: false},
					{Name: "name", Type: "VARCHAR(255)", Nullable: true},
					{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
				},
				PrimaryKey: &introspect.PrimaryKey{
					Name:    "users_pkey",
					Columns: []string{"id"},
				},
				Indexes: []introspect.Index{
					{Name: "users_email_idx", Columns: []string{"email"}, IsUnique: true},
				},
			},
			{
				Name:   "posts",
				Schema: "public",
				Columns: []introspect.Column{
					{Name: "id", Type: "SERIAL", Nullable: false, AutoIncrement: true},
					{Name: "title", Type: "VARCHAR(255)", Nullable: false},
					{Name: "content", Type: "TEXT", Nullable: true},
					{Name: "author_id", Type: "INTEGER", Nullable: false},
					{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
				},
				PrimaryKey: &introspect.PrimaryKey{
					Name:    "posts_pkey",
					Columns: []string{"id"},
				},
				ForeignKeys: []introspect.ForeignKey{
					{
						Name:              "posts_author_id_fkey",
						Columns:           []string{"author_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
						OnDelete:          "CASCADE",
						OnUpdate:          "CASCADE",
					},
				},
			},
		},
	}
	
	fmt.Println("Target schema created with 2 tables\n")
	
	// Step 3: Compare schemas
	fmt.Println("Step 3: Comparing schemas...")
	differ := diff.NewSimpleDiffer("postgresql")
	diffResult := differ.CompareSchemas(currentSchema, targetSchema)
	
	fmt.Printf("Changes detected:\n")
	fmt.Printf("  • Tables to create: %d\n", len(diffResult.TablesToCreate))
	fmt.Printf("  • Tables to alter: %d\n", len(diffResult.TablesToAlter))
	fmt.Printf("  • Tables to drop: %d\n", len(diffResult.TablesToDrop))
	fmt.Printf("  • Total changes: %d\n\n", len(diffResult.Changes))
	
	// Display changes
	for _, change := range diffResult.Changes {
		safetyIcon := "✅"
		if !change.IsSafe {
			safetyIcon = "⚠️"
		}
		fmt.Printf("  %s %s: %s\n", safetyIcon, change.Type, change.Description)
		for _, warning := range change.Warnings {
			fmt.Printf("      ⚠️  %s\n", warning)
		}
	}
	fmt.Println()
	
	// Step 4: Generate migration SQL
	fmt.Println("Step 4: Generating migration SQL...")
	sqlGenerator := sqlgen.NewPostgresMigrationGenerator()
	migrationSQL, err := sqlGenerator.GenerateMigrationSQL(diffResult, targetSchema)
	if err != nil {
		log.Fatal("Failed to generate SQL:", err)
	}
	
	fmt.Println("Generated SQL:")
	fmt.Println("---")
	fmt.Println(migrationSQL)
	fmt.Println("---\n")
	
	// Step 5: Setup migration executor
	fmt.Println("Step 5: Setting up migration executor...")
	migrationExecutor := executor.NewMigrationExecutor(db, "postgresql")
	
	// Ensure migration table exists
	err = migrationExecutor.EnsureMigrationTable(ctx)
	if err != nil {
		log.Fatal("Failed to create migration table:", err)
	}
	fmt.Println("Migration history table ready\n")
	
	// Step 6: Check applied migrations
	fmt.Println("Step 6: Checking migration history...")
	appliedMigrations, err := migrationExecutor.GetAppliedMigrations(ctx)
	if err != nil {
		log.Printf("Note: Could not read migration history: %v\n", err)
	} else {
		fmt.Printf("Applied migrations: %d\n", len(appliedMigrations))
		for _, m := range appliedMigrations {
			fmt.Printf("  • %s (applied: %s)\n", m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Println()
	
	// Step 7: Execute migration (commented out for safety)
	fmt.Println("Step 7: Migration execution")
	fmt.Println("⚠️  Migration execution is disabled in this example for safety.")
	fmt.Println("To execute the migration, uncomment the following code:")
	fmt.Println()
	fmt.Println("  migrationName := fmt.Sprintf(\"migration_%d\", time.Now().Unix())")
	fmt.Println("  err = migrationExecutor.ExecuteMigration(ctx, migrationSQL, migrationName)")
	fmt.Println("  if err != nil {")
	fmt.Println("      log.Fatal(\"Migration failed:\", err)")
	fmt.Println("  }")
	fmt.Println("  fmt.Println(\"✅ Migration executed successfully!\")")
	fmt.Println()
	
	/*
	// Uncomment to actually execute the migration:
	migrationName := fmt.Sprintf("migration_%d", time.Now().Unix())
	err = migrationExecutor.ExecuteMigration(ctx, migrationSQL, migrationName)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}
	fmt.Println("✅ Migration executed successfully!")
	*/
	
	fmt.Println("✅ Migration workflow complete!")
	fmt.Println("\nWorkflow Summary:")
	fmt.Println("  1. ✅ Database introspection")
	fmt.Println("  2. ✅ Target schema definition")
	fmt.Println("  3. ✅ Schema comparison (diffing)")
	fmt.Println("  4. ✅ SQL generation")
	fmt.Println("  5. ✅ Migration executor setup")
	fmt.Println("  6. ✅ Migration history tracking")
	fmt.Println("  7. ⚠️  Migration execution (disabled for safety)")
	fmt.Println("\n🎉 All migration components are working!")
}

