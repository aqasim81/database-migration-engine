package analyzer

import (
	"fmt"

	"github.com/aqasim81/database-migration-engine/internal/migration"
	"github.com/aqasim81/database-migration-engine/internal/parser"
)

// Option configures the Analyzer.
type Option func(*Analyzer)

// Analyzer runs registered rules against parsed migrations.
type Analyzer struct {
	registry  *Registry
	parseFn   func(string) (*parser.ParseResult, error)
	pgVersion int
}

// New creates a new Analyzer with the given options.
func New(opts ...Option) *Analyzer {
	a := &Analyzer{
		registry:  NewRegistry(),
		parseFn:   parser.Parse,
		pgVersion: 14, //nolint:mnd // default PostgreSQL version
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// WithRegistry sets a custom rule registry.
func WithRegistry(r *Registry) Option {
	return func(a *Analyzer) { a.registry = r }
}

// WithPGVersion sets the target PostgreSQL version.
func WithPGVersion(v int) Option {
	return func(a *Analyzer) { a.pgVersion = v }
}

// WithParser overrides the SQL parser function (useful for testing).
func WithParser(fn func(string) (*parser.ParseResult, error)) Option {
	return func(a *Analyzer) { a.parseFn = fn }
}

// Analyze parses and analyzes a single migration, returning all findings.
func (a *Analyzer) Analyze(m *migration.Migration) (*AnalysisResult, error) {
	result, err := a.parseFn(m.UpSQL)
	if err != nil {
		return nil, fmt.Errorf("parsing migration %s: %w", m.Version, err)
	}

	var findings []Finding

	maxSeverity := Safe

	for i, stmt := range result.Stmts {
		ctx := &RuleContext{
			Migration:       m,
			TargetPGVersion: a.pgVersion,
			StmtIndex:       i,
			SQL:             m.UpSQL,
		}

		for _, rule := range a.registry.Rules() {
			fs := rule.Check(stmt, ctx)
			for j := range fs {
				if fs[j].Severity > maxSeverity {
					maxSeverity = fs[j].Severity
				}
			}

			findings = append(findings, fs...)
		}
	}

	return &AnalysisResult{
		Migration:   m,
		Findings:    findings,
		MaxSeverity: maxSeverity,
	}, nil
}

// AnalyzeAll analyzes multiple migrations and returns results for each.
func (a *Analyzer) AnalyzeAll(migrations []migration.Migration) ([]AnalysisResult, error) {
	results := make([]AnalysisResult, 0, len(migrations))

	for i := range migrations {
		r, err := a.Analyze(&migrations[i])
		if err != nil {
			return nil, err
		}

		results = append(results, *r)
	}

	return results, nil
}
