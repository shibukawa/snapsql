package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
)

// Error definitions
var (
	ErrDatabaseConnection        = errors.New("database connection failed")
	ErrQueryExecution            = errors.New("query execution failed")
	ErrInvalidOutputFormat       = errors.New("invalid output format")
	ErrInvalidParams             = errors.New("invalid parameters")
	ErrDangerousQuery            = errors.New("dangerous query detected")
	ErrUnsupportedTemplateFormat = errors.New("unsupported template extension")
)

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	FormatTable    OutputFormat = "table"
	FormatJSON     OutputFormat = "json"
	FormatCSV      OutputFormat = "csv"
	FormatYAML     OutputFormat = "yaml"
	FormatMarkdown OutputFormat = "markdown"
)

// QueryOptions contains options for query execution
type QueryOptions struct {
	// Database connection options
	Driver           string
	ConnectionString string
	Timeout          int

	// Query execution options
	Explain        bool
	ExplainAnalyze bool
	Limit          int
	Offset         int

	// Safety options
	ExecuteDangerousQuery bool

	// Output options
	Format     OutputFormat
	OutputFile string
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	// Query information
	SQL        string        `json:"sql"`
	Parameters []interface{} `json:"parameters"`
	Duration   time.Duration `json:"duration"`

	// Result data
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Count   int             `json:"count"`

	// For EXPLAIN queries
	ExplainPlan string `json:"explain_plan,omitempty"`
}

// Executor executes SQL queries using templates
type Executor struct {
	db *sql.DB
}

// NewExecutor creates a new query executor
func NewExecutor(db *sql.DB) *Executor {
	return &Executor{
		db: db,
	}
}

// LoadIntermediateFormat is now implemented in template_loader.go

// ExecuteWithTemplate executes a query using a template file
func (e *Executor) ExecuteWithTemplate(ctx context.Context, templateFile string, params map[string]interface{}, options QueryOptions) (*QueryResult, error) {
	// Load template
	format, err := LoadIntermediateFormat(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Preflight parameter validation before any evaluation
	if err := ValidateParameters(format, params); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	// Static fast-path: if there are no dynamic instructions and no CEL expressions,
	// fall back to executing the original SQL text. This preserves constructs like CTEs
	// that are not yet reconstructed by the instruction pipeline.
	dialect := getDialectFromDriver(options.Driver)

	optimized, _ := intermediate.OptimizeInstructions(format.Instructions, dialect)
	if !intermediate.HasDynamicInstructions(optimized) && len(format.CELExpressions) == 0 {
		sqlText, readErr := readOriginalSQL(templateFile)
		if readErr == nil && sqlText != "" {
			// Dangerous query check
			sqlText = addLimitOffsetIfNeeded(sqlText, options)

			sqlText = FormatSQLForDriver(sqlText, options.Driver)
			if IsDangerousQuery(sqlText) && !options.ExecuteDangerousQuery {
				return nil, fmt.Errorf("%w: query contains DELETE/UPDATE without WHERE clause. Use --execute-dangerous-query flag to execute anyway", ErrDangerousQuery)
			}

			return e.executeSQL(ctx, sqlText, nil, options)
		}
		// If reading original SQL failed, fall back to pipeline execution
	}

	return e.Execute(ctx, format, params, options)
}

// IsDangerousQuery checks if a query is dangerous (DELETE/UPDATE without WHERE)
func IsDangerousQuery(sql string) bool {
	// Normalize SQL by removing extra whitespace and converting to uppercase
	normalizedSQL := strings.ToUpper(strings.TrimSpace(sql))

	// Check for DELETE without WHERE
	if strings.HasPrefix(normalizedSQL, "DELETE FROM") && !strings.Contains(normalizedSQL, "WHERE") {
		return true
	}

	// Check for UPDATE without WHERE
	if strings.HasPrefix(normalizedSQL, "UPDATE") && !strings.Contains(normalizedSQL, "WHERE") {
		return true
	}

	return false
}

// isWriteWithoutReturning detects INSERT/UPDATE/DELETE without RETURNING clause
func isWriteWithoutReturning(sql string) bool {
	s := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(s, "INSERT") || strings.HasPrefix(s, "UPDATE") || strings.HasPrefix(s, "DELETE") {
		// crude check: no RETURNING keyword
		return !strings.Contains(s, " RETURNING ") && !strings.HasSuffix(s, " RETURNING")
	}

	return false
}

// buildSQLFromOptimized builds SQL from optimized instructions
func (e *Executor) buildSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat, params map[string]interface{}) (string, []interface{}, error) {
	var (
		builder strings.Builder
		args    []interface{}
		// Boundary handling state
		deferredTokens    []string // tokens seen as EMIT_UNLESS_BOUNDARY since last boundary
		hasContentSinceBd bool     // true if any EMIT_STATIC appended since last boundary
	)

	// Create parameter map for evaluation
	paramMap := make(map[string]interface{})
	for k, v := range params {
		paramMap[k] = v
	}

	// Create CEL programs for expressions
	celPrograms := make(map[int]*cel.Program)

	// Declare variables for CEL: params map + individual keys
	decls := []cel.EnvOption{cel.Variable("params", cel.MapType(cel.StringType, cel.AnyType))}
	for k := range paramMap {
		decls = append(decls, cel.Variable(k, cel.AnyType))
	}

	env, err := cel.NewEnv(decls...)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	for i, expr := range format.CELExpressions {
		ast, issues := env.Compile(expr.Expression)
		if issues.Err() != nil {
			return "", nil, fmt.Errorf("failed to compile expression %d (%s): %w", i, expr.Expression, issues.Err())
		}

		program, err := env.Program(ast)
		if err != nil {
			return "", nil, fmt.Errorf("failed to create program for expression %d: %w", i, err)
		}

		celPrograms[i] = &program
	}

	flushDeferred := func() {
		if len(deferredTokens) > 0 {
			for _, tok := range deferredTokens {
				builder.WriteString(tok)
			}

			deferredTokens = nil
		}
	}

	// Process optimized instructions
	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			// When we see the first content after deferred boundary tokens, flush them first.
			if len(deferredTokens) > 0 && !isOnlyWhitespace(inst.Value) {
				flushDeferred()
			}

			if inst.Value != "" {
				builder.WriteString(inst.Value)

				if !isOnlyWhitespace(inst.Value) {
					hasContentSinceBd = true
				}
			}

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				// Fast path: direct param lookup by expression string
				if *inst.ExprIndex >= 0 && *inst.ExprIndex < len(format.CELExpressions) {
					exprStr := format.CELExpressions[*inst.ExprIndex].Expression
					if v, ok := paramMap[exprStr]; ok {
						args = append(args, v)
						break
					}
				}

				program, exists := celPrograms[*inst.ExprIndex]
				if !exists {
					return "", nil, fmt.Errorf("%w: %d", snapsql.ErrExpressionIndexNotFound, *inst.ExprIndex)
				}
				// Provide both params map and individual variables
				evalParams := map[string]interface{}{"params": paramMap}
				for k, v := range paramMap {
					evalParams[k] = v
				}

				result, _, err := (*program).Eval(evalParams)
				if err != nil {
					return "", nil, fmt.Errorf("failed to evaluate expression %d: %w", *inst.ExprIndex, err)
				}

				args = append(args, result.Value())
			}

		case "EMIT_UNLESS_BOUNDARY":
			// Defer emission until we know content appears before the next boundary
			if inst.Value != "" {
				deferredTokens = append(deferredTokens, inst.Value)
			}

		case "BOUNDARY":
			// Reached boundary; emit deferred tokens only if content has appeared since previous boundary
			if hasContentSinceBd {
				flushDeferred()
			} else {
				// Drop deferred tokens
				deferredTokens = nil
			}
			// Reset for next region
			hasContentSinceBd = false

		default:
			// Ignore other control flow ops here (IF/ELSE/END/LOOP_* are resolved at optimization or not supported yet)
		}
	}

	// End: if we still have deferred tokens and we emitted content, flush them
	if hasContentSinceBd && len(deferredTokens) > 0 {
		flushDeferred()
	}

	return builder.String(), args, nil
}

// isOnlyWhitespace reports whether s consists of only whitespace characters
func isOnlyWhitespace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
	}

	return len(s) > 0
}

// getDialectFromDriver converts database driver name to dialect
func getDialectFromDriver(driver string) string {
	switch driver {
	case "postgres", "pgx":
		return "postgresql"
	case "mysql":
		return "mysql"
	case "sqlite3":
		return "sqlite"
	default:
		return "postgresql" // default
	}
}

// Execute executes a query using an intermediate format
func (e *Executor) Execute(ctx context.Context, format *intermediate.IntermediateFormat, params map[string]interface{}, options QueryOptions) (*QueryResult, error) {
	// Generate SQL from intermediate format using optimized instructions
	dialect := getDialectFromDriver(options.Driver)

	optimizedInstructions, err := intermediate.OptimizeInstructions(format.Instructions, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize instructions: %w", err)
	}

	// Build SQL and arguments
	sql, args, err := e.buildSQLFromOptimized(optimizedInstructions, format, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL: %w", err)
	}

	// Apply optional LIMIT/OFFSET for SELECT when not present in SQL
	sql = addLimitOffsetIfNeeded(sql, options)
	// Convert placeholders and ensure readability (shared logic)
	sql = FormatSQLForDriver(sql, options.Driver)

	// Check for dangerous queries
	if IsDangerousQuery(sql) && !options.ExecuteDangerousQuery {
		return nil, fmt.Errorf("%w: query contains DELETE/UPDATE without WHERE clause. Use --execute-dangerous-query flag to execute anyway", ErrDangerousQuery)
	}

	return e.executeSQL(ctx, sql, args, options)
}

// executeSQL runs the given SQL with args and formats the result according to options
func (e *Executor) executeSQL(ctx context.Context, sql string, args []interface{}, options QueryOptions) (*QueryResult, error) {
	// Create query context with timeout
	queryCtx := ctx

	if options.Timeout > 0 {
		var cancel context.CancelFunc

		queryCtx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Second)
		defer cancel()
	}

	startTime := time.Now()

	if isWriteWithoutReturning(sql) {
		res, err := e.db.ExecContext(queryCtx, sql, args...)
		duration := time.Since(startTime)

		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrQueryExecution, err)
		}

		ra, _ := res.RowsAffected()
		li, _ := res.LastInsertId()

		return &QueryResult{
			SQL:        sql,
			Parameters: args,
			Duration:   duration,
			Columns:    []string{"rows_affected", "last_insert_id"},
			Rows:       [][]interface{}{{ra, li}},
			Count:      1,
		}, nil
	}

	rows, err := e.db.QueryContext(queryCtx, sql, args...)
	duration := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueryExecution, err)
	}

	defer rows.Close()

	result := &QueryResult{SQL: sql, Parameters: args, Duration: duration}

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	result.Columns = columns

	if options.Explain {
		// Re-run with EXPLAIN prefix depending on driver
		_ = rows.Close()

		var explainSQL string

		switch options.Driver {
		case "sqlite3":
			explainSQL = "EXPLAIN QUERY PLAN " + sql
		case "postgres", "pgx":
			if options.ExplainAnalyze {
				explainSQL = "EXPLAIN ANALYZE " + sql
			} else {
				explainSQL = "EXPLAIN " + sql
			}
		default:
			explainSQL = "EXPLAIN " + sql
		}

		rows2, err2 := e.db.QueryContext(queryCtx, explainSQL, args...)
		if err2 != nil {
			return nil, fmt.Errorf("%w: %w", ErrQueryExecution, err2)
		}
		defer rows2.Close()

		// Aggregate rows into a plan string (driver-agnostic)
		cols, _ := rows2.Columns()
		vals := make([]interface{}, len(cols))

		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}

		var lines []string

		for rows2.Next() {
			if err := rows2.Scan(ptrs...); err != nil {
				return nil, fmt.Errorf("failed to scan explain row: %w", err)
			}
			// join values by space
			var sb strings.Builder

			for i, v := range vals {
				if i > 0 {
					sb.WriteString(" ")
				}

				sb.WriteString(formatValue(v))
			}

			lines = append(lines, sb.String())
		}

		if err := rows2.Err(); err != nil {
			return nil, fmt.Errorf("error during explain iteration: %w", err)
		}

		result.ExplainPlan = strings.Join(lines, "\n")

		return result, nil
	}

	var resultRows [][]interface{}

	values := make([]interface{}, len(columns))

	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowValues := make([]interface{}, len(columns))
		for i, v := range values {
			rowValues[i] = convertSQLValue(v)
		}

		resultRows = append(resultRows, rowValues)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	result.Rows = resultRows
	result.Count = len(resultRows)

	return result, nil
}

// addLimitOffsetIfNeeded appends LIMIT/OFFSET to SELECT when not present
func addLimitOffsetIfNeeded(sql string, options QueryOptions) string {
	if options.Limit <= 0 && options.Offset <= 0 {
		return sql
	}

	s := strings.TrimSpace(sql)

	upper := strings.ToUpper(s)
	if !strings.HasPrefix(upper, "SELECT") {
		return sql
	}
	// Skip if already contains LIMIT or OFFSET (naive check)
	if strings.Contains(upper, " LIMIT ") || strings.Contains(upper, " OFFSET ") {
		return sql
	}
	// Handle trailing semicolon
	hasSemi := strings.HasSuffix(s, ";")
	if hasSemi {
		s = strings.TrimSuffix(s, ";")
	}

	if options.Limit > 0 {
		s = s + fmt.Sprintf(" LIMIT %d", options.Limit)
	}

	if options.Offset > 0 {
		s = s + fmt.Sprintf(" OFFSET %d", options.Offset)
	}

	if hasSemi {
		s += ";"
	}

	return s
}

// convertPlaceholdersForDriver converts '?' placeholders to driver-specific syntax
func convertPlaceholdersForDriver(sql string, driver string) string {
	d := strings.ToLower(driver)
	if d != "postgres" && d != "pgx" && d != "postgresql" {
		return sql
	}

	var b strings.Builder

	n := 1
	inSingle := false
	inDouble := false

	for i := range len(sql) {
		ch := sql[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle

			b.WriteByte(ch)

			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble

			b.WriteByte(ch)

			continue
		}

		if ch == '?' && !inSingle && !inDouble {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))

			n++

			continue
		}

		b.WriteByte(ch)
	}

	return b.String()
}

// readOriginalSQL reads SQL content from .snap.sql or extracts SQL from .snap.md
func readOriginalSQL(path string) (string, error) {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".snap.sql") || strings.HasSuffix(lower, ".sql") {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}

	if strings.HasSuffix(lower, ".snap.md") || strings.HasSuffix(lower, ".md") {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		doc, err := markdownparser.Parse(f)
		if err != nil {
			return "", err
		}

		return doc.SQL, nil
	}

	return "", fmt.Errorf("%w: %s", ErrUnsupportedTemplateFormat, path)
}

// convertSQLValue converts SQL values to appropriate Go types
func convertSQLValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch value := v.(type) {
	case []byte:
		// Try to convert []byte to string or JSON
		str := string(value)

		// Check if it's a JSON object or array
		if (str[0] == '{' && str[len(str)-1] == '}') || (str[0] == '[' && str[len(str)-1] == ']') {
			var jsonValue interface{}

			err := json.Unmarshal(value, &jsonValue)
			if err == nil {
				return jsonValue
			}
		}

		return str
	default:
		return value
	}
}

// OpenDatabase opens a database connection
func OpenDatabase(driver, connectionString string, timeout int) (*sql.DB, error) {
	// Open database connection
	db, err := sql.Open(driver, connectionString)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
	}

	// Set connection parameters
	db.SetConnMaxLifetime(time.Duration(timeout) * time.Second)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
	}

	return db, nil
}
