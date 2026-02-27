package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestVacuumFullRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewVacuumFullRule()
	assert.Equal(t, "vacuum-full", rule.ID())
}

func TestVacuumFullRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		wantCount    int
		wantSeverity analyzer.Severity
		wantTable    string
	}{
		{
			name:         "VACUUM FULL is HIGH",
			sql:          "VACUUM FULL users;",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:         "VACUUM (FULL) is HIGH",
			sql:          "VACUUM (FULL) users;",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:      "plain VACUUM is safe",
			sql:       "VACUUM users;",
			wantCount: 0,
		},
		{
			name:      "VACUUM ANALYZE is safe",
			sql:       "VACUUM ANALYZE users;",
			wantCount: 0,
		},
		{
			name:      "non-VACUUM statement ignored",
			sql:       "CREATE TABLE users (id INT);",
			wantCount: 0,
		},
	}

	rule := rules.NewVacuumFullRule()

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
					assert.Equal(t, tt.wantTable, findings[0].Table)
				}
			}
		})
	}
}
