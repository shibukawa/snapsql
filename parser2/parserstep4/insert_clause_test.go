package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	"github.com/shibukawa/snapsql/parser2/parserstep2"
	"github.com/shibukawa/snapsql/parser2/parserstep3"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// TestInsertWithoutColumnList tests INSERT statements without column list.
func TestInsertWithoutColumnList(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantErr     bool
		wantTable   cmn.TableReference
		wantColumns []string
	}{
		{
			name:    "insert without column list",
			sql:     "INSERT INTO users VALUES (1, 'Alice', 20);",
			wantErr: true,
		},
		{
			name:        "insert with column list",
			sql:         "INSERT INTO users (id, name, age) VALUES (1, 'Alice', 20);",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{"id", "name", "age"},
		},
		{
			name:        "insert with select without column list",
			sql:         "INSERT INTO users SELECT id, name, age FROM tmp;",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{},
		},
		{
			name:        "insert with select with column list",
			sql:         "INSERT INTO users (id, name, age) SELECT id, name, age FROM tmp;",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{"id", "name", "age"},
		},
		{
			name:        "insert with schema.table",
			sql:         "INSERT INTO public.users (id, name, age) VALUES (1, 'Alice', 20);",
			wantErr:     false,
			wantTable:   cmn.TableReference{SchemaName: "public", Name: "users"},
			wantColumns: []string{"id", "name", "age"},
		},
		{
			name:        "insert with db.table (MySQL/SQLite)",
			sql:         "INSERT INTO mydb.users (id, name, age) VALUES (2, 'Bob', 30);",
			wantErr:     false,
			wantTable:   cmn.TableReference{SchemaName: "mydb", Name: "users"},
			wantColumns: []string{"id", "name", "age"},
		},
		{
			name:        "insert with quoted schema and table",
			sql:         "INSERT INTO \"public\".\"users\" (id, name, age) VALUES (3, 'Carol', 25);",
			wantErr:     false,
			wantTable:   cmn.TableReference{SchemaName: `"public"`, Name: `"users"`},
			wantColumns: []string{"id", "name", "age"},
		},
		{
			name:    "insert with invalid table name (number)",
			sql:     "INSERT INTO 123 (id, name, age) VALUES (1, 'Alice', 20);",
			wantErr: true,
		},
		{
			name:    "insert with invalid table name (string literal)",
			sql:     "INSERT INTO 'users' (id, name, age) VALUES (1, 'Alice', 20);",
			wantErr: true,
		},
		{
			name:    "insert with invalid column name (number)",
			sql:     "INSERT INTO users (1, name, age) VALUES (1, 'Alice', 20);",
			wantErr: true,
		},
		{
			name:    "insert with invalid column name (string literal)",
			sql:     "INSERT INTO users ('id', name, age) VALUES (1, 'Alice', 20);",
			wantErr: true,
		},
		{
			name:    "insert with duplicate column names",
			sql:     "INSERT INTO users (id, id, name) VALUES (1, 2, 'Alice');",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, _ := tok.Tokenize(tt.sql)
			stmt, err := parserstep2.Execute(tokens)
			if err != nil {
				panic("unexpected error: " + err.Error())
			}
			err = parserstep3.Execute(stmt)
			if err != nil {
				panic("unexpected error: " + err.Error())
			}
			insertStmt, ok := stmt.(*cmn.InsertIntoStatement)
			if !ok {
				panic("cast should be success")
			}
			perr := &cmn.ParseError{}
			FinalizeInsertIntoClause(insertStmt.Into, insertStmt.Select, perr)
			if tt.wantErr {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				for _, e := range perr.Errors {
					t.Logf("Error: %s", e.Error())
				}
				assert.Equal(t, 0, len(perr.Errors))
				assert.Equal(t, tt.wantTable, insertStmt.Into.Table)
				assert.Equal(t, tt.wantColumns, insertStmt.Into.Columns)
			}
		})
	}
}
