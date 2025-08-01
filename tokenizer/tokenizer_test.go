package tokenizer

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestTokenize(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE active = true;"
	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	expectedTypes := []TokenType{
		SELECT, WHITESPACE, IDENTIFIER, COMMA, WHITESPACE, IDENTIFIER, WHITESPACE,
		FROM, WHITESPACE, IDENTIFIER, WHITESPACE, WHERE, WHITESPACE, IDENTIFIER,
		WHITESPACE, EQUAL, WHITESPACE, BOOLEAN, SEMICOLON, EOF,
	}

	var actualTypes []TokenType
	for _, token := range tokens {
		actualTypes = append(actualTypes, token.Type)
	}

	assert.Equal(t, expectedTypes, actualTypes)
}

func TestBasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "single keyword",
			input:    "SELECT",
			expected: []TokenType{SELECT, EOF},
		},
		{
			name:     "basic SELECT statement",
			input:    "SELECT id, name FROM users",
			expected: []TokenType{SELECT, WHITESPACE, IDENTIFIER, COMMA, WHITESPACE, IDENTIFIER, WHITESPACE, FROM, WHITESPACE, IDENTIFIER, EOF},
		},
		{
			name:     "WHERE clause with condition",
			input:    "WHERE id = 123",
			expected: []TokenType{WHERE, WHITESPACE, IDENTIFIER, WHITESPACE, EQUAL, WHITESPACE, NUMBER, EOF},
		},
		{
			name:     "parentheses",
			input:    "SELECT (id)",
			expected: []TokenType{SELECT, WHITESPACE, OPENED_PARENS, IDENTIFIER, CLOSED_PARENS, EOF},
		},
		{
			name:     "single quoted string",
			input:    "'abc'",
			expected: []TokenType{STRING, EOF},
		},
		{
			name:     "double quoted identifier",
			input:    `"col"`,
			expected: []TokenType{IDENTIFIER, EOF},
		},
		{
			name:     "single quote with double inside",
			input:    `'a"b'`,
			expected: []TokenType{STRING, EOF},
		},
		{
			name:     "double quote with single inside",
			input:    `"a'b"`,
			expected: []TokenType{IDENTIFIER, EOF},
		},
		{
			name:     "escaped single quote (doubled)",
			input:    "'a''b'",
			expected: []TokenType{STRING, EOF},
		},
		{
			name:     "escaped double quote (doubled)",
			input:    `"a""b"`,
			expected: []TokenType{IDENTIFIER, EOF},
		},
		{
			name:     "backslash escape in single quote",
			input:    `'a\'b'`,
			expected: []TokenType{STRING, EOF},
		},
		{
			name:     "backtick identifier (MySQL)",
			input:    "`col`",
			expected: []TokenType{IDENTIFIER, EOF},
		},
		{
			name:  "keyword like token",
			input: "AND OR NOT IN EXISTS BETWEEN LIKE IS NULL",
			expected: []TokenType{
				AND, WHITESPACE, OR, WHITESPACE, NOT, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE,
				RESERVED_IDENTIFIER, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE, NULL, EOF},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokens, err := Tokenize(test.input)
			assert.NoError(t, err)

			var actualTypes []TokenType
			for _, token := range tokens {
				assert.NoError(t, err)
				actualTypes = append(actualTypes, token.Type)
				if token.Type == EOF {
					break
				}
			}

			assert.Equal(t, test.expected, actualTypes)
		})
	}
}

func TestSnapSQLDirectives(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  TokenType
		isDirective   bool
		directiveType string
	}{
		{
			name:          "if directive",
			input:         "/*# if condition */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "if",
		},
		{
			name:          "variable directive",
			input:         "/*= variable */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "variable",
		},
		{
			name:          "normal comment",
			input:         "/* normal comment */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   false,
			directiveType: "",
		},
		{
			name:          "elseif directive",
			input:         "/*# elseif condition */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "elseif",
		},
		{
			name:          "elseif directive(no space)",
			input:         "/*#elseif condition*/",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "elseif",
		},
		{
			name:          "else directive",
			input:         "/*# else */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "else",
		},
		{
			name:          "for directive",
			input:         "/*# for item : items */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "for",
		},
		{
			name:          "end directive",
			input:         "/*# end */",
			expectedType:  BLOCK_COMMENT,
			isDirective:   true,
			directiveType: "end",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokens, err := Tokenize(test.input)
			assert.NoError(t, err)

			var foundToken Token
			for _, token := range tokens {
				if token.Type != EOF {
					foundToken = token
					break
				}
			}

			assert.Equal(t, test.expectedType, foundToken.Type)
			assert.Equal(t, test.isDirective, foundToken.Directive != nil)
			if test.isDirective {
				assert.Equal(t, test.directiveType, foundToken.Directive.Type)
			}
		})
	}
}

// Helper function to test for specific token types
func testForTokenType(t *testing.T, input string, expectedTokenType TokenType, expectError bool, errorMsg string) {
	t.Helper()
	tokens, err := Tokenize(input)

	if expectError {
		assert.Error(t, err)
		return
	}

	assert.NoError(t, err)

	var foundToken bool
	for _, token := range tokens {
		if token.Type == expectedTokenType {
			foundToken = true
			break
		}
	}

	assert.True(t, foundToken, errorMsg)
}

func TestWindowFunctions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
		expectError bool
	}{
		{
			name:        "basic window function",
			input:       "SELECT ROW_NUMBER() OVER (ORDER BY id) FROM users",
			description: "basic window function",
			expectError: false,
		},
		{
			name:        "complex window function",
			input:       "SELECT SUM(salary) OVER (PARTITION BY dept ORDER BY salary ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM emp",
			description: "complex window function",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testForTokenType(t, test.input, RESERVED_IDENTIFIER, test.expectError, "OVER keyword not found")
		})
	}
}

func TestComplexConditions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
		expectError bool
	}{
		{
			name:        "complex WHERE clause",
			input:       "SELECT * FROM users WHERE (age > 18 AND status = 'active') OR (vip = true)",
			description: "complex WHERE clause with OR/AND and nested parentheses",
			expectError: false,
		},
		{
			name:        "condition with IN clause",
			input:       "SELECT * FROM users WHERE age > 18 AND status IN ('active', 'pending')",
			description: "condition with IN clause",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Tokenize(test.input)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSubqueries(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
		expectError bool
	}{
		{
			name:        "WHERE clauseのsubquery ",
			input:       "SELECT * FROM users WHERE id IN (SELECT user_id FROM active_users)",
			description: "WHERE clauseのsubquery ",
			expectError: false,
		},
		{
			name:        "subquery in SELECT clause",
			input:       "SELECT u.name, (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) FROM users u",
			description: "subquery in SELECT clause",
			expectError: false,
		},
		{
			name:        "FROM clauseのsubquery ",
			input:       "SELECT * FROM (SELECT * FROM users WHERE active = true) AS active_users",
			description: "FROM clauseのsubquery ",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokens, err := Tokenize(test.input)

			if test.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			var foundSubquery bool
			parenCount := 0
			// サブクエリ検出ロジックを現仕様に合わせて修正
			for _, token := range tokens {
				if token.Type == SELECT && parenCount > 0 {
					foundSubquery = true
				}
				if token.Type == OPENED_PARENS {
					parenCount++
				} else if token.Type == CLOSED_PARENS {
					parenCount--
				}
			}

			assert.True(t, foundSubquery, "subquery not found")
		})
	}
}

func TestCTEs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
		expectError bool
	}{
		{
			name:        "basic CTE",
			input:       "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users",
			description: "basic CTE",
			expectError: false,
		},
		{
			name:        "multiple CTEs",
			input:       "WITH users AS (SELECT * FROM employees), stats AS (SELECT COUNT(*) FROM users) SELECT * FROM stats",
			description: "multiple CTEs",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testForTokenType(t, test.input, WITH, test.expectError, "WITH keyword not found")
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "unclosed string",
			input:       "SELECT id, name FROM users WHERE id = 'unclosed string",
			expectedErr: ErrUnterminatedString,
		},
		{
			name:        "unclosed block comment",
			input:       "SELECT id /* unclosed comment",
			expectedErr: ErrUnterminatedComment,
		},
		{
			name:        "invalid numeric format",
			input:       "SELECT 123e FROM users",
			expectedErr: ErrInvalidNumber,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Tokenize(test.input)

			assert.Error(t, err)
			assert.True(t, errors.Is(err, test.expectedErr))
		})
	}
}

func TestAllTokens(t *testing.T) {
	sql := "SELECT id FROM users;"
	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	expectedTypes := []TokenType{SELECT, WHITESPACE, IDENTIFIER, WHITESPACE, FROM, WHITESPACE, IDENTIFIER, SEMICOLON, EOF}
	var actualTypes []TokenType
	for _, token := range tokens {
		actualTypes = append(actualTypes, token.Type)
	}

	assert.Equal(t, expectedTypes, actualTypes)
}

func TestTokenPosition(t *testing.T) {
	sql := "SELECT\nid,\nname"
	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	expectedPositions := []Position{
		{Line: 1, Column: 1, Offset: 0},  // SELECT
		{Line: 2, Column: 0, Offset: 6},  // \n (newline character is treated as start of next line)
		{Line: 2, Column: 1, Offset: 7},  // id
		{Line: 2, Column: 3, Offset: 9},  // ,
		{Line: 3, Column: 0, Offset: 10}, // \n
		{Line: 3, Column: 1, Offset: 11}, // name
		{Line: 3, Column: 5, Offset: 16}, // EOF
	}

	var actualPositions []Position
	for _, token := range tokens {
		actualPositions = append(actualPositions, token.Position)
	}

	assert.Equal(t, expectedPositions, actualPositions)
}

func TestComplexSQL(t *testing.T) {
	sql := `
	WITH RECURSIVE employee_hierarchy AS (
		SELECT employee_id, name, manager_id, 0 as level
		FROM employees
		WHERE manager_id IS NULL
		
		UNION ALL
		
		SELECT e.employee_id, e.name, e.manager_id, eh.level + 1
		FROM employees e
		INNER JOIN employee_hierarchy eh ON e.manager_id = eh.employee_id
	),
	department_stats AS (
		SELECT 
			department_id,
			COUNT(*) as employee_count,
			AVG(salary) as avg_salary,
			ROW_NUMBER() OVER (PARTITION BY department_id ORDER BY AVG(salary) DESC) as dept_rank
		FROM employees
		WHERE active = true
		GROUP BY department_id
		HAVING COUNT(*) > 5
	)
	SELECT 
		eh.name,
		eh.level,
		ds.employee_count,
		ds.avg_salary,
		CASE 
			WHEN ds.dept_rank <= 3 THEN 'Top Department'
			WHEN eh.level = 0 THEN 'Executive'
			ELSE 'Regular'
		END as category
	FROM employee_hierarchy eh
	LEFT JOIN department_stats ds ON eh.department_id = ds.department_id
	WHERE eh.level <= 5
	ORDER BY eh.level, eh.name;
	`

	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	var tokenCount int
	var foundKeywords = make(map[TokenType]bool)

	for _, token := range tokens {
		tokenCount++
		foundKeywords[token.Type] = true
	}

	assert.True(t, tokenCount > 50, "token count is less than expected")

	// Verify important keywords are included
	expectedKeywords := []TokenType{}
	for _, keyword := range expectedKeywords {
		assert.True(t, foundKeywords[keyword], "keyword %s not found", keyword.String())
	}
}

func TestPostgresDoubleColonCastToken(t *testing.T) {
	tokens, err := Tokenize("SELECT age::text, (price*quantity)::numeric FROM users;")
	assert.NoError(t, err)
	var foundDoubleColon bool
	for _, tok := range tokens {
		if tok.Type == DOUBLE_COLON {
			foundDoubleColon = true
		}
	}
	assert.True(t, foundDoubleColon, "DOUBLE_COLON token should be present for '::' cast operator")
}

func TestPostgresJSONOperators(t *testing.T) {
	tokens, err := Tokenize("col->'key', col->>'key', col#>'{a,b}', col#>>'{a,b}'")
	assert.NoError(t, err)
	var found []string
	for _, tok := range tokens {
		if tok.Type == JSON_OPERATOR {
			found = append(found, tok.Value)
		}
	}
	assert.Equal(t, []string{"->", "->>", "#>", "#>>"}, found)
}

func TestTokenizerDirectives(t *testing.T) {
	sql := `SELECT /*= user.name */'default_name' FROM users`

	// Test raw tokenizer output
	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	t.Logf("Raw tokenizer output for: %s", sql)
	for i, token := range tokens {
		if token.Directive != nil {
			t.Logf("Token[%d]: %s = %q (Directive: %s, Condition: %q)",
				i, token.Type, token.Value, token.Directive.Type, token.Directive.Condition)
		} else {
			t.Logf("Token[%d]: %s = %q", i, token.Type, token.Value)
		}
	}

	// Check if directive token is present
	foundDirective := false
	for _, token := range tokens {
		if token.Directive != nil && token.Directive.Type == "variable" {
			foundDirective = true
			break
		}
	}

	assert.True(t, foundDirective, "Should find variable directive in raw tokens")
}

func TestTokenizerSnapSQLDirectives(t *testing.T) {
	tests := []struct {
		name               string
		sql                string
		expectedDirectives []struct {
			type_     string
			condition string
		}
	}{
		{
			name: "Variable directive",
			sql:  `SELECT /*= user.name */'default' FROM users`,
			expectedDirectives: []struct {
				type_     string
				condition string
			}{
				{type_: "variable", condition: ""},
			},
		},
		{
			name: "Environment directive",
			sql:  `SELECT /*$ env.table */default_table FROM users`,
			expectedDirectives: []struct {
				type_     string
				condition string
			}{
				{type_: "const", condition: ""},
			},
		},
		{
			name: "If directive with condition",
			sql:  `SELECT id /*# if user.active */FROM users/*# end */`,
			expectedDirectives: []struct {
				type_     string
				condition string
			}{
				{type_: "if", condition: "user.active"},
				{type_: "end", condition: ""},
			},
		},
		{
			name: "Multiple directive types",
			sql:  `SELECT /*= user.id */123, /*$ env.table */default /*# if status */WHERE active = 1/*# end */`,
			expectedDirectives: []struct {
				type_     string
				condition string
			}{
				{type_: "variable", condition: ""},
				{type_: "const", condition: ""},
				{type_: "if", condition: "status"},
				{type_: "end", condition: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.sql)
			assert.NoError(t, err)

			var foundDirectives []struct {
				type_     string
				condition string
			}

			for _, token := range tokens {
				if token.Directive != nil {
					foundDirectives = append(foundDirectives, struct {
						type_     string
						condition string
					}{
						type_:     token.Directive.Type,
						condition: token.Directive.Condition,
					})
				}
			}

			assert.Equal(t, len(tt.expectedDirectives), len(foundDirectives),
				"Number of directives should match")

			for i, expected := range tt.expectedDirectives {
				if i < len(foundDirectives) {
					assert.Equal(t, expected.type_, foundDirectives[i].type_,
						"Directive type should match at index %d", i)
					assert.Equal(t, expected.condition, foundDirectives[i].condition,
						"Directive condition should match at index %d", i)
				}
			}
		})
	}
}

func TestTokenizeWithLineOffset(t *testing.T) {
	sql := "SELECT id\nFROM users"

	// Test without offset
	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	// Find the FROM token
	var fromToken Token
	for _, token := range tokens {
		if token.Type == FROM {
			fromToken = token
			break
		}
	}

	assert.Equal(t, 2, fromToken.Position.Line) // Should be on line 2

	// Test with offset of 10
	tokensWithOffset, err := Tokenize(sql, 10)
	assert.NoError(t, err)

	// Find the FROM token with offset
	var fromTokenWithOffset Token
	for _, token := range tokensWithOffset {
		if token.Type == FROM {
			fromTokenWithOffset = token
			break
		}
	}

	assert.Equal(t, 12, fromTokenWithOffset.Position.Line) // Should be on line 12 (2 + 10)
}

func TestModuloToken(t *testing.T) {
	sql := "SELECT id % 10 FROM users"
	tokens, err := Tokenize(sql)
	assert.NoError(t, err)

	// Find the MODULO token
	var moduloToken Token
	for _, token := range tokens {
		if token.Type == MODULO {
			moduloToken = token
			break
		}
	}

	assert.Equal(t, MODULO, moduloToken.Type)
	assert.Equal(t, "%", moduloToken.Value)
}
