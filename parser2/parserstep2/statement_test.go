package parserstep2

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func TestSubQuery(t *testing.T) {
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
			tz := tokenizer.NewSqlTokenizer(tt.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true
			consumed, _, err := subQuery()(pctx, pcTokens)
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
			name:        "insert into select",
			sql:         `INSERT INTO users (id, name) SELECT id, name FROM tmp WHERE id > 10 ORDER BY id DESC LIMIT 5 OFFSET 2;`,
			wantClauses: 6, // INSERT INTO, SELECT, WHERE, ORDER BY, LIMIT, OFFSET
			wantType:    cmn.INSERT_INTO_STATEMENT,
		},
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
			tz := tokenizer.NewSqlTokenizer(tt.sql)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			consumed, got, err := ParseStatement()(pctx, pcTokens)
			assert.NoError(t, err)
			assert.Equal(t, len(pcTokens), consumed, "should consume all tokens")
			assert.Equal(t, 1, len(got), "should return exactly one statement")
			stmt, ok := got[0].Val.NewValue.(cmn.StatementNode)
			assert.True(t, ok, "should be StatementNode")
			assert.Equal(t, tt.wantClauses, len(stmt.Clauses()), "should return correct number of clauses")
			assert.Equal(t, tt.wantType, stmt.Type(), "should return correct statement type")
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tz := tokenizer.NewSqlTokenizer(tt.args.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			consumed, got, err := ParseStatement()(pctx, pcTokens)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, len(pcTokens), consumed, "ParseGroup() should consume all tokens")
			assert.Equal(t, len(got), 1, "ParseGroup() should return exactly one statement")
			assert.Equal(t, tt.wantType, got[0].Val.NewValue.Type(), "ParseGroup() should return correct node type")
			stmt := got[0].Val.NewValue.(cmn.StatementNode)
			assert.True(t, stmt.CTE() != nil)
			assert.Equal(t, tt.wantRecursive, stmt.CTE().Recursive, "ParseGroup() should return correct recursive flag")
			assert.Equal(t, tt.wantCTEs, len(stmt.CTE().CTEs), "ParseGroup() should return correct number of CTEs")
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
			name: "select with subquery",
			args: args{
				src: `SELECT id FROM (SELECT id FROM users) AS sub;`,
			},
			wantType:    cmn.SELECT_STATEMENT,
			wantClauses: 2,
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
			tz := tokenizer.NewSqlTokenizer(tt.args.src)
			tokens, err := tz.AllTokens()
			assert.NoError(t, err)
			pcTokens := TokenToEntity(tokens)
			pctx := &pc.ParseContext[Entity]{}
			pctx.MaxDepth = 30
			pctx.TraceEnable = true
			pctx.OrMode = pc.OrModeTryFast
			pctx.CheckTransformSafety = true

			consumed, got, err := ParseStatement()(pctx, pcTokens)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, len(pcTokens), consumed, "ParseGroup() should consume all tokens")
			assert.Equal(t, len(got), 1, "ParseGroup() should return exactly one statement")
			assert.Equal(t, tt.wantType, got[0].Val.NewValue.Type(), "ParseGroup() should return correct node type")
			stmt := got[0].Val.NewValue.(cmn.StatementNode)
			assert.Equal(t, tt.wantClauses, len(stmt.Clauses()))
		})
	}
}
