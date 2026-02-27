package analyzer_test

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
)

func TestTableName_withSchema(t *testing.T) {
	t.Parallel()

	rv := &pg_query.RangeVar{Schemaname: "public", Relname: "users"}
	assert.Equal(t, "public.users", analyzer.TableName(rv))
}

func TestTableName_withoutSchema(t *testing.T) {
	t.Parallel()

	rv := &pg_query.RangeVar{Relname: "orders"}
	assert.Equal(t, "orders", analyzer.TableName(rv))
}

func TestTableName_nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "<unknown>", analyzer.TableName(nil))
}

func TestTruncateSQL_shortString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "SELECT 1", analyzer.TruncateSQL("SELECT 1", 100))
}

func TestTruncateSQL_exactLength(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "SELECT 1", analyzer.TruncateSQL("SELECT 1", 8))
}

func TestTruncateSQL_truncated(t *testing.T) {
	t.Parallel()

	result := analyzer.TruncateSQL("SELECT * FROM very_long_table_name WHERE id = 1", 20)
	assert.Equal(t, "SELECT * FROM ver...", result)
	assert.Len(t, result, 20)
}

func TestTruncateSQL_maxLenTooSmall(t *testing.T) {
	t.Parallel()

	// maxLen < 4 returns the full string to avoid panic
	assert.Equal(t, "SELECT 1", analyzer.TruncateSQL("SELECT 1", 3))
}

func TestExtractStmtSQL_singleStatement(t *testing.T) {
	t.Parallel()

	fullSQL := "CREATE TABLE users (id INT);"
	stmts := parseStmts(t, fullSQL)

	result := analyzer.ExtractStmtSQL(stmts, 0, fullSQL)
	assert.Equal(t, "CREATE TABLE users (id INT);", result)
}

func TestExtractStmtSQL_multipleStatements(t *testing.T) {
	t.Parallel()

	fullSQL := "CREATE TABLE a (id INT); CREATE TABLE b (id INT);"
	stmts := parseStmts(t, fullSQL)

	first := analyzer.ExtractStmtSQL(stmts, 0, fullSQL)
	assert.Equal(t, "CREATE TABLE a (id INT);", first)

	second := analyzer.ExtractStmtSQL(stmts, 1, fullSQL)
	assert.Equal(t, "CREATE TABLE b (id INT);", second)
}

func TestExtractStmtSQL_emptyStmts(t *testing.T) {
	t.Parallel()

	assert.Empty(t, analyzer.ExtractStmtSQL(nil, 0, "SELECT 1"))
}

func TestExtractStmtSQL_outOfBounds(t *testing.T) {
	t.Parallel()

	fullSQL := "SELECT 1;"
	stmts := parseStmts(t, fullSQL)

	assert.Empty(t, analyzer.ExtractStmtSQL(stmts, 5, fullSQL))
	assert.Empty(t, analyzer.ExtractStmtSQL(stmts, -1, fullSQL))
}

func TestHasHighOrCritical(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity analyzer.Severity
		expected bool
	}{
		{"safe", analyzer.Safe, false},
		{"low", analyzer.Low, false},
		{"medium", analyzer.Medium, false},
		{"high", analyzer.High, true},
		{"critical", analyzer.Critical, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &analyzer.AnalysisResult{MaxSeverity: tt.severity}
			assert.Equal(t, tt.expected, r.HasHighOrCritical())
		})
	}
}

// parseStmts is a test helper that parses SQL and returns the raw statements.
func parseStmts(t *testing.T, sql string) []*pg_query.RawStmt {
	t.Helper()

	result, err := pg_query.Parse(sql)
	if err != nil {
		t.Fatalf("failed to parse SQL: %v", err)
	}

	return result.Stmts
}
