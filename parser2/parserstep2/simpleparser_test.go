package parserstep2

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
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
				identifier(),
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

			consumed, result, err := test.parser(pctx, pcTokens)
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
