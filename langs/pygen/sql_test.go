package pygen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

func TestGenerateStaticSQL(t *testing.T) {
	tests := []struct {
		name         string
		format       *intermediate.IntermediateFormat
		dialect      snapsql.Dialect
		wantSQL      string
		wantArgs     []string
		wantIsStatic bool
	}{
		{
			name: "simple SELECT with PostgreSQL",
			format: &intermediate.IntermediateFormat{
				FunctionName: "get_user",
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(0)},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "user_id"},
				},
				Expressions: stubExpressions("user_id"),
			},
			dialect:      "postgres",
			wantSQL:      "SELECT * FROM users WHERE id = $1",
			wantArgs:     []string{"_eval_explang_expression(0, param_map)"},
			wantIsStatic: true,
		},
		{
			name: "simple SELECT with MySQL",
			format: &intermediate.IntermediateFormat{
				FunctionName: "get_user",
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(0)},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "user_id"},
				},
				Expressions: stubExpressions("user_id"),
			},
			dialect:      "mysql",
			wantSQL:      "SELECT * FROM users WHERE id = %s",
			wantArgs:     []string{"_eval_explang_expression(0, param_map)"},
			wantIsStatic: true,
		},
		{
			name: "simple SELECT with SQLite",
			format: &intermediate.IntermediateFormat{
				FunctionName: "get_user",
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(0)},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "user_id"},
				},
				Expressions: stubExpressions("user_id"),
			},
			dialect:      "sqlite",
			wantSQL:      "SELECT * FROM users WHERE id = ?",
			wantArgs:     []string{"_eval_explang_expression(0, param_map)"},
			wantIsStatic: true,
		},
		{
			name: "INSERT with multiple parameters",
			format: &intermediate.IntermediateFormat{
				FunctionName: "insert_user",
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "INSERT INTO users (username, email) VALUES ("},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(0)},
					{Op: codegenerator.OpEmitStatic, Value: ", "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(1)},
					{Op: codegenerator.OpEmitStatic, Value: ")"},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "username"},
					{Expression: "email"},
				},
				Expressions: stubExpressions("username", "email"),
			},
			dialect:      "postgres",
			wantSQL:      "INSERT INTO users (username, email) VALUES ($1, $2)",
			wantArgs:     []string{"_eval_explang_expression(0, param_map)", "_eval_explang_expression(1, param_map)"},
			wantIsStatic: true,
		},
		{
			name: "UPDATE with system parameter",
			format: &intermediate.IntermediateFormat{
				FunctionName: "update_user",
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "UPDATE users SET username = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(0)},
					{Op: codegenerator.OpEmitStatic, Value: ", updated_by = "},
					{Op: codegenerator.OpEmitSystemValue, SystemField: "updated_by"},
					{Op: codegenerator.OpEmitStatic, Value: " WHERE id = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(1)},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "username"},
					{Expression: "user_id"},
				},
				Expressions: stubExpressions("username", "user_id"),
			},
			dialect:      "postgres",
			wantSQL:      "UPDATE users SET username = $1, updated_by = $2 WHERE id = $3",
			wantArgs:     []string{"_eval_explang_expression(0, param_map)", "updated_by", "_eval_explang_expression(1, param_map)"},
			wantIsStatic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processSQLBuilder(tt.format, tt.dialect)
			if err != nil {
				t.Fatalf("processSQLBuilder() error = %v", err)
			}

			if result.IsStatic != tt.wantIsStatic {
				t.Errorf("IsStatic = %v, want %v", result.IsStatic, tt.wantIsStatic)
			}

			if result.StaticSQL != tt.wantSQL {
				t.Errorf("StaticSQL = %q, want %q", result.StaticSQL, tt.wantSQL)
			}

			if len(result.Args) != len(tt.wantArgs) {
				t.Errorf("Args length = %d, want %d", len(result.Args), len(tt.wantArgs))
			} else {
				for i, arg := range result.Args {
					if arg != tt.wantArgs[i] {
						t.Errorf("Args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
					}
				}
			}
		})
	}
}

func TestGenerateDynamicSQL(t *testing.T) {
	tests := []struct {
		name             string
		format           *intermediate.IntermediateFormat
		dialect          string
		wantIsStatic     bool
		wantCodeContains []string
	}{
		{
			name: "conditional WHERE clause",
			format: &intermediate.IntermediateFormat{
				FunctionName: "search_users",
				Parameters:   []intermediate.Parameter{{Name: "username"}},
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users"},
					{Op: codegenerator.OpIf, ExprIndex: intPtr(0)},
					{Op: codegenerator.OpEmitStatic, Value: " WHERE username = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(1)},
					{Op: codegenerator.OpEnd},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "username_present"},
					{Expression: "username"},
				},
				Expressions: stubExpressions("username_present", "username"),
			},
			dialect:      "postgres",
			wantIsStatic: false,
			wantCodeContains: []string{
				"sql_parts = []",
				"args = []",
				"param_map = {",
				"'username': username",
				"cond_value = _eval_explang_expression(0, param_map)",
				"if _truthy(cond_value):",
				"sql_parts.append(\"SELECT * FROM users\")",
				"sql_parts.append(\" WHERE username = $1\")",
				"args.append(_eval_explang_expression(1, param_map))",
				"sql = ''.join(sql_parts)",
			},
		},
		{
			name: "loop with EMIT_UNLESS_BOUNDARY",
			format: &intermediate.IntermediateFormat{
				FunctionName: "get_users_by_ids",
				Parameters:   []intermediate.Parameter{{Name: "user_ids"}},
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users WHERE id IN ("},
					{Op: codegenerator.OpLoopStart, Variable: "id", CollectionExprIndex: intPtr(0)},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(1)},
					{Op: codegenerator.OpEmitUnlessBoundary, Value: ", "},
					{Op: codegenerator.OpLoopEnd},
					{Op: codegenerator.OpEmitStatic, Value: ")"},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "user_ids"},
					{Expression: "id"},
				},
				Expressions: stubExpressions("user_ids", "id"),
			},
			dialect:      "postgres",
			wantIsStatic: false,
			wantCodeContains: []string{
				"param_map = {",
				"'user_ids': user_ids",
				"for id in _as_iterable(id_collection):",
				"param_map['id'] = id",
				"sql_parts.append(\"$",
				"args.append(_eval_explang_expression(1, param_map))",
			},
		},
		{
			name: "IF-ELSE structure",
			format: &intermediate.IntermediateFormat{
				FunctionName: "search_users_advanced",
				Parameters:   []intermediate.Parameter{{Name: "include_active"}},
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users WHERE"},
					{Op: codegenerator.OpIf, ExprIndex: intPtr(0)},
					{Op: codegenerator.OpEmitStatic, Value: " active = true"},
					{Op: codegenerator.OpElse},
					{Op: codegenerator.OpEmitStatic, Value: " active = false"},
					{Op: codegenerator.OpEnd},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "include_active"},
				},
				Expressions: stubExpressions("include_active"),
			},
			dialect:      "postgres",
			wantIsStatic: false,
			wantCodeContains: []string{
				"param_map = {",
				"'include_active': include_active",
				"cond_value = _eval_explang_expression(0, param_map)",
				"if _truthy(cond_value):",
				"else:",
				"sql_parts.append(\" active = true\")",
				"sql_parts.append(\" active = false\")",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processSQLBuilder(tt.format, snapsql.Dialect(tt.dialect))
			if err != nil {
				t.Fatalf("processSQLBuilder() error = %v", err)
			}

			if result.IsStatic != tt.wantIsStatic {
				t.Errorf("IsStatic = %v, want %v", result.IsStatic, tt.wantIsStatic)
			}

			if result.IsStatic {
				t.Skip("Test is for dynamic SQL")
			}

			for _, want := range tt.wantCodeContains {
				if !strings.Contains(result.DynamicCode, want) {
					t.Errorf("DynamicCode does not contain %q\nGot:\n%s", want, result.DynamicCode)
				}
			}
		})
	}
}

func TestPythonIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "snake_case",
			input: "user_id",
			want:  "user_id",
		},
		{
			name:  "dotted expression",
			input: "user.id",
			want:  "user_id",
		},
		{
			name:  "uppercase",
			input: "USER_ID",
			want:  "user_id",
		},
		{
			name:  "leading digit",
			input: "123value",
			want:  "_123value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pythonIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("pythonIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func stubExpressions(names ...string) []intermediate.ExplangExpression {
	exprs := make([]intermediate.ExplangExpression, len(names))
	for i, name := range names {
		exprs[i] = intermediate.ExplangExpression{
			ID: fmt.Sprintf("expr_%d", i),
			Steps: []intermediate.Expressions{
				{
					Kind:       intermediate.StepIdentifier,
					Identifier: name,
				},
			},
		}
	}

	return exprs
}
