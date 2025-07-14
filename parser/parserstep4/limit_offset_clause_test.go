package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/parser2/parserstep3"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestFinalizeLimitOffsetClause(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{
			name:    "valid limit and offset",
			src:     "SELECT * FROM users LIMIT 10 OFFSET 0",
			wantErr: false,
		},
		{
			name:    "valid limit and offset",
			src:     "SELECT * FROM users LIMIT 100 OFFSET 20",
			wantErr: false,
		},
		{
			name:    "negative limit",
			src:     "SELECT * FROM users LIMIT -1 OFFSET 0",
			wantErr: true,
		},
		{
			name:    "negative offset",
			src:     "SELECT * FROM users LIMIT 10 OFFSET -5",
			wantErr: true,
		},
		{
			name:    "both negative",
			src:     "SELECT * FROM users LIMIT -10 OFFSET -20",
			wantErr: true,
		},
		{
			name:    "non-numeric limit",
			src:     "SELECT * FROM users LIMIT 'abc' OFFSET 0",
			wantErr: true,
		},
		{
			name:    "non-numeric offset",
			src:     "SELECT * FROM users LIMIT 10 OFFSET 'xyz'",
			wantErr: true,
		},
		{
			name:    "both non-numeric",
			src:     "SELECT * FROM users LIMIT 'foo' OFFSET 'bar'",
			wantErr: true,
		},
		{
			name:    "MySQL LIMIT comma syntax (invalid)",
			src:     "SELECT * FROM users LIMIT 20, 10",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, _ := tok.Tokenize(tt.src)
			stmt, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(stmt)
			assert.NoError(t, err)
			selectStmt, ok := stmt.(*cmn.SelectStatement)
			assert.True(t, ok)
			perr := &cmn.ParseError{}
			finalizeLimitOffsetClause(selectStmt.Limit, selectStmt.Offset, perr)
			if tt.wantErr {
				assert.NotEqual(t, 0, len(perr.Errors), "should have errors")
			} else {
				assert.Equal(t, 0, len(perr.Errors), "should not have errors")
			}
		})
	}
}
