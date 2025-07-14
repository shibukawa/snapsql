package parserstep2

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestExtractIfCondition(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		expectedClauses   int
		expectedCondition string
	}{
		{
			name:              "WHERE clause with if condition",
			sql:               `SELECT id, name FROM users /*# if user_id */ WHERE id = 1 /*# end */`,
			expectedClauses:   3, // SELECT, FROM, WHERE
			expectedCondition: "user_id",
		},
		{
			name:              "ORDER BY clause with if condition",
			sql:               `SELECT id, name FROM users /*# if sort_enabled */ ORDER BY name /*# end */`,
			expectedClauses:   3, // SELECT, FROM, ORDER BY
			expectedCondition: "sort_enabled",
		},
		{
			name:              "LIMIT clause with if condition",
			sql:               `SELECT id, name FROM users WHERE id > 0 /*# if limit_enabled */ LIMIT 10 /*# end */`,
			expectedClauses:   4, // SELECT, FROM, WHERE, LIMIT
			expectedCondition: "limit_enabled",
		},
		{
			name:              "No if/end directives",
			sql:               `SELECT id, name FROM users WHERE id = 1`,
			expectedClauses:   3, // SELECT, FROM, WHERE
			expectedCondition: "",
		},
		{
			name:              "if without end",
			sql:               `SELECT id, name FROM users /*# if user_id */ WHERE id = 1`,
			expectedClauses:   3, // SELECT, FROM, WHERE
			expectedCondition: "",
		},
		{
			name:              "end without if",
			sql:               `SELECT id, name FROM users WHERE id = 1 /*# end */`,
			expectedClauses:   3, // SELECT, FROM, WHERE
			expectedCondition: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize
			tokens, err := tok.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Convert to Entity tokens
			pcTokens := tokenToEntity(tokens) // Parse statement
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = false // Disable trace for cleaner output
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			consume, result, err := ParseStatement()(pctx, pcTokens)
			assert.NoError(t, err)
			assert.True(t, consume > 0)
			assert.True(t, len(result) > 0)

			// Check if result has expected number of clauses
			if stmt, ok := result[0].Val.NewValue.(cmn.StatementNode); ok {
				clauses := stmt.Clauses()
				assert.Equal(t, tt.expectedClauses, len(clauses))

				// Find WHERE/ORDER BY/LIMIT/OFFSET clause and check condition
				foundCondition := ""
				for _, clause := range clauses {
					switch c := clause.(type) {
					case *cmn.WhereClause:
						foundCondition = c.IfCondition()
					case *cmn.OrderByClause:
						foundCondition = c.IfCondition()
					case *cmn.LimitClause:
						foundCondition = c.IfCondition()
					case *cmn.OffsetClause:
						foundCondition = c.IfCondition()
					}
				}
				assert.Equal(t, tt.expectedCondition, foundCondition)
			} else {
				t.Fatalf("Expected StatementNode, got %T", result[0].Val.NewValue)
			}
		})
	}
}
