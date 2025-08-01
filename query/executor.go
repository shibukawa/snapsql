package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

// LoadIntermediateFormat loads an intermediate format from a file
func LoadIntermediateFormat(templateFile string) (*intermediate.IntermediateFormat, error) {
	// Read the file
	data, err := intermediate.FromJSON([]byte(templateFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}
	return data, nil
}

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

// Compiler is a simple interface for SQL template compilation
type Compiler interface {
	CompileInstructions(instructions []intermediate.Instruction) error
	Execute(params map[string]interface{}) (string, []interface{}, error)
}

// SimpleCompiler is a basic implementation of the Compiler interface
type SimpleCompiler struct {
	instructions []intermediate.Instruction
}

// NewCompiler creates a new SimpleCompiler
func NewCompiler() *SimpleCompiler {
	return &SimpleCompiler{}
}

// CompileInstructions compiles the given instructions
func (c *SimpleCompiler) CompileInstructions(instructions []intermediate.Instruction) error {
	c.instructions = instructions
	return nil
}

// Execute executes the compiled instructions with the given parameters
func (c *SimpleCompiler) Execute(params map[string]interface{}) (string, []interface{}, error) {
	// This is a simplified implementation
	// In a real implementation, this would execute the instructions
	// For now, we'll just return a placeholder SQL
	return "SELECT 1", nil, nil
}

// Execute executes a query using an intermediate format
func (e *Executor) Execute(ctx context.Context, format *intermediate.IntermediateFormat, params map[string]interface{}, options QueryOptions) (*QueryResult, error) {
	// Create a compiler
	compiler := NewCompiler()

	// Compile the template
	if err := compiler.CompileInstructions(format.Instructions); err != nil {
		return nil, fmt.Errorf("failed to compile template: %w", err)
	}

	// Execute the template with parameters
	sql, args, err := compiler.Execute(params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
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
