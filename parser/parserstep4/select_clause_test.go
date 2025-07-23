package parserstep4

import (
	"log"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser/parsercommon"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/tokenizer"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

type testCase struct {
	name              string
	sql               string
	wantError         bool
	distinct          bool
	distinctOn        []string
	wantFieldTypes    []cmn.FieldType
	wantFieldNames    []string
	wantExplicitNames []bool
	wantTypeName      []string
	wantExplicitTypes []bool
}

func TestFinalizeSelectClause(t *testing.T) {
	tests := []testCase{
		// --- forbidden literal tests ---
		{
			name:              "SELECT literal number is forbidden",
			sql:               "SELECT 123 FROM users",
			wantFieldTypes:    []cmn.FieldType{cmn.LiteralField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "SELECT literal string is forbidden",
			sql:               "SELECT 'abc' FROM users",
			wantFieldTypes:    []cmn.FieldType{cmn.LiteralField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "SELECT literal boolean is forbidden",
			sql:               "SELECT true FROM users",
			wantFieldTypes:    []cmn.FieldType{cmn.LiteralField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "SELECT literal NULL is forbidden",
			sql:               "SELECT NULL FROM users",
			wantFieldTypes:    []cmn.FieldType{cmn.LiteralField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "SELECT literal with alias is forbidden",
			sql:               "SELECT 1 AS one FROM users",
			wantFieldTypes:    []cmn.FieldType{cmn.LiteralField},
			wantFieldNames:    []string{"one"},
			wantExplicitNames: []bool{true},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "SELECT literal with cast is forbidden",
			sql:               "SELECT 1::int FROM users",
			wantFieldTypes:    []cmn.FieldType{cmn.LiteralField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"int"},
			wantExplicitTypes: []bool{true},
		},
		/*{
			name:      "SELECT literal in expression is forbidden",
			sql:       "SELECT 1+2 FROM users",
			wantError: true,
		},*/
		// --- alias and cast tests ---
		{
			name:              "Single field with alias",
			sql:               "SELECT name AS n FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"n"},
			wantExplicitNames: []bool{true},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "Single field with alias (no AS)",
			sql:               "SELECT name n FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"n"},
			wantExplicitNames: []bool{true},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "Multiple fields with alias",
			sql:               "SELECT name AS n, age AS a FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField, cmn.SingleField},
			wantFieldNames:    []string{"n", "a"},
			wantExplicitNames: []bool{true, true},
			wantTypeName:      []string{"", ""},
			wantExplicitTypes: []bool{false, false},
		},
		{
			name:              "Alias with reserved word (should be allowed if quoted)",
			sql:               "SELECT name AS \"select\" FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{`"select"`}, // keep quotes
			wantExplicitNames: []bool{true},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "Field with type cast (standard CAST)",
			sql:               "SELECT CAST(age AS TEXT) FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"TEXT"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL ::)",
			sql:               "SELECT age::text FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with spaces)",
			sql:               "SELECT age :: text FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with newline)",
			sql:               "SELECT age\n::\ntext FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with parens)",
			sql:               "SELECT (age)::text FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with parens and spaces)",
			sql:               "SELECT ( age ) :: text FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with qualified identifier)",
			sql:               "SELECT u.age::text FROM users u",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.TableField},
			wantFieldNames:    []string{"age"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with function call)",
			sql:               "SELECT sum(age)::numeric FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.FunctionField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"numeric"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "Field with type cast (PostgreSQL :: with JSON path)",
			sql:               "SELECT data->>'age'::text FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.ComplexField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"text"},
			wantExplicitTypes: []bool{true},
		},
		{
			name:              "ALL (default, no error)",
			sql:               "SELECT ALL name FROM users",
			wantError:         false,
			distinct:          false,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"name"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:      "DISTINCT ALL is not allowed",
			sql:       "SELECT DISTINCT ALL name FROM users",
			wantError: true,
		},
		{
			name:              "DISTINCT ON", //  PostgreSQL only
			sql:               "SELECT DISTINCT ON (user_id) user_id, created_at FROM orders ORDER BY user_id, created_at DESC",
			distinct:          true,
			distinctOn:        []string{"user_id"},
			wantError:         false, // PostgreSQL only, adjust if unsupported
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField, cmn.SingleField},
			wantFieldNames:    []string{"user_id", "created_at"},
			wantExplicitNames: []bool{false, false},
			wantTypeName:      []string{"", ""},
			wantExplicitTypes: []bool{false, false},
		},
		{
			name:              "COUNT(*) is allowed",
			sql:               "SELECT COUNT(*) FROM users",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.FunctionField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"integer"},
			wantExplicitTypes: []bool{false},
		},
		{
			name:      "SELECT * is forbidden",
			sql:       "SELECT * FROM users",
			wantError: true,
		},
		{
			name:      "SELECT t.* is forbidden",
			sql:       "SELECT t.* FROM users t",
			wantError: true,
		},
		{
			name:              "COUNT(DISTINCT *) is allowed",
			sql:               "SELECT COUNT(DISTINCT *) FROM users",
			wantError:         false,
			distinct:          false,
			wantFieldTypes:    []cmn.FieldType{cmn.FunctionField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"integer"},
			wantExplicitTypes: []bool{false},
		},
		{
			name:      "SELECT *, id is forbidden",
			sql:       "SELECT *, id FROM users",
			wantError: true,
		},
		{
			name:              "Subquery in SELECT with * is allowed",
			sql:               "SELECT (SELECT COUNT(*) FROM orders WHERE user_id = u.id) as order_count FROM users u",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.ComplexField},
			wantFieldNames:    []string{"order_count"},
			wantExplicitNames: []bool{true},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "DISTINCT ON with table-qualified column",
			sql:               "SELECT DISTINCT ON (t.name) t.name AS name2, t.age FROM users t ORDER BY t.name, t.age DESC",
			wantError:         false, // PostgreSQL only, valid usage
			distinct:          true,
			distinctOn:        []string{"t.name"},
			wantFieldTypes:    []cmn.FieldType{cmn.TableField, cmn.TableField},
			wantFieldNames:    []string{"name2", "age"},
			wantExplicitNames: []bool{true, false},
			wantTypeName:      []string{"", ""},
			wantExplicitTypes: []bool{false, false},
		},
		{
			name:      "DISTINCT ON with alias (not allowed)",
			sql:       "SELECT DISTINCT ON (name2) t.name AS name2, t.age FROM users t ORDER BY t.name, t.age DESC",
			wantError: true, // alias in DISTINCT ON is not allowed
		},
		{
			name:              "DISTINCT single column",
			sql:               "SELECT DISTINCT name FROM users",
			wantError:         false,
			distinct:          true,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField},
			wantFieldNames:    []string{"name"},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{""},
			wantExplicitTypes: []bool{false},
		},
		{
			name:              "DISTINCT multiple columns",
			sql:               "SELECT DISTINCT name, age FROM users",
			wantError:         false,
			distinct:          true,
			wantFieldTypes:    []cmn.FieldType{cmn.SingleField, cmn.SingleField},
			wantFieldNames:    []string{"name", "age"},
			wantExplicitNames: []bool{false, false},
			wantTypeName:      []string{"", ""},
			wantExplicitTypes: []bool{false, false},
		},
		{
			name:              "COUNT(DISTINCT user_id) is allowed",
			sql:               "SELECT COUNT(DISTINCT user_id) FROM orders",
			wantError:         false,
			wantFieldTypes:    []cmn.FieldType{cmn.FunctionField},
			wantFieldNames:    []string{""},
			wantExplicitNames: []bool{false},
			wantTypeName:      []string{"integer"},
			wantExplicitTypes: []bool{false},
		},
		{
			name:      "DISTINCT with duplicate column name is forbidden",
			sql:       "SELECT DISTINCT name, name FROM users",
			wantError: true,
		},
		{
			name:      "duplicate column name is forbidden",
			sql:       "SELECT name, name FROM users",
			wantError: true,
		},
		{
			name:      "duplicate column name is forbidden even if table name qualified",
			sql:       "SELECT users.name, name FROM users",
			wantError: true,
		},
		{
			name:      "duplicate column name is forbidden even if table name qualified (2)",
			sql:       "SELECT users.name, members.name FROM users LEFT OUTER JOIN members ON users.id = members.user_id",
			wantError: true,
		},
		{
			name:      "duplicate column name is forbidden even if on of the name is alias",
			sql:       "SELECT name, family_name as name FROM users",
			wantError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(tc.sql)
			assert.NoError(t, err)
			ast, err := parserstep2.Execute(tokens)
			assert.NoError(t, err)
			err = parserstep3.Execute(ast)
			assert.NoError(t, err)

			selectStmt, ok := ast.(*parsercommon.SelectStatement)
			assert.True(t, ok)
			selectClause := selectStmt.Select
			perr := &parsercommon.ParseError{}
			finalizeSelectClause(selectClause, perr)
			if tc.wantError {
				assert.True(t, len(perr.Errors) > 0, "should return error")
			} else {
				assert.True(t, len(perr.Errors) == 0, "should not return error")
				assert.Equal(t, tc.distinct, selectClause.Distinct, "distinct flag")
				assert.Equal(t, len(tc.distinctOn), len(selectClause.DistinctOn), "distinctOn length")

				gotFieldTypes := make([]cmn.FieldType, 0, len(selectClause.Fields))
				gotFieldNames := make([]string, 0, len(selectClause.Fields))
				gotImplicitNames := make([]bool, 0, len(selectClause.Fields))
				gotTypeNames := make([]string, 0, len(selectClause.Fields))
				gotImplicitTypes := make([]bool, 0, len(selectClause.Fields))
				for _, field := range selectClause.Fields {
					gotFieldTypes = append(gotFieldTypes, field.FieldKind)
					gotFieldNames = append(gotFieldNames, field.FieldName)
					gotImplicitNames = append(gotImplicitNames, field.ExplicitName)
					gotTypeNames = append(gotTypeNames, field.TypeName)
					gotImplicitTypes = append(gotImplicitTypes, field.ExplicitType)
				}
				assert.Equal(t, tc.wantFieldTypes, gotFieldTypes, "Field types do not match")
				assert.Equal(t, tc.wantFieldNames, gotFieldNames, "Field names do not match")
				assert.Equal(t, tc.wantExplicitNames, gotImplicitNames, "Implicit names do not match")
				assert.Equal(t, tc.wantTypeName, gotTypeNames, "Type names do not match")
				assert.Equal(t, tc.wantExplicitTypes, gotImplicitTypes, "Implicit types do not match")
			}
		})
	}
}
