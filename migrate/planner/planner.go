// Package planner generates migration plans from schema diffs.
package planner

import (
	"fmt"

	"github.com/satishbabariya/prisma-go/migrate/diff"
	"github.com/satishbabariya/prisma-go/migrate/introspect"
	"github.com/satishbabariya/prisma-go/migrate/sqlgen"
)

// MigrationPlan represents a planned migration with steps
type MigrationPlan struct {
	Name     string
	Steps    []MigrationStep
	IsSafe   bool
	Warnings []string
}

// MigrationStep represents a single migration step
type MigrationStep struct {
	Type        string
	Description string
	SQL         string
	Rollback    string
	IsSafe      bool
	Warnings    []string
}

// Planner generates migration plans
type Planner struct {
	provider  string
	generator sqlgen.MigrationGenerator
}

// NewPlanner creates a new migration planner
func NewPlanner(provider string) (*Planner, error) {
	generator, err := sqlgen.NewMigrationGenerator(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration generator: %w", err)
	}

	return &Planner{
		provider:  provider,
		generator: generator,
	}, nil
}

// Plan generates a migration plan from a diff result
func (p *Planner) Plan(diffResult *diff.DiffResult, migrationName string, targetSchema *introspect.DatabaseSchema) (*MigrationPlan, error) {
	plan := &MigrationPlan{
		Name:     migrationName,
		Steps:    []MigrationStep{},
		IsSafe:   true,
		Warnings: []string{},
	}

	// GenerateMigrationSQL generates SQL for the migration
	migrationSQL, err := p.generator.GenerateMigrationSQL(diffResult, targetSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration SQL: %w", err)
	}

	// Generate rollback SQL
	rollbackSQL, err := p.generator.GenerateRollbackSQL(diffResult, targetSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rollback SQL: %w", err)
	}

	// Convert changes to steps
	for _, change := range diffResult.Changes {
		step := MigrationStep{
			Type:        change.Type,
			Description: change.Description,
			IsSafe:      change.IsSafe,
			Warnings:    change.Warnings,
		}

		// Extract SQL for this change from the generated SQL
		// For now, we'll use the full SQL and rollback SQL
		// In a more sophisticated implementation, we'd split by change
		if len(plan.Steps) == 0 {
			step.SQL = migrationSQL
			step.Rollback = rollbackSQL
		}

		plan.Steps = append(plan.Steps, step)

		// Collect warnings
		if !change.IsSafe {
			plan.IsSafe = false
		}
		if len(change.Warnings) > 0 {
			plan.Warnings = append(plan.Warnings, change.Warnings...)
		}
	}

	// If no changes, add a no-op step
	if len(plan.Steps) == 0 {
		plan.Steps = append(plan.Steps, MigrationStep{
			Type:        "NoOp",
			Description: "No changes detected",
			SQL:         "-- No changes to apply",
			Rollback:    "-- No changes to rollback",
			IsSafe:      true,
		})
	}

	return plan, nil
}
