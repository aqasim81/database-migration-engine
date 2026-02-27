package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

func TestAddConstraintRule_ID(t *testing.T) {
	t.Parallel()

	rule := rules.NewAddConstraintRule()
	assert.Equal(t, "add-constraint-without-not-valid", rule.ID())
}

func TestAddConstraintRule_Check(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sql          string
		wantCount    int
		wantSeverity analyzer.Severity
	}{
		{
			name:         "CHECK without NOT VALID is HIGH",
			sql:          "ALTER TABLE users ADD CONSTRAINT chk_age CHECK (age > 0);",
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:      "CHECK with NOT VALID is safe",
			sql:       "ALTER TABLE users ADD CONSTRAINT chk_age CHECK (age > 0) NOT VALID;",
			wantCount: 0,
		},
		{
			name:         "FOREIGN KEY without NOT VALID is HIGH",
			sql:          "ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id);",
			wantCount:    1,
			wantSeverity: analyzer.High,
		},
		{
			name:      "FOREIGN KEY with NOT VALID is safe",
			sql:       "ALTER TABLE orders ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;",
			wantCount: 0,
		},
		{
			name:      "UNIQUE constraint is not flagged",
			sql:       "ALTER TABLE users ADD CONSTRAINT uq_email UNIQUE (email);",
			wantCount: 0,
		},
		{
			name:      "PRIMARY KEY is not flagged",
			sql:       "ALTER TABLE t ADD CONSTRAINT pk PRIMARY KEY (id);",
			wantCount: 0,
		},
		{
			name:      "non-ALTER statement ignored",
			sql:       "CREATE TABLE users (id INT);",
			wantCount: 0,
		},
	}

	rule := rules.NewAddConstraintRule()

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
			}
		})
	}
}
