package parserstep5

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "Simple SELECT with LIMIT variable",
			sql:  "SELECT id FROM users LIMIT /*= limit */10",
		},
		{
			name: "SELECT with variable directive",
			sql:  "SELECT /*= user.name */name FROM users",
		},
		{
			name: "Valid template with simple variable",
			sql:  "SELECT id, name FROM users WHERE status = /*= status */1",
		},
		{
			name: "Template with environment variable",
			sql:  "SELECT id, name FROM users WHERE role = /*@ role */admin",
		},
		{
			name: "Template with LIMIT implicit condition",
			sql:  "SELECT id, name FROM users ORDER BY id LIMIT /*= limit */10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Parse statement using parserstep2
			statement, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)

			// Parse statement using parserstep3
			err = parserstep3.Execute(statement)
			assert.NoError(t, err)
			// Parse statement using parserstep4
			err = parserstep4.Execute(statement)
			assert.NoError(t, err)

			// Execute parserstep5 (includes parserstep3 and parserstep4)
			err = Execute(statement)
			assert.NoError(t, err)
		})
	}
}
