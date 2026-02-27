package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestSetNotNullRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewSetNotNullRule()
	assert.Equal(t, "set-not-null", rule.ID())
}

func TestSetNotNullRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		pgVersion    int
		wantCount    int
		wantSeverity analyzer.Severity
	}{
		{
			name:         "SET NOT NULL on PG14 is MEDIUM",
			sql:          "ALTER TABLE users ALTER COLUMN status SET NOT NULL;",
			pgVersion:    14,
			wantCount:    1,
			wantSeverity: analyzer.Medium,
		},
		{
			name:         "SET NOT NULL on PG12 is MEDIUM",
			sql:          "ALTER TABLE users ALTER COLUMN status SET NOT NULL;",
			pgVersion:    12,
			wantCount:    1,
			wantSeverity: analyzer.Medium,
		},
		{
			name:         "SET NOT NULL on PG11 is HIGH",
			sql:          "ALTER TABLE users ALTER COLUMN status SET NOT NULL;",
			pgVersion:    11,
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:         "SET NOT NULL on PG10 is HIGH",
			sql:          "ALTER TABLE users ALTER COLUMN status SET NOT NULL;",
			pgVersion:    10,
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:      "non-ALTER statement ignored",
			sql:       "CREATE TABLE users (id INT);",
			pgVersion: 14,
			wantCount: 0,
		},
		{
			name:      "ADD COLUMN is not flagged",
			sql:       "ALTER TABLE users ADD COLUMN bio TEXT;",
			pgVersion: 14,
			wantCount: 0,
		},
	}

	rule := rules.NewSetNotNullRule()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parser.Parse(tt.sql)
			require.NoError(t, err)
			require.Len(t, result.Stmts, 1)

			ctx := &analyzer.RuleContext{
				TargetPGVersion: tt.pgVersion,
				StmtIndex:       0,
			}

			findings := rule.Check(result.Stmts[0], ctx)
			assert.Len(t, findings, tt.wantCount)

			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantSeverity, findings[0].Severity)
				assert.Equal(t, rule.ID(), findings[0].Rule)
			}
		})
	}
}
