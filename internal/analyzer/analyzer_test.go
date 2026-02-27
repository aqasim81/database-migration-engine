package analyzer_test

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer"
	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

// stubRule is a test rule that always returns a finding.
type stubRule struct{}

func (r *stubRule) ID() string { return "test-stub" }

func (r *stubRule) Check(_ *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	return []analyzer.Finding{{
		Rule:      r.ID(),
		Severity:  analyzer.High,
		Message:   "stub finding",
		StmtIndex: ctx.StmtIndex,
	}}
}

func TestAnalyze_safeMigration_noFindings(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "create_users",
		UpSQL:   "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);",
	}

	a := analyzer.New() // no rules registered

	result, err := a.Analyze(m)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	assert.Equal(t, analyzer.Safe, result.MaxSeverity)
}

func TestAnalyze_withStubRule_returnsFindings(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "create_users",
		UpSQL:   "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);",
	}

	registry := analyzer.NewRegistry()
	registry.Register(&stubRule{})

	a := analyzer.New(analyzer.WithRegistry(registry))

	result, err := a.Analyze(m)
	require.NoError(t, err)
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, analyzer.High, result.MaxSeverity)
	assert.Equal(t, "test-stub", result.Findings[0].Rule)
}

func TestAnalyze_invalidSQL_returnsError(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "bad_sql",
		UpSQL:   "NOT VALID SQL AT ALL;;;",
	}

	a := analyzer.New()

	_, err := a.Analyze(m)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing migration 001")
}

func TestAnalyze_emptyMigration_noFindings(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "empty",
		UpSQL:   "",
	}

	a := analyzer.New()

	result, err := a.Analyze(m)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	assert.Equal(t, analyzer.Safe, result.MaxSeverity)
}

func TestAnalyzeAll_multipleMigrations_correctResultCount(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "first", UpSQL: "CREATE TABLE a (id INT);"},
		{Version: "002", Name: "second", UpSQL: "CREATE TABLE b (id INT);"},
	}

	a := analyzer.New()

	results, err := a.AnalyzeAll(migrations)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestAnalyzeAll_errorInOne_returnsWrappedError(t *testing.T) {
	t.Parallel()

	migrations := []migration.Migration{
		{Version: "001", Name: "good", UpSQL: "CREATE TABLE a (id INT);"},
		{Version: "002", Name: "bad", UpSQL: "INVALID SQL;;;"},
	}

	a := analyzer.New()

	_, err := a.AnalyzeAll(migrations)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration 002")
	assert.Contains(t, err.Error(), "parsing migration 002")
}

func TestAnalyze_multiStatement_runsRulesOnEach(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "multi",
		UpSQL:   "CREATE TABLE a (id INT); CREATE TABLE b (id INT);",
	}

	registry := analyzer.NewRegistry()
	registry.Register(&stubRule{})

	a := analyzer.New(analyzer.WithRegistry(registry))

	result, err := a.Analyze(m)
	require.NoError(t, err)
	assert.Len(t, result.Findings, 2)
	assert.Equal(t, 0, result.Findings[0].StmtIndex)
	assert.Equal(t, 1, result.Findings[1].StmtIndex)
}

func TestAnalyze_populatesStatementField(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "test",
		UpSQL:   "CREATE TABLE users (id BIGSERIAL PRIMARY KEY);",
	}

	registry := analyzer.NewRegistry()
	registry.Register(&stubRule{})

	a := analyzer.New(analyzer.WithRegistry(registry))

	result, err := a.Analyze(m)
	require.NoError(t, err)
	require.Len(t, result.Findings, 1)
	assert.NotEmpty(t, result.Findings[0].Statement)
	assert.Contains(t, result.Findings[0].Statement, "CREATE TABLE users")
}

func TestWithPGVersion_setsVersion(t *testing.T) {
	t.Parallel()

	m := &migration.Migration{
		Version: "001",
		Name:    "test",
		UpSQL:   "CREATE TABLE a (id INT);",
	}

	// Use a rule that captures the PG version from context
	capturedVersion := 0
	capturingRule := &versionCapturingRule{captured: &capturedVersion}

	registry := analyzer.NewRegistry()
	registry.Register(capturingRule)

	a := analyzer.New(
		analyzer.WithRegistry(registry),
		analyzer.WithPGVersion(10), //nolint:mnd // test value
	)

	_, err := a.Analyze(m)
	require.NoError(t, err)
	assert.Equal(t, 10, capturedVersion)
}

func TestWithParser_overridesParser(t *testing.T) {
	t.Parallel()

	customParseCalled := false
	customParse := func(sql string) (*parser.ParseResult, error) {
		customParseCalled = true
		return parser.Parse(sql)
	}

	m := &migration.Migration{
		Version: "001",
		Name:    "test",
		UpSQL:   "CREATE TABLE a (id INT);",
	}

	a := analyzer.New(analyzer.WithParser(customParse))

	_, err := a.Analyze(m)
	require.NoError(t, err)
	assert.True(t, customParseCalled)
}

// versionCapturingRule captures the PG version from context for testing.
type versionCapturingRule struct {
	captured *int
}

func (r *versionCapturingRule) ID() string { return "version-capture" }

func (r *versionCapturingRule) Check(_ *pg_query.RawStmt, ctx *analyzer.RuleContext) []analyzer.Finding {
	*r.captured = ctx.TargetPGVersion
	return nil
}
