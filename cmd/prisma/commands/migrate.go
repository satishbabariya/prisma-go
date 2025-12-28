// Package commands implements CLI commands.
package commands

import (
	"context"
	"fmt"

	"github.com/satishbabariya/prisma-go/v3/internal/service"
	"github.com/satishbabariya/prisma-go/v3/internal/utils/container"
	"github.com/spf13/cobra"
)

// NewMigrateCommand creates the migrate command with subcommands.
func NewMigrateCommand(c *container.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
		Long:  "Create, apply, and manage Prisma database migrations",
	}

	cmd.AddCommand(newMigrateDevCommand(c))
	cmd.AddCommand(newMigrateDeployCommand(c))
	cmd.AddCommand(newMigrateStatusCommand(c))
	cmd.AddCommand(newMigrateResetCommand(c))

	return cmd
}

func newMigrateDevCommand(c *container.Container) *cobra.Command {
	var schemaPath string
	var name string

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Create and apply migrations in development",
		Long:  "Create a migration from changes in Prisma schema, apply it to the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateDev(c, schemaPath, name)
		},
	}

	cmd.Flags().StringVar(&schemaPath, "schema", "prisma/schema.prisma", "Path to schema file")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Name for the migration")
	cmd.MarkFlagRequired("name")

	return cmd
}

func newMigrateDeployCommand(c *container.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Apply pending migrations",
		Long:  "Apply all pending migrations to the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateDeploy(c)
		},
	}

	return cmd
}

func newMigrateStatusCommand(c *container.Container) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check migration status",
		Long:  "Display the status of migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateStatus(c)
		},
	}

	return cmd
}

func runMigrateDev(c *container.Container, schemaPath, name string) error {
	fmt.Println("Creating migration...")

	migrationService := c.MigrationService()
	if migrationService == nil {
		return fmt.Errorf("migration service not initialized")
	}

	// Create migration
	migration, err := migrationService.CreateMigration(context.Background(), service.CreateMigrationInput{
		Name:       name,
		SchemaPath: schemaPath,
	})
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	fmt.Printf("✓ Created migration: %s\n", migration.ID)

	// Apply migration
	fmt.Println("Applying migration...")
	if err := migrationService.ApplyMigration(context.Background(), migration.ID); err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	fmt.Println("✓ Migration applied successfully")

	return nil
}

func runMigrateDeploy(c *container.Container) error {
	fmt.Println("Applying pending migrations...")

	migrationService := c.MigrationService()
	if migrationService == nil {
		return fmt.Errorf("migration service not initialized")
	}

	// Get migration status
	status, err := migrationService.GetMigrationStatus(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	if len(status.Pending) == 0 {
		fmt.Println("✓ No pending migrations")
		return nil
	}

	// Apply each pending migration
	for _, migration := range status.Pending {
		fmt.Printf("Applying migration: %s\n", migration.ID)
		if err := migrationService.ApplyMigration(context.Background(), migration.ID); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.ID, err)
		}
	}

	fmt.Printf("✓ Applied %d migration(s)\n", len(status.Pending))

	return nil
}

func runMigrateStatus(c *container.Container) error {
	migrationService := c.MigrationService()
	if migrationService == nil {
		return fmt.Errorf("migration service not initialized")
	}

	status, err := migrationService.GetMigrationStatus(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("Migration Status:\n")
	fmt.Printf("  Total migrations: %d\n", status.Total)
	fmt.Printf("  Applied: %d\n", len(status.Applied))
	fmt.Printf("  Pending: %d\n", len(status.Pending))

	if len(status.Pending) > 0 {
		fmt.Println("\nPending migrations:")
		for _, migration := range status.Pending {
			fmt.Printf("  - %s (%s)\n", migration.ID, migration.Name)
		}
	}

	if len(status.Applied) > 0 {
		fmt.Println("\nApplied migrations:")
		for _, migration := range status.Applied {
			appliedAt := "unknown"
			if migration.AppliedAt != nil {
				appliedAt = migration.AppliedAt.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("  - %s (%s) - %s\n", migration.ID, migration.Name, appliedAt)
		}
	}

	return nil
}

func newMigrateResetCommand(c *container.Container) *cobra.Command {
	var force bool
	var skipSeed bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database",
		Long: `Reset the database by dropping all data, rolling back all migrations,
and then re-applying all migrations. This is a destructive operation.

Use with caution - all data will be lost!`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateReset(c, force, skipSeed)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&skipSeed, "skip-seed", false, "Skip running seed after reset")

	return cmd
}

func runMigrateReset(c *container.Container, force, skipSeed bool) error {
	if !force {
		fmt.Println("⚠️  Warning: This will delete ALL data in the database!")
		fmt.Println("Run with --force to confirm.")
		return fmt.Errorf("reset cancelled - use --force to confirm")
	}

	fmt.Println("🔄 Resetting database...")

	migrationService := c.MigrationService()
	if migrationService == nil {
		return fmt.Errorf("migration service not initialized")
	}

	// Get current migration status
	status, err := migrationService.GetMigrationStatus(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	// Rollback all applied migrations in reverse order
	fmt.Printf("Rolling back %d applied migrations...\n", len(status.Applied))
	for i := len(status.Applied) - 1; i >= 0; i-- {
		migration := status.Applied[i]
		fmt.Printf("  Rolling back: %s\n", migration.ID)
		if err := migrationService.RollbackMigration(context.Background(), migration.ID); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.ID, err)
		}
	}

	fmt.Println("✅ All migrations rolled back")

	// Re-apply all migrations
	fmt.Println("\n🚀 Re-applying all migrations...")
	allMigrations, err := migrationService.GetMigrationStatus(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get migrations: %w", err)
	}

	for _, migration := range allMigrations.Pending {
		fmt.Printf("  Applying: %s\n", migration.ID)
		if err := migrationService.ApplyMigration(context.Background(), migration.ID); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.ID, err)
		}
	}

	fmt.Printf("✅ Applied %d migrations\n", len(allMigrations.Pending))

	if !skipSeed {
		fmt.Println("\n💡 Run your seed script to populate initial data")
	}

	fmt.Println("\n✅ Database reset complete!")
	return nil
}
