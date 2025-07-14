package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/parser2/parserstep3"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestFinalizeDeleteFromClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{"delete ok", "DELETE FROM users WHERE id = 1", false},
		{"delete error (no table)", "DELETE FROM WHERE id = 1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err)
			ast, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(ast)
			assert.NoError(t, err)
			selectStmt, ok := ast.(*cmn.DeleteFromStatement)
			assert.True(t, ok)
			perr := &cmn.ParseError{}
			FinalizeDeleteFromClause(selectStmt.From, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
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
			FinalizeUpdateClause(selectStmt.Update, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
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
			FinalizeSetClause(selectStmt.Set, perr)
			if tc.wantError {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				assert.Equal(t, 0, len(perr.Errors))
			}
		})
	}
}
