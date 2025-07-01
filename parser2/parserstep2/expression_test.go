package parserstep2

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestOperatorExpr(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{"plus", "+", "+"},
		{"minus", "-", "-"},
		{"equal", "=", "="},
		{"multiply", "*", "*"},
		{"divide", "/", "/"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, tok.NewSQLiteDialect())
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := operator()(pctx, pcTokens)
			assert.NoError(t, err)
			assert.True(t, consumed > 0)
			assert.Equal(t, test.expected, result[0].Raw)
		})
	}
}

func TestExpression(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		exprStr       string
		rawTokenCount int
	}{
		{"add", "1 + 2", "1 + 2", 3},
		{"sub", "3 - 4", "3 - 4", 3},
		{"mul", "5 * 6", "5 * 6", 3},
		{"div", "8 / 2", "8 / 2", 3},
		{"paren", "( 1 )", "( 1 )", 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, tok.NewSQLiteDialect())
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 20
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true
			consumed, result, err := expression(pctx, pcTokens)
			if err != nil {
				pctx.DumpTrace()
			}
			assert.NoError(t, err)
			assert.True(t, consumed > 0)
			assert.Equal(t, "expression", result[0].Type)
			assert.Equal(t, 1, len(result))
			assert.Equal(t, test.rawTokenCount, len(result[0].Val.rawTokens))
			var nodes []string
			for _, t := range result[0].Val.rawTokens {
				nodes = append(nodes, t.Value)
			}
			assert.Equal(t, test.exprStr, strings.Join(nodes, " "))
		})
	}
}
