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
	"github.com/google/cel-go/cel"
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

// executeDryRun generates SQL without executing it
func (q *QueryCmd) executeDryRun(ctx *Context, params map[string]any, options query.QueryOptions) error {
	// Load intermediate format
	format, err := query.LoadIntermediateFormat(q.TemplateFile)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Generate SQL using the same logic as actual execution
	dialect := q.getDialectFromOptions(options)
	optimizedInstructions, err := intermediate.OptimizeInstructions(format.Instructions, dialect)
	if err != nil {
		return fmt.Errorf("failed to optimize instructions: %w", err)
	}

	// Build SQL and arguments
	sql, args, err := q.buildSQLFromOptimized(optimizedInstructions, format, params)
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	// Display results
	if !ctx.Quiet {
		color.Blue("Template File: %s", q.TemplateFile)
		fmt.Println()

		color.Blue("Generated SQL:")
		fmt.Println(sql)
		fmt.Println()

		if len(args) > 0 {
			color.Blue("Parameters:")
			for i, arg := range args {
				fmt.Printf("  $%d: %v (%T)\n", i+1, arg, arg)
			}
			fmt.Println()
		}

		if len(params) > 0 {
			color.Blue("Input Parameters:")
			q.printParameters(params, "")
			fmt.Println()
		}

		// Check for dangerous queries
		if q.isDangerousQuery(sql) {
			color.Red("⚠️  WARNING: This query appears to be dangerous (DELETE/UPDATE without WHERE clause)")
			color.Red("   Use --execute-dangerous-query flag to execute it anyway")
		}
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

// getDialectFromOptions determines the dialect from query options
func (q *QueryCmd) getDialectFromOptions(options query.QueryOptions) string {
	switch options.Driver {
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

// buildSQLFromOptimized builds SQL from optimized instructions (for dry-run)
func (q *QueryCmd) buildSQLFromOptimized(instructions []intermediate.OptimizedInstruction, format *intermediate.IntermediateFormat, params map[string]any) (string, []any, error) {
	var builder strings.Builder
	var args []any

	// Create parameter map for evaluation
	paramMap := make(map[string]any)
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
