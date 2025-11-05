package pygen

import (
	"strings"
	"testing"
)

func TestGenerateQueryLoggerProtocol(t *testing.T) {
	protocol := generateQueryLoggerProtocol()

	// Check for Protocol definition
	if !strings.Contains(protocol, "class QueryLogger(Protocol)") {
		t.Error("Protocol definition not found")
	}

	// Check for set_query method
	if !strings.Contains(protocol, "def set_query(self, sql: str, args: List[Any])") {
		t.Error("set_query method not found")
	}

	// Check for write method
	if !strings.Contains(protocol, "async def write") {
		t.Error("write method not found")
	}

	// Check for QueryLogMetadata dataclass
	if !strings.Contains(protocol, "@dataclass") {
		t.Error("dataclass decorator not found")
	}

	if !strings.Contains(protocol, "class QueryLogMetadata:") {
		t.Error("QueryLogMetadata class not found")
	}

	// Check for metadata fields
	requiredFields := []string{"func_name", "source_file", "dialect", "query_type"}
	for _, field := range requiredFields {
		if !strings.Contains(protocol, field+": str") {
			t.Errorf("QueryLogMetadata field %s not found", field)
		}
	}
}

func TestGenerateDefaultQueryLogger(t *testing.T) {
	logger := generateDefaultQueryLogger()

	// Check for class definition
	if !strings.Contains(logger, "class DefaultQueryLogger:") {
		t.Error("DefaultQueryLogger class not found")
	}

	// Check for __init__ method
	if !strings.Contains(logger, "def __init__(self, slow_query_threshold_ms: float = 100.0)") {
		t.Error("__init__ method not found or incorrect signature")
	}

	// Check for set_query method
	if !strings.Contains(logger, "def set_query(self, sql: str, args: List[Any])") {
		t.Error("set_query method not found")
	}

	// Check for write method
	if !strings.Contains(logger, "async def write") {
		t.Error("write method not found")
	}

	// Check for helper methods
	helperMethods := []string{"get_logs", "get_slow_queries", "get_errors", "clear"}
	for _, method := range helperMethods {
		if !strings.Contains(logger, "def "+method) {
			t.Errorf("Helper method %s not found", method)
		}
	}

	// Check for slow query detection
	if !strings.Contains(logger, "is_slow") {
		t.Error("Slow query detection not found")
	}

	// Check for console output
	if !strings.Contains(logger, "[ERROR]") || !strings.Contains(logger, "[SLOW]") {
		t.Error("Console output for errors and slow queries not found")
	}
}

func TestUpdateSnapSQLContextForLogging(t *testing.T) {
	context := updateSnapSQLContextForLogging()

	// Check for SnapSQLContext class
	if !strings.Contains(context, "class SnapSQLContext:") {
		t.Error("SnapSQLContext class not found")
	}

	// Check for query logging fields
	if !strings.Contains(context, "enable_query_log: bool = False") {
		t.Error("enable_query_log field not found")
	}

	if !strings.Contains(context, "query_logger: Optional[QueryLogger] = None") {
		t.Error("query_logger field not found")
	}

	// Check for other context fields
	requiredFields := []string{"system_values", "mock_mode", "mock_data", "row_lock_mode"}
	for _, field := range requiredFields {
		if !strings.Contains(context, field) {
			t.Errorf("Context field %s not found", field)
		}
	}
}

func TestGenerateQueryExecutionWithLogging(t *testing.T) {
	tests := []struct {
		name             string
		dialect          string
		responseAffinity string
		functionName     string
		queryType        string
	}{
		{
			name:             "PostgreSQL SELECT with one affinity",
			dialect:          "postgres",
			responseAffinity: "one",
			functionName:     "get_user_by_id",
			queryType:        "select",
		},
		{
			name:             "MySQL INSERT with none affinity",
			dialect:          "mysql",
			responseAffinity: "none",
			functionName:     "insert_user",
			queryType:        "insert",
		},
		{
			name:             "SQLite SELECT with many affinity",
			dialect:          "sqlite",
			responseAffinity: "many",
			functionName:     "list_users",
			queryType:        "select",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateQueryExecutionWithLogging(
				tt.dialect,
				tt.responseAffinity,
				tt.functionName,
				tt.queryType,
			)

			// Check for logging setup
			if !strings.Contains(code, "ctx = get_snapsql_context()") {
				t.Error("Context retrieval not found")
			}

			if !strings.Contains(code, "logger = ctx.query_logger") {
				t.Error("Logger retrieval not found")
			}

			if !strings.Contains(code, "logger.set_query(sql, args)") {
				t.Error("set_query call not found")
			}

			// Check for timing
			if !strings.Contains(code, "import time") {
				t.Error("time import not found")
			}

			if !strings.Contains(code, "start_time = time.time()") {
				t.Error("start_time not found")
			}

			// Check for error handling
			if !strings.Contains(code, "try:") {
				t.Error("try block not found")
			}

			if !strings.Contains(code, "except Exception as e:") {
				t.Error("except block not found")
			}

			if !strings.Contains(code, "finally:") {
				t.Error("finally block not found")
			}

			// Check for metadata creation
			if !strings.Contains(code, "metadata = QueryLogMetadata(") {
				t.Error("QueryLogMetadata creation not found")
			}

			if !strings.Contains(code, "func_name=") {
				t.Error("func_name not set in metadata")
			}

			if !strings.Contains(code, tt.functionName) {
				t.Errorf("Function name %s not found in metadata", tt.functionName)
			}

			if !strings.Contains(code, "dialect=") {
				t.Error("dialect not set in metadata")
			}

			if !strings.Contains(code, tt.dialect) {
				t.Errorf("Dialect %s not found in metadata", tt.dialect)
			}

			if !strings.Contains(code, "query_type=") {
				t.Error("query_type not set in metadata")
			}

			if !strings.Contains(code, tt.queryType) {
				t.Errorf("Query type %s not found in metadata", tt.queryType)
			}

			// Check for write call
			if !strings.Contains(code, "await logger.write(metadata, duration_ms, row_count, error)") {
				t.Error("logger.write call not found")
			}

			// Check for duration calculation
			if !strings.Contains(code, "duration_ms = (time.time() - start_time) * 1000") {
				t.Error("duration_ms calculation not found")
			}
		})
	}
}

func TestGenerateNoneAffinityExecutionWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		dialect     string
		withLogging bool
	}{
		{"PostgreSQL with logging", "postgres", true},
		{"PostgreSQL without logging", "postgres", false},
		{"MySQL with logging", "mysql", true},
		{"MySQL without logging", "mysql", false},
		{"SQLite with logging", "sqlite", true},
		{"SQLite without logging", "sqlite", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateNoneAffinityExecutionWithLogging(tt.dialect, tt.withLogging)

			// Check for dialect-specific execution
			switch tt.dialect {
			case "postgres":
				if !strings.Contains(code, "await conn.execute(sql, *args)") {
					t.Error("PostgreSQL execute not found")
				}
			case "mysql":
				if !strings.Contains(code, "async with cursor.execute(sql, args)") {
					t.Error("MySQL execute not found")
				}
			case "sqlite":
				if !strings.Contains(code, "async with conn.execute(sql, args)") {
					t.Error("SQLite execute not found")
				}
			}

			// Check for row_count handling
			if tt.withLogging {
				if !strings.Contains(code, "row_count") {
					t.Error("row_count not found in logging version")
				}
			}

			// Check for return statement
			if !strings.Contains(code, "return") {
				t.Error("return statement not found")
			}
		})
	}
}

func TestGenerateOneAffinityExecutionWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		dialect     string
		withLogging bool
	}{
		{"PostgreSQL with logging", "postgres", true},
		{"MySQL with logging", "mysql", true},
		{"SQLite with logging", "sqlite", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateOneAffinityExecutionWithLogging(tt.dialect, tt.withLogging)

			// Check for NotFoundError
			if !strings.Contains(code, "raise NotFoundError") {
				t.Error("NotFoundError not found")
			}

			// Check for row_count in logging version
			if tt.withLogging {
				if !strings.Contains(code, "row_count = 1") {
					t.Error("row_count = 1 not found in logging version")
				}
			}

			// Check for result_class instantiation
			if !strings.Contains(code, "result_class(**") {
				t.Error("result_class instantiation not found")
			}
		})
	}
}

func TestGenerateManyAffinityExecutionWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		dialect     string
		withLogging bool
	}{
		{"PostgreSQL with logging", "postgres", true},
		{"MySQL with logging", "mysql", true},
		{"SQLite with logging", "sqlite", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateManyAffinityExecutionWithLogging(tt.dialect, tt.withLogging)

			// Check for yield statement
			if !strings.Contains(code, "yield result_class(**") {
				t.Error("yield statement not found")
			}

			// Check for row_count increment in logging version
			if tt.withLogging {
				if !strings.Contains(code, "row_count = 0") {
					t.Error("row_count initialization not found")
				}

				if !strings.Contains(code, "row_count += 1") {
					t.Error("row_count increment not found")
				}
			}

			// Check for async iteration
			if !strings.Contains(code, "async for") && !strings.Contains(code, "await") {
				t.Error("async iteration not found")
			}
		})
	}
}

func TestGetQueryTypeFromStatementType(t *testing.T) {
	tests := []struct {
		stmtType string
		expected string
	}{
		{"select", "select"},
		{"SELECT", "select"},
		{"insert", "insert"},
		{"INSERT", "insert"},
		{"update", "update"},
		{"UPDATE", "update"},
		{"delete", "delete"},
		{"DELETE", "delete"},
		{"unknown", "exec"},
		{"", "exec"},
	}

	for _, tt := range tests {
		t.Run(tt.stmtType, func(t *testing.T) {
			result := getQueryTypeFromStatementType(tt.stmtType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
