package query

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestSQLGenerator_Generate_BasicOperations(t *testing.T) {
	testCases := []struct {
		name         string
		instructions []intermediate.Instruction
		expressions  []intermediate.CELExpression
		params       map[string]any
		expectedSQL  string
		expectedArgs []any
		expectError  bool
	}{
		{
			name: "static text only",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users"},
			},
			expressions:  []intermediate.CELExpression{},
			params:       map[string]any{},
			expectedSQL:  "SELECT * FROM users",
			expectedArgs: []any{},
			expectError:  false,
		},
		{
			name: "static text with parameter",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(0)},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "user_id"},
			},
			params: map[string]any{
				"user_id": 123,
			},
			expectedSQL:  "SELECT * FROM users WHERE id = ?",
			expectedArgs: []any{123},
			expectError:  false,
		},
		{
			name: "multiple parameters",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " AND name = "},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(1)},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "user_id"},
				{Expression: "user_name"},
			},
			params: map[string]any{
				"user_id":   123,
				"user_name": "John",
			},
			expectedSQL:  "SELECT * FROM users WHERE id = ? AND name = ?",
			expectedArgs: []any{123, "John"},
			expectError:  false,
		},
		{
			name: "missing parameter",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(0)},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "user_id"},
			},
			params:      map[string]any{},
			expectedSQL: "",
			expectError: true,
		},
		{
			name: "invalid expression index",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users WHERE id = "},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(99)},
			},
			expressions: []intermediate.CELExpression{},
			params:      map[string]any{},
			expectedSQL: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				Instructions:   tc.instructions,
				CELExpressions: tc.expressions,
			}
			generator := NewSQLGenerator(format, snapsql.DialectPostgres)

			sql, args, err := generator.Generate(tc.params)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSQL, sql)
				assert.Equal(t, tc.expectedArgs, args)
			}
		})
	}
}

func TestSQLGenerator_Generate_ConditionalOperations(t *testing.T) {
	testCases := []struct {
		name         string
		instructions []intermediate.Instruction
		expressions  []intermediate.CELExpression
		params       map[string]any
		expectedSQL  string
		expectedArgs []any
		expectError  bool
	}{
		{
			name: "if condition true",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users"},
				{Op: intermediate.OpIf, ExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " WHERE active = true"},
				{Op: intermediate.OpEnd},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "include_active"},
			},
			params: map[string]any{
				"include_active": true,
			},
			expectedSQL:  "SELECT * FROM users WHERE active = true",
			expectedArgs: []any{},
			expectError:  false,
		},
		{
			name: "if condition false",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users"},
				{Op: intermediate.OpIf, ExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " WHERE active = true"},
				{Op: intermediate.OpEnd},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "include_active"},
			},
			params: map[string]any{
				"include_active": false,
			},
			expectedSQL:  "SELECT * FROM users",
			expectedArgs: []any{},
			expectError:  false,
		},
		{
			name: "if-else condition",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users"},
				{Op: intermediate.OpIf, ExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " WHERE active = true"},
				{Op: intermediate.OpElse},
				{Op: intermediate.OpEmitStatic, Value: " WHERE active = false"},
				{Op: intermediate.OpEnd},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "show_active"},
			},
			params: map[string]any{
				"show_active": false,
			},
			expectedSQL:  "SELECT * FROM users WHERE active = false",
			expectedArgs: []any{},
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				Instructions:   tc.instructions,
				CELExpressions: tc.expressions,
			}
			generator := NewSQLGenerator(format, snapsql.DialectPostgres)

			sql, args, err := generator.Generate(tc.params)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSQL, sql)
				assert.Equal(t, tc.expectedArgs, args)
			}
		})
	}
}

func TestSQLGenerator_SystemValueDefaults(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		Instructions: []intermediate.Instruction{
			{Op: intermediate.OpEmitStatic, Value: "INSERT INTO logs (created_at, created_by) VALUES ("},
			{Op: intermediate.OpEmitSystemValue, SystemField: "created_at"},
			{Op: intermediate.OpEmitStatic, Value: ", "},
			{Op: intermediate.OpEmitSystemValue, SystemField: "created_by"},
			{Op: intermediate.OpEmitStatic, Value: ")"},
		},
		SystemFields: []intermediate.SystemFieldInfo{
			{
				Name: "created_at",
				OnInsert: &intermediate.SystemFieldOperationInfo{
					Default: "NOW()",
				},
			},
			{
				Name: "created_by",
				OnInsert: &intermediate.SystemFieldOperationInfo{
					Parameter: "implicit",
				},
			},
		},
		ImplicitParameters: []intermediate.ImplicitParameter{
			{Name: "created_by", Type: "string"},
		},
	}

	generator := NewSQLGenerator(format, snapsql.DialectPostgres)
	params := map[string]any{}

	sql, args, err := generator.Generate(params)
	assert.NoError(t, err)
	assert.Equal(t, "INSERT INTO logs (created_at, created_by) VALUES (?, ?)", sql)

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	if _, ok := args[0].(time.Time); !ok {
		t.Fatalf("expected time.Time for created_at, got %T", args[0])
	}

	if _, ok := args[1].(string); !ok {
		t.Fatalf("expected string for created_by, got %T", args[1])
	}

	assert.Equal(t, args[0], params["created_at"])
	assert.Equal(t, args[1], params["created_by"])
}

func TestSQLGenerator_Generate_LoopOperations(t *testing.T) {
	testCases := []struct {
		name         string
		instructions []intermediate.Instruction
		expressions  []intermediate.CELExpression
		params       map[string]any
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name: "single level loop",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "INSERT"},
				{Op: intermediate.OpLoopStart, Variable: "value", CollectionExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " ["},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(1)},
				{Op: intermediate.OpEmitStatic, Value: " ]"},
				{Op: intermediate.OpLoopEnd},
				{Op: intermediate.OpEmitStatic, Value: " ;"},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "items"},
				{Expression: "value"},
			},
			params: map[string]any{
				"items": []any{"foo", "bar"},
			},
			expectedSQL:  "INSERT [? ] [? ] ;",
			expectedArgs: []any{"foo", "bar"},
		},
		{
			name: "nested loops",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "BEGIN"},
				{Op: intermediate.OpLoopStart, Variable: "group", CollectionExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " ["},
				{Op: intermediate.OpLoopStart, Variable: "member", CollectionExprIndex: intPtr(1)},
				{Op: intermediate.OpEmitStatic, Value: " {"},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(2)},
				{Op: intermediate.OpEmitStatic, Value: " }"},
				{Op: intermediate.OpLoopEnd},
				{Op: intermediate.OpEmitStatic, Value: " ]"},
				{Op: intermediate.OpLoopEnd},
				{Op: intermediate.OpEmitStatic, Value: " END"},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "groups"},
				{Expression: "group"},
				{Expression: "member"},
			},
			params: map[string]any{
				"groups": []any{
					[]any{"a", "b"},
					[]any{"c"},
				},
			},
			expectedSQL:  "BEGIN [ {? } {? } ] [ {? } ] END",
			expectedArgs: []any{"a", "b", "c"},
		},
		{
			name: "empty collection",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "DELETE FROM t"},
				{Op: intermediate.OpLoopStart, Variable: "value", CollectionExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: " SHOULD_NOT_APPEAR"},
				{Op: intermediate.OpLoopEnd},
				{Op: intermediate.OpEmitStatic, Value: " WHERE flag = true"},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "items"},
			},
			params: map[string]any{
				"items": []any{},
			},
			expectedSQL:  "DELETE FROM t WHERE flag = true",
			expectedArgs: []any{},
		},
		{
			name: "loop removes trailing comma",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "INSERT INTO t VALUES"},
				{Op: intermediate.OpLoopStart, Variable: "row", CollectionExprIndex: intPtr(0)},
				{Op: intermediate.OpEmitStatic, Value: "("},
				{Op: intermediate.OpEmitEval, ExprIndex: intPtr(1)},
				{Op: intermediate.OpEmitStatic, Value: ")"},
				{Op: intermediate.OpEmitStatic, Value: ",\n"},
				{Op: intermediate.OpLoopEnd},
				{Op: intermediate.OpEmitStatic, Value: "RETURNING id"},
			},
			expressions: []intermediate.CELExpression{
				{Expression: "rows"},
				{Expression: "row"},
			},
			params: map[string]any{
				"rows": []any{1, 2},
			},
			expectedSQL:  "INSERT INTO t VALUES(?),\n(?)\nRETURNING id",
			expectedArgs: []any{1, 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				Instructions:   tc.instructions,
				CELExpressions: tc.expressions,
			}

			generator := NewSQLGenerator(format, snapsql.DialectPostgres)
			sql, args, err := generator.Generate(tc.params)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedSQL, sql)
			assert.Equal(t, tc.expectedArgs, args)
		})
	}
}

func TestLoadIntermediateFormat_SupportedFileTypes(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name        string
		filename    string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "JSON file",
			filename:    "test.json",
			content:     "",
			expectError: true,
			errorMsg:    "template file not found", // File doesn't exist, but format is supported
		},
		{
			name:        "SQL file",
			filename:    "test.snap.sql",
			content:     "",
			expectError: true,
			errorMsg:    "template file not found", // File doesn't exist, but format is supported
		},
		{
			name:        "Markdown file",
			filename:    "test.snap.md",
			content:     "",
			expectError: true,
			errorMsg:    "template file not found", // File doesn't exist, but format is supported
		},
		{
			name:        "Unsupported file",
			filename:    "test.txt",
			content:     "some content",
			expectError: true,
			errorMsg:    "unsupported template file format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var filePath string
			if tc.content != "" {
				// Create the file for unsupported format test
				filePath = filepath.Join(tmpDir, tc.filename)
				err := os.WriteFile(filePath, []byte(tc.content), 0644)
				assert.NoError(t, err)
			} else {
				// Use non-existent file path
				filePath = tc.filename
			}

			_, err := LoadIntermediateFormat(filePath)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
