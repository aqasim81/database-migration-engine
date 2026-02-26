package parser //nolint:revive // intentional: does not conflict with go/parser in internal package

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// ParseResult holds the parsed AST and original SQL.
type ParseResult struct {
	Stmts []*pg_query.RawStmt
	SQL   string
}

// Parse parses a PostgreSQL SQL string and returns the AST.
// Returns an empty result (zero statements) for empty or whitespace-only input.
func Parse(sql string) (*ParseResult, error) {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return &ParseResult{SQL: sql}, nil
	}

	tree, err := pg_query.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parsing SQL: %w", err)
	}

	return &ParseResult{
		Stmts: tree.Stmts,
		SQL:   sql,
	}, nil
}
