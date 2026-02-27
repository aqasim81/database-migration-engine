package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestDropTableRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewDropTableRule()
	assert.Equal(t, "drop-table", rule.ID())
}

func TestDropTableRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		wantCount    int
		wantSeverity analyzer.Severity
		wantTable    string
	}{
		{
			name:         "DROP TABLE is CRITICAL",
			sql:          "DROP TABLE users;",
			wantCount:    1,
			wantSeverity: analyzer.Critical,
			wantTable:    "users",
		},
		{
			name:         "DROP TABLE IF EXISTS is CRITICAL",
			sql:          "DROP TABLE IF EXISTS users;",
			wantCount:    1,
			wantSeverity: analyzer.Critical,
			wantTable:    "users",
		},
		{
			name:         "TRUNCATE is CRITICAL",
			sql:          "TRUNCATE users;",
			wantCount:    1,
			wantSeverity: analyzer.Critical,
			wantTable:    "users",
		},
		{
			name:      "DROP INDEX is not flagged",
			sql:       "DROP INDEX idx_users_email;",
			wantCount: 0,
		},
		{
			name:      "DROP VIEW is not flagged",
			sql:       "DROP VIEW user_view;",
			wantCount: 0,
		},
		{
			name:      "CREATE TABLE is not flagged",
			sql:       "CREATE TABLE users (id INT);",
			wantCount: 0,
		},
	}

	rule := rules.NewDropTableRule()

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

				if tt.wantTable != "" {
					assert.Contains(t, findings[0].Table, tt.wantTable)
				}
			}
		})
	}
}
