package planner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/planner"
)

const (
	sizeSmall   int64 = 50 * 1024 * 1024       // 50 MB
	sizeMedium  int64 = 500 * 1024 * 1024      // 500 MB
	sizeLarge   int64 = 2 * 1024 * 1024 * 1024 // 2 GB
	sizeUnknown int64 = -1
)

func TestEstimateImpact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ruleID         string
		tableSize      int64
		wantDuration   string
		wantLockType   string
		wantRewrite    bool
		wantScan       bool
		wantConcurrent bool
	}{
		// create-index-not-concurrent
		{
			name:         "create_index_small_table",
			ruleID:       "create-index-not-concurrent",
			tableSize:    sizeSmall,
			wantDuration: planner.Duration1to10s,
			wantLockType: "SHARE",
		},
		{
			name:         "create_index_medium_table",
			ruleID:       "create-index-not-concurrent",
			tableSize:    sizeMedium,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "SHARE",
		},
		{
			name:         "create_index_large_table",
			ruleID:       "create-index-not-concurrent",
			tableSize:    sizeLarge,
			wantDuration: planner.DurationOver5min,
			wantLockType: "SHARE",
		},
		{
			name:         "create_index_unknown_size",
			ruleID:       "create-index-not-concurrent",
			tableSize:    sizeUnknown,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "SHARE",
		},

		// add-column-volatile-default
		{
			name:         "add_column_default_small",
			ruleID:       "add-column-volatile-default",
			tableSize:    sizeSmall,
			wantDuration: planner.Duration1to10s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},
		{
			name:         "add_column_default_large",
			ruleID:       "add-column-volatile-default",
			tableSize:    sizeLarge,
			wantDuration: planner.DurationOver5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},
		{
			name:         "add_column_default_medium",
			ruleID:       "add-column-volatile-default",
			tableSize:    sizeMedium,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},
		{
			name:         "add_column_default_unknown",
			ruleID:       "add-column-volatile-default",
			tableSize:    sizeUnknown,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},

		// add-constraint-without-not-valid
		{
			name:         "add_constraint_small",
			ruleID:       "add-constraint-without-not-valid",
			tableSize:    sizeSmall,
			wantDuration: planner.DurationUnder1s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},
		{
			name:         "add_constraint_medium",
			ruleID:       "add-constraint-without-not-valid",
			tableSize:    sizeMedium,
			wantDuration: planner.Duration1to10s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},
		{
			name:         "add_constraint_large",
			ruleID:       "add-constraint-without-not-valid",
			tableSize:    sizeLarge,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},

		// alter-column-type
		{
			name:         "alter_column_type_small",
			ruleID:       "alter-column-type",
			tableSize:    sizeSmall,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},
		{
			name:         "alter_column_type_large",
			ruleID:       "alter-column-type",
			tableSize:    sizeLarge,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},

		// set-not-null
		{
			name:         "set_not_null_small",
			ruleID:       "set-not-null",
			tableSize:    sizeSmall,
			wantDuration: planner.DurationUnder1s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},
		{
			name:         "set_not_null_medium",
			ruleID:       "set-not-null",
			tableSize:    sizeMedium,
			wantDuration: planner.Duration1to10s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},
		{
			name:         "set_not_null_large",
			ruleID:       "set-not-null",
			tableSize:    sizeLarge,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},
		{
			name:         "set_not_null_unknown",
			ruleID:       "set-not-null",
			tableSize:    sizeUnknown,
			wantDuration: planner.Duration1to10s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantScan:     true,
		},

		// drop-table
		{
			name:         "drop_table_any_size",
			ruleID:       "drop-table",
			tableSize:    sizeLarge,
			wantDuration: planner.DurationUnder1s,
			wantLockType: "ACCESS EXCLUSIVE",
		},

		// vacuum-full
		{
			name:         "vacuum_full_small",
			ruleID:       "vacuum-full",
			tableSize:    sizeSmall,
			wantDuration: planner.Duration1to10s,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},
		{
			name:         "vacuum_full_medium",
			ruleID:       "vacuum-full",
			tableSize:    sizeMedium,
			wantDuration: planner.Duration10sTo5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},
		{
			name:         "vacuum_full_large",
			ruleID:       "vacuum-full",
			tableSize:    sizeLarge,
			wantDuration: planner.DurationOver5min,
			wantLockType: "ACCESS EXCLUSIVE",
			wantRewrite:  true,
		},

		// lock-table
		{
			name:         "lock_table_any",
			ruleID:       "lock-table",
			tableSize:    sizeUnknown,
			wantDuration: planner.Duration1to10s,
			wantLockType: "EXPLICIT",
		},

		// rename
		{
			name:         "rename_any",
			ruleID:       "rename",
			tableSize:    sizeUnknown,
			wantDuration: planner.DurationUnder1s,
			wantLockType: "ACCESS EXCLUSIVE",
		},

		// unknown rule
		{
			name:         "unknown_rule_defaults",
			ruleID:       "some-unknown-rule",
			tableSize:    sizeUnknown,
			wantDuration: planner.DurationUnder1s,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			finding := &analyzer.Finding{
				Rule:  tt.ruleID,
				Table: "test_table",
			}

			impact := planner.EstimateImpact(finding, tt.tableSize)

			assert.Equal(t, tt.wantDuration, impact.EstimatedLockDuration)
			assert.Equal(t, tt.wantRewrite, impact.RequiresFullRewrite)
			assert.Equal(t, tt.wantScan, impact.RequiresFullScan)
			assert.Equal(t, tt.wantConcurrent, impact.CanRunConcurrently)
			assert.Equal(t, "test_table", impact.AffectedTable)
			assert.Equal(t, tt.tableSize, impact.TableSizeBytes)

			if tt.wantLockType != "" {
				assert.Equal(t, tt.wantLockType, impact.LockType)
			}
		})
	}
}

func TestEstimateImpact_findingLockTypeOverridesDefault(t *testing.T) {
	t.Parallel()

	finding := &analyzer.Finding{
		Rule:     "create-index-not-concurrent",
		Table:    "users",
		LockType: "SHARE UPDATE EXCLUSIVE",
	}

	impact := planner.EstimateImpact(finding, sizeSmall)

	assert.Equal(t, "SHARE UPDATE EXCLUSIVE", impact.LockType)
}

func TestEstimateImpact_emptyFindingLockType_usesDefault(t *testing.T) {
	t.Parallel()

	finding := &analyzer.Finding{
		Rule:  "create-index-not-concurrent",
		Table: "users",
	}

	impact := planner.EstimateImpact(finding, sizeSmall)

	assert.Equal(t, "SHARE", impact.LockType)
}
