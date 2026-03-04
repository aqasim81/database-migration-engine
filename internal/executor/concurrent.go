package executor

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/parser"
)

// containsConcurrentOp parses the SQL and returns true if any statement
// is a CREATE INDEX CONCURRENTLY or DROP INDEX CONCURRENTLY. Such statements
// cannot run inside a transaction block and must be executed directly on the pool.
func containsConcurrentOp(sql string) (bool, error) {
	// Fast path: skip CGO parser overhead when keyword is absent.
	if !strings.Contains(strings.ToUpper(sql), "CONCURRENTLY") {
		return false, nil
	}

	result, err := parser.Parse(sql)
	if err != nil {
		return false, fmt.Errorf("parsing SQL for concurrent operation detection: %w", err)
	}

	for _, stmt := range result.Stmts {
		switch node := stmt.Stmt.Node.(type) {
		case *pg_query.Node_IndexStmt:
			if node.IndexStmt != nil && node.IndexStmt.Concurrent {
				return true, nil
			}
		case *pg_query.Node_DropStmt:
			if node.DropStmt != nil && node.DropStmt.Concurrent {
				return true, nil
			}
		}
	}

	return false, nil
}
