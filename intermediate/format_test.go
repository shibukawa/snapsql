package intermediate

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestNewFormat(t *testing.T) {
	format := NewFormat()
	assert.NotZero(t, format)
	assert.Zero(t, format.Source.File)
	assert.Zero(t, format.Source.Content)
	assert.Zero(t, format.InterfaceSchema)
}

func TestSetSource(t *testing.T) {
	format := NewFormat()
	format.SetSource("test.sql", "SELECT * FROM users")

	assert.Equal(t, "test.sql", format.Source.File)
	assert.Equal(t, "SELECT * FROM users", format.Source.Content)
}

func TestSetInterfaceSchemaWithNil(t *testing.T) {
	format := NewFormat()
	format.SetInterfaceSchema(nil)

	assert.Zero(t, format.InterfaceSchema)
}

func TestSetInterfaceSchemaWithValidSchema(t *testing.T) {
	format := NewFormat()

	// Create a mock InterfaceSchema
	schema := &parser.InterfaceSchema{
		Name:          "TestQuery",
		Description:   "Test query description",
		FunctionName:  "testQuery",
		OrderedParams: parser.NewOrderedParameters(),
	}

	// Add some parameters (simplified - no path or order)
	schema.OrderedParams.Add("active", "bool")
	schema.OrderedParams.Add("name", "string")

	format.SetInterfaceSchema(schema)

	assert.NotZero(t, format.InterfaceSchema)
	assert.Equal(t, "TestQuery", format.InterfaceSchema.Name)
	assert.Equal(t, "Test query description", format.InterfaceSchema.Description)
	assert.Equal(t, "testQuery", format.InterfaceSchema.FunctionName)
	assert.Equal(t, 2, len(format.InterfaceSchema.Parameters))
}

func TestConvertType(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string type",
			input:    "string",
			expected: "string",
		},
		{
			name:     "bool type",
			input:    "bool",
			expected: "bool",
		},
		{
			name:     "array type",
			input:    []any{"string"},
			expected: "array",
		},
		{
			name:     "empty array type",
			input:    []any{},
			expected: "array",
		},
		{
			name:     "object type",
			input:    map[string]any{"field": "string"},
			expected: "object",
		},
		{
			name:     "unknown type",
			input:    123,
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractChildren(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int // number of children
	}{
		{
			name:     "string type - no children",
			input:    "string",
			expected: 0,
		},
		{
			name:     "array with one element",
			input:    []any{"string"},
			expected: 1,
		},
		{
			name:     "object with two fields",
			input:    map[string]any{"field1": "string", "field2": "bool"},
			expected: 2,
		},
		{
			name:     "nested object",
			input:    map[string]any{"nested": map[string]any{"inner": "string"}},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			children := extractChildren(tt.input)
			assert.Equal(t, tt.expected, len(children))
		})
	}
}

func TestGenerateObjectName(t *testing.T) {
	counter := 1

	name1 := generateObjectName(&counter)
	name2 := generateObjectName(&counter)

	assert.Equal(t, "o1", name1)
	assert.Equal(t, "o2", name2)
	assert.Equal(t, 3, counter)
}

func TestConvertASTNodeWithNil(t *testing.T) {
	result := convertASTNode(nil)
	assert.Equal(t, "", result.Type)
	assert.Equal(t, [3]int{0, 0, 0}, result.Pos)
}

func TestConvertASTNodeWithIdentifier(t *testing.T) {
	// Use a simple SQL to create a real AST node
	sql := "SELECT test_field FROM users"

	// Create tokenizer
	tokenizer := tokenizer.NewSqlTokenizer(sql, tokenizer.NewPostgreSQLDialect())
	tokens, err := tokenizer.AllTokens()
	assert.NoError(t, err)

	// Create parser
	parser := parser.NewSqlParser(tokens, nil)
	ast, err := parser.Parse()
	assert.NoError(t, err)

	result := convertASTNode(ast)

	assert.Equal(t, "SELECT_STATEMENT", result.Type)
	assert.Equal(t, [3]int{1, 1, 0}, result.Pos)

	// Check that select_clause exists
	selectClause, ok := result.Children["select_clause"]
	assert.True(t, ok)
	assert.NotZero(t, selectClause)
}

func TestWriteJSON(t *testing.T) {
	format := NewFormat()
	format.SetSource("test.sql", "SELECT * FROM users")

	// Test compact JSON
	var compactBuf bytes.Buffer
	err := format.WriteJSON(&compactBuf, false)
	assert.NoError(t, err)

	compactJSON := compactBuf.String()
	assert.True(t, len(compactJSON) > 0)
	// Note: json.Encoder always adds a trailing newline

	// Test pretty JSON
	var prettyBuf bytes.Buffer
	err = format.WriteJSON(&prettyBuf, true)
	assert.NoError(t, err)

	prettyJSON := prettyBuf.String()
	assert.True(t, len(prettyJSON) > 0)
	assert.True(t, strings.Contains(prettyJSON, "  ")) // Should have indentation

	// Pretty JSON should be longer due to formatting
	assert.True(t, len(prettyJSON) > len(compactJSON))

	// Verify both contain the same data
	var compactData, prettyData map[string]interface{}
	err = json.Unmarshal([]byte(compactJSON), &compactData)
	assert.NoError(t, err)
	err = json.Unmarshal([]byte(prettyJSON), &prettyData)
	assert.NoError(t, err)

	assert.Equal(t, compactData["source"], prettyData["source"])
}

func TestToJSON(t *testing.T) {
	format := NewFormat()
	format.SetSource("test.sql", "SELECT id FROM users")

	jsonData, err := format.ToJSON()
	assert.NoError(t, err)
	assert.NotZero(t, jsonData)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	assert.NoError(t, err)

	source, ok := result["source"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test.sql", source["file"])
	assert.Equal(t, "SELECT id FROM users", source["content"])
}

func TestWriteJSONUsageExample(t *testing.T) {
	// Create a format with complex data
	format := NewFormat()
	format.SetSource("complex.sql", "SELECT * FROM users WHERE active = /*= active */true")

	// Add interface schema
	schema := &parser.InterfaceSchema{
		Name:          "UserQuery",
		Description:   "Query users with filters",
		FunctionName:  "queryUsers",
		OrderedParams: parser.NewOrderedParameters(),
	}
	schema.OrderedParams.Add("active", "bool")
	schema.OrderedParams.Add("filters", map[string]any{
		"name": "string",
		"age":  "int",
	})
	format.SetInterfaceSchema(schema)

	// Test writing to different outputs
	t.Run("write to stdout simulation", func(t *testing.T) {
		var buf bytes.Buffer
		err := format.WriteJSON(&buf, true)
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "UserQuery")
		assert.Contains(t, output, "queryUsers")
		assert.Contains(t, output, "complex.sql")
	})

	t.Run("write compact for network transmission", func(t *testing.T) {
		var buf bytes.Buffer
		err := format.WriteJSON(&buf, false)
		assert.NoError(t, err)

		compactOutput := buf.String()
		// Compact should be shorter
		var prettyBuf bytes.Buffer
		format.WriteJSON(&prettyBuf, true)
		assert.True(t, len(compactOutput) < len(prettyBuf.String()))
	})
}

func TestToJSONPretty(t *testing.T) {
	format := NewFormat()
	format.SetSource("test.sql", "SELECT id FROM users")

	jsonData, err := format.ToJSONPretty()
	assert.NoError(t, err)
	assert.NotZero(t, jsonData)

	// Pretty JSON should contain newlines and indentation
	jsonString := string(jsonData)
	assert.Contains(t, jsonString, "\n")
	assert.Contains(t, jsonString, "  ")
}

func TestIntegrationWithSimpleSQL(t *testing.T) {
	// Parse a simple SQL statement
	sql := "SELECT id, name FROM users WHERE active = /*= active */true"

	// Create tokenizer
	tokenizer := tokenizer.NewSqlTokenizer(sql, tokenizer.NewPostgreSQLDialect())
	tokens, err := tokenizer.AllTokens()
	assert.NoError(t, err)

	// Create parser
	parser := parser.NewSqlParser(tokens, nil)
	ast, err := parser.Parse()
	assert.NoError(t, err)

	// Create intermediate format
	format := NewFormat()
	format.SetSource("test.sql", sql)
	format.SetAST(ast)

	// Convert to JSON
	jsonData, err := format.ToJSONPretty()
	assert.NoError(t, err)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	assert.NoError(t, err)

	// Check source
	source := result["source"].(map[string]interface{})
	assert.Equal(t, "test.sql", source["file"])
	assert.Equal(t, sql, source["content"].(string))

	// Check AST
	astResult := result["ast"].(map[string]interface{})
	assert.Equal(t, "SELECT_STATEMENT", astResult["type"])

	// Verify position format
	pos := astResult["pos"].([]interface{})
	assert.Equal(t, 3, len(pos))
	assert.Equal(t, 1.0, pos[0]) // line
	assert.Equal(t, 1.0, pos[1]) // column
	assert.Equal(t, 0.0, pos[2]) // offset
}
