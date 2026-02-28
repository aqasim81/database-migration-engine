package executor

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"github.com/aqasim81/database-migration-engine/internal/parser"
)

// containsConcurrentIndex parses the SQL and returns true if any statement
// is a CREATE INDEX CONCURRENTLY. Such statements cannot run inside a
// transaction block and must be executed directly on the pool.
func containsConcurrentIndex(sql string) (bool, error) {
	result, err := parser.Parse(sql)
	if err != nil {
		return false, fmt.Errorf("parsing SQL for concurrent index detection: %w", err)
	}

	for _, stmt := range result.Stmts {
		node, ok := stmt.Stmt.Node.(*pg_query.Node_IndexStmt)
		if !ok {
			continue
		}

		if node.IndexStmt != nil && node.IndexStmt.Concurrent {
			return true, nil
		}
	}

	return false, nil
}
