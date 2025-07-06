package parserstep3

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	step2 "github.com/shibukawa/snapsql/parser2/parserstep2"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestClauseOrderValidation(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantErr   bool
		wantOrder []cmn.NodeType
	}{
		{
			name:    "SELECT correct order",
			sql:     "SELECT id FROM users WHERE id = 1 ORDER BY id LIMIT 10 OFFSET 2 RETURNING id",
			wantErr: false,
			wantOrder: []cmn.NodeType{
				cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.WHERE_CLAUSE, cmn.ORDER_BY_CLAUSE, cmn.LIMIT_CLAUSE, cmn.OFFSET_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "SELECT wrong order",
			sql:     "SELECT id FROM users ORDER BY id WHERE id = 1 LIMIT 10 OFFSET 2 RETURNING id",
			wantErr: true,
			wantOrder: []cmn.NodeType{
				cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.ORDER_BY_CLAUSE, cmn.WHERE_CLAUSE, cmn.LIMIT_CLAUSE, cmn.OFFSET_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "INSERT VALUES correct order",
			sql:     "INSERT INTO users (id) VALUES (1) RETURNING id",
			wantErr: false,
			wantOrder: []cmn.NodeType{
				cmn.INSERT_INTO_CLAUSE, cmn.VALUES_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "INSERT VALUES wrong order",
			sql:     "INSERT INTO users (id) RETURNING id VALUES (1)",
			wantErr: true,
			wantOrder: []cmn.NodeType{
				cmn.INSERT_INTO_CLAUSE, cmn.RETURNING_CLAUSE, cmn.VALUES_CLAUSE,
			},
		},
		{
			name:    "INSERT SELECT correct order",
			sql:     "INSERT INTO users (id) SELECT id FROM tmp WHERE id > 1 ORDER BY id LIMIT 5 OFFSET 2 RETURNING id",
			wantErr: false,
			wantOrder: []cmn.NodeType{
				cmn.INSERT_INTO_CLAUSE, cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.WHERE_CLAUSE, cmn.ORDER_BY_CLAUSE, cmn.LIMIT_CLAUSE, cmn.OFFSET_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "INSERT SELECT wrong order",
			sql:     "INSERT INTO users (id) SELECT id FROM tmp ORDER BY id WHERE id > 1 LIMIT 5 OFFSET 2 RETURNING id",
			wantErr: true,
			wantOrder: []cmn.NodeType{
				cmn.INSERT_INTO_CLAUSE, cmn.SELECT_CLAUSE, cmn.FROM_CLAUSE, cmn.ORDER_BY_CLAUSE, cmn.WHERE_CLAUSE, cmn.LIMIT_CLAUSE, cmn.OFFSET_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "UPDATE correct order",
			sql:     "UPDATE users SET name = 'a' WHERE id = 1 RETURNING id",
			wantErr: false,
			wantOrder: []cmn.NodeType{
				cmn.UPDATE_CLAUSE, cmn.SET_CLAUSE, cmn.WHERE_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "UPDATE wrong order",
			sql:     "UPDATE users WHERE id = 1 SET name = 'a' RETURNING id",
			wantErr: true,
			wantOrder: []cmn.NodeType{
				cmn.UPDATE_CLAUSE, cmn.WHERE_CLAUSE, cmn.SET_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "DELETE correct order",
			sql:     "DELETE FROM users WHERE id = 1 RETURNING id",
			wantErr: false,
			wantOrder: []cmn.NodeType{
				cmn.DELETE_FROM_CLAUSE, cmn.WHERE_CLAUSE, cmn.RETURNING_CLAUSE,
			},
		},
		{
			name:    "DELETE wrong order",
			sql:     "DELETE FROM users RETURNING id WHERE id = 1",
			wantErr: true,
			wantOrder: []cmn.NodeType{
				cmn.DELETE_FROM_CLAUSE, cmn.RETURNING_CLAUSE, cmn.WHERE_CLAUSE,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tok.Tokenize(tt.sql)
			if err != nil {
				t.Fatalf("tokenize error: %v", err)
			}
			node, err := step2.Execute(tokens)
			if err != nil {
				t.Fatalf("parserstep2 error: %v", err)
			}
			var perr cmn.ParseError
			clauses := ValidateClausePresence(node.Type(), node.Clauses(), &perr)
			var gotOrder []cmn.NodeType
			for _, c := range clauses {
				gotOrder = append(gotOrder, c.Type())
			}
			if len(gotOrder) != len(tt.wantOrder) {
				t.Errorf("clause count mismatch: got %v, want %v", gotOrder, tt.wantOrder)
			}
			for i := range gotOrder {
				if gotOrder[i] != tt.wantOrder[i] {
					t.Errorf("clause order mismatch at %d: got %v, want %v", i, gotOrder[i], tt.wantOrder[i])
				}
			}
		})
	}
}
