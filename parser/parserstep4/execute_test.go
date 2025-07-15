package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestExecuteParserStep4(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		// SELECT
		{"select ok", "SELECT id FROM users", false},
		{"select error", "SELECT * FROM users", true},
		// INSERT
		{"insert ok", "INSERT INTO users(id) VALUES(1)", false},
		{"insert error", "INSERT INTO users(id, id) VALUES(1, 2)", true},
		// UPDATE
		{"update ok", "UPDATE users SET name = 'foo' WHERE id = 1", false},
		{"update error", "UPDATE users SET name = 'foo', name = 'bar' WHERE id = 1", true},
		// DELETE
		{"delete ok", "DELETE FROM users WHERE id = 1", false},
		{"delete error", "DELETE FROM users WHERE", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			if err != nil {
				t.Fatalf("tokenize error: %v", err)
			}
			ast, err := parserstep2.Execute(tokens)
			if err != nil {
				t.Fatalf("parserstep2 error: %v", err)
			}
			err = parserstep3.Execute(ast)
			if err != nil {
				t.Fatalf("parserstep3 error: %v", err)
			}
			err = Execute(ast)
			if tc.wantError {
				perr, ok := cmn.AsParseError(err)
				assert.True(t, ok, "expected *ParseError, got %v", err)
				assert.NotEqual(t, 0, len(perr.Errors), "expected errors, got none")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
