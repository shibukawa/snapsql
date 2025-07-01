package parserstep2

import (
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

func TestExpression_Binary(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		exprStr string
	}{
		{"add", "1 + 2", "1 + 2"},
		{"sub", "3 - 4", "3 - 4"},
		{"mul", "5 * 6", "5 * 6"},
		{"div", "8 / 2", "8 / 2"},
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
			consumed, result, err := expression(pctx, pcTokens)
			if err != nil {
				pctx.DumpTrace()
			}
			assert.NoError(t, err)
			assert.True(t, consumed > 0)
			ast := result[0].Val.NewValue
			assert.Equal(t, test.exprStr, ast.String())
		})
	}
}

func TestExpression_Paren(t *testing.T) {
	tz := tok.NewSqlTokenizer("( 1 )", tok.NewSQLiteDialect())
	tokens, err := tz.AllTokens()
	assert.NoError(t, err)
	pcTokens := TokenToEntity(tokens)
	pctx := &pc.ParseContext[Entity]{}
	pctx.MaxDepth = 20
	pctx.TraceEnable = true
	consumed, result, err := expression(pctx, pcTokens)
	if err != nil {
		pctx.DumpTrace()
	}
	assert.NoError(t, err)
	assert.True(t, consumed > 0)
	ast := result[0].Val.NewValue
	assert.Equal(t, "1", ast.String())
}
