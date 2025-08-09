package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestFinalizeOrderByClause(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantErr        bool
		wantFieldNames []cmn.FieldName
		wantDescs      []bool
	}{
		{
			name:           "ORDER BY column name",
			input:          "SELECT id, name FROM users ORDER BY name",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY table alias column",
			input:          "SELECT id, name FROM users u ORDER BY u.name",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{TableName: "u", Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY ASC explicit",
			input:          "SELECT id, name FROM users ORDER BY name ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY table alias column ASC",
			input:          "SELECT id, name FROM users u ORDER BY u.name ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{TableName: "u", Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:    "ORDER BY position",
			input:   "SELECT id, name FROM users ORDER BY 1",
			wantErr: true,
		},
		{
			name:    "ORDER BY expression",
			input:   "SELECT id, name FROM users ORDER BY age + 1",
			wantErr: true,
		},
		{
			name:           "ORDER BY multiple columns",
			input:          "SELECT id, name FROM users ORDER BY name, age DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}, {Name: "age"}},
			wantDescs:      []bool{false, true},
		},
		{
			name:    "ORDER BY multiple columns, but have same names",
			input:   "SELECT id, name FROM users ORDER BY name, name DESC",
			wantErr: true,
		},
		{
			name:           "ORDER BY with NULLS FIRST",
			input:          "SELECT id, name FROM users ORDER BY age DESC NULLS FIRST",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{true},
		},
		{
			name:           "ORDER BY with NULLS LAST",
			input:          "SELECT id, name FROM users ORDER BY age ASC NULLS LAST",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY FIELD function (MySQL)",
			input:          "SELECT id, name, status FROM users ORDER BY FIELD(status, 'active', 'pending', 'inactive')",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "status"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE (SQLite)",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE NOCASE",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY type cast (PostgreSQL)",
			input:          "SELECT id, name, age FROM users ORDER BY age::text",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY CASE expression",
			input:          "SELECT id, name, age FROM users ORDER BY CASE WHEN age > 30 THEN 1 ELSE 2 END",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{false},
		},
		{
			name:    "ORDER BY invalid position (zero)",
			input:   "SELECT id, name FROM users ORDER BY 0",
			wantErr: true,
		},
		{
			name:    "ORDER BY invalid position (negative)",
			input:   "SELECT id, name FROM users ORDER BY -1",
			wantErr: true,
		},
		{
			name:    "ORDER BY invalid position (out of range)",
			input:   "SELECT id, name FROM users ORDER BY 100",
			wantErr: true,
		},
		{
			name:           "ORDER BY multiple FIELD functions",
			input:          "SELECT id, name, a, b FROM users ORDER BY FIELD(a,1,2), FIELD(b,3,4)",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "a"}, {Name: "b"}},
			wantDescs:      []bool{false, false},
		},
		{
			name:           "ORDER BY multiple CASE expressions",
			input:          "SELECT id, name, a, b FROM users ORDER BY CASE WHEN a>0 THEN 1 ELSE 2 END, CASE WHEN b>0 THEN 3 ELSE 4 END",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "a"}, {Name: "b"}},
			wantDescs:      []bool{false, false},
		},
		{
			name:           "ORDER BY multiple NULLS FIRST/LAST",
			input:          "SELECT id, name, a, b FROM users ORDER BY a DESC NULLS FIRST, b ASC NULLS LAST",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "a"}, {Name: "b"}},
			wantDescs:      []bool{true, false},
		},
		{
			name:           "ORDER BY multiple COLLATE",
			input:          "SELECT id, name, address FROM users ORDER BY name COLLATE NOCASE, address COLLATE NOCASE",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}, {Name: "address"}},
			wantDescs:      []bool{false, false},
		},
		{
			name:           "ORDER BY COLLATE BINARY",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE BINARY",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE RTRIM",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE RTRIM",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE locale (PostgreSQL)",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE \"ja_JP\"",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE charset (MySQL)",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE utf8mb4_unicode_ci",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY CAST function (MySQL/PostgreSQL)",
			input:          "SELECT id, name, age FROM users ORDER BY CAST(age AS CHAR)",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}}, // 関数なので空
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE locale DESC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE \"ja_JP\" DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{true},
		},
		{
			name:           "ORDER BY COLLATE charset ASC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE utf8mb4_unicode_ci ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY CAST function DESC",
			input:          "SELECT id, name, age FROM users ORDER BY CAST(age AS CHAR) DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{true},
		},
		{
			name:           "ORDER BY CASE expression ASC",
			input:          "SELECT id, name, age FROM users ORDER BY CASE WHEN age > 30 THEN 1 ELSE 2 END ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY CASE expression DESC",
			input:          "SELECT id, name, age FROM users ORDER BY CASE WHEN age > 30 THEN 1 ELSE 2 END DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
			wantDescs:      []bool{true},
		},
		{
			name:           "ORDER BY COLLATE NOCASE ASC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE NOCASE ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE NOCASE DESC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE NOCASE DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{true},
		},
		{
			name:           "ORDER BY COLLATE BINARY ASC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE BINARY ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE BINARY DESC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE BINARY DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{true},
		},
		{
			name:           "ORDER BY COLLATE RTRIM ASC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE RTRIM ASC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{false},
		},
		{
			name:           "ORDER BY COLLATE RTRIM DESC",
			input:          "SELECT id, name FROM users ORDER BY name COLLATE RTRIM DESC",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
			wantDescs:      []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, _ := tok.Tokenize(tt.input)
			stmt, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(stmt)
			assert.NoError(t, err)

			selectStmt, ok := stmt.(*cmn.SelectStatement)
			assert.True(t, ok)

			orderByClause := selectStmt.OrderBy
			perr := &cmn.ParseError{}

			finalizeOrderByClause(orderByClause, perr)

			if tt.wantErr {
				assert.NotEqual(t, 0, len(perr.Errors), "should have errors")
			} else {
				assert.Equal(t, 0, len(perr.Errors), "should not have errors: %s", perr.Error())

				gotFieldNames := []cmn.FieldName{}
				gotDescs := []bool{}

				for _, item := range orderByClause.Fields {
					gotFieldNames = append(gotFieldNames, item.Field)
					gotDescs = append(gotDescs, item.Desc)
				}

				assert.Equal(t, tt.wantFieldNames, gotFieldNames, "field names should match")
				assert.Equal(t, tt.wantDescs, gotDescs, "descs should match")
			}
		})
	}
}
