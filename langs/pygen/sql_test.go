package pygen

import (
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
			},
			dialect:      "postgres",
			wantSQL:      "SELECT * FROM users WHERE id = $1",
			wantArgs:     []string{"userId"},
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
			},
			dialect:      "mysql",
			wantSQL:      "SELECT * FROM users WHERE id = %s",
			wantArgs:     []string{"userId"},
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
			},
			dialect:      "sqlite",
			wantSQL:      "SELECT * FROM users WHERE id = ?",
			wantArgs:     []string{"userId"},
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
			},
			dialect:      "postgres",
			wantSQL:      "INSERT INTO users (username, email) VALUES ($1, $2)",
			wantArgs:     []string{"username", "email"},
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
			},
			dialect:      "postgres",
			wantSQL:      "UPDATE users SET username = $1, updated_by = $2 WHERE id = $3",
			wantArgs:     []string{"username", "updated_by", "userId"},
			wantIsStatic: true,
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
				Instructions: []codegenerator.Instruction{
					{Op: codegenerator.OpEmitStatic, Value: "SELECT * FROM users"},
					{Op: codegenerator.OpIf, ExprIndex: intPtr(0)},
					{Op: codegenerator.OpEmitStatic, Value: " WHERE username = "},
					{Op: codegenerator.OpEmitEval, ExprIndex: intPtr(1)},
					{Op: codegenerator.OpEnd},
				},
				CELExpressions: []intermediate.CELExpression{
					{Expression: "username != null"},
					{Expression: "username"},
				},
			},
			dialect:      "postgres",
			wantIsStatic: false,
			wantCodeContains: []string{
				"sql_parts = []",
				"args = []",
				"if username is not None:",
				"sql_parts.append(\"SELECT * FROM users\")",
				"sql_parts.append(\" WHERE username = $1\")",
				"args.append(username)",
				"sql = ' '.join(sql_parts)",
			},
		},
		{
			name: "loop with EMIT_UNLESS_BOUNDARY",
			format: &intermediate.IntermediateFormat{
				FunctionName: "get_users_by_ids",
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
			},
			dialect:      "postgres",
			wantIsStatic: false,
			wantCodeContains: []string{
				"sql_parts = []",
				"args = []",
				"for id_idx, id in enumerate(_collection):",
				"id_is_last = (id_idx == len(_collection) - 1)",
				"if not id_is_last:",
				"sql_parts.append(\", \")",
			},
		},
		{
			name: "IF-ELSE structure",
			format: &intermediate.IntermediateFormat{
				FunctionName: "search_users_advanced",
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
			},
			dialect:      "postgres",
			wantIsStatic: false,
			wantCodeContains: []string{
				"if includeActive:",
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

func TestConvertCELExpressionToPython(t *testing.T) {
	tests := []struct {
		name     string
		celExpr  string
		wantExpr string
	}{
		{
			name:     "simple null check",
			celExpr:  "username != null",
			wantExpr: "username is not None",
		},
		{
			name:     "logical AND",
			celExpr:  "active && verified",
			wantExpr: "active and verified",
		},
		{
			name:     "logical OR",
			celExpr:  "admin || moderator",
			wantExpr: "admin or moderator",
		},
		{
			name:     "negation",
			celExpr:  "!deleted",
			wantExpr: "not deleted",
		},
		{
			name:     "complex expression",
			celExpr:  "user_id != null && active",
			wantExpr: "userId is not None and active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertCELExpressionToPython(tt.celExpr)
			if got != tt.wantExpr {
				t.Errorf("convertCELExpressionToPython(%q) = %q, want %q", tt.celExpr, got, tt.wantExpr)
			}
		})
	}
}

func TestSnakeToCamelLower(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple snake_case",
			input: "user_id",
			want:  "userId",
		},
		{
			name:  "multiple underscores",
			input: "first_name_last_name",
			want:  "firstNameLastName",
		},
		{
			name:  "already camelCase",
			input: "userId",
			want:  "userId",
		},
		{
			name:  "single word",
			input: "username",
			want:  "username",
		},
		{
			name:  "with uppercase",
			input: "USER_ID",
			want:  "userId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snakeToCamelLower(tt.input)
			if got != tt.want {
				t.Errorf("snakeToCamelLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
