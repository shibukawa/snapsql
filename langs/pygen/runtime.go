package pygen

import (
	"bytes"
	"text/template"
	"time"
)

type runtimeTemplateData struct {
	Timestamp string
}

// RuntimePublicSymbols lists the names exported from snapsql_runtime.py and re-exported via __all__.
var RuntimePublicSymbols = []string{
	"SnapSQLContext",
	"get_snapsql_context",
	"QueryLogMetadata",
	"QueryLogger",
	"DefaultQueryLogger",
	"SnapSQLError",
	"NotFoundError",
	"ValidationError",
	"DatabaseError",
	"UnsafeQueryError",
	"set_system_value",
	"update_system_values",
	"set_row_lock_mode",
	"set_mock_mode",
	"allow_unsafe_queries",
	"set_query_logger",
	"enable_query_logging",
	"ROW_LOCK_NONE",
	"ensure_row_lock_allowed",
	"build_row_lock_clause_postgres",
	"build_row_lock_clause_mysql",
	"build_row_lock_clause_mariadb",
	"build_row_lock_clause_sqlite",
	"RowLockError",
}

// RenderRuntimeModule returns the shared Python runtime module content.
func RenderRuntimeModule() (string, error) {
	data := runtimeTemplateData{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	tmpl, err := template.New("python_runtime").Parse(pythonRuntimeTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
