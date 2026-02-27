package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestLockTableRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewLockTableRule()
	assert.Equal(t, "lock-table", rule.ID())
}

func TestLockTableRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		wantCount    int
		wantSeverity analyzer.Severity
		wantTable    string
	}{
		{
			name:         "ACCESS EXCLUSIVE lock is HIGH",
			sql:          "LOCK TABLE users IN ACCESS EXCLUSIVE MODE;",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:         "SHARE MODE lock is HIGH",
			sql:          "LOCK TABLE users IN SHARE MODE;",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:         "ROW SHARE lock is HIGH",
			sql:          "LOCK TABLE users IN ROW SHARE MODE;",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:      "non-LOCK statement ignored",
			sql:       "CREATE TABLE users (id INT);",
			wantCount: 0,
		},
	}

	rule := rules.NewLockTableRule()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parser.Parse(tt.sql)
			require.NoError(t, err)
			require.Len(t, result.Stmts, 1)

			ctx := &analyzer.RuleContext{
				TargetPGVersion: 14, //nolint:mnd // test default
				StmtIndex:       0,
			}

			findings := rule.Check(result.Stmts[0], ctx)
			assert.Len(t, findings, tt.wantCount)

			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantSeverity, findings[0].Severity)
				assert.Equal(t, rule.ID(), findings[0].Rule)
				assert.Equal(t, tt.wantTable, findings[0].Table)
			}
		})
	}
}
