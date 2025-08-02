package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/shibukawa/snapsql/intermediate"
)

// Error definitions
var (
	ErrDatabaseConnection   = errors.New("database connection failed")
	ErrQueryExecution       = errors.New("query execution failed")
	ErrInvalidOutputFormat  = errors.New("invalid output format")
	ErrMissingRequiredParam = errors.New("missing required parameter")
	ErrInvalidParams        = errors.New("invalid parameters")
	ErrDangerousQuery       = errors.New("dangerous query detected")
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

	// Apply system directives based on options
	// Note: This functionality needs to be reimplemented with the new format structure
	// For now, we'll skip this part

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

// buildSQLFromOptimized builds SQL from optimized instructions
func (e *Executor) buildSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat, params map[string]interface{}) (string, []interface{}, error) {
	var builder strings.Builder
	var args []interface{}

	// Create parameter map for evaluation
	paramMap := make(map[string]interface{})
	for k, v := range params {
		paramMap[k] = v
	}

	// Create CEL programs for expressions
	celPrograms := make(map[int]*cel.Program)
	env, err := cel.NewEnv()
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

	// Process optimized instructions
	for _, inst := range instructions {
		switch inst.Op {
		case "EMIT_STATIC":
			builder.WriteString(inst.Value)

		case "ADD_PARAM":
			if inst.ExprIndex != nil {
				program, exists := celPrograms[*inst.ExprIndex]
				if !exists {
					return "", nil, fmt.Errorf("expression index %d not found", *inst.ExprIndex)
				}

				// Evaluate expression
				result, _, err := (*program).Eval(paramMap)
				if err != nil {
					return "", nil, fmt.Errorf("failed to evaluate expression %d: %w", *inst.ExprIndex, err)
				}

				builder.WriteString("?")
				args = append(args, result.Value())
			}

		default:
			// For now, ignore other operations (they should be handled by optimization)
		}
	}

	return builder.String(), args, nil
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

	// Check for dangerous queries
	if IsDangerousQuery(sql) && !options.ExecuteDangerousQuery {
		return nil, fmt.Errorf("%w: query contains DELETE/UPDATE without WHERE clause. Use --execute-dangerous-query flag to execute anyway", ErrDangerousQuery)
	}

	// Create query context with timeout
	queryCtx := ctx
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Second)
		defer cancel()
	}

	// Execute query
	startTime := time.Now()
	rows, err := e.db.QueryContext(queryCtx, sql, args...)
	duration := time.Since(startTime)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryExecution, err)
	}
	defer rows.Close()

	// Process results
	result := &QueryResult{
		SQL:        sql,
		Parameters: args,
		Duration:   duration,
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}
	result.Columns = columns

	// For EXPLAIN queries, handle differently
	if options.Explain {
		// For EXPLAIN queries, we expect a single column with the plan
		var plan string
		var plans []string

		for rows.Next() {
			if err := rows.Scan(&plan); err != nil {
				return nil, fmt.Errorf("failed to scan explain plan: %w", err)
			}
			plans = append(plans, plan)
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error during row iteration: %w", err)
		}

		// Combine all plan lines
		result.ExplainPlan = ""
		for _, line := range plans {
			result.ExplainPlan += line + "\n"
		}

		return result, nil
	}

	// For regular queries, process all rows
	var resultRows [][]interface{}
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		// Scan row values
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Copy values
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
			if err := json.Unmarshal(value, &jsonValue); err == nil {
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
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
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
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	return db, nil
}
