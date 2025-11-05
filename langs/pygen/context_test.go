package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestSnapSQLContextDataclass(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify SnapSQLContext dataclass structure
	expectedStrings := []string{
		"@dataclass",
		"class SnapSQLContext:",
		"system_values: Dict[str, Any] = None",
		"enable_query_log: bool = False",
		"query_logger: Optional[Any] = None",
		"mock_mode: bool = False",
		"mock_data: Optional[Dict[str, Any]] = None",
		"row_lock_mode: str = \"none\"",
		"def __post_init__(self):",
		"if self.system_values is None:",
		"self.system_values = {}",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing SnapSQLContext element: %q", want)
		}
	}
}

func TestSnapSQLContextMethods(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify SnapSQLContext methods
	expectedMethods := []string{
		"def get_system_value(self, name: str, default: Any = None) -> Any:",
		"return self.system_values.get(name, default)",
		"def set_system_value(self, name: str, value: Any) -> None:",
		"self.system_values[name] = value",
	}

	for _, want := range expectedMethods {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing SnapSQLContext method: %q", want)
		}
	}
}

func TestContextVarInitialization(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify contextvars initialization
	expectedStrings := []string{
		"import contextvars",
		"snapsql_ctx: contextvars.ContextVar[Optional[SnapSQLContext]] =",
		"contextvars.ContextVar('snapsql', default=None)",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing contextvars element: %q", want)
		}
	}
}

func TestGetSnapSQLContextHelper(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify get_snapsql_context() helper function
	expectedStrings := []string{
		"def get_snapsql_context() -> SnapSQLContext:",
		`"""Get current SnapSQL context or create default"""`,
		"ctx = snapsql_ctx.get()",
		"if ctx is None:",
		"ctx = SnapSQLContext()",
		"snapsql_ctx.set(ctx)",
		"return ctx",
	}

	for _, want := range expectedStrings {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing get_snapsql_context element: %q", want)
		}
	}
}

func TestContextSystemValues(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify system_values field is properly defined
	if !strings.Contains(output, "system_values: Dict[str, Any]") {
		t.Error("Generate() output missing system_values field")
	}

	// Verify initialization in __post_init__
	if !strings.Contains(output, "if self.system_values is None:") {
		t.Error("Generate() output missing system_values initialization")
	}
}

func TestContextQueryLogging(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify query logging fields
	expectedFields := []string{
		"enable_query_log: bool = False",
		"query_logger: Optional[Any] = None",
	}

	for _, want := range expectedFields {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing query logging field: %q", want)
		}
	}
}

func TestContextMockMode(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify mock mode fields
	expectedFields := []string{
		"mock_mode: bool = False",
		"mock_data: Optional[Dict[str, Any]] = None",
	}

	for _, want := range expectedFields {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing mock mode field: %q", want)
		}
	}
}

func TestContextRowLocking(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify row locking field
	if !strings.Contains(output, `row_lock_mode: str = "none"`) {
		t.Error("Generate() output missing row_lock_mode field")
	}
}

func TestContextDocstring(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify docstrings are present
	expectedDocstrings := []string{
		`"""SnapSQL execution context for managing system values and settings"""`,
		`"""Get system column value"""`,
		`"""Set system column value"""`,
		`"""Get current SnapSQL context or create default"""`,
	}

	for _, want := range expectedDocstrings {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing docstring: %q", want)
		}
	}
}

func TestContextIntegration(t *testing.T) {
	// Test that context management is properly integrated into the template
	format := &intermediate.IntermediateFormat{
		FunctionName: "TestFunc",
		Description:  "Test function",
	}

	gen := New(format, WithDialect(snapsql.DialectPostgres))

	var buf bytes.Buffer

	err := gen.Generate(&buf)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	output := buf.String()

	// Verify the order of sections
	contextIdx := strings.Index(output, "# Context Management")
	errorIdx := strings.Index(output, "# Error Classes")
	responseIdx := strings.Index(output, "# Response Structures")
	functionIdx := strings.Index(output, "# Generated Functions")

	if contextIdx == -1 {
		t.Error("Generate() output missing Context Management section")
	}

	if errorIdx == -1 {
		t.Error("Generate() output missing Error Classes section")
	}

	if responseIdx == -1 {
		t.Error("Generate() output missing Response Structures section")
	}

	if functionIdx == -1 {
		t.Error("Generate() output missing Generated Functions section")
	}

	// Verify sections are in correct order
	if contextIdx > errorIdx {
		t.Error("Context Management should come before Error Classes")
	}

	if errorIdx > responseIdx {
		t.Error("Error Classes should come before Response Structures")
	}

	if responseIdx > functionIdx {
		t.Error("Response Structures should come before Generated Functions")
	}
}
