package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestClauseValidation(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid if statement within SELECT clause",
			sql: `SELECT 
				id,
				name,
				/*# if include_email */
					email,
				/*# end */
			FROM users;`,
			expectError: false,
		},
		{
			name: "valid for statement within WHERE clause",
			sql: `SELECT id FROM users 
			WHERE active = true
				/*# for filter : filters */
				AND /*= filter.field */ = /*= filter.value */
				/*# end */;`,
			expectError: false,
		},
		{
			name: "valid for statement within ORDER BY clause",
			sql: `SELECT id FROM users 
			ORDER BY 
				/*# for sort : sort_fields */
					/*= sort.field */ /*= sort.direction */,
				/*# end */
				created_at ASC;`,
			expectError: false,
		},
		{
			name: "if statement across clauses (SELECT→FROM)",
			sql: `SELECT 
				id,
				/*# if include_table_info */
					name
			FROM /*= table_name */
				/*# else */
					'default' as name
			FROM default_table
				/*# end */;`,
			expectError: true,
			errorMsg:    "SnapSQL directive spans multiple SQL clauses",
		},
		{
			name: "for statement across clauses (FROM→WHERE)",
			sql: `SELECT id 
			FROM 
				/*# for table : tables */
					/*= table.name */ /*= table.alias */
			WHERE /*= table.condition */
				/*# end */;`,
			expectError: true,
			errorMsg:    "SnapSQL directive spans multiple SQL clauses",
		},
		{
			name: "if statement across clauses (WHERE→ORDER BY)",
			sql: `SELECT id FROM users 
			WHERE 
				/*# if has_filter */
					active = true
			ORDER BY 
					priority DESC
				/*# else */
					1 = 1
			ORDER BY 
					created_at ASC
				/*# end */;`,
			expectError: true,
			errorMsg:    "SnapSQL directive spans multiple SQL clauses",
		},
		{
			name: "nested if statements within same clause",
			sql: `SELECT 
				id,
				/*# if include_contact */
					/*# if include_email */
						email,
					/*# else */
						phone,
					/*# end */
				/*# end */
			FROM users;`,
			expectError: false,
		},
		{
			name: "nested for statements within same clause",
			sql: `SELECT 
				/*# for category : categories */
					/*# for field : category.fields */
						/*= field */,
					/*# end */
				/*# end */
			FROM users;`,
			expectError: false,
		},
		{
			name: "multiple independent if statements (different clauses)",
			sql: `SELECT 
				id,
				/*# if include_name */
					name,
				/*# end */
			FROM users
			WHERE active = true
				/*# if include_filter */
				AND department = /*= department */
				/*# end */;`,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Tokenize
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			// Validate clause constraints
			// validator := NewClauseValidator(tokens)
			errors := ValidateDirectiveClauseConstraints(tokens)

			if test.expectError {
				assert.True(t, len(errors) > 0, "Expected validation errors but got none")
				if len(errors) > 0 && test.errorMsg != "" {
					found := false
					for _, err := range errors {
						if err.Message != "" && len(err.Message) > 0 {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error message not found")
				}
			} else {
				if len(errors) > 0 {
					t.Logf("Unexpected validation errors:")
					for _, err := range errors {
						t.Logf("- %s", err.Message)
					}
				}
				assert.Equal(t, 0, len(errors), "Expected no validation errors")
			}
		})
	}
}

func TestClauseDetection(t *testing.T) {
	sql := `SELECT id, name 
		FROM users u 
		LEFT JOIN posts p ON u.id = p.user_id
		WHERE u.active = true 
		GROUP BY u.department 
		HAVING COUNT(*) > 5
		ORDER BY u.name ASC 
		LIMIT 10 
		OFFSET 5;`

	tok := tokenizer.NewSqlTokenizer(sql, tokenizer.NewSQLiteDialect())
	tokens, err := tok.AllTokens()
	assert.NoError(t, err)

	clauseMap := buildClauseMap(tokens)

	// Verify that major keyword clauses are correctly detected
	selectFound := false
	fromFound := false
	whereFound := false
	groupByFound := false
	havingFound := false
	orderByFound := false
	limitFound := false
	offsetFound := false

	for i, token := range tokens {
		clause := clauseMap[i]
		switch token.Type {
		case tokenizer.SELECT:
			assert.Equal(t, SELECT_CLAUSE_SECTION, clause)
			selectFound = true
		case tokenizer.FROM:
			assert.Equal(t, FROM_CLAUSE_SECTION, clause)
			fromFound = true
		case tokenizer.WHERE:
			assert.Equal(t, WHERE_CLAUSE_SECTION, clause)
			whereFound = true
		case tokenizer.GROUP:
			assert.Equal(t, GROUP_BY_CLAUSE_SECTION, clause)
			groupByFound = true
		case tokenizer.HAVING:
			assert.Equal(t, HAVING_CLAUSE_SECTION, clause)
			havingFound = true
		case tokenizer.ORDER:
			assert.Equal(t, ORDER_BY_CLAUSE_SECTION, clause)
			orderByFound = true
		case tokenizer.WORD:
			switch token.Value {
			case "LIMIT":
				assert.Equal(t, LIMIT_CLAUSE_SECTION, clause)
				limitFound = true
			case "OFFSET":
				assert.Equal(t, OFFSET_CLAUSE_SECTION, clause)
				offsetFound = true
			}
		}
	}

	// Verify that all major clauses were detected
	assert.True(t, selectFound, "SELECT clause not detected")
	assert.True(t, fromFound, "FROM clause not detected")
	assert.True(t, whereFound, "WHERE clause not detected")
	assert.True(t, groupByFound, "GROUP BY clause not detected")
	assert.True(t, havingFound, "HAVING clause not detected")
	assert.True(t, orderByFound, "ORDER BY clause not detected")
	assert.True(t, limitFound, "LIMIT clause not detected")
	assert.True(t, offsetFound, "OFFSET clause not detected")
}

func TestIntegratedClauseValidation(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name: "valid SnapSQL template",
			sql: `/*@
parameters:
 include_email: bool
 filters:
 active: bool
 departments: list[str]
 sort_fields:
 - field: str
 direction: str
*/

SELECT 
				id,
				name,
				/*# if include_email */
					email,
				/*# end */
			FROM users_/*@ table_suffix */
			WHERE active = /*= filters.active */
				/*# if filters.departments */
				AND department IN (/*= filters.departments */)
				/*# end */
			ORDER BY 
				/*# for sort : sort_fields */
					/*= sort.field */ /*= sort.direction */,
				/*# end */
				created_at ASC;`,
			expectError: false,
		},
		{
			name: "SnapSQL template with clause constraint violation",
			sql: `SELECT 
				id,
				/*# if include_complex */
					name
			FROM users
			WHERE active = true
				/*# end */;`,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up environment information

			// Tokenize SQL first
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err, "Failed to tokenize SQL")

			is, err := NewInterfaceSchemaFromSQL(tokens)
			assert.NoError(t, err, "Failed to parse interface schema")

			ns := NewNamespace(is)
			ns.SetVariable("table_suffix", "test")

			// Integration validation with parameter schema parser
			parser := NewSqlParser(tokens, ns, nil)

			_, err = parser.Parse()

			if test.expectError {
				assert.Error(t, err, "Expected parsing error due to clause constraint violation")
			} else {
				assert.NoError(t, err, "Expected no parsing error")
			}
		})
	}
}
