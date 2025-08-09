package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func TestFinalizeGroupByClause(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		wantErr              bool
		wantFieldNames       []cmn.FieldName
		wantNull             bool
		wantAdvancedGrouping bool
	}{
		{
			name:           "GROUP BY column name",
			input:          "SELECT id, name FROM users GROUP BY name",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}},
		},
		{
			name:           "GROUP BY table alias column",
			input:          "SELECT id, name FROM users u GROUP BY u.name",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{TableName: "u", Name: "name"}},
		},
		{
			name:    "GROUP BY number",
			input:   "SELECT id, name FROM users GROUP BY 1",
			wantErr: true,
		},
		{
			name:    "GROUP BY expression",
			input:   "SELECT id, name FROM users GROUP BY age + 1",
			wantErr: true,
		},
		{
			name:           "GROUP BY multiple columns",
			input:          "SELECT id, name FROM users GROUP BY name, age",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "name"}, {Name: "age"}},
		},
		{
			name:    "GROUP BY multiple columns with same name",
			input:   "SELECT id, name FROM users GROUP BY name, age, name",
			wantErr: true,
		},
		{
			name:                 "GROUP BY ROLLUP",
			input:                "SELECT department, job_title, COUNT(*) FROM employees GROUP BY ROLLUP(department, job_title)",
			wantErr:              false,
			wantFieldNames:       []cmn.FieldName{{Name: "department"}, {Name: "job_title"}},
			wantAdvancedGrouping: true,
		},
		{
			name:                 "GROUP BY CUBE",
			input:                "SELECT department, job_title, COUNT(*) FROM employees GROUP BY CUBE(department, job_title)",
			wantErr:              false,
			wantFieldNames:       []cmn.FieldName{{Name: "department"}, {Name: "job_title"}},
			wantAdvancedGrouping: true,
		},
		{
			name:                 "GROUP BY GROUPING SETS",
			input:                "SELECT department, job_title, COUNT(*) FROM employees GROUP BY GROUPING SETS ((department), (job_title))",
			wantErr:              false,
			wantFieldNames:       []cmn.FieldName{{Name: "department"}, {Name: "job_title"}},
			wantAdvancedGrouping: true,
		},
		{
			name:           "GROUP BY CASE expression",
			input:          "SELECT CASE WHEN age > 30 THEN 'over' ELSE 'under' END, COUNT(*) FROM users GROUP BY CASE WHEN age > 30 THEN 'over' ELSE 'under' END",
			wantErr:        false,
			wantFieldNames: []cmn.FieldName{{Name: "age"}},
		},
		{
			name:           "GROUP BY NULL",
			input:          "SELECT COUNT(*) FROM users GROUP BY NULL",
			wantErr:        false,
			wantNull:       true,
			wantFieldNames: []cmn.FieldName{},
		},
		{
			name:    "GROUP BY JSON expression",
			input:   "SELECT data->>'type', COUNT(*) FROM events GROUP BY data->>'type'",
			wantErr: true,
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

			groupByClause := selectStmt.GroupBy
			perr := &cmn.ParseError{}

			finalizeGroupByClause(groupByClause, perr)

			if tt.wantErr {
				assert.NotEqual(t, 0, len(perr.Errors), "should have errors")
			} else {
				assert.Equal(t, 0, len(perr.Errors), "should not have errors: %s", perr.Error())

				gotFieldNames := []cmn.FieldName{}
				for _, item := range groupByClause.Fields {
					gotFieldNames = append(gotFieldNames, cmn.FieldName{
						Name:      item.Name,
						TableName: item.TableName,
					})
				}

				assert.Equal(t, tt.wantFieldNames, gotFieldNames, "field names should match")
				assert.Equal(t, tt.wantAdvancedGrouping, groupByClause.AdvancedGrouping, "advanced grouping should match")
				assert.Equal(t, tt.wantNull, groupByClause.Null, "null grouping should match")
			}
		})
	}
}
