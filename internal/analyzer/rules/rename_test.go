package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestRenameRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewRenameRule()
	assert.Equal(t, "rename", rule.ID())
}

func TestRenameRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		wantCount    int
		wantSeverity analyzer.Severity
		wantTable    string
	}{
		{
			name:         "RENAME COLUMN is MEDIUM",
			sql:          "ALTER TABLE users RENAME COLUMN email TO email_address;",
			wantCount:    1,
			wantSeverity: analyzer.Medium,
			wantTable:    "users",
		},
		{
			name:         "RENAME TABLE is MEDIUM",
			sql:          "ALTER TABLE users RENAME TO customers;",
			wantCount:    1,
			wantSeverity: analyzer.Medium,
			wantTable:    "users",
		},
		{
			name:      "RENAME INDEX is not flagged",
			sql:       "ALTER INDEX idx_users RENAME TO idx_customers;",
			wantCount: 0,
		},
		{
			name:      "non-RENAME statement ignored",
			sql:       "CREATE TABLE users (id INT);",
			wantCount: 0,
		},
	}

	rule := rules.NewRenameRule()

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
