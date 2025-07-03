package parserstep2

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func TestSimpleParsers(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		parser   pc.Parser[Entity]
		expected []string
	}{
		{
			name:     "identifier",
			src:      "user_id ",
			parser:   identifier(),
			expected: []string{"identifier"},
		},
		{
			name:     "number",
			src:      "123 ",
			parser:   number(),
			expected: []string{"number"},
		},
		{
			name:     "string",
			src:      "'hello' ",
			parser:   str(),
			expected: []string{"string"},
		},
		{
			name:     "comma",
			src:      ", ",
			parser:   comma(),
			expected: []string{"comma"},
		},
		{
			name:     "semicolon",
			src:      "; ",
			parser:   semicolon(),
			expected: []string{"semicolon"},
		},
		{
			name:     "operator equal",
			src:      "= ",
			parser:   operator(),
			expected: []string{"operator"},
		},
		{
			name:     "operator plus",
			src:      "+ ",
			parser:   operator(),
			expected: []string{"operator"},
		},
		{
			name:     "space",
			src:      " ",
			parser:   space(),
			expected: []string{"space"},
		},
		{
			name:     "comment line",
			src:      "-- comment",
			parser:   comment(),
			expected: []string{"comment"},
		},
		{
			name:     "comment block",
			src:      "/* comment */",
			parser:   comment(),
			expected: []string{"comment"},
		},
		{
			name:     "literalExpr number",
			src:      "42 ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "literalExpr string",
			src:      "'abc' ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "literalExpr boolean true",
			src:      "TRUE ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "literalExpr boolean false",
			src:      "FALSE ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "literalExpr boolean mixed case",
			src:      "True ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "literalExpr null",
			src:      "NULL ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "literalExpr null mixed case",
			src:      "null ",
			parser:   literal(),
			expected: []string{"literal"},
		},
		{
			name:     "paren open",
			src:      "(",
			parser:   parenOpen(),
			expected: []string{"parenOpen"},
		},
		{
			name:     "paren close",
			src:      ")",
			parser:   parenClose(),
			expected: []string{"parenClose"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Generate tokens from source
			tz := tok.NewSqlTokenizer(test.src, tok.NewSQLiteDialect())
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)

			// Convert to parsercombinator tokens
			pcTokens := TokenToEntity(tokens)

			// Parse using the specified parser
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := test.parser(pctx, pcTokens)
			assert.NoError(t, err)
			assert.True(t, consumed > 0, "parser should consume at least one token")

			// Extract type strings from result
			var actualTypes []string
			for _, token := range result {
				actualTypes = append(actualTypes, token.Type)
			}

			assert.Equal(t, test.expected, actualTypes)
		})
	}
}

func TestComplexParsing(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		parser   pc.Parser[Entity]
		expected []string
	}{
		{
			name:     "identifier with trailing spaces",
			src:      "user_id   ",
			parser:   identifier(),
			expected: []string{"identifier"},
		},
		{
			name:     "number with trailing comment",
			src:      "123 -- this is a number",
			parser:   number(),
			expected: []string{"number"},
		},
		{
			name: "multiple tokens",
			src:  "id = 123",
			parser: pc.Seq(
				ws(identifier()),
				operator(),
				number(),
			),
			expected: []string{"identifier", "operator", "number"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Generate tokens from source
			tz := tok.NewSqlTokenizer(test.src, tok.NewSQLiteDialect())
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			// Convert to parsercombinator tokens
			pcTokens := TokenToEntity(tokens)

			// Parse sequentially using the specified parsers
			var actualTypes []string
			pctx := &pc.ParseContext[Entity]{}
			pctx.TraceEnable = true // Enable tracing for debugging

			pctx.MaxDepth = 20
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			consumed, result, err := test.parser(pctx, pcTokens)
			if err != nil {
				pctx.DumpTrace()
			}

			assert.NoError(t, err)
			assert.True(t, consumed > 0, "parser should consume at least one token")
			for _, token := range result {
				actualTypes = append(actualTypes, token.Type)
			}
			assert.Equal(t, test.expected, actualTypes)
		})
	}
}

func TestParserNotMatch(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		parser pc.Parser[Entity]
	}{
		{
			name:   "identifier parser with number",
			src:    "123",
			parser: identifier(),
		},
		{
			name:   "number parser with identifier",
			src:    "abc",
			parser: number(),
		},
		{
			name:   "operator parser with identifier",
			src:    "abc",
			parser: operator(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Generate tokens from source
			tz := tok.NewSqlTokenizer(test.src, tok.NewSQLiteDialect())
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)

			// Convert to parsercombinator tokens
			pcTokens := TokenToEntity(tokens)

			// Parse using the specified parser - should return ErrNotMatch
			pctx := &pc.ParseContext[Entity]{}
			consumed, _, err := test.parser(pctx, pcTokens)
			assert.Error(t, err)
			assert.Equal(t, 0, consumed)
			assert.Equal(t, pc.ErrNotMatch, err)
		})
	}
}

func TestSnapSQLDirectiveParsers(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		parser   pc.Parser[Entity]
		expected []string
	}{
		{
			name:     "if directive",
			src:      "/*# if user_id */",
			parser:   ifDirective(),
			expected: []string{"if-directive"},
		},
		{
			name:     "end directive",
			src:      "/*# end */",
			parser:   endDirective(),
			expected: []string{"end-directive"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Generate tokens from source
			tz := tok.NewSqlTokenizer(test.src, tok.NewSQLiteDialect())
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)

			// Convert to parsercombinator tokens
			pcTokens := TokenToEntity(tokens)

			// Parse using the specified parser
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := test.parser(pctx, pcTokens)
			assert.NoError(t, err)
			assert.True(t, consumed > 0, "parser should consume at least one token")

			// Extract type strings from result
			var actualTypes []string
			for _, token := range result {
				actualTypes = append(actualTypes, token.Type)
			}

			assert.Equal(t, test.expected, actualTypes)
		})
	}
}

func TestAtomic(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		expectedType string
		expectedRaw  string
		shouldError  bool
		dialect      tok.SqlDialect
	}{
		// Literal tests
		{"number", "42", "literal", "42", false, tok.SQLiteDialect},
		{"string", "'hello'", "literal", "'hello'", false, tok.SQLiteDialect},
		{"boolean_true", "TRUE", "literal", "TRUE", false, tok.SQLiteDialect},
		{"null", "NULL", "literal", "NULL", false, tok.SQLiteDialect},

		// Column reference tests
		{"simple_column", "col", "column-reference", "col", false, tok.SQLiteDialect},
		{"qualified_column", "table_name.col", "column-reference", "table_name.col", false, tok.SQLiteDialect},
		{"quoted_reserved", "\"select\".id", "column-reference", "\"select\".id", false, tok.SQLiteDialect},
		{"backtick_mysql", "`from`.column", "column-reference", "`from`.column", false, tok.MySQLDialect},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, test.dialect)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := atomic()(pctx, pcTokens)

			if test.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, consumed > 0)
				assert.Equal(t, test.expectedType, result[0].Type)
				assert.Equal(t, test.expectedRaw, result[0].Raw)
			}
		})
	}
}

func TestColumnReferenceNode(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		expectedTable string
		expectedCol   string
		dialect       tok.SqlDialect
	}{
		{"simple", "age", "", "age", tok.SQLiteDialect},
		{"qualified", "users.age", "users", "age", tok.SQLiteDialect},
		{"quoted_table", "\"users\".age", "\"users\"", "age", tok.SQLiteDialect},
		{"quoted_column", "users.\"user_id\"", "users", "\"user_id\"", tok.SQLiteDialect},
		{"mysql_backtick", "`order`.id", "`order`", "id", tok.MySQLDialect},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tz := tok.NewSqlTokenizer(test.src, test.dialect)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			consumed, result, err := columnReference()(pctx, pcTokens)

			assert.NoError(t, err)
			assert.True(t, consumed > 0)
			assert.Equal(t, "column-reference", result[0].Type)

			// Check AST node
			colRef, ok := result[0].Val.NewValue.(*ColumnReferenceNode)
			assert.True(t, ok)
			assert.Equal(t, test.expectedTable, colRef.TableName)
			assert.Equal(t, test.expectedCol, colRef.ColumnName)
			assert.Equal(t, cmn.COLUMN_REFERENCE, colRef.Type())
		})
	}
}
