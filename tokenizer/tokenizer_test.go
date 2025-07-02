package tokenizer

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestTokenIterator(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE active = true;"
	tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())

	expectedTypes := []TokenType{
		RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, COMMA, WHITESPACE, IDENTIFIER, WHITESPACE,
		RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER,
		WHITESPACE, EQUAL, WHITESPACE, IDENTIFIER, SEMICOLON, EOF,
	}

	var actualTypes []TokenType
	for token, err := range tokenizer.Tokens() {
		assert.NoError(t, err)

		actualTypes = append(actualTypes, token.Type)

		if token.Type == EOF {
			break
		}
	}

	assert.Equal(t, expectedTypes, actualTypes)
}

func TestTokenIteratorWithOptions(t *testing.T) {
	sql := "SELECT id, name FROM users -- comment\nWHERE active = true;"
	tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect(), TokenizerOptions{
		SkipWhitespace: true,
		SkipComments:   true,
	})

	expectedTypes := []TokenType{
		RESERVED_IDENTIFIER, IDENTIFIER, COMMA, IDENTIFIER, RESERVED_IDENTIFIER, IDENTIFIER, RESERVED_IDENTIFIER, IDENTIFIER, EQUAL, IDENTIFIER, SEMICOLON, EOF,
	}

	var actualTypes []TokenType
	for token, err := range tokenizer.Tokens() {
		assert.NoError(t, err)

		actualTypes = append(actualTypes, token.Type)

		if token.Type == EOF {
			break
		}
	}

	assert.Equal(t, expectedTypes, actualTypes)
}

func TestIteratorEarlyTermination(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE active = true;"
	tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())

	count := 0
	for _, err := range tokenizer.Tokens() {
		assert.NoError(t, err)

		count++

		// 5つ目のトークンで終了
		if count >= 5 {
			break
		}
	}

	assert.Equal(t, 5, count)
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
			expected: []TokenType{RESERVED_IDENTIFIER, EOF},
		},
		{
			name:     "basic SELECT statement",
			input:    "SELECT id, name FROM users",
			expected: []TokenType{RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, COMMA, WHITESPACE, IDENTIFIER, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, EOF},
		},
		{
			name:     "WHERE clause with condition",
			input:    "WHERE id = 123",
			expected: []TokenType{RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, WHITESPACE, EQUAL, WHITESPACE, NUMBER, EOF},
		},
		{
			name:     "parentheses",
			input:    "SELECT (id)",
			expected: []TokenType{RESERVED_IDENTIFIER, WHITESPACE, OPENED_PARENS, IDENTIFIER, CLOSED_PARENS, EOF},
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
				AND, WHITESPACE, OR, WHITESPACE, NOT, WHITESPACE, IN, WHITESPACE, EXISTS, WHITESPACE,
				BETWEEN, WHITESPACE, LIKE, WHITESPACE, IS, WHITESPACE, NULL, EOF},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokenizer := NewSqlTokenizer(test.input, NewSQLiteDialect())

			var actualTypes []TokenType
			for token, err := range tokenizer.Tokens() {
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
			tokenizer := NewSqlTokenizer(test.input, NewSQLiteDialect())

			var foundToken Token
			for token, err := range tokenizer.Tokens() {
				assert.NoError(t, err)
				if token.Type != EOF {
					foundToken = token
					break
				}
			}

			assert.Equal(t, test.expectedType, foundToken.Type)
			assert.Equal(t, test.isDirective, foundToken.IsSnapSQLDirective)
			assert.Equal(t, test.directiveType, foundToken.DirectiveType)
		})
	}
}

// Helper function to test for specific token types
func testForTokenType(t *testing.T, input string, expectedTokenType TokenType, expectError bool, errorMsg string) {
	t.Helper()
	tokenizer := NewSqlTokenizer(input, NewSQLiteDialect())

	var hasError bool
	var foundToken bool
	for token, err := range tokenizer.Tokens() {
		if err != nil {
			hasError = true
			break
		}
		if token.Type == expectedTokenType {
			foundToken = true
		}
		if token.Type == EOF {
			break
		}
	}

	assert.Equal(t, expectError, hasError)
	if !expectError {
		assert.True(t, foundToken, errorMsg)
	}
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
			tokenizer := NewSqlTokenizer(test.input, NewSQLiteDialect())

			var hasError bool
			for _, err := range tokenizer.Tokens() {
				if err != nil {
					hasError = true
					break
				}
			}

			assert.Equal(t, test.expectError, hasError)
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
			tokenizer := NewSqlTokenizer(test.input, NewSQLiteDialect())

			var hasError bool
			var foundSubquery bool
			parenCount := 0
			for token, err := range tokenizer.Tokens() {
				if err != nil {
					hasError = true
					break
				}
				if token.Type == OPENED_PARENS {
					parenCount++
				} else if token.Type == CLOSED_PARENS {
					parenCount--
				} else if token.Type == RESERVED_IDENTIFIER && parenCount > 0 {
					foundSubquery = true
				}
				if token.Type == EOF {
					break
				}
			}

			assert.Equal(t, test.expectError, hasError)
			if !test.expectError {
				assert.True(t, foundSubquery, "subquery not found")
			}
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
			testForTokenType(t, test.input, RESERVED_IDENTIFIER, test.expectError, "WITH keyword not found")
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
			tokenizer := NewSqlTokenizer(test.input, NewSQLiteDialect())

			var foundError error
			for _, err := range tokenizer.Tokens() {
				if err != nil {
					foundError = err
					break
				}
			}

			assert.Error(t, foundError)
			assert.True(t, errors.Is(foundError, test.expectedErr))
		})
	}
}

func TestAllTokens(t *testing.T) {
	sql := "SELECT id FROM users;"
	tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())

	tokens, err := tokenizer.AllTokens()
	assert.NoError(t, err)

	expectedTypes := []TokenType{RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, WHITESPACE, RESERVED_IDENTIFIER, WHITESPACE, IDENTIFIER, SEMICOLON, EOF}
	var actualTypes []TokenType
	for _, token := range tokens {
		actualTypes = append(actualTypes, token.Type)
	}

	assert.Equal(t, expectedTypes, actualTypes)
}

func TestTokenPosition(t *testing.T) {
	sql := "SELECT\nid,\nname"
	tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())

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
	for token, err := range tokenizer.Tokens() {
		assert.NoError(t, err)
		actualPositions = append(actualPositions, token.Position)
		if token.Type == EOF {
			break
		}
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

	tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())

	var tokenCount int
	var hasError bool
	var foundKeywords = make(map[TokenType]bool)

	for token, err := range tokenizer.Tokens() {
		if err != nil {
			hasError = true
			break
		}

		tokenCount++
		foundKeywords[token.Type] = true

		if token.Type == EOF {
			break
		}
	}

	assert.False(t, hasError, "error occurred while parsing complex SQL")
	assert.True(t, tokenCount > 50, "token count is less than expected")

	// Verify important keywords are included
	expectedKeywords := []TokenType{}
	for _, keyword := range expectedKeywords {
		assert.True(t, foundKeywords[keyword], "keyword %s not found", keyword.String())
	}
}
