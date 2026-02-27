package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestAddColumnRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewAddColumnRule()
	assert.Equal(t, "add-column-volatile-default", rule.ID())
}

func TestAddColumnRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		pgVersion    int
		wantCount    int
		wantSeverity analyzer.Severity
	}{
		{
			name:      "literal default on PG14 is safe",
			sql:       "ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';",
			pgVersion: 14,
			wantCount: 0,
		},
		{
			name:         "literal default on PG10 is HIGH",
			sql:          "ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';",
			pgVersion:    10,
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:         "volatile default now() on PG14 is HIGH",
			sql:          "ALTER TABLE users ADD COLUMN created_at TIMESTAMPTZ DEFAULT now();",
			pgVersion:    14,
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:         "volatile default now() on PG10 is HIGH",
			sql:          "ALTER TABLE users ADD COLUMN created_at TIMESTAMPTZ DEFAULT now();",
			pgVersion:    10,
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:      "no default is safe",
			sql:       "ALTER TABLE users ADD COLUMN bio TEXT;",
			pgVersion: 14,
			wantCount: 0,
		},
		{
			name:      "integer default on PG14 is safe",
			sql:       "ALTER TABLE users ADD COLUMN count INT DEFAULT 0;",
			pgVersion: 14,
			wantCount: 0,
		},
		{
			name:         "gen_random_uuid() on PG14 is HIGH",
			sql:          "ALTER TABLE t ADD COLUMN id UUID DEFAULT gen_random_uuid();",
			pgVersion:    14,
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:      "boolean default on PG11 is safe",
			sql:       "ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT true;",
			pgVersion: 11,
			wantCount: 0,
		},
		{
			name:         "boolean default on PG10 is HIGH",
			sql:          "ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT true;",
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
	}

	rule := rules.NewAddColumnRule()

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
