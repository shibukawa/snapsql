package parser

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestCaseSensitivity(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "uppercase keywords",
			sql:         "SELECT id, name FROM users WHERE active = true ORDER BY name ASC;",
			expectError: false,
		},
		{
			name:        "lowercase keywords",
			sql:         "select id, name from users where active = true order by name asc;",
			expectError: false,
		},
		{
			name:        "mixed case keywords",
			sql:         "Select id, Name From users Where active = True Order By name Asc;",
			expectError: false,
		},
		{
			name:        "lowercase with GROUP BY and HAVING",
			sql:         "select department, count(*) from users group by department having count(*) > 5;",
			expectError: false,
		},
		{
			name:        "mixed case with LIMIT and OFFSET",
			sql:         "Select id From users Limit 10 Offset 5;",
			expectError: false,
		},
		{
			name:        "lowercase INSERT statement",
			sql:         "insert into users (name, email) values ('John', 'john@example.com');",
			expectError: false,
		},
		{
			name:        "mixed case UPDATE statement",
			sql:         "Update users Set name = 'Jane' Where id = 1;",
			expectError: false,
		},
		{
			name:        "lowercase DELETE statement",
			sql:         "delete from users where active = false;",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Tokenize
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			if test.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.True(t, len(tokens) > 0)

			// Parse
			parser := NewSqlParser(tokens, nil)
			stmt, err := parser.Parse()
			if test.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.True(t, stmt != nil)

			// Verify that the statement was parsed correctly
			switch stmt := stmt.(type) {
			case *SelectStatement:
				assert.True(t, stmt.SelectClause != nil, "SELECT clause should be parsed")
				assert.True(t, stmt.FromClause != nil, "FROM clause should be parsed")
			case *InsertStatement:
				assert.True(t, stmt.Table != nil, "Table should be parsed")
			case *UpdateStatement:
				assert.True(t, stmt.Table != nil, "Table should be parsed")
			case *DeleteStatement:
				assert.True(t, stmt.Table != nil, "Table should be parsed")
			}
		})
	}
}

func TestKeywordCaseInsensitiveComparison(t *testing.T) {
	// Test that the same SQL with different cases produces equivalent AST structures
	upperCaseSQL := "SELECT id, name FROM users WHERE active = true ORDER BY name ASC;"
	lowerCaseSQL := "select id, name from users where active = true order by name asc;"

	// Parse uppercase version
	upperTok := tokenizer.NewSqlTokenizer(upperCaseSQL, tokenizer.NewSQLiteDialect())
	upperTokens, err := upperTok.AllTokens()
	assert.NoError(t, err)
	upperParser := NewSqlParser(upperTokens, nil)
	upperStmt, err := upperParser.Parse()
	assert.NoError(t, err)

	// Parse lowercase version
	lowerTok := tokenizer.NewSqlTokenizer(lowerCaseSQL, tokenizer.NewSQLiteDialect())
	lowerTokens, err := lowerTok.AllTokens()
	assert.NoError(t, err)
	lowerParser := NewSqlParser(lowerTokens, nil)
	lowerStmt, err := lowerParser.Parse()
	assert.NoError(t, err)

	// Both should be SelectStatement
	upperSelect, ok1 := upperStmt.(*SelectStatement)
	lowerSelect, ok2 := lowerStmt.(*SelectStatement)
	assert.True(t, ok1 && ok2, "Both statements should be SelectStatement")

	// Compare basic structure
	assert.Equal(t, len(upperSelect.SelectClause.Fields), len(lowerSelect.SelectClause.Fields))
	assert.Equal(t, len(upperSelect.FromClause.Tables), len(lowerSelect.FromClause.Tables))
	assert.True(t, upperSelect.WhereClause != nil && lowerSelect.WhereClause != nil)
	assert.True(t, upperSelect.OrderByClause != nil && lowerSelect.OrderByClause != nil)
}

func TestComplexCaseSensitivity(t *testing.T) {
	// Test complex SQL with various case combinations
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "all uppercase",
			sql:  "SELECT U.ID, U.NAME FROM USERS U LEFT JOIN POSTS P ON U.ID = P.USER_ID WHERE U.ACTIVE = TRUE ORDER BY U.NAME ASC LIMIT 10;",
		},
		{
			name: "all lowercase",
			sql:  "select u.id, u.name from users u left join posts p on u.id = p.user_id where u.active = true order by u.name asc limit 10;",
		},
		{
			name: "mixed case",
			sql:  "Select u.Id, u.Name From Users u Left Join Posts p On u.Id = p.User_Id Where u.Active = True Order By u.Name Asc Limit 10;",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tok := tokenizer.NewSqlTokenizer(test.sql, tokenizer.NewSQLiteDialect())
			tokens, err := tok.AllTokens()
			assert.NoError(t, err)

			parser := NewSqlParser(tokens, nil)
			stmt, err := parser.Parse()
			assert.NoError(t, err)
			assert.True(t, stmt != nil)

			if selectStmt, ok := stmt.(*SelectStatement); ok {
				assert.True(t, selectStmt.SelectClause != nil)
				assert.True(t, selectStmt.FromClause != nil)
				assert.True(t, selectStmt.WhereClause != nil)
				assert.True(t, selectStmt.OrderByClause != nil)
				assert.True(t, selectStmt.LimitClause != nil)
			}
		})
	}
}
