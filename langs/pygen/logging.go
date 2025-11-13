package pygen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
)

// QueryLogMetadata represents metadata for query logging in generated Python code
type QueryLogMetadata struct {
	FuncName   string
	SourceFile string
	Dialect    string
	QueryType  string // "select", "insert", "update", "delete"
}

// generateQueryLoggerProtocol generates Python Protocol definition for QueryLogger
func generateQueryLoggerProtocol() string {
	return `
from typing import Protocol, Optional, List, Any
from dataclasses import dataclass
from datetime import datetime


@dataclass
class QueryLogMetadata:
    """Query metadata for logging"""
    func_name: str
    source_file: str
    dialect: str
    query_type: str  # "select", "insert", "update", "delete"


class QueryLogger(Protocol):
    """Query logger protocol for logging database operations"""
    
    def set_query(self, sql: str, args: List[Any]) -> None:
        """Set query and arguments before execution"""
        ...
    
    async def write(
        self,
        metadata: QueryLogMetadata,
        duration_ms: float,
        row_count: Optional[int],
        error: Optional[Exception]
    ) -> None:
        """Write query log after execution"""
        ...
`
}

// generateDefaultQueryLogger generates Python implementation of DefaultQueryLogger
func generateDefaultQueryLogger() string {
	return `
class DefaultQueryLogger:
    """Default query logger implementation"""
    
    def __init__(self, slow_query_threshold_ms: float = 100.0):
        self.sql: Optional[str] = None
        self.args: Optional[List[Any]] = None
        self.logs: List[Dict[str, Any]] = []
        self.slow_query_threshold_ms = slow_query_threshold_ms
    
    def set_query(self, sql: str, args: List[Any]) -> None:
        """Set query and arguments before execution"""
        self.sql = sql
        self.args = args if args else []
    
    async def write(
        self,
        metadata: QueryLogMetadata,
        duration_ms: float,
        row_count: Optional[int],
        error: Optional[Exception]
    ) -> None:
        """Write query log after execution"""
        log_entry = {
            "timestamp": datetime.now().isoformat(),
            "func_name": metadata.func_name,
            "source_file": metadata.source_file,
            "dialect": metadata.dialect,
            "query_type": metadata.query_type,
            "sql": self.sql,
            "args": self.args,
            "duration_ms": duration_ms,
            "row_count": row_count,
            "error": str(error) if error else None,
            "is_slow": duration_ms > self.slow_query_threshold_ms
        }
        self.logs.append(log_entry)
        
        # Console output for errors and slow queries
        if error:
            print(f"[ERROR] {metadata.func_name}: {error} ({duration_ms:.2f}ms)")
        elif duration_ms > self.slow_query_threshold_ms:
            print(f"[SLOW] {metadata.func_name}: {duration_ms:.2f}ms")
    
    def get_logs(self) -> List[Dict[str, Any]]:
        """Get all logged entries"""
        return self.logs
    
    def get_slow_queries(self) -> List[Dict[str, Any]]:
        """Get slow query entries"""
        return [log for log in self.logs if log.get("is_slow", False)]
    
    def get_errors(self) -> List[Dict[str, Any]]:
        """Get error entries"""
        return [log for log in self.logs if log.get("error") is not None]
    
    def clear(self) -> None:
        """Clear all logs"""
        self.logs.clear()
        self.sql = None
        self.args = None
`
}

// updateSnapSQLContextForLogging updates the SnapSQLContext to include query logging
func updateSnapSQLContextForLogging() string {
	return `
@dataclass
class SnapSQLContext:
    """SnapSQL execution context for managing system values and settings"""
    # System column values (created_by, updated_by, etc.)
    system_values: Dict[str, Any] = None
    
    # Query logging
    enable_query_log: bool = False
    query_logger: Optional[QueryLogger] = None
    
    # Mock execution
    mock_mode: bool = False
    mock_data: Optional[Dict[str, Any]] = None
    
    # Row locking
    row_lock_mode: str = "none"  # none, share, update
    
    def __post_init__(self):
        if self.system_values is None:
            self.system_values = {}
    
    def get_system_value(self, name: str, default: Any = None) -> Any:
        """Get system column value"""
        return self.system_values.get(name, default)
    
    def set_system_value(self, name: str, value: Any) -> None:
        """Set system column value"""
        self.system_values[name] = value
`
}

// generateQueryExecutionWithLogging generates query execution code with logging support
func generateQueryExecutionWithLogging(
	dialect snapsql.Dialect,
	responseAffinity string,
	functionName string,
	queryType string,
) string {
	var code strings.Builder

	// Start timing
	code.WriteString("    # Query logging setup\n")
	code.WriteString("    ctx = get_snapsql_context()\n")
	code.WriteString("    logger = ctx.query_logger if ctx and ctx.enable_query_log else None\n")
	code.WriteString("    \n")
	code.WriteString("    if logger:\n")
	code.WriteString("        logger.set_query(sql, args)\n")
	code.WriteString("    \n")
	code.WriteString("    import time\n")
	code.WriteString("    start_time = time.time()\n")
	code.WriteString("    error: Optional[Exception] = None\n")
	code.WriteString("    row_count: Optional[int] = None\n")
	code.WriteString("    \n")
	code.WriteString("    try:\n")

	// Query execution based on dialect and affinity
	switch responseAffinity {
	case "none":
		code.WriteString(generateNoneAffinityExecutionWithLogging(dialect, true))
	case "one":
		code.WriteString(generateOneAffinityExecutionWithLogging(dialect, true))
	case "many":
		code.WriteString(generateManyAffinityExecutionWithLogging(dialect, true))
	}

	// Error handling and logging
	code.WriteString("    except Exception as e:\n")
	code.WriteString("        error = e\n")
	code.WriteString("        raise\n")
	code.WriteString("    finally:\n")
	code.WriteString("        if logger:\n")
	code.WriteString("            duration_ms = (time.time() - start_time) * 1000\n")
	code.WriteString("            metadata = QueryLogMetadata(\n")
	code.WriteString(fmt.Sprintf("                func_name=%q,\n", functionName))
	code.WriteString(fmt.Sprintf("                source_file=\"queries/%s\",\n", functionName))
	code.WriteString(fmt.Sprintf("                dialect=%q,\n", dialect))
	code.WriteString(fmt.Sprintf("                query_type=%q\n", queryType))
	code.WriteString("            )\n")
	code.WriteString("            await logger.write(metadata, duration_ms, row_count, error)\n")

	return code.String()
}

// generateNoneAffinityExecutionWithLogging generates execution code for none affinity with logging
func generateNoneAffinityExecutionWithLogging(dialect snapsql.Dialect, withLogging bool) string {
	indent := "        "
	if !withLogging {
		indent = "    "
	}

	var code strings.Builder

	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(indent + "# Execute query (PostgreSQL)\n")
		code.WriteString(indent + "result = await conn.execute(sql, *args)\n")

		if withLogging {
			code.WriteString(indent + "row_count = int(result.split()[-1]) if result else 0\n")
		}

		code.WriteString(indent + "return row_count if row_count is not None else 0\n")

	case snapsql.DialectMySQL:
		code.WriteString(indent + "# Execute query (MySQL)\n")
		code.WriteString(indent + "async with cursor.execute(sql, args) as cur:\n")
		code.WriteString(indent + "    row_count = cur.rowcount\n")
		code.WriteString(indent + "return row_count if row_count is not None else 0\n")

	case snapsql.DialectSQLite:
		code.WriteString(indent + "# Execute query (SQLite)\n")
		code.WriteString(indent + "async with conn.execute(sql, args) as cur:\n")
		code.WriteString(indent + "    row_count = cur.rowcount\n")
		code.WriteString(indent + "await conn.commit()\n")
		code.WriteString(indent + "return row_count if row_count is not None else 0\n")
	default:
		panic(fmt.Sprintf("unsupported dialect for logging: %s", dialect))
	}

	return code.String()
}

// generateOneAffinityExecutionWithLogging generates execution code for one affinity with logging
func generateOneAffinityExecutionWithLogging(dialect snapsql.Dialect, withLogging bool) string {
	indent := "        "
	if !withLogging {
		indent = "    "
	}

	var code strings.Builder

	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(indent + "# Execute query and fetch one (PostgreSQL)\n")
		code.WriteString(indent + "row = await conn.fetchrow(sql, *args)\n")
		code.WriteString(indent + "if row is None:\n")
		code.WriteString(indent + "    raise NotFoundError(f\"Record not found\")\n")

		if withLogging {
			code.WriteString(indent + "row_count = 1\n")
		}

		code.WriteString(indent + "return result_class(**dict(row))\n")

	case snapsql.DialectMySQL:
		code.WriteString(indent + "# Execute query and fetch one (MySQL)\n")
		code.WriteString(indent + "await cursor.execute(sql, args)\n")
		code.WriteString(indent + "row = await cursor.fetchone()\n")
		code.WriteString(indent + "if row is None:\n")
		code.WriteString(indent + "    raise NotFoundError(f\"Record not found\")\n")

		if withLogging {
			code.WriteString(indent + "row_count = 1\n")
		}

		code.WriteString(indent + "return result_class(**row)\n")

	case snapsql.DialectSQLite:
		code.WriteString(indent + "# Execute query and fetch one (SQLite)\n")
		code.WriteString(indent + "async with conn.execute(sql, args) as cur:\n")
		code.WriteString(indent + "    row = await cur.fetchone()\n")
		code.WriteString(indent + "if row is None:\n")
		code.WriteString(indent + "    raise NotFoundError(f\"Record not found\")\n")

		if withLogging {
			code.WriteString(indent + "row_count = 1\n")
		}

		code.WriteString(indent + "return result_class(**dict(row))\n")
	default:
		panic(fmt.Sprintf("unsupported dialect for logging: %s", dialect))
	}

	return code.String()
}

// generateManyAffinityExecutionWithLogging generates execution code for many affinity with logging
func generateManyAffinityExecutionWithLogging(dialect snapsql.Dialect, withLogging bool) string {
	indent := "        "
	if !withLogging {
		indent = "    "
	}

	var code strings.Builder

	// For many affinity with logging, we need to count rows as we yield
	if withLogging {
		code.WriteString(indent + "row_count = 0\n")
	}

	switch dialect {
	case snapsql.DialectPostgres:
		code.WriteString(indent + "# Execute query and yield rows (PostgreSQL)\n")
		code.WriteString(indent + "async for row in conn.cursor(sql, *args):\n")

		if withLogging {
			code.WriteString(indent + "    row_count += 1\n")
		}

		code.WriteString(indent + "    yield result_class(**dict(row))\n")

	case snapsql.DialectMySQL:
		code.WriteString(indent + "# Execute query and yield rows (MySQL)\n")
		code.WriteString(indent + "await cursor.execute(sql, args)\n")
		code.WriteString(indent + "while True:\n")
		code.WriteString(indent + "    row = await cursor.fetchone()\n")
		code.WriteString(indent + "    if row is None:\n")
		code.WriteString(indent + "        break\n")

		if withLogging {
			code.WriteString(indent + "    row_count += 1\n")
		}

		code.WriteString(indent + "    yield result_class(**row)\n")

	case snapsql.DialectSQLite:
		code.WriteString(indent + "# Execute query and yield rows (SQLite)\n")
		code.WriteString(indent + "async with conn.execute(sql, args) as cur:\n")
		code.WriteString(indent + "    async for row in cur:\n")

		if withLogging {
			code.WriteString(indent + "        row_count += 1\n")
		}

		code.WriteString(indent + "        yield result_class(**dict(row))\n")
	default:
		panic(fmt.Sprintf("unsupported dialect for logging: %s", dialect))
	}

	return code.String()
}

// getQueryTypeFromStatementType converts statement type to query type for logging
func getQueryTypeFromStatementType(stmtType string) string {
	switch strings.ToLower(stmtType) {
	case "select":
		return "select"
	case "insert":
		return "insert"
	case "update":
		return "update"
	case "delete":
		return "delete"
	default:
		return "exec"
	}
}
