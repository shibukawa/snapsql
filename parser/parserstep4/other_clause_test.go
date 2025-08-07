package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/tokenizer"
)

// testClauseFinalization is a helper function for testing clause finalization
func testClauseFinalization(t *testing.T, tests []struct {
	name      string
	sql       string
	wantError bool
}, statementType string, finalizeFunc func(interface{}, *cmn.ParseError)) {
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err)
			ast, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(ast)
			assert.NoError(t, err)

			perr := &cmn.ParseError{}

			switch statementType {
			case "DELETE":
				stmt, ok := ast.(*cmn.DeleteFromStatement)
				assert.True(t, ok)
				finalizeFunc(stmt.From, perr)
			case "UPDATE":
				stmt, ok := ast.(*cmn.UpdateStatement)
				assert.True(t, ok)
				finalizeFunc(stmt.Update, perr)
			}

			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
}

func TestFinalizeDeleteFromClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{"delete ok", "DELETE FROM users WHERE id = 1", false},
		{"delete error (no table)", "DELETE FROM WHERE id = 1", true},
	}

	testClauseFinalization(t, tests, "DELETE", func(clause interface{}, perr *cmn.ParseError) {
		finalizeDeleteFromClause(clause.(*cmn.DeleteFromClause), perr)
	})
}

func TestFinalizeUpdateClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{"update ok", "UPDATE users SET name = 'foo' WHERE id = 1", false},
		{"update error (no table)", "UPDATE SET name = 'foo' WHERE id = 1", true},
	}

	testClauseFinalization(t, tests, "UPDATE", func(clause interface{}, perr *cmn.ParseError) {
		finalizeUpdateClause(clause.(*cmn.UpdateClause), perr)
	})
}

func TestFinalizeSetClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{"set ok", "UPDATE users SET name = 'foo', age = 20 WHERE id = 1", false},
		{"set error (duplicate)", "UPDATE users SET name = 'foo', name = 'bar' WHERE id = 1", true},
		{"set error (empty)", "UPDATE users SET WHERE id = 1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err)
			ast, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(ast)
			assert.NoError(t, err)
			selectStmt, ok := ast.(*cmn.UpdateStatement)
			assert.True(t, ok)
			perr := &cmn.ParseError{}
			finalizeSetClause(selectStmt.Set, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
}

func TestFinalizeReturningClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{
			name:      "returning ok",
			sql:       "UPDATE users SET name = 'foo' RETURNING id, name",
			wantError: false,
		},
		{
			name:      "returning alias",
			sql:       "UPDATE users SET name = 'foo' RETURNING id AS user_id, name AS user_name",
			wantError: false,
		},
		{
			name:      "returning cast(1)",
			sql:       "UPDATE users SET name = 'foo' RETURNING CAST(id AS TEXT)",
			wantError: false,
		},
		{
			name:      "returning cast(2)",
			sql:       "UPDATE users SET name = 'foo' RETURNING id::text",
			wantError: false,
		},
		{
			name:      "returning cast(3)",
			sql:       "UPDATE users SET name = 'foo' RETURNING (id)::text",
			wantError: false,
		},
		{
			name:      "returning error (duplicate)",
			sql:       "UPDATE users SET name = 'foo' RETURNING id, id",
			wantError: true,
		},
		{
			name:      "returning error (empty)",
			sql:       "UPDATE users SET name = 'foo' RETURNING",
			wantError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err)
			ast, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(ast)
			assert.NoError(t, err)
			selectStmt, ok := ast.(*cmn.UpdateStatement)
			assert.True(t, ok)
			perr := &cmn.ParseError{}
			finalizeReturningClause(selectStmt.Returning, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
}
