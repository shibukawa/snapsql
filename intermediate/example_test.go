package intermediate

import (
	"fmt"
	"testing"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestExampleOutput(t *testing.T) {
	// Simple SQL example
	sql := `SELECT id, name FROM users WHERE active = /*= active */true`

	// Create tokenizer and parser
	tokenizer := tokenizer.NewSqlTokenizer(sql, tokenizer.NewPostgreSQLDialect())
	tokens, err := tokenizer.AllTokens()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := parser.NewSqlParser(tokens, nil, nil)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	// Create intermediate format
	format := NewFormat()
	format.SetSource("queries/users.snap.sql", sql)
	format.SetAST(ast)

	// Output JSON
	jsonData, err := format.ToJSONPretty()
	if err != nil {
		t.Fatalf("JSON serialization failed: %v", err)
	}

	fmt.Println("=== Intermediate Format Output ===")
	fmt.Println(string(jsonData))
}
