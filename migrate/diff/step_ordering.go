// Package diff provides dependency-aware migration step ordering
package diff

import (
	"sort"
)

// MigrationStep represents a single migration step with dependencies
type MigrationStep struct {
	Change       Change
	Dependencies []string // Table/column/index names this step depends on
	Provides     []string // Table/column/index names this step creates
}

// OrderChanges orders changes based on dependencies
func OrderChanges(changes []Change) []Change {
	if len(changes) == 0 {
		return changes
	}

	// Convert changes to steps with dependency information
	steps := make([]MigrationStep, 0, len(changes))
	stepMap := make(map[int]*MigrationStep)

	for i, change := range changes {
		step := &MigrationStep{
			Change:       change,
			Dependencies: extractDependencies(change),
			Provides:     extractProvides(change),
		}
		steps = append(steps, *step)
		stepMap[i] = step
	}

	// Topological sort
	ordered := topologicalSort(steps)

	// Convert back to changes
	result := make([]Change, 0, len(ordered))
	for _, step := range ordered {
		result = append(result, step.Change)
	}

	return result
}

// extractDependencies extracts what a change depends on
func extractDependencies(change Change) []string {
	var deps []string

	switch change.Type {
	case ChangeTypeCreateForeignKey:
		// FK creation depends on referenced table existing
		// Note: We'd need FK metadata to get referenced table
		// For now, we'll handle this in the ordering logic
	case ChangeTypeAlterTable, ChangeTypeAlterColumn, ChangeTypeAddColumn:
		// Depends on table existing
		if change.Table != "" {
			deps = append(deps, "table:"+change.Table)
		}
	case ChangeTypeCreateIndex, ChangeTypeDropIndex, ChangeTypeRenameIndex:
		// Depends on table existing
		if change.Table != "" {
			deps = append(deps, "table:"+change.Table)
		}
	case ChangeTypeDropForeignKey:
		// Depends on FK existing (no dependency needed, can drop anytime)
	case ChangeTypeDropTable:
		// Drop table should happen after dropping FKs that reference it
		// This is handled in special ordering
	case ChangeTypeDropColumn:
		// Depends on table existing
		if change.Table != "" {
			deps = append(deps, "table:"+change.Table)
		}
	}

	return deps
}

// extractProvides extracts what a change provides/creates
func extractProvides(change Change) []string {
	var provides []string

	switch change.Type {
	case ChangeTypeCreateTable:
		if change.Table != "" {
			provides = append(provides, "table:"+change.Table)
		}
	case ChangeTypeAddColumn:
		if change.Table != "" && change.Column != "" {
			provides = append(provides, "column:"+change.Table+"."+change.Column)
		}
	case ChangeTypeCreateIndex:
		if change.Table != "" && change.Index != "" {
			provides = append(provides, "index:"+change.Table+"."+change.Index)
		}
	case ChangeTypeCreateForeignKey:
		if change.Table != "" {
			provides = append(provides, "fk:"+change.Table+"."+change.Index)
		}
	}

	return provides
}

// topologicalSort performs topological sorting of migration steps
func topologicalSort(steps []MigrationStep) []MigrationStep {
	// Build dependency graph
	inDegree := make(map[int]int)
	graph := make(map[int][]int) // step index -> dependent step indices

	for i := range steps {
		inDegree[i] = 0
		graph[i] = []int{}
	}

	// Build graph: for each step, find steps that depend on what it provides
	for i, step := range steps {
		for _, provided := range step.Provides {
			for j, otherStep := range steps {
				if i == j {
					continue
				}
				for _, dep := range otherStep.Dependencies {
					if dep == provided {
						graph[i] = append(graph[i], j)
						inDegree[j]++
					}
				}
			}
		}
	}

	// Apply special ordering rules
	applySpecialOrderingRules(steps, graph, inDegree)

	// Kahn's algorithm for topological sort
	var queue []int
	for i := range steps {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	var result []MigrationStep
	for len(queue) > 0 {
		// Sort queue for deterministic output
		sort.Ints(queue)
		current := queue[0]
		queue = queue[1:]

		result = append(result, steps[current])

		// Reduce in-degree of dependent nodes
		for _, dependent := range graph[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Add any remaining steps (shouldn't happen in valid dependency graph)
	// Use a map to track which steps we've added
	added := make(map[int]bool)
	for _, step := range result {
		// Find index of this step
		for i, s := range steps {
			if step.Change.Type == s.Change.Type &&
				step.Change.Table == s.Change.Table &&
				step.Change.Column == s.Change.Column &&
				step.Change.Index == s.Change.Index {
				added[i] = true
				break
			}
		}
	}
	for i := range steps {
		if !added[i] {
			result = append(result, steps[i])
		}
	}

	return result
}

// applySpecialOrderingRules applies provider-specific ordering rules
func applySpecialOrderingRules(steps []MigrationStep, graph map[int][]int, inDegree map[int]int) {
	// Rule 1: Create tables before creating indexes/foreign keys on them
	// Rule 2: Drop foreign keys before dropping tables
	// Rule 3: Drop indexes before dropping tables (for some providers)
	// Rule 4: Drop unique indexes before creating primary keys (special case)

	for i, step := range steps {
		switch step.Change.Type {
		case ChangeTypeCreateTable:
			// Ensure indexes/FKs created after table creation
			tableName := step.Change.Table
			for j, otherStep := range steps {
				if i == j {
					continue
				}
				if (otherStep.Change.Type == ChangeTypeCreateIndex ||
					otherStep.Change.Type == ChangeTypeCreateForeignKey ||
					otherStep.Change.Type == ChangeTypeRenameIndex) &&
					otherStep.Change.Table == tableName {
					// Add edge: table creation -> index/FK creation
					if !containsInt(graph[i], j) {
						graph[i] = append(graph[i], j)
						inDegree[j]++
					}
				}
			}

		case ChangeTypeDropTable:
			// Ensure FKs/indexes dropped before table drop
			tableName := step.Change.Table
			for j, otherStep := range steps {
				if i == j {
					continue
				}
				if (otherStep.Change.Type == ChangeTypeDropForeignKey ||
					otherStep.Change.Type == ChangeTypeDropIndex ||
					otherStep.Change.Type == ChangeTypeRenameIndex) &&
					otherStep.Change.Table == tableName {
					// Add edge: FK/index drop -> table drop
					if !containsInt(graph[j], i) {
						graph[j] = append(graph[j], i)
						inDegree[i]++
					}
				}
			}

		case ChangeTypeDropIndex:
			// Special case: Drop unique index before creating primary key
			// This is handled by checking if the index is unique and if there's a PK creation
			// For now, we'll handle this in a simpler way by ensuring drop happens before alter
			tableName := step.Change.Table
			for j, otherStep := range steps {
				if i == j {
					continue
				}
				if otherStep.Change.Type == ChangeTypeAlterTable &&
					otherStep.Change.Table == tableName {
					// Drop index before altering table (which might create PK)
					if !containsInt(graph[i], j) {
						graph[i] = append(graph[i], j)
						inDegree[j]++
					}
				}
			}
		}
	}
}

// containsInt checks if slice contains value
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
