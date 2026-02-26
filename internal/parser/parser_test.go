package parser_test

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ahmad/migrate/internal/parser"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sql       string
		wantErr   bool
		wantStmts int
		checkNode func(t *testing.T, result *parser.ParseResult)
	}{
		{
			name:      "valid CREATE TABLE returns one statement",
			sql:       "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				_, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_CreateStmt)
				assert.True(t, ok, "expected CreateStmt node")
			},
		},
		{
			name:      "multi-statement SQL returns correct count",
			sql:       "CREATE TABLE a (id INT); CREATE TABLE b (id INT); CREATE TABLE c (id INT);",
			wantStmts: 3,
		},
		{
			name:      "CREATE INDEX CONCURRENTLY parses correctly",
			sql:       "CREATE INDEX CONCURRENTLY idx_name ON users (email);",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				node, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_IndexStmt)
				require.True(t, ok, "expected IndexStmt node")
				assert.True(t, node.IndexStmt.Concurrent, "expected Concurrent to be true")
			},
		},
		{
			name:      "ALTER TABLE ADD COLUMN parses correctly",
			sql:       "ALTER TABLE users ADD COLUMN status TEXT;",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				_, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_AlterTableStmt)
				assert.True(t, ok, "expected AlterTableStmt node")
			},
		},
		{
			name:    "invalid SQL returns error",
			sql:     "SELECT * FROM WHERE;",
			wantErr: true,
		},
		{
			name:      "empty string returns zero statements",
			sql:       "",
			wantStmts: 0,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				assert.Empty(t, result.SQL)
			},
		},
		{
			name:      "whitespace-only returns zero statements",
			sql:       "   \n\t  ",
			wantStmts: 0,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				assert.Equal(t, "   \n\t  ", result.SQL, "original SQL preserved")
			},
		},
		{
			name:      "VACUUM FULL parses as VacuumStmt",
			sql:       "VACUUM FULL users;",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				_, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_VacuumStmt)
				assert.True(t, ok, "expected VacuumStmt node")
			},
		},
		{
			name:      "LOCK TABLE parses as LockStmt",
			sql:       "LOCK TABLE users IN ACCESS EXCLUSIVE MODE;",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				_, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_LockStmt)
				assert.True(t, ok, "expected LockStmt node")
			},
		},
		{
			name:      "DROP TABLE parses as DropStmt",
			sql:       "DROP TABLE users;",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				_, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_DropStmt)
				assert.True(t, ok, "expected DropStmt node")
			},
		},
		{
			name:      "RENAME COLUMN parses as RenameStmt",
			sql:       "ALTER TABLE users RENAME COLUMN email TO email_address;",
			wantStmts: 1,
			checkNode: func(t *testing.T, result *parser.ParseResult) {
				t.Helper()
				_, ok := result.Stmts[0].Stmt.Node.(*pg_query.Node_RenameStmt)
				assert.True(t, ok, "expected RenameStmt node")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parser.Parse(tt.sql)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Stmts, tt.wantStmts)
			assert.Equal(t, tt.sql, result.SQL)

			if tt.checkNode != nil {
				tt.checkNode(t, result)
			}
		})
	}
}
