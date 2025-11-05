package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

// TestSystemColumnsIntegration tests the full integration of system columns
// from intermediate format through code generation
func TestSystemColumnsIntegration(t *testing.T) {
	tests := []struct {
		name               string
		implicitParameters []intermediate.ImplicitParameter
		wantInCode         []string
	}{
		{
			name: "created_by from context",
			implicitParameters: []intermediate.ImplicitParameter{
				{
					Name: "createdBy",
					Type: "string",
				},
			},
			wantInCode: []string{
				"created_by: Optional[str] = None",
				"if created_by is None:",
				"created_by = ctx.get_system_value('created_by')",
			},
		},
		{
			name: "created_at with default value",
			implicitParameters: []intermediate.ImplicitParameter{
				{
					Name:    "createdAt",
					Type:    "timestamp",
					Default: "datetime.now()",
				},
			},
			wantInCode: []string{
				"created_at: Optional[datetime] = None",
				"if created_at is None:",
				"created_at = ctx.get_system_value('created_at')",
				"if created_at is None:",
				"created_at = datetime.now()",
			},
		},
		{
			name: "multiple system columns",
			implicitParameters: []intermediate.ImplicitParameter{
				{
					Name: "createdBy",
					Type: "string",
				},
				{
					Name: "updatedBy",
					Type: "string",
				},
				{
					Name:    "createdAt",
					Type:    "timestamp",
					Default: "datetime.now()",
				},
			},
			wantInCode: []string{
				"created_by: Optional[str] = None",
				"updated_by: Optional[str] = None",
				"created_at: Optional[datetime] = None",
				"if created_by is None:",
				"created_by = ctx.get_system_value('created_by')",
				"if updated_by is None:",
				"updated_by = ctx.get_system_value('updated_by')",
				"if created_at is None:",
				"created_at = ctx.get_system_value('created_at')",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create intermediate format with implicit parameters
			format := &intermediate.IntermediateFormat{
				FunctionName:       "insert_user",
				Description:        "Insert a new user",
				ImplicitParameters: tt.implicitParameters,
				Parameters: []intermediate.Parameter{
					{
						Name: "username",
						Type: "string",
					},
				},
			}

			// Create generator
			gen := New(format, WithDialect(snapsql.DialectPostgres))

			// Generate code
			var buf bytes.Buffer

			err := gen.Generate(&buf)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			generatedCode := buf.String()

			// Verify all expected strings are in the generated code
			for _, want := range tt.wantInCode {
				if !strings.Contains(generatedCode, want) {
					t.Errorf("Generated code missing expected string: %q", want)
					t.Logf("Generated code:\n%s", generatedCode)
				}
			}
		})
	}
}

// TestSystemColumnsContextRetrieval tests that system columns are properly
// retrieved from context in the generated code
func TestSystemColumnsContextRetrieval(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "insert_user",
		Description:  "Insert a new user with system columns",
		Parameters: []intermediate.Parameter{
			{
				Name: "username",
				Type: "string",
			},
		},
		ImplicitParameters: []intermediate.ImplicitParameter{
			{
				Name: "createdBy",
				Type: "string",
			},
			{
				Name:    "createdAt",
				Type:    "timestamp",
				Default: "datetime.now()",
			},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	generatedCode := buf.String()

	// Verify context retrieval pattern
	expectedPatterns := []string{
		// Function signature should have implicit params with Optional type
		"created_by: Optional[str] = None",
		"created_at: Optional[datetime] = None",

		// Context should be retrieved
		"ctx = get_snapsql_context()",

		// Values should be retrieved from context
		"created_by = ctx.get_system_value('created_by')",
		"created_at = ctx.get_system_value('created_at')",

		// Default value should be applied
		"created_at = datetime.now()",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generatedCode, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}

	// Verify the order: context retrieval happens before default value application
	ctxGetIndex := strings.Index(generatedCode, "ctx.get_system_value('created_at')")
	defaultIndex := strings.Index(generatedCode, "created_at = datetime.now()")

	if ctxGetIndex == -1 || defaultIndex == -1 {
		t.Fatal("Could not find context retrieval or default value application")
	}

	if ctxGetIndex >= defaultIndex {
		t.Error("Context retrieval should happen before default value application")
	}
}

// TestSystemColumnsWithRegularParameters tests that system columns work
// correctly alongside regular parameters
func TestSystemColumnsWithRegularParameters(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "insert_user",
		Description:  "Insert a new user",
		Parameters: []intermediate.Parameter{
			{
				Name:     "username",
				Type:     "string",
				Optional: false,
			},
			{
				Name:     "email",
				Type:     "string",
				Optional: false,
			},
			{
				Name:     "isActive",
				Type:     "bool",
				Optional: true,
			},
		},
		ImplicitParameters: []intermediate.ImplicitParameter{
			{
				Name: "createdBy",
				Type: "string",
			},
		},
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	generatedCode := buf.String()

	// Verify function signature has correct parameter order:
	// 1. Connection parameter
	// 2. Regular required parameters
	// 3. Regular optional parameters
	// 4. Keyword-only separator (*)
	// 5. Implicit parameters

	expectedSignature := []string{
		"conn: asyncpg.Connection",
		"username: str",
		"email: str",
		"is_active: bool = None",
		"*,",
		"created_by: Optional[str] = None",
	}

	for _, sig := range expectedSignature {
		if !strings.Contains(generatedCode, sig) {
			t.Errorf("Generated code missing expected signature part: %q", sig)
		}
	}

	// Verify implicit parameter is retrieved from context
	if !strings.Contains(generatedCode, "created_by = ctx.get_system_value('created_by')") {
		t.Error("Generated code missing context retrieval for created_by")
	}
}

// TestSystemColumnsTypeConversion tests that system column types are
// correctly converted to Python types
func TestSystemColumnsTypeConversion(t *testing.T) {
	tests := []struct {
		name         string
		implicitType string
		wantPyType   string
	}{
		{"string type", "string", "str"},
		{"int type", "int", "int"},
		{"timestamp type", "timestamp", "datetime"},
		{"bool type", "bool", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := &intermediate.IntermediateFormat{
				FunctionName: "test_func",
				Description:  "Test function",
				ImplicitParameters: []intermediate.ImplicitParameter{
					{
						Name: "testParam",
						Type: tt.implicitType,
					},
				},
			}

			gen := New(format, WithDialect(snapsql.DialectPostgres))

			var buf bytes.Buffer

			err := gen.Generate(&buf)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			generatedCode := buf.String()

			// Verify the type hint in function signature
			expectedTypeHint := "test_param: Optional[" + tt.wantPyType + "] = None"
			if !strings.Contains(generatedCode, expectedTypeHint) {
				t.Errorf("Generated code missing expected type hint: %q", expectedTypeHint)
				t.Logf("Generated code:\n%s", generatedCode)
			}
		})
	}
}
