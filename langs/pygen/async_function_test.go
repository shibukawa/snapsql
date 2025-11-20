package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestAsyncFunctionSignature(t *testing.T) {
	tests := []struct {
		name         string
		functionName string
		dialect      snapsql.Dialect
		wantContains []string
	}{
		{
			name:         "postgres async function",
			functionName: "GetUserById",
			dialect:      "postgres",
			wantContains: []string{
				"async def get_user_by_id(",
				"conn: asyncpg.Connection,",
				") -> ",
			},
		},
		{
			name:         "mysql async function",
			functionName: "ListUsers",
			dialect:      "mysql",
			wantContains: []string{
				"async def list_users(",
				"cursor: Any,",
				") -> ",
			},
		},
		{
			name:         "sqlite async function",
			functionName: "UpdateUser",
			dialect:      "sqlite",
			wantContains: []string{
				"async def update_user(",
				"cursor: Union[aiosqlite.cursor.Cursor, aiosqlite.Connection],",
				") -> ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				FunctionName: tt.functionName,
				Description:  "Test function",
			}

			gen := New(format, WithDialect(tt.dialect))

			var buf bytes.Buffer

			err := gen.Generate(&buf)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			output := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing: %q", want)
				}
			}
		})
	}
}

func TestAsyncFunctionWithParameters(t *testing.T) {
	// This test will be fully implemented when parameter processing is done
	// For now, we verify the template structure supports parameters
	format := &intermediate.IntermediateFormat{
		FunctionName: "GetUser",
		Description:  "Get user by ID",
		Parameters: []intermediate.Parameter{
			{
				Name:     "user_id",
				Type:     "int",
				Optional: false,
			},
			{
				Name:     "include_deleted",
				Type:     "bool",
				Optional: true,
			},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify async function structure
	expectedStrings := []string{
		"async def get_user(",
		"conn: asyncpg.Connection,",
		") -> ",
		`"""`,
		"Get user by ID",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing: %q", want)
		}
	}
}

func TestAsyncGeneratorReturnType(t *testing.T) {
	// Test that AsyncGenerator is imported and available
	format := &intermediate.IntermediateFormat{
		FunctionName: "StreamUsers",
		Description:  "Stream users",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify AsyncGenerator is imported
	if !strings.Contains(output, "from typing import Optional, List, Any, Dict, AsyncGenerator") {
		t.Error("Generate() output missing AsyncGenerator import")
	}
}

func TestAsyncFunctionDocstring(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "GetUserById",
		Description:  "Retrieve a user by their unique identifier",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify docstring is present
	expectedStrings := []string{
		`"""`,
		"Retrieve a user by their unique identifier",
		"Args:",
		"Returns:",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing docstring element: %q", want)
		}
	}
}

func TestAsyncFunctionTypeHints(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "ProcessData",
		Description:  "Process data",
	}

	tests := []struct {
		name    string
		dialect snapsql.Dialect
		want    string
	}{
		{
			name:    "postgres connection type",
			dialect: snapsql.DialectPostgres,
			want:    "conn: asyncpg.Connection,",
		},
		{
			name:    "mysql cursor type",
			dialect: snapsql.DialectMySQL,
			want:    "cursor: Any,",
		},
		{
			name:    "sqlite cursor type",
			dialect: snapsql.DialectSQLite,
			want:    "cursor: Union[aiosqlite.cursor.Cursor, aiosqlite.Connection],",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := New(format, WithDialect(tt.dialect))

			var buf bytes.Buffer

			err := gen.Generate(&buf)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			output := buf.String()

			if !strings.Contains(output, tt.want) {
				t.Errorf("Generate() output missing type hint: %q", tt.want)
			}
		})
	}
}

func TestAsyncFunctionReturnTypeAnnotation(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "GetData",
		Description:  "Get data",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify return type annotation is present (type may vary)
	if !strings.Contains(output, ") -> ") {
		t.Error("Generate() output missing return type annotation")
	}
}
