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
		dialect  tok.SqlDialect
	}{
		{"plus", "+", "+", tok.SQLiteDialect},
		{"minus", "-", "-", tok.SQLiteDialect},
		{"equal", "=", "=", tok.SQLiteDialect},
		{"multiply", "*", "*", tok.SQLiteDialect},
		{"divide", "/", "/", tok.SQLiteDialect},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, test.dialect)
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
		dialect       tok.SqlDialect
	}{
		{"simple_column", "col", "col", 1, tok.SQLiteDialect},
		{"qualified_column", "table_name.col", "table_name . col", 3, tok.SQLiteDialect},
		{"underscore_column", "user_id", "user_id", 1, tok.SQLiteDialect},
		{"qualified_underscore", "users.user_id", "users . user_id", 3, tok.SQLiteDialect},
		{"sqlite_non_reserved_table", "table.id", "table . id", 3, tok.SQLiteDialect},
		{"mysql_non_reserved_constraint", "constraint.id", "constraint . id", 3, tok.MySQLDialect},
		{"mysql_non_reserved_order", "order.id", "order . id", 3, tok.MySQLDialect},
		{"quoted_reserved_select", "\"select\".id", `"select" . id`, 3, tok.SQLiteDialect},
		{"backtick_reserved_from", "`from`.column", "`from` . column", 3, tok.MySQLDialect},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, test.dialect)
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
		dialect       tok.SqlDialect
	}{
		{"add", "1 + 2", "1 + 2", 3, tok.SQLiteDialect},
		{"minus", "1 - 2", "1 - 2", 3, tok.SQLiteDialect},
		{"mul", "5 * 6", "5 * 6", 3, tok.SQLiteDialect},
		{"div", "8 / 2", "8 / 2", 3, tok.SQLiteDialect},
		{"and", "8 and 2", "8 and 2", 3, tok.SQLiteDialect},
		{"or", "8 or 2", "8 or 2", 3, tok.SQLiteDialect},
		{"like", "8 like 2", "8 like 2", 3, tok.SQLiteDialect},
		{"not like", `'abc' NOT LIKE '%c'`, `'abc' NOT LIKE '%c'`, 4, tok.SQLiteDialect},

		{"between", "age between 18 and 60", "age between 18 and 60", 5, tok.SQLiteDialect},
		{"between with paren", "age between (18 + 2) and 60", "age between ( 18 + 2 ) and 60", 9, tok.SQLiteDialect},
		{"paren", "(1)", "( 1 )", 3, tok.SQLiteDialect},
		{"column", "age", "age", 1, tok.SQLiteDialect},
		{"qualified_column", "users.id", "users . id", 3, tok.SQLiteDialect},
		{"column_arithmetic", "age + 1", "age + 1", 3, tok.SQLiteDialect},
		{"qualified_arithmetic", "users.age * 2", "users . age * 2", 5, tok.SQLiteDialect},
		{"boolean_true", "TRUE", "TRUE", 1, tok.SQLiteDialect},
		{"boolean_false", "FALSE", "FALSE", 1, tok.SQLiteDialect},
		{"boolean_mixed_case", "True", "True", 1, tok.SQLiteDialect},
		{"null_literal", "NULL", "NULL", 1, tok.SQLiteDialect},
		{"null_mixed_case", "Null", "Null", 1, tok.SQLiteDialect},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, test.dialect)
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

func TestAnyIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		expected    string
		shouldError bool
		dialect     tok.SqlDialect
	}{
		{"regular_identifier", "user_id", "user_id", false, tok.SQLiteDialect},
		{"sqlite_non_reserved_table", "table", "table", false, tok.SQLiteDialect},              // Non-reserved in SQLite
		{"mysql_non_reserved_constraint", "constraint", "constraint", false, tok.MySQLDialect}, // Non-reserved in MySQL
		{"mysql_non_reserved_order", "order", "order", false, tok.MySQLDialect},                // Non-reserved in MySQL
		{"quoted_reserved_keyword", "\"select\"", `"select"`, false, tok.SQLiteDialect},
		{"backtick_reserved_keyword", "`from`", "`from`", false, tok.MySQLDialect},
		{"mysql_non_reserved_index", "index", "index", false, tok.MySQLDialect}, // Non-reserved in MySQL
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, test.dialect)
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
