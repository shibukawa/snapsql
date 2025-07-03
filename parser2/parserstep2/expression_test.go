package parserstep2

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
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
			tz := tok.NewSqlTokenizer(test.src)
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

func TestColumnReference(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		exprStr       string
		rawTokenCount int
	}{
		{"simple_column", "col", "col", 1},
		{"qualified_column", "table_name.col", "table_name . col", 3},
		{"underscore_column", "user_id", "user_id", 1},
		{"qualified_underscore", "users.user_id", "users . user_id", 3},
		{"quoted_reserved_select", "\"select\".id", `"select" . id`, 3},
		{"backtick_reserved_from", "`from`.column", "`from` . column", 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 20
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true
			consumed, result, err := columnReference()(pctx, pcTokens)
			if err != nil {
				pctx.DumpTrace()
			}
			assert.NoError(t, err)
			assert.True(t, consumed > 0)
			assert.Equal(t, "column-reference", result[0].Type)
			assert.Equal(t, 1, len(result))
			assert.Equal(t, test.rawTokenCount, len(result[0].Val.rawTokens))

			// Check that it returns ColumnReferenceNode
			colRef, ok := result[0].Val.NewValue.(*ColumnReferenceNode)
			assert.True(t, ok)
			assert.Equal(t, cmn.COLUMN_REFERENCE, colRef.Type())

			var nodes []string
			for _, t := range result[0].Val.rawTokens {
				nodes = append(nodes, t.Value)
			}
			assert.Equal(t, test.exprStr, strings.Join(nodes, " "))
		})
	}
}

// Update TestExpression to include column references
func TestExpression(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		exprStr       string
		rawTokenCount int
	}{
		{"add", "1 + 2", "1 + 2", 3},
		{"minus", "1 - 2", "1 - 2", 3},
		{"mul", "5 * 6", "5 * 6", 3},
		{"div", "8 / 2", "8 / 2", 3},
		{"and", "8 and 2", "8 and 2", 3},
		{"or", "8 or 2", "8 or 2", 3},
		{"like", "8 like 2", "8 like 2", 3},
		{"not like", `'abc' NOT LIKE '%c'`, `'abc' NOT LIKE '%c'`, 4},

		{"between", "age between 18 and 60", "age between 18 and 60", 5},
		{"between with paren", "age between (18 + 2) and 60", "age between ( 18 + 2 ) and 60", 9},
		{"paren", "(1)", "( 1 )", 3},
		{"column", "age", "age", 1},
		{"qualified_column", "users.id", "users . id", 3},
		{"column_arithmetic", "age + 1", "age + 1", 3},
		{"qualified_arithmetic", "users.age * 2", "users . age * 2", 5},
		{"boolean_true", "TRUE", "TRUE", 1},
		{"boolean_false", "FALSE", "FALSE", 1},
		{"boolean_mixed_case", "True", "True", 1},
		{"null_literal", "NULL", "NULL", 1},
		{"null_mixed_case", "Null", "Null", 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true
			consumed, result, err := expression(SelectClause)(pctx, pcTokens) // Add SelectClause parameter
			if err != nil {
				pctx.DumpTrace()
			}
			assert.NoError(t, err)
			assert.Equal(t, consumed, len(pcTokens))
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

func TestIsExpression(t *testing.T) {
	tests := []struct {
		name           string
		src            string
		expectedTokens int
		shouldError    bool
	}{
		{"is_null", "age IS NULL", 3, false},
		{"is_true", "flag IS TRUE", 3, false},
		{"is_false", "flag IS FALSE", 3, false},
		{"is_not_null", "age IS NOT NULL", 4, false},
		{"is_not_true", "flag IS NOT TRUE", 4, false},
		{"is_not_false", "flag IS NOT FALSE", 4, false},
		// {"not_is_expr", "age is 1", 0, true}, // this step passes, but later step should fail
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := expression(SelectClause)(pctx, pcTokens)

			if test.shouldError {
				assert.Error(t, err)
				assert.Equal(t, 0, consumed)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(pcTokens), consumed)
				assert.Equal(t, test.expectedTokens, len(result[0].Val.rawTokens))
			}
		})
	}
}

func TestAnyIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		expected    string
		shouldError bool
	}{
		{"regular_identifier", "user_id", "user_id", false},
		{"quoted_reserved_keyword", "\"select\"", `"select"`, false},
		{"backtick_reserved_keyword", "`from`", "`from`", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := anyIdentifier()(pctx, pcTokens)

			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, consumed > 0)
				assert.Equal(t, "identifier", result[0].Type)
				assert.Equal(t, test.expected, result[0].Raw)
			}
		})
	}
}
