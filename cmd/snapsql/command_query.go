package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/query"
)

// Error definitions
var (
	ErrTemplateNotFound     = errors.New("template file not found")
	ErrInvalidParams        = errors.New("invalid parameters")
	ErrDatabaseConnection   = errors.New("database connection failed")
	ErrQueryExecution       = errors.New("query execution failed")
	ErrInvalidOutputFormat  = errors.New("invalid output format")
	ErrOutputFileCreation   = errors.New("failed to create output file")
	ErrMissingRequiredParam = errors.New("missing required parameter")
)

// QueryCmd represents the query command
type QueryCmd struct {
	TemplateFile          string   `arg:"" help:"SQL template file (.snap.sql or .snap.md)" type:"path"`
	ParamsFile            string   `short:"p" long:"params" help:"Parameters file (JSON/YAML)" type:"path"`
	Param                 []string `long:"param" help:"Individual parameter (key=value format)"`
	ConstFiles            []string `long:"const" help:"Constant definition files" type:"path"`
	DBConnection          string   `long:"db" help:"Database connection string"`
	Environment           string   `long:"env" help:"Environment name from config"`
	Format                string   `long:"format" help:"Output format (table, json, csv, yaml, markdown)" default:"table"`
	OutputFile            string   `short:"o" long:"output" help:"Output file (defaults to stdout)" type:"path"`
	Timeout               int      `long:"timeout" help:"Query timeout in seconds" default:"30"`
	Explain               bool     `long:"explain" help:"Show query execution plan"`
	ExplainAnalyze        bool     `long:"explain-analyze" help:"Show detailed query execution plan with actual execution statistics"`
	Limit                 int      `long:"limit" help:"Limit number of rows returned"`
	Offset                int      `long:"offset" help:"Offset for result set"`
	ExecuteDangerousQuery bool     `long:"execute-dangerous-query" help:"Execute DELETE/UPDATE queries without WHERE clause (dangerous!)"`
	DryRun                bool     `long:"dry-run" help:"Show generated SQL without executing"`
}

// Run executes the query command
func (q *QueryCmd) Run(ctx *Context) error {
	// Load configuration
	config, err := LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Verify template file exists
	if !fileExists(q.TemplateFile) {
		return fmt.Errorf("%w: %s", ErrTemplateNotFound, q.TemplateFile)
	}

	// Load parameters
	params, err := q.loadParameters(ctx)
	if err != nil {
		return fmt.Errorf("failed to load parameters: %w", err)
	}

	// Load constants
	constants, err := q.loadConstants(config, ctx)
	if err != nil {
		return fmt.Errorf("failed to load constants: %w", err)
	}

	// Merge constants into parameters
	for k, v := range constants {
		// Don't override explicit parameters
		if _, exists := params[k]; !exists {
			params[k] = v
		}
	}

	// Create query options
	options := query.QueryOptions{
		Timeout:               q.Timeout,
		Format:                query.OutputFormat(strings.ToLower(q.Format)),
		OutputFile:            q.OutputFile,
		Explain:               q.Explain,
		ExplainAnalyze:        q.ExplainAnalyze,
		Limit:                 q.Limit,
		Offset:                q.Offset,
		ExecuteDangerousQuery: q.ExecuteDangerousQuery,
	}

	// If explain-analyze is set, ensure explain is also set
	if options.ExplainAnalyze {
		options.Explain = true
	}

	// If ExecuteDangerousQuery is not set in command line, check config
	if !q.ExecuteDangerousQuery {
		options.ExecuteDangerousQuery = config.Query.ExecuteDangerousQuery
	} else {
		options.ExecuteDangerousQuery = true
	}

	// Validate output format
	if !query.IsValidOutputFormat(q.Format) {
		return fmt.Errorf("%w: %s", ErrInvalidOutputFormat, q.Format)
	}

	// If dry run, just generate SQL and exit (no database connection needed)
	if q.DryRun {
		return q.executeDryRun(ctx, params, options)
	}

	// Get database connection (only needed for actual execution)
	driver, connectionString, err := q.getDatabaseConnection(config, ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}
	options.Driver = driver
	options.ConnectionString = connectionString

	// Execute query
	return q.executeQuery(ctx, params, options)
}

// loadParameters loads parameters from file and command line
func (q *QueryCmd) loadParameters(ctx *Context) (map[string]any, error) {
	params := make(map[string]any)

	// Load from file if specified
	if q.ParamsFile != "" {
		if !fileExists(q.ParamsFile) {
			return nil, fmt.Errorf("parameters file not found: %s", q.ParamsFile)
		}

		data, err := os.ReadFile(q.ParamsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read parameters file: %w", err)
		}

		// Determine format based on extension
		ext := strings.ToLower(filepath.Ext(q.ParamsFile))
		if ext == ".json" {
			if err := json.Unmarshal(data, &params); err != nil {
				return nil, fmt.Errorf("failed to parse JSON parameters: %w", err)
			}
		} else if ext == ".yaml" || ext == ".yml" {
			if err := yaml.Unmarshal(data, &params); err != nil {
				return nil, fmt.Errorf("failed to parse YAML parameters: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unsupported parameters file format: %s", ext)
		}

		if ctx.Verbose {
			color.Blue("Loaded parameters from %s", q.ParamsFile)
		}
	}

	// Add command line parameters (overriding file parameters)
	for _, param := range q.Param {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: parameter must be in key=value format: %s", ErrInvalidParams, param)
		}

		key := parts[0]
		value := parts[1]

		// Try to parse as JSON if it looks like a complex value
		if (strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}")) ||
			(strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]")) {
			var jsonValue any
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				params[key] = jsonValue
				continue
			}
		}

		// Handle boolean values
		if value == "true" {
			params[key] = true
			continue
		}
		if value == "false" {
			params[key] = false
			continue
		}

		// Handle numeric values
		if strings.Contains(value, ".") {
			// Try as float
			if floatVal, err := parseFloat(value); err == nil {
				params[key] = floatVal
				continue
			}
		} else {
			// Try as integer
			if intVal, err := parseInt(value); err == nil {
				params[key] = intVal
				continue
			}
		}

		// Default to string
		params[key] = value
	}

	return params, nil
}

// loadConstants loads constant files
func (q *QueryCmd) loadConstants(config *Config, ctx *Context) (map[string]any, error) {
	constants := make(map[string]any)

	// Combine constant files from config and command line
	constFiles := append([]string{}, config.ConstantFiles...)
	constFiles = append(constFiles, q.ConstFiles...)

	// Load each constant file
	for _, file := range constFiles {
		if !fileExists(file) {
			if ctx.Verbose {
				color.Yellow("Constant file not found: %s", file)
			}
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read constant file %s: %w", file, err)
		}

		var fileConstants map[string]any
		if err := yaml.Unmarshal(data, &fileConstants); err != nil {
			return nil, fmt.Errorf("failed to parse constant file %s: %w", file, err)
		}

		// Merge constants
		for k, v := range fileConstants {
			constants[k] = v
		}

		if ctx.Verbose {
			color.Blue("Loaded constants from %s", file)
		}
	}

	return constants, nil
}

// getDatabaseConnection gets database connection information
func (q *QueryCmd) getDatabaseConnection(config *Config, ctx *Context) (string, string, error) {
	var connectionString string
	var driver string

	// Get connection string from environment or direct specification
	if q.Environment != "" {
		// Get from config
		dbConfig, exists := config.Databases[q.Environment]
		if !exists {
			return "", "", fmt.Errorf("environment not found in config: %s", q.Environment)
		}
		connectionString = dbConfig.Connection
		driver = dbConfig.Driver
	} else if q.DBConnection != "" {
		// Direct connection string
		connectionString = q.DBConnection
		// Try to determine driver from connection string
		driver = determineDriver(connectionString)
	} else {
		// Try default environment from config
		if config.Query.DefaultEnvironment != "" {
			dbConfig, exists := config.Databases[config.Query.DefaultEnvironment]
			if !exists {
				return "", "", fmt.Errorf("default environment not found in config: %s", config.Query.DefaultEnvironment)
			}
			connectionString = dbConfig.Connection
			driver = dbConfig.Driver
		} else {
			return "", "", fmt.Errorf("no database connection specified")
		}
	}

	if ctx.Verbose {
		color.Blue("Using database driver: %s", driver)
	}

	return driver, connectionString, nil
}

// executeDryRun generates SQL without executing it
func (q *QueryCmd) executeDryRun(ctx *Context, params map[string]any, options query.QueryOptions) error {
	// For dry run, we don't need a database connection
	// Just show the template and parameters
	
	// Read template file
	content, err := os.ReadFile(q.TemplateFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Print template content and parameters
	if !ctx.Quiet {
		color.Blue("Template File: %s", q.TemplateFile)
		fmt.Println()
		
		color.Blue("Template Content:")
		fmt.Println(string(content))
		fmt.Println()

		if len(params) > 0 {
			color.Blue("Parameters:")
			q.printParameters(params, "")
		}
		
		color.Yellow("\nNote: This is a dry run. The actual SQL would be generated by processing")
		color.Yellow("the template with the provided parameters at runtime.")
	}

	return nil
}

// executeQuery executes the query and outputs results
func (q *QueryCmd) executeQuery(ctx *Context, params map[string]any, options query.QueryOptions) error {
	// Open database connection
	db, err := query.OpenDatabase(options.Driver, options.ConnectionString, options.Timeout)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}
	defer db.Close()

	// Create executor
	executor := query.NewExecutor(db)

	// Execute query
	result, err := executor.ExecuteWithTemplate(context.Background(), q.TemplateFile, params, options)
	if err != nil {
		// Special handling for dangerous query errors
		if strings.Contains(err.Error(), "dangerous query detected") {
			if !ctx.Quiet {
				color.Red("ERROR: %v", err)
				color.Red("\nThis query contains DELETE or UPDATE without a WHERE clause, which could affect all rows in the table.")
				color.Red("To execute this query anyway, use the --execute-dangerous-query flag.")
			}
			return err
		}
		return fmt.Errorf("%w: %v", ErrQueryExecution, err)
	}

	// Determine output destination
	var output *os.File
	if q.OutputFile != "" {
		// Create output file
		file, err := os.Create(q.OutputFile)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrOutputFileCreation, err)
		}
		defer file.Close()
		output = file
	} else {
		// Use stdout
		output = os.Stdout
	}

	// Create formatter
	formatter := query.NewFormatter(options.Format)

	// Format and output results
	if options.Explain {
		if err := formatter.FormatExplain(result, output); err != nil {
			return fmt.Errorf("failed to format explain results: %w", err)
		}
	} else {
		if err := formatter.Format(result, output); err != nil {
			return fmt.Errorf("failed to format results: %w", err)
		}
	}

	return nil
}

// Helper functions

// determineDriver determines the database driver from connection string
func determineDriver(connectionString string) string {
	if strings.HasPrefix(connectionString, "postgres://") {
		return "postgres"
	}
	if strings.HasPrefix(connectionString, "mysql://") {
		return "mysql"
	}
	if strings.HasPrefix(connectionString, "sqlite://") || strings.HasSuffix(connectionString, ".db") {
		return "sqlite3"
	}
	// Default to postgres
	return "postgres"
}

// parseInt parses a string as an integer
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// parseFloat parses a string as a float
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// generateSQLFromTemplate generates SQL from template without database connection
func (q *QueryCmd) generateSQLFromTemplate(templateContent string, params map[string]any) (string, []any, error) {
	// Use the intermediate format to generate SQL
	reader := strings.NewReader(templateContent)
	
	// Generate intermediate format
	format, err := intermediate.GenerateFromSQL(reader, params, q.TemplateFile, ".", nil, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate intermediate format: %w", err)
	}

	// Execute the instructions to generate SQL
	sql, sqlParams, err := q.executeInstructions(format.Instructions, format.Expressions, params)
	if err != nil {
		return "", nil, fmt.Errorf("failed to execute instructions: %w", err)
	}

	return sql, sqlParams, nil
}

// executeInstructions executes intermediate format instructions to generate SQL
func (q *QueryCmd) executeInstructions(instructions []intermediate.Instruction, expressions []string, params map[string]any) (string, []any, error) {
	var result strings.Builder
	var sqlParams []any
	
	for _, instr := range instructions {
		switch instr.Op {
		case intermediate.OpEmitStatic:
			result.WriteString(instr.Value)
		case intermediate.OpEmitEval:
			if instr.ExprIndex != nil && *instr.ExprIndex >= 0 && *instr.ExprIndex < len(expressions) {
				expr := expressions[*instr.ExprIndex]
				value, err := q.evaluateExpression(expr, params)
				if err != nil {
					return "", nil, fmt.Errorf("failed to evaluate expression %s: %w", expr, err)
				}
				// Format the value appropriately
				formatted := q.formatValue(value)
				result.WriteString(formatted)
			} else if instr.ExprIndex != nil && *instr.ExprIndex == -1 {
				// Handle special case for -1 index (might be a placeholder)
				result.WriteString("NULL")
			}
		case intermediate.OpEmitUnlessBoundary:
			// For dry run, we'll emit the value
			result.WriteString(instr.Value)
		case intermediate.OpBoundary:
			// Skip boundary markers in dry run
		case intermediate.OpIf:
			// For dry run, we'll include conditional content
			// In a real implementation, this would need proper condition evaluation
		case intermediate.OpEnd:
			// End of conditional block
		case intermediate.OpEmitIfDialect:
			// For dry run, emit the SQL fragment (assume PostgreSQL)
			if len(instr.Dialects) > 0 && contains(instr.Dialects, "postgresql") {
				result.WriteString(instr.SqlFragment)
			} else if len(instr.Dialects) > 0 {
				result.WriteString(instr.SqlFragment)
			}
		default:
			// For other operations, we'll skip them in dry run mode
		}
	}
	
	return result.String(), sqlParams, nil
}

// evaluateExpression evaluates a simple expression against parameters
func (q *QueryCmd) evaluateExpression(expr string, params map[string]any) (any, error) {
	// Simple dot notation evaluation
	parts := strings.Split(expr, ".")
	current := params
	
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the value
			if val, ok := current[part]; ok {
				return val, nil
			}
			return nil, fmt.Errorf("parameter %s not found", expr)
		} else {
			// Intermediate part - navigate deeper
			if val, ok := current[part]; ok {
				if nested, ok := val.(map[string]any); ok {
					current = nested
				} else {
					return nil, fmt.Errorf("parameter %s is not an object", strings.Join(parts[:i+1], "."))
				}
			} else {
				return nil, fmt.Errorf("parameter %s not found", strings.Join(parts[:i+1], "."))
			}
		}
	}
	
	return nil, fmt.Errorf("parameter %s not found", expr)
}

// isDangerousQuery checks if a query is potentially dangerous
func (q *QueryCmd) isDangerousQuery(sql string) bool {
	sqlUpper := strings.ToUpper(strings.TrimSpace(sql))
	
	// Check for DELETE without WHERE
	if strings.HasPrefix(sqlUpper, "DELETE") && !strings.Contains(sqlUpper, "WHERE") {
		return true
	}
	
	// Check for UPDATE without WHERE
	if strings.HasPrefix(sqlUpper, "UPDATE") && !strings.Contains(sqlUpper, "WHERE") {
		return true
	}
	
	return false
}

// formatValue formats a value for SQL output
func (q *QueryCmd) formatValue(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case []any:
		var parts []string
		for _, item := range v {
			parts = append(parts, q.formatValue(item))
		}
		return strings.Join(parts, ", ")
	case []string:
		var parts []string
		for _, item := range v {
			parts = append(parts, fmt.Sprintf("'%s'", strings.ReplaceAll(item, "'", "''")))
		}
		return strings.Join(parts, ", ")
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// printParameters recursively prints parameters with indentation
func (q *QueryCmd) printParameters(params map[string]any, indent string) {
	for key, value := range params {
		switch v := value.(type) {
		case map[string]any:
			fmt.Printf("%s%s:\n", indent, key)
			q.printParameters(v, indent+"  ")
		case []any:
			fmt.Printf("%s%s: [", indent, key)
			for i, item := range v {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%v", item)
			}
			fmt.Println("]")
		default:
			fmt.Printf("%s%s: %v\n", indent, key, value)
		}
	}
}
