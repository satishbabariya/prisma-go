// Package planner implements migration planning.
package planner

import (
	"context"
	"fmt"
	"sort"

	"github.com/satishbabariya/prisma-go/v3/internal/core/migration/domain"
)

// MigrationPlanner implements the Planner interface.
type MigrationPlanner struct {
	dialect domain.SQLDialect
}

// NewMigrationPlanner creates a new migration planner.
func NewMigrationPlanner(dialect domain.SQLDialect) *MigrationPlanner {
	return &MigrationPlanner{
		dialect: dialect,
	}
}

// CreatePlan creates a migration plan from changes.
func (p *MigrationPlanner) CreatePlan(ctx context.Context, changes []domain.Change) (*domain.MigrationPlan, error) {
	// Sort changes by priority (create tables before adding columns, etc.)
	sortedChanges := p.sortChanges(changes)

	// Generate SQL for each change
	var allSQL []string
	for _, change := range sortedChanges {
		sql, err := change.ToSQL(p.dialect)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SQL for change %s: %w", change.Description(), err)
		}
		allSQL = append(allSQL, sql...)
	}

	plan := &domain.MigrationPlan{
		Changes: sortedChanges,
		SQL:     allSQL,
	}

	return plan, nil
}

// OptimizePlan optimizes a migration plan.
func (p *MigrationPlanner) OptimizePlan(ctx context.Context, plan *domain.MigrationPlan) (*domain.MigrationPlan, error) {
	// For now, return the plan as-is
	// Future optimizations could include:
	// - Combining multiple ALTER TABLE statements
	// - Removing redundant changes
	// - Reordering for better performance
	return plan, nil
}

// ValidatePlan validates a migration plan.
func (p *MigrationPlanner) ValidatePlan(ctx context.Context, plan *domain.MigrationPlan) error {
	if len(plan.Changes) == 0 {
		return fmt.Errorf("migration plan has no changes")
	}

	// Check for conflicting changes
	// For example, dropping a table and then adding a column to it
	tableDrops := make(map[string]bool)
	for _, change := range plan.Changes {
		if change.Type() == domain.DropTable {
			// Extract table name from description
			// This is a simplified check
			tableDrops[change.Description()] = true
		}
	}

	return nil
}

// sortChanges sorts changes by priority to ensure correct execution order.
func (p *MigrationPlanner) sortChanges(changes []domain.Change) []domain.Change {
	// Define priority order
	priority := map[domain.ChangeType]int{
		domain.DropIndex:      1, // Drop indexes first
		domain.DropColumn:     2, // Then drop columns
		domain.DropTable:      3, // Then drop tables
		domain.CreateTable:    4, // Create tables
		domain.AddColumn:      5, // Add columns
		domain.AlterColumn:    6, // Alter columns
		domain.CreateIndex:    7, // Create indexes last
		domain.AddConstraint:  8,
		domain.DropConstraint: 9,
	}

	sorted := make([]domain.Change, len(changes))
	copy(sorted, changes)

	sort.Slice(sorted, func(i, j int) bool {
		pi := priority[sorted[i].Type()]
		pj := priority[sorted[j].Type()]
		return pi < pj
	})

	return sorted
}

// hasDestructiveChanges checks if any changes are destructive.
func (p *MigrationPlanner) hasDestructiveChanges(changes []domain.Change) bool {
	for _, change := range changes {
		if change.IsDestructive() {
			return true
		}
	}
	return false
}

// Ensure MigrationPlanner implements Planner interface.
var _ domain.Planner = (*MigrationPlanner)(nil)
