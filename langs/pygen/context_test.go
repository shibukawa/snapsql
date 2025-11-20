package pygen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestSnapSQLContextDataclass(t *testing.T) {
	output := renderRuntimeCode(t)

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
		"allow_unsafe_mutations: bool = False",
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
	output := renderRuntimeCode(t)

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
	output := renderRuntimeCode(t)

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
	output := renderRuntimeCode(t)

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

func TestContextHelperFunctions(t *testing.T) {
	output := renderRuntimeCode(t)

	expectedHelpers := []string{
		"def set_system_value",
		"def update_system_values",
		"def set_row_lock_mode",
		"def set_mock_mode",
		"def allow_unsafe_queries",
		"def set_query_logger",
		"def enable_query_logging",
	}

	for _, helper := range expectedHelpers {
		if !strings.Contains(output, helper) {
			t.Errorf("Generate() output missing runtime helper: %q", helper)
		}
	}
}

func TestRuntimeRowLockHelpers(t *testing.T) {
	output := renderRuntimeCode(t)

	expected := []string{
		"ROW_LOCK_NONE = \"none\"",
		"ensure_row_lock_allowed",
		"build_row_lock_clause_postgres",
		"RowLockError",
	}

	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing row lock helper: %q", want)
		}
	}
}

func TestContextSystemValues(t *testing.T) {
	output := renderRuntimeCode(t)

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
	output := renderRuntimeCode(t)

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
	output := renderRuntimeCode(t)

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
	output := renderRuntimeCode(t)

	// Verify row locking field
	if !strings.Contains(output, `row_lock_mode: str = "none"`) {
		t.Error("Generate() output missing row_lock_mode field")
	}
}

func TestContextDocstring(t *testing.T) {
	output := renderRuntimeCode(t)

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
	// Ensure that runtime import and remaining sections are present
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

	responseIdx := strings.Index(output, "# Response Structures")
	functionIdx := strings.Index(output, "# Generated Functions")

	if responseIdx == -1 {
		t.Error("Generate() output missing Response Structures section")
	}

	if functionIdx == -1 {
		t.Error("Generate() output missing Generated Functions section")
	}

	if responseIdx > functionIdx {
		t.Error("Response Structures should come before Generated Functions")
	}
}
