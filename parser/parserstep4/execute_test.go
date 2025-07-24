package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/testhelper"
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

func TestExecuteParserStep4WithDummyLiterals(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantError bool
	}{
		{
			name:      "select with dummy literal" + testhelper.GetCaller(t),
			sql:       "SELECT /*= user_id */ FROM users",
			wantError: false,
		},
		{
			name:      "select with multiple dummy literals" + testhelper.GetCaller(t),
			sql:       "SELECT /*= user_id */, /*= user_name */ FROM users",
			wantError: false,
		},
		{
			name:      "insert with dummy literal in values" + testhelper.GetCaller(t),
			sql:       "INSERT INTO users(id, name) VALUES(/*= user_id */, /*= user_name */)",
			wantError: false,
		},
		{
			name:      "update with dummy literal in set" + testhelper.GetCaller(t),
			sql:       "UPDATE users SET name = /*= new_name */ WHERE id = /*= user_id */",
			wantError: false,
		},
		{
			name:      "delete with dummy literal in where" + testhelper.GetCaller(t),
			sql:       "DELETE FROM users WHERE id = /*= user_id */",
			wantError: false,
		},
		{
			name:      "select with dummy literal and alias" + testhelper.GetCaller(t),
			sql:       "SELECT /*= user_id */ AS id, /*= user_name */ AS name FROM users",
			wantError: false,
		},
		{
			name:      "select with dummy literal in function call" + testhelper.GetCaller(t),
			sql:       "SELECT COUNT(/*= user_id */) FROM users",
			wantError: false,
		},
		{
			name:      "select with complex dummy literal expression" + testhelper.GetCaller(t),
			sql:       "SELECT /*= user.profile.name */ FROM users",
			wantError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Tokenize
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err, "tokenize should not fail")

			// Step 2: Run parserstep1 (includes dummy literal insertion)
			processedTokens, err := parserstep1.Execute(tokens)
			assert.NoError(t, err, "parserstep1 should not fail")

			// Verify DUMMY_LITERAL tokens were inserted
			hasDummyLiteral := false
			for _, token := range processedTokens {
				if token.Type == tokenizer.DUMMY_LITERAL {
					hasDummyLiteral = true
					assert.True(t, len(token.Value) > 0, "DUMMY_LITERAL should have variable name in Value")
					break
				}
			}
			assert.True(t, hasDummyLiteral, "should contain DUMMY_LITERAL tokens")

			// Step 3: Run parserstep2
			ast, err := parserstep2.Execute(processedTokens)
			assert.NoError(t, err, "parserstep2 should not fail")

			// Step 4: Run parserstep3
			err = parserstep3.Execute(ast)
			assert.NoError(t, err, "parserstep3 should not fail")

			// Step 5: Run parserstep4 (our focus)
			err = Execute(ast)
			if tc.wantError {
				perr, ok := cmn.AsParseError(err)
				assert.True(t, ok, "expected *ParseError, got %v", err)
				assert.NotEqual(t, 0, len(perr.Errors), "expected errors, got none")
			} else {
				assert.NoError(t, err, "parserstep4 should handle DUMMY_LITERAL tokens")
			}
		})
	}
}
