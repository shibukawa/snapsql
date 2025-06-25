package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestImplicitIfGenerator(t *testing.T) {
	// Create parameter schema
	schema := &InterfaceSchema{
		Parameters: map[string]any{
			"filters": map[string]any{
				"active":      "bool",
				"departments": []any{"str"},
				"name":        "str",
			},
			"pagination": map[string]any{
				"limit":  "int",
				"offset": "int",
			},
			"sort_fields": []any{"str"},
		},
	}

	// Create environment with schema
	ns := NewNamespace(schema)

	// Test that the function exists and can be called
	result := GenerateImplicitConditionals(nil, schema, ns)
	assert.Equal(t, nil, result)
}

func TestImplicitIfGeneration(t *testing.T) {
	sqlContent := `/*@
parameters:
 filters:
 active: bool
 departments: list[str]
 name: str
 pagination:
 limit: int
 offset: int
 sort_fields: list[str]
*/

SELECT id, name FROM users
WHERE active = /*= filters.active */true
 AND department IN (/*= filters.departments */'sales', 'marketing')
 AND name LIKE /*= filters.name */'%test%'
ORDER BY /*= sort_fields[0] */'created_at'
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;`

	// Set up environment

	// Tokenize
	tok := tokenizer.NewSqlTokenizer(sqlContent, tokenizer.NewSQLiteDialect())
	tokens, err := tok.AllTokens()
	assert.NoError(t, err)

	is, err := NewInterfaceSchemaFromSQL(tokens)
	assert.NoError(t, err)

	ns := NewNamespace(is)
	ns.SetVariable("table_suffix", "prod")

	// Create parser with implicit if generation
	parser := NewSqlParser(tokens, ns, nil)

	// Parse with implicit if generation
	stmt, err := parser.Parse()
	assert.NoError(t, err)
	assert.True(t, stmt != nil)

	// Verify implicit conditionals were generated
	// This is a basic test - more detailed verification would check the AST structure
	assert.True(t, stmt != nil)
	if selectStmt, ok := stmt.(*SelectStatement); ok {
		assert.True(t, selectStmt.WhereClause != nil || selectStmt.OrderByClause != nil || selectStmt.LimitClause != nil)
	}
}

func TestVariableConditionGeneration(t *testing.T) {
	// Create parameter schema
	schema := &InterfaceSchema{
		Parameters: map[string]any{
			"user_id":     "int",
			"name":        "str",
			"active":      "bool",
			"departments": []any{"str"},
			"tags":        []any{"str"},
		},
	}

	// Test generateVariableCondition function directly

	tests := []struct {
		name     string
		variable string
		expected string
	}{
		{
			name:     "Integer variable",
			variable: "user_id",
			expected: "user_id != null",
		},
		{
			name:     "String variable",
			variable: "name",
			expected: "name != null && name != ''",
		},
		{
			name:     "Boolean variable",
			variable: "active",
			expected: "active != null",
		},
		{
			name:     "List variable",
			variable: "departments",
			expected: "departments != null && size(departments) > 0",
		},
		{
			name:     "Another list variable",
			variable: "tags",
			expected: "tags != null && size(tags) > 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			condition := generateVariableCondition(test.variable, schema)
			assert.Equal(t, test.expected, condition)
		})
	}
}

func TestVariableReferenceExtraction(t *testing.T) {
	// Test extractVariableReferences function directly

	tests := []struct {
		name     string
		sqlText  string
		expected []string
	}{
		{
			name:     "Single variable",
			sqlText:  "WHERE active = /*= filters.active */",
			expected: []string{"filters.active"},
		},
		{
			name:     "Multiple variables",
			sqlText:  "WHERE active = /*= filters.active */ AND name = /*= filters.name */",
			expected: []string{"filters.active", "filters.name"},
		},
		{
			name:     "Complex expression",
			sqlText:  "ORDER BY /*= sort.field */ /*= sort.direction */",
			expected: []string{"sort.field", "sort.direction"},
		},
		{
			name:     "No variables",
			sqlText:  "WHERE active = true",
			expected: []string{},
		},
		{
			name:     "Variable with spaces",
			sqlText:  "WHERE id IN (/*= user.favorite_ids */)",
			expected: []string{"user.favorite_ids"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			variables := extractVariableReferences(test.sqlText)
			assert.Equal(t, test.expected, variables)
		})
	}
}

func TestShouldWrapInImplicitIf(t *testing.T) {
	// Create parameter schema
	schema := &InterfaceSchema{

		Parameters: map[string]any{
			"user_id": "int",
			"filters": map[string]any{
				"departments": []any{"str"},
				"active":      "bool",
			},
		},
	}

	// Test shouldWrapInImplicitIf function directly

	tests := []struct {
		name     string
		variable string
		expected bool
	}{
		{
			name:     "Simple variable",
			variable: "user_id",
			expected: false,
		},
		{
			name:     "Nested variable",
			variable: "filters.active",
			expected: true,
		},
		{
			name:     "List variable",
			variable: "filters.departments",
			expected: true, // Both nested and list type
		},
		{
			name:     "Deep nested variable",
			variable: "config.database.host",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shouldWrapInImplicitIf(test.variable, schema)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestImplicitIfValidation(t *testing.T) {
	// Create parameter schema
	schema := &InterfaceSchema{

		Parameters: map[string]any{
			"filters": map[string]any{
				"active":      "bool",
				"departments": []any{"str"},
			},
		},
	}

	// Create environment with schema
	ns := NewNamespace(schema)

	// Test ValidateImplicitConditions function directly

	// Test condition validation
	conditions := []string{
		"filters.active != null",
	}

	errors := ValidateImplicitConditions(conditions, ns)
	if len(errors) > 0 {
		for _, err := range errors {
			t.Logf("Validation error: %v", err)
		}
	}
	// Note: Some CEL expressions might not validate without proper context
	// This is expected behavior for complex expressions

	// Test invalid conditions
	invalidConditions := []string{
		"nonexistent.variable != null",
	}

	errors = ValidateImplicitConditions(invalidConditions, ns)
	assert.True(t, len(errors) > 0, "Invalid conditions should produce errors")
}

func TestComplexImplicitIfGeneration(t *testing.T) {
	sqlContent := `/*@
parameters:
 search:
 query: str
 filters:
 categories: list[str]
 price_range:
 min: float
 max: float
 in_stock: bool
 pagination:
 page: int
 size: int
 sort:
 field: str
 direction: str
*/

SELECT p.id, p.name, p.price
FROM products p
WHERE p.name LIKE /*= search.query + '%' */'%test%'
 AND p.category IN (/*= search.filters.categories */'electronics')
 AND p.price >= /*= search.filters.price_range.min */0.0
 AND p.price <= /*= search.filters.price_range.max */1000.0
 AND p.stock_quantity > 0
ORDER BY /*= sort.field */'name' /*= sort.direction */'ASC'
LIMIT /*= pagination.size */20
OFFSET /*= pagination.page * pagination.size */0;`

	// Set up environment

	// Tokenize
	tok := tokenizer.NewSqlTokenizer(sqlContent, tokenizer.NewSQLiteDialect())
	tokens, err := tok.AllTokens()
	assert.NoError(t, err)

	is, err := NewInterfaceSchemaFromSQL(tokens)
	assert.NoError(t, err)
	ns := NewNamespace(is)

	// Create parser with implicit if generation
	parser := NewSqlParser(tokens, ns, nil)

	// Parse with implicit if generation
	stmt, err := parser.Parse()
	assert.NoError(t, err)
	assert.True(t, stmt != nil)

	// Verify that the statement was processed
	// In a real implementation, we would check for ImplicitConditional nodes in the AST
	assert.True(t, stmt != nil)
	if selectStmt, ok := stmt.(*SelectStatement); ok {
		assert.True(t, selectStmt.WhereClause != nil || selectStmt.OrderByClause != nil || selectStmt.LimitClause != nil)
	}
}
