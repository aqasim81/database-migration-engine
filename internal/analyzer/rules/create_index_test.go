package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestCreateIndexRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewCreateIndexRule()
	assert.Equal(t, "create-index-not-concurrent", rule.ID())
}

func TestCreateIndexRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		wantCount    int
		wantSeverity analyzer.Severity
		wantTable    string
	}{
		{
			name:         "non-concurrent index is HIGH",
			sql:          "CREATE INDEX idx_users_email ON users (email);",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:      "concurrent index is safe",
			sql:       "CREATE INDEX CONCURRENTLY idx_users_email ON users (email);",
			wantCount: 0,
		},
		{
			name:         "unique non-concurrent index is HIGH",
			sql:          "CREATE UNIQUE INDEX idx_users_email ON users (email);",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:         "partial index non-concurrent is HIGH",
			sql:          "CREATE INDEX idx_active ON users (email) WHERE active = true;",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "users",
		},
		{
			name:         "schema-qualified table",
			sql:          "CREATE INDEX idx ON myschema.users (email);",
			wantCount:    1,
			wantSeverity: analyzer.High,
			wantTable:    "myschema.users",
		},
		{
			name:      "non-index statement ignored",
			sql:       "CREATE TABLE users (id INT);",
			wantCount: 0,
		},
	}

	rule := rules.NewCreateIndexRule()

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
				assert.Equal(t, tt.wantTable, findings[0].Table)
				assert.Equal(t, "SHARE", findings[0].LockType)
				assert.Equal(t, rule.ID(), findings[0].Rule)
			}
		})
	}
}
