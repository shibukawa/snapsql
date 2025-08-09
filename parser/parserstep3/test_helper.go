package parserstep3

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	step2 "github.com/shibukawa/snapsql/parser/parserstep2"
	tokenizer "github.com/shibukawa/snapsql/tokenizer"
)

// parseClausesFromSQL parses SQL and returns ClauseNode slice for testing.
func parseClausesFromSQL(t *testing.T, sql string) []cmn.ClauseNode {
	t.Helper()

	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	ast, err := step2.Execute(tokens)
	if err != nil {
		t.Fatalf("parserstep2.Execute error: %v", err)
	}

	return ast.Clauses()
}
