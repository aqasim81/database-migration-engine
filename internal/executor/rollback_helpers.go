package executor

import (
	"fmt"

	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

// reverseApplied returns a copy of applied migrations in reverse order
// (last applied first) for rollback processing.
func reverseApplied(applied []tracker.AppliedMigration) []tracker.AppliedMigration {
	reversed := make([]tracker.AppliedMigration, len(applied))
	for i, a := range applied {
		reversed[len(applied)-1-i] = a
	}

	return reversed
}

// appliedAfterVersion returns all applied migrations with version > target,
// in reverse order (last applied first). Returns ErrTargetNotFound if the
// target version is not in the applied list.
func appliedAfterVersion(applied []tracker.AppliedMigration, target string) ([]tracker.AppliedMigration, error) {
	found := false

	var after []tracker.AppliedMigration

	for _, a := range applied {
		if a.Version == target {
			found = true

			continue
		}

		if a.Version > target {
			after = append(after, a)
		}
	}

	if !found {
		return nil, fmt.Errorf("version %s: %w", target, ErrTargetNotFound)
	}

	// Reverse so last applied is rolled back first.
	for i, j := 0, len(after)-1; i < j; i, j = i+1, j-1 {
		after[i], after[j] = after[j], after[i]
	}

	return after, nil
}

// buildMigrationLookup creates a version -> Migration map for O(1) lookups.
func buildMigrationLookup(migrations []migration.Migration) map[string]*migration.Migration {
	lookup := make(map[string]*migration.Migration, len(migrations))
	for i := range migrations {
		lookup[migrations[i].Version] = &migrations[i]
	}

	return lookup
}
