package parserstep2

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestSubQuery(t *testing.T) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	tests := []struct {
		name      string
		src       string
		wantCount int
		wantErr   error
	}{
		{
			name:      "not subquery",
			src:       `SELECT * FROM users`,
			wantCount: 0,
			wantErr:   pc.ErrNotMatch,
		},
		{
			name:      "subquery with correct parentheses",
			src:       `( SELECT * FROM users)`,
			wantCount: 10,
			wantErr:   nil,
		},
		{
			name:      "missing closing parenthesis",
			src:       `( SELECT * FROM users`,
			wantCount: 0,
			wantErr:   pc.ErrCritical,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.src)
			assert.NoError(t, err)
			pcTokens := tokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true
			consumed, _, err := subQuery(pctx, pcTokens)
			if tt.wantErr != nil {
				assert.IsError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCount, consumed)
		})
	}
}

func TestParseStatementWithAllClauses(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantClauses int
		wantType    cmn.NodeType
	}{
		{
			name:        "all select clauses",
			sql:         `SELECT a, b, c FROM users WHERE a > 1 GROUP BY a, b HAVING COUNT(*) > 1 ORDER BY a DESC, b ASC LIMIT 10 OFFSET 5 RETURNING a, b;`,
			wantClauses: 9, // SELECT, FROM, WHERE, GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET, RETURNING
			wantType:    cmn.SELECT_STATEMENT,
		},
		{
			name:        "select for update",
			sql:         `SELECT id, name FROM users WHERE id = 1 FOR UPDATE;`,
			wantClauses: 4, // SELECT, FROM, WHERE, FOR UPDATE
			wantType:    cmn.SELECT_STATEMENT,
		},
		{
			name:        "select for share",
			sql:         `SELECT id, name FROM users WHERE id = 1 FOR SHARE;`,
			wantClauses: 4, // SELECT, FROM, WHERE, FOR SHARE
			wantType:    cmn.SELECT_STATEMENT,
		},
		{
			name:        "select for no key update",
			sql:         `SELECT id, name FROM users WHERE id = 1 FOR NO KEY UPDATE;`,
			wantClauses: 4, // SELECT, FROM, WHERE, FOR NO KEY UPDATE
			wantType:    cmn.SELECT_STATEMENT,
		},
		{
			name:        "select for key share",
			sql:         `SELECT id, name FROM users WHERE id = 1 FOR KEY SHARE;`,
			wantClauses: 4, // SELECT, FROM, WHERE, FOR KEY SHARE
			wantType:    cmn.SELECT_STATEMENT,
		},
		{
			name:        "select for update nowait",
			sql:         `SELECT id, name FROM users WHERE id = 1 FOR UPDATE NOWAIT;`,
			wantClauses: 4, // SELECT, FROM, WHERE, FOR (FOR UPDATE NOWAIT)
			wantType:    cmn.SELECT_STATEMENT,
		},
		{
			name:        "select for share skip locked",
			sql:         `SELECT id, name FROM users WHERE id = 1 FOR SHARE SKIP LOCKED;`,
			wantClauses: 4, // SELECT, FROM, WHERE, FOR (FOR SHARE SKIP LOCKED)
			wantType:    cmn.SELECT_STATEMENT,
		},
		/*{
			name: "not support: select insert",
			args: args{
				src: `SELECT id, name INTO archived_users FROM users WHERE deleted = 1;`,
			},
			wantType: cmn.SELECT_STATEMENT,
			wantCTEs: 0,
			wantErr:  false,
		},*/
		{
			name:        "all insert clauses",
			sql:         `INSERT INTO users (a, b, c) VALUES (1, 2, 3) WHERE a > 1 ON CONFLICT (a) DO UPDATE SET b = EXCLUDED.b RETURNING a, b;`,
			wantClauses: 5, // INSERT INTO, VALUES, WHERE, ON CONFLICT, RETURNING
			wantType:    cmn.INSERT_INTO_STATEMENT,
		},
		{
			name:        "insert into select",
			sql:         `INSERT INTO users (id, name) SELECT id, name FROM tmp WHERE id > 10 ORDER BY id DESC LIMIT 5 OFFSET 2;`,
			wantClauses: 7, // INSERT INTO, SELECT, FROM, WHERE, ORDER BY, LIMIT, OFFSET
			wantType:    cmn.INSERT_INTO_STATEMENT,
		},
		{
			name:        "all update clauses",
			sql:         `UPDATE users SET name = 'Bob', age = 20 WHERE id = 1 RETURNING id, name;`,
			wantClauses: 4, // UPDATE, SET, WHERE, RETURNING
			wantType:    cmn.UPDATE_STATEMENT,
		},
		{
			name:        "all delete clauses",
			sql:         `DELETE FROM users WHERE id = 1 RETURNING id, name;`,
			wantClauses: 3, // DELETE FROM, WHERE, RETURNING
			wantType:    cmn.DELETE_FROM_STATEMENT,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.sql)
			assert.NoError(t, err)
			pcTokens := tokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			perr := &cmn.ParseError{}

			consumed, got, err := ParseStatement(perr)(pctx, pcTokens)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(perr.Errors), "should have no parse errors")
			assert.Equal(t, len(pcTokens), consumed, "should consume all tokens")
			assert.Equal(t, 1, len(got), "should return exactly one statement")
			stmt, ok := got[0].Val.NewValue.(cmn.StatementNode)
			assert.True(t, ok, "should be StatementNode")
			assert.Equal(t, tt.wantClauses, len(stmt.Clauses()), "should return correct number of clauses")
			assert.Equal(t, tt.wantType, stmt.Type(), "should return correct statement type")
		})
	}
}

func TestClauseSourceText(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		wantSrcTexts []string
		wantClauses  int
		wantType     cmn.NodeType
	}{
		{
			name:         "all select clauses",
			sql:          `select a, b, c from users where a > 1 group BY a, b having COUNT(*) > 1 order BY a DESC, b ASC limit 10 offset 5 returning a, b;`,
			wantSrcTexts: []string{"select", "from", "where", "group BY", "having", "order BY", "limit", "offset", "returning"},
		},
		{
			name:         "all insert clauses",
			sql:          `INSERT INTO users (a, b, c) VALUES (1, 2, 3) WHERE a > 1 ON CONFLICT (a) DO UPDATE SET b = EXCLUDED.b RETURNING a, b;`,
			wantSrcTexts: []string{"INTO", "VALUES", "WHERE", "ON CONFLICT", "RETURNING"},
		},
		{
			name:         "all update clauses",
			sql:          `UPDATE users SET name = 'Bob', age = 20 WHERE id = 1 RETURNING id, name;`,
			wantSrcTexts: []string{"UPDATE", "SET", "WHERE", "RETURNING"},
		},
		{
			name:         "all delete clauses",
			sql:          `DELETE FROM users WHERE id = 1 RETURNING id, name;`,
			wantSrcTexts: []string{"FROM", "WHERE", "RETURNING"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.sql)
			assert.NoError(t, err)
			pcTokens := tokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			perr := &cmn.ParseError{}

			consumed, got, err := ParseStatement(perr)(pctx, pcTokens)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(perr.Errors), "should have no parse errors")
			assert.Equal(t, len(pcTokens), consumed, "should consume all tokens")
			assert.Equal(t, 1, len(got), "should return exactly one statement")
			stmt, ok := got[0].Val.NewValue.(cmn.StatementNode)
			assert.True(t, ok, "should be StatementNode")
			for i, clause := range stmt.Clauses() {
				assert.Equal(t, tt.wantSrcTexts[i], clause.SourceText(), "should return correct source text for clause %d", i)
			}
		})
	}
}

func TestParseStatementWithCTE(t *testing.T) {
	type args struct {
		src string
	}
	// Remove stray closing brace so the rest of the function is inside
	tests := []struct {
		name          string
		args          args
		wantType      cmn.NodeType
		wantCTEs      int
		wantRecursive bool
		wantErr       bool
	}{
		{
			name: "select with CTE",
			args: args{
				src: `WITH tmp AS (SELECT id FROM users) SELECT * FROM tmp;`,
			},
			wantType: cmn.SELECT_STATEMENT,
			wantCTEs: 1,
			wantErr:  false,
		},
		{
			name: "select with multiple CTEs",
			args: args{
				src: `WITH a AS (SELECT 1), b AS (SELECT 2) SELECT * FROM a JOIN b ON a.col = b.col;`,
			},
			wantType: cmn.SELECT_STATEMENT,
			wantCTEs: 2,
			wantErr:  false,
		},
		{
			name: "select with multiple CTEs (3 CTEs)",
			args: args{
				src: `WITH a AS (SELECT 1), b AS (SELECT 2), c AS (SELECT 3) SELECT * FROM a JOIN b ON a.col = b.col JOIN c ON b.col = c.col;`,
			},
			wantType: cmn.SELECT_STATEMENT,
			wantCTEs: 3,
			wantErr:  false,
		},
		{
			name: "select with recursive CTE",
			args: args{
				src: `WITH RECURSIVE tmp AS (SELECT id FROM users) SELECT * FROM tmp;`,
			},
			wantType:      cmn.SELECT_STATEMENT,
			wantCTEs:      1,
			wantRecursive: true,
			wantErr:       false,
		},
		{
			name: "select with CTE and extra comma",
			args: args{
				src: `WITH tmp AS (SELECT id FROM users), SELECT * FROM tmp;`,
			},
			wantType: cmn.SELECT_STATEMENT,
			wantCTEs: 1,
			wantErr:  false,
		},
		{
			name: "insert with CTE",
			args: args{
				src: `WITH tmp AS (SELECT id, name FROM users) INSERT INTO users (id, name) SELECT id, name FROM tmp;`,
			},
			wantType: cmn.INSERT_INTO_STATEMENT,
			wantCTEs: 1,
			wantErr:  false,
		},
		{
			name: "update with CTE",
			args: args{
				src: `WITH tmp AS (SELECT id, name FROM users) UPDATE users SET name = (SELECT name FROM tmp WHERE tmp.id = users.id) WHERE id = 1;`,
			},
			wantType: cmn.UPDATE_STATEMENT,
			wantCTEs: 1,
			wantErr:  false,
		},
		{
			name: "delete with CTE",
			args: args{
				src: `WITH tmp AS (SELECT id FROM users) DELETE FROM users WHERE id IN (SELECT id FROM tmp);`,
			},
			wantType: cmn.DELETE_FROM_STATEMENT,
			wantCTEs: 1,
			wantErr:  false,
		},
		{
			name: "regression test for CTE parsing",
			args: args{
				src: `WITH user_stats AS (
					SELECT department, COUNT(*) as dept_count
					FROM users
					WHERE age >= /*= min_age */18
					GROUP BY department
					)
					SELECT u.name, u.department, s.dept_count
					FROM users u
					JOIN user_stats s ON u.department = s.department`,
			},
			wantType: cmn.SELECT_STATEMENT,
			wantCTEs: 1,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.args.src)
			assert.NoError(t, err)
			pcTokens := tokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			perr := &cmn.ParseError{}
			consumed, got, err := ParseStatement(perr)(pctx, pcTokens)

			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
				assert.Equal(t, 1, len(perr.Errors), "should have one parse error")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
				assert.Equal(t, len(pcTokens), consumed, "ParseStatement() should consume all tokens")
				assert.Equal(t, len(got), 1, "ParseStatement() should return exactly one statement")
				assert.Equal(t, tt.wantType, got[0].Val.NewValue.Type(), "ParseStatement() should return correct node type")
				stmt := got[0].Val.NewValue.(cmn.StatementNode)
				assert.True(t, stmt.CTE() != nil)
				assert.Equal(t, tt.wantRecursive, stmt.CTE().Recursive, "ParseStatement() should return correct recursive flag")
				assert.Equal(t, tt.wantCTEs, len(stmt.CTE().CTEs), "ParseStatement() should return correct number of CTEs")
			}
		})
	}
}

func TestParseStatementWithSubQuery(t *testing.T) {
	type args struct {
		src string
	}
	// Remove stray closing brace so the rest of the function is inside
	tests := []struct {
		name        string
		args        args
		wantType    cmn.NodeType
		wantClauses int
		wantErr     bool
	}{
		{
			name: "select with subquery in select clause",
			args: args{
				src: `SELECT id, (SELECT name FROM users WHERE id = 1) AS name FROM users;`,
			},
			wantType:    cmn.SELECT_STATEMENT,
			wantClauses: 2,
			wantErr:     false,
		},
		{
			name: "select with subquery in from clause",
			args: args{
				src: `SELECT id FROM (SELECT id FROM users) AS sub;`,
			},
			wantType:    cmn.SELECT_STATEMENT,
			wantClauses: 2,
			wantErr:     false,
		},
		{
			name: "select with subquery in where clause",
			args: args{
				src: `SELECT id FROM users WHERE id IN (SELECT id FROM tmp);`,
			},
			wantType:    cmn.SELECT_STATEMENT,
			wantClauses: 3,
			wantErr:     false,
		},
		{
			name: "select with subquery in having clause",
			args: args{
				src: `SELECT id FROM users GROUP BY id HAVING id IN (SELECT id FROM
tmp);`,
			},
			wantType:    cmn.SELECT_STATEMENT,
			wantClauses: 4,
			wantErr:     false,
		},
		{
			name: "update with subquery",
			args: args{
				src: `UPDATE users SET name = (SELECT name FROM tmp WHERE tmp.id = users.id) WHERE id = 1;`,
			},
			wantType:    cmn.UPDATE_STATEMENT,
			wantClauses: 3,
			wantErr:     false,
		},
		{
			name: "delete with subquery",
			args: args{
				src: `DELETE FROM users WHERE id IN (SELECT id FROM tmp);`,
			},
			wantType:    cmn.DELETE_FROM_STATEMENT,
			wantClauses: 2,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.args.src)
			assert.NoError(t, err)
			pcTokens := tokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			perr := &cmn.ParseError{}
			consumed, got, err := ParseStatement(perr)(pctx, pcTokens)

			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
				assert.Equal(t, 0, len(perr.Errors), "should have no parse errors")
				assert.Equal(t, len(pcTokens), consumed, "ParseStatement() should consume all tokens")
				assert.Equal(t, len(got), 1, "ParseStatement() should return exactly one statement")
				assert.Equal(t, tt.wantType, got[0].Val.NewValue.Type(), "ParseStatement() should return correct node type")
				stmt := got[0].Val.NewValue.(cmn.StatementNode)
				assert.Equal(t, tt.wantClauses, len(stmt.Clauses()))
			}
		})
	}
}

func TestExtractIfConditionUnit(t *testing.T) {
	tests := []struct {
		name             string
		prevClauseSQL    string
		currentClauseSQL string
		wantErr          bool
		wantCondition    string
	}{
		{
			name:             "Valid if/end pair",
			prevClauseSQL:    `FROM users /*# if user_id */`,
			currentClauseSQL: `WHERE id = 1 /*# end */`,
			wantCondition:    "user_id",
			wantErr:          false,
		},
		{
			name:             "No if directive",
			prevClauseSQL:    `FROM users`,
			currentClauseSQL: `WHERE id = 1 /*# end */`,
			wantErr:          true,
		},
		{
			name:             "No end directive",
			prevClauseSQL:    `FROM users /*# if user_id */`,
			currentClauseSQL: `WHERE id = 1`,
			wantErr:          true,
		},
		{
			name:             "Both directives missing",
			prevClauseSQL:    `FROM users`,
			currentClauseSQL: `WHERE id = 1`,
			wantErr:          false,
			wantCondition:    ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize previous clause
			prevTokens, err := tok.Tokenize(tt.prevClauseSQL)
			assert.NoError(t, err)
			prevEntityTokens := tokenToEntity(prevTokens)

			// Tokenize current clause
			currentTokens, err := tok.Tokenize(tt.currentClauseSQL)
			assert.NoError(t, err)
			currentEntityTokens := tokenToEntity(currentTokens)

			// Find WHERE clause head and body
			var clauseHead, clauseBody []pc.Token[Entity]
			for i, token := range currentEntityTokens {
				if token.Val.Original.Type == tok.WHERE {
					clauseHead = currentEntityTokens[i : i+1]
					clauseBody = currentEntityTokens[i+1:]
					break
				}
			}

			// Extract previous clause body (everything after FROM)
			var prevClauseBody []pc.Token[Entity]
			for i, token := range prevEntityTokens {
				if token.Val.Original.Type == tok.FROM {
					prevClauseBody = prevEntityTokens[i+1:]
					break
				}
			}

			// Test the function
			condition, _, _, err := detectWrappedIfCondition(clauseHead, clauseBody, prevClauseBody)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Expected no error but got one")
				assert.Equal(t, tt.wantCondition, condition, "Condition mismatch")
			}
		})
	}
}

// TestExtractIfConditionWithFullSQL tests the extractIfCondition function with complete SQL statements
func TestExtractIfConditionWithFullSQL(t *testing.T) {
	tests := []struct {
		name                string
		sql                 string
		expectedIfCondition string
		clauseType          cmn.NodeType
	}{
		{
			name: "WHERE clause with if condition",
			sql: `SELECT id, name, email FROM users 
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */`,
			expectedIfCondition: "filters.active",
			clauseType:          cmn.WHERE_CLAUSE,
		},
		{
			name: "ORDER BY clause with if condition",
			sql: `SELECT id, name FROM users 
/*# if sort_by */
ORDER BY /*= sort_by */name
/*# end */`,
			expectedIfCondition: "sort_by",
			clauseType:          cmn.ORDER_BY_CLAUSE,
		},
		{
			name: "LIMIT clause with if condition",
			sql: `SELECT id, name FROM users 
/*# if page_size */
LIMIT /*= page_size */10
/*# end */`,
			expectedIfCondition: "page_size",
			clauseType:          cmn.LIMIT_CLAUSE,
		},
		{
			name: "OFFSET clause with if condition",
			sql: `SELECT id, name FROM users 
/*# if page_offset */
OFFSET /*= page_offset */0
/*# end */`,
			expectedIfCondition: "page_offset",
			clauseType:          cmn.OFFSET_CLAUSE,
		},
		{
			name: "No conditional clause",
			sql: `SELECT id, name FROM users 
WHERE active = true`,
			expectedIfCondition: "",
			clauseType:          cmn.WHERE_CLAUSE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Tokenize the SQL
			tokens, err := tok.Tokenize(tt.sql)
			assert.NoError(t, err)

			// Parse the SQL
			stmt, err := Execute(tokens)
			assert.NoError(t, err)

			// Find the clause of the specified type
			var foundClause cmn.ClauseNode
			for _, clause := range stmt.Clauses() {
				if clause.Type() == tt.clauseType {
					foundClause = clause
					break
				}
			}

			// Check if the clause has the expected if condition
			if tt.expectedIfCondition == "" {
				if foundClause == nil {
					// If no condition expected and no clause found, that's fine
					return
				}

				// If clause exists, check that it has no condition
				assert.Equal(t, "", foundClause.IfCondition())
			} else {
				// If condition expected, clause must exist
				if foundClause == nil {
					t.Fatalf("Clause of type %s not found", tt.clauseType)
				}

				// Check the condition
				assert.Equal(t, tt.expectedIfCondition, foundClause.IfCondition())

				// Debug output
				t.Logf("Found clause of type %s with condition: %q", tt.clauseType, foundClause.IfCondition())
			}
		})
	}
}

// TestMultipleConditionalClauses tests the handling of multiple conditional clauses in a single SQL statement
func TestMultipleConditionalClauses(t *testing.T) {
	sql := `SELECT id, name FROM users 
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */
/*# if sort_by */
ORDER BY /*= sort_by */name
/*# end */
/*# if page_size */
LIMIT /*= page_size */10
/*# end */`
	// Tokenize the SQL
	tokens, err := tok.Tokenize(sql)
	assert.NoError(t, err)

	// Parse the SQL
	stmt, err := Execute(tokens)
	assert.NoError(t, err)

	// デバッグ情報を出力
	t.Log("Statement clauses:")
	for i, clause := range stmt.Clauses() {
		t.Logf("Clause[%d]: Type=%s", i, clause.Type())
		t.Logf("  Content tokens:")
		for j, token := range clause.ContentTokens() {
			if token.Directive != nil {
				t.Logf("    Token[%d]: Type=%s, Value=%s, Directive.Type=%s",
					j, token.Type, token.Value, token.Directive.Type)
			} else {
				t.Logf("    Token[%d]: Type=%s, Value=%s", j, token.Type, token.Value)
			}
		}
	}

	// Check each clause type and its condition
	clauseChecks := []struct {
		clauseType          cmn.NodeType
		expectedIfCondition string
	}{
		{cmn.WHERE_CLAUSE, "filters.active"},
		{cmn.ORDER_BY_CLAUSE, "sort_by"},
		{cmn.LIMIT_CLAUSE, "page_size"},
	}

	for _, check := range clauseChecks {
		var foundClause cmn.ClauseNode
		for _, clause := range stmt.Clauses() {
			if clause.Type() == check.clauseType {
				foundClause = clause
				break
			}
		}

		if foundClause == nil {
			t.Fatalf("Clause of type %s not found", check.clauseType)
		}

		// Check the condition
		assert.Equal(t, check.expectedIfCondition, foundClause.IfCondition())

		// Debug output
		t.Logf("Found clause of type %s with condition: %q", check.clauseType, foundClause.IfCondition())
	}
}
