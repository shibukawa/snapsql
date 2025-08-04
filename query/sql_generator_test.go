package query

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestSQLGenerator_Generate_BasicOperations(t *testing.T) {
	testCases := []struct {
		name         string
		instructions []intermediate.Instruction
		expressions  []intermediate.CELExpression
		params       map[string]interface{}
		expectedSQL  string
		expectedArgs []interface{}
		expectError  bool
	}{
		{
			name: "static text only",
			instructions: []intermediate.Instruction{
				{Op: intermediate.OpEmitStatic, Value: "SELECT * FROM users"},
			},
			expressions:  []intermediate.CELExpression{},
			params:       map[string]interface{}{},
			expectedSQL:  "SELECT * FROM users",
			expectedArgs: []interface{}{},
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
			params: map[string]interface{}{
				"user_id": 123,
			},
			expectedSQL:  "SELECT * FROM users WHERE id = ?",
			expectedArgs: []interface{}{123},
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
			params: map[string]interface{}{
				"user_id":   123,
				"user_name": "John",
			},
			expectedSQL:  "SELECT * FROM users WHERE id = ? AND name = ?",
			expectedArgs: []interface{}{123, "John"},
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
			params:      map[string]interface{}{},
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
			params:      map[string]interface{}{},
			expectedSQL: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generator := NewSQLGenerator(tc.instructions, tc.expressions, "postgresql")

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
		params       map[string]interface{}
		expectedSQL  string
		expectedArgs []interface{}
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
			params: map[string]interface{}{
				"include_active": true,
			},
			expectedSQL:  "SELECT * FROM users WHERE active = true",
			expectedArgs: []interface{}{},
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
			params: map[string]interface{}{
				"include_active": false,
			},
			expectedSQL:  "SELECT * FROM users",
			expectedArgs: []interface{}{},
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
			params: map[string]interface{}{
				"show_active": false,
			},
			expectedSQL:  "SELECT * FROM users WHERE active = false",
			expectedArgs: []interface{}{},
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			generator := NewSQLGenerator(tc.instructions, tc.expressions, "postgresql")

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
