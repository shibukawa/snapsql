package parserstep3

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	step2 "github.com/shibukawa/snapsql/parser/parserstep2"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestStatementClausePresence(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantErr     bool
		wantClauses int
	}{
		{
			name:        "SELECT with WHERE",
			sql:         "SELECT id FROM users WHERE id = 1",
			wantErr:     false,
			wantClauses: 3, // SELECT, FROM, WHERE
		},
		{
			name:        "INSERT with WHERE (invalid)",
			sql:         "INSERT INTO users (id) SELECT id FROM new_users WHERE id = 1",
			wantErr:     false,
			wantClauses: 4, // INSERT INTO, VALUES, WHERE
		},
		{
			name:        "INSERT with WHERE (invalid)",
			sql:         "INSERT INTO users (id) VALUES (1) WHERE id = 1",
			wantErr:     true,
			wantClauses: 2, // INSERT INTO, VALUES, WHERE
		},
		{
			name:        "UPDATE with GROUP BY (invalid)",
			sql:         "UPDATE users SET name = 'a' GROUP BY id",
			wantErr:     true,
			wantClauses: 2, // UPDATE, SET
		},
		{
			name:        "DELETE with ORDER BY (invalid)",
			sql:         "DELETE FROM users ORDER BY id",
			wantErr:     true,
			wantClauses: 1, // DELETE FROM
		},
		{
			name:        "SELECT with ORDER BY",
			sql:         "SELECT id FROM users ORDER BY id",
			wantErr:     false,
			wantClauses: 3, // SELECT, FROM, ORDER BY
		},
		{
			name:        "INSERT with RETURNING",
			sql:         "INSERT INTO users (id) VALUES (1) RETURNING id",
			wantErr:     false,
			wantClauses: 3, // INSERT INTO, VALUES, RETURNING
		},
		{
			name:        "UPDATE with RETURNING",
			sql:         "UPDATE users SET name = 'a' RETURNING id",
			wantErr:     false,
			wantClauses: 3, // UPDATE, SET, and RETURNING
		},
		{
			name:        "DELETE with RETURNING",
			sql:         "DELETE FROM users RETURNING id",
			wantErr:     false,
			wantClauses: 2, // DELETE FROM and RETURNING
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.sql)
			if err != nil {
				t.Fatalf("tokenize error: %v", err)
			}

			node, err := step2.Execute(tokens)
			if err != nil {
				t.Fatalf("parserstep2 error: %v", err)
			}

			var perr cmn.ParseError

			clauses := ValidateClausePresence(node.Type(), node.Clauses(), &perr)

			if tt.wantErr {
				assert.NotZero(t, len(perr.Errors), "expected error but got none")
			} else {
				assert.Zero(t, len(perr.Errors), "unexpected error: %v", perr.Error())
			}

			assert.Equal(t, tt.wantClauses, len(clauses), "expected number of clauses does not match")
		})
	}
}
