package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/google/cel-go/cel"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/explain"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/query"
)

// Error definitions
var (
	ErrTemplateNotFound           = errors.New("template file not found")
	ErrInvalidParams              = errors.New("invalid parameters")
	ErrDatabaseConnection         = errors.New("database connection failed")
	ErrQueryExecution             = errors.New("query execution failed")
	ErrInvalidOutputFormat        = errors.New("invalid output format")
	ErrOutputFileCreation         = errors.New("failed to create output file")
	ErrMissingRequiredParam       = errors.New("missing required parameter")
	ErrParametersFileNotFound     = errors.New("parameters file not found")
	ErrUnsupportedParamsFormat    = errors.New("unsupported parameters file format")
	ErrDefaultEnvironmentNotFound = errors.New("default environment not found in config")
	ErrNoDatabaseConnection       = errors.New("no database connection specified")
	ErrExpressionIndexNotFound    = errors.New("expression index not found")
	ErrInvalidDialect             = errors.New("invalid dialect")
)

// QueryCmd represents the query command
type QueryCmd struct {
	TemplateFile string   `arg:"" help:"SQL template file (.snap.sql or .snap.md)" type:"path"`
	ParamsFile   string   `short:"P" long:"params" help:"Parameters file (JSON/YAML)" type:"path"`
	Param        []string `short:"p" long:"param" help:"Individual parameter (key=value format)"`
	ConstFiles   []string `long:"const" help:"Constant definition files" type:"path"`
	DBConnection string   `long:"db" help:"Database connection string"`
	// Environment flag removed; use --tbls-config to point to a tbls config if needed
	Format                string `long:"format" help:"Output format (table, json, csv, yaml, markdown)" default:"table"`
	OutputFile            string `short:"o" long:"output" help:"Output file (defaults to stdout)" type:"path"`
	Timeout               int    `long:"timeout" help:"Query timeout in seconds" default:"30"`
	Explain               bool   `long:"explain" help:"Show query execution plan"`
	ExplainAnalyze        bool   `long:"explain-analyze" help:"Show detailed query execution plan with actual execution statistics"`
	Limit                 int    `long:"limit" help:"Limit number of rows returned"`
	Offset                int    `long:"offset" help:"Offset for result set"`
	ExecuteDangerousQuery bool   `long:"execute-dangerous-query" help:"Execute DELETE/UPDATE queries without WHERE clause (dangerous!)"`
	DryRun                bool   `long:"dry-run" help:"Show generated SQL without executing"`
	Dialect               string `long:"dialect" help:"SQL dialect for dry-run or when no DB (postgresql|mysql|sqlite|mariadb)"`
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

	tableMetadata := buildTableMetadataFromConfig(config.Tables)

	slowThreshold := config.Performance.SlowQueryThreshold
	if slowThreshold <= 0 {
		slowThreshold = 3 * time.Second
	}

	slowThreshold = q.extractSlowQueryThreshold(slowThreshold)

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
		return fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
	}

	options.Driver = driver
	options.ConnectionString = connectionString

	// Execute query
	return q.executeQuery(ctx, params, options, slowThreshold, tableMetadata)
}

// loadParameters loads parameters from file and command line
func (q *QueryCmd) loadParameters(ctx *Context) (map[string]any, error) {
	params := make(map[string]any)

	// Load from file if specified
	if q.ParamsFile != "" {
		if !fileExists(q.ParamsFile) {
			return nil, fmt.Errorf("%w: %s", ErrParametersFileNotFound, q.ParamsFile)
		}

		data, err := os.ReadFile(q.ParamsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read parameters file: %w", err)
		}

		// Determine format based on extension
		ext := strings.ToLower(filepath.Ext(q.ParamsFile))
		switch ext {
		case ".json":
			err := json.Unmarshal(data, &params)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON parameters: %w", err)
			}
		case ".yaml", ".yml":
			err := yaml.Unmarshal(data, &params)
			if err != nil {
				return nil, fmt.Errorf("failed to parse YAML parameters: %w", err)
			}
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedParamsFormat, ext)
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

			err := json.Unmarshal([]byte(value), &jsonValue)
			if err == nil {
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
func (q *QueryCmd) loadConstants(config *snapsql.Config, ctx *Context) (map[string]any, error) {
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
		maps.Copy(constants, fileConstants)

		if ctx.Verbose {
			color.Blue("Loaded constants from %s", file)
		}
	}

	return constants, nil
}

// getDatabaseConnection gets database connection information
func (q *QueryCmd) getDatabaseConnection(config *snapsql.Config, ctx *Context) (string, string, error) {
	var (
		connectionString string
		driver           string
	)

	// Get connection string from direct specification or tbls runtime (via --tbls-config / --config)

	if q.DBConnection != "" {
		// Direct connection string
		connectionString = q.DBConnection
		// Try to determine driver from connection string
		driver = determineDriver(connectionString)
	} else if config.Query.DefaultEnvironment != "" {
		// Resolve default environment using tbls runtime (config.Databases ignored)
		if ctx.Verbose {
			color.Blue("Resolving default environment '%s' via .tbls.yaml", config.Query.DefaultEnvironment)
		}

		fallback, err := resolveDatabaseFromTbls(ctx)
		if err != nil {
			if errors.Is(err, ErrTblsDatabaseUnavailable) {
				return "", "", fmt.Errorf("%w: tbls config unavailable for default environment %s", ErrDefaultEnvironmentNotFound, config.Query.DefaultEnvironment)
			}

			return "", "", fmt.Errorf("failed to resolve database from tbls: %w", err)
		}

		connectionString = fallback.Connection
		driver = fallback.Driver
	}

	if strings.TrimSpace(connectionString) == "" {
		fallback, err := resolveDatabaseFromTbls(ctx)
		if err != nil {
			if errors.Is(err, ErrTblsDatabaseUnavailable) {
				return "", "", ErrNoDatabaseConnection
			}

			return "", "", err
		}

		connectionString = fallback.Connection
		if driver == "" {
			driver = fallback.Driver
		}

		if ctx.Verbose {
			color.Blue("Using database connection from tbls config")
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
		return normalizeSQLDriverName("postgres")
	}

	if strings.HasPrefix(connectionString, "mysql://") {
		return normalizeSQLDriverName("mysql")
	}

	if strings.HasPrefix(connectionString, "sqlite://") || strings.HasSuffix(connectionString, ".db") {
		return normalizeSQLDriverName("sqlite3")
	}
	// Default to postgres
	return normalizeSQLDriverName("postgres")
}

// executeDryRun generates SQL without executing it
func (q *QueryCmd) executeDryRun(ctx *Context, params map[string]any, options query.QueryOptions) error {
	// Load intermediate format
	format, err := query.LoadIntermediateFormat(q.TemplateFile)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Preflight parameter validation
	if err := query.ValidateParameters(format, params); err != nil {
		return fmt.Errorf("%w", err)
	}

	// Determine dialect for dry-run
	dialect := strings.ToLower(strings.TrimSpace(q.Dialect))
	if dialect == "" {
		dialect = q.getDialectFromOptions(options)
	}

	optimizedInstructions, err := codegenerator.OptimizeInstructions(format.Instructions, snapsql.Dialect(dialect))
	if err != nil {
		return fmt.Errorf("failed to optimize instructions: %w", err)
	}

	// Build SQL and arguments
	sql, args, err := q.buildSQLFromOptimized(optimizedInstructions, format, params)
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	// Format SQL for display (shared with executor)
	sql = query.FormatSQLForDialect(sql, snapsql.Dialect(dialect))

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

// removed local formatting helpers; using shared query.FormatSQLForDialect

// executeQuery executes the query and outputs results
func (q *QueryCmd) executeQuery(ctx *Context, params map[string]any, options query.QueryOptions, slowThreshold time.Duration, tableMetadata map[string]explain.TableMetadata) error {
	// Open database connection
	db, err := query.OpenDatabase(options.Driver, options.ConnectionString, options.Timeout)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDatabaseConnection, err)
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

		return fmt.Errorf("%w: %w", ErrQueryExecution, err)
	}

	analyzeEvaluation := q.analyzePerformance(db, options, result, slowThreshold, tableMetadata)

	// Determine output destination
	var output *os.File

	if q.OutputFile != "" {
		// Create output file
		file, err := os.Create(q.OutputFile)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrOutputFileCreation, err)
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
		err := formatter.FormatExplain(result, output)
		if err != nil {
			return fmt.Errorf("failed to format explain results: %w", err)
		}
	} else {
		err := formatter.Format(result, output)
		if err != nil {
			return fmt.Errorf("failed to format results: %w", err)
		}
	}

	if analyzeEvaluation != nil {
		q.printPerformanceWarnings(ctx, analyzeEvaluation, result.TableReferences, tableMetadata)
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
func (q *QueryCmd) buildSQLFromOptimized(instructions []codegenerator.OptimizedInstruction, format *intermediate.IntermediateFormat, params map[string]any) (string, []any, error) {
	var (
		builder           strings.Builder
		args              []any
		deferredTokens    []string
		hasContentSinceBd bool
	)

	// Create parameter map for evaluation
	paramMap := make(map[string]any)
	maps.Copy(paramMap, params)

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
					return "", nil, fmt.Errorf("%w: %d", ErrExpressionIndexNotFound, *inst.ExprIndex)
				}

				evalParams := map[string]any{"params": paramMap}
				maps.Copy(evalParams, paramMap)

				result, _, err := (*program).Eval(evalParams)
				if err != nil {
					return "", nil, fmt.Errorf("failed to evaluate expression %d: %w", *inst.ExprIndex, err)
				}

				args = append(args, result.Value())
			}

		case "EMIT_UNLESS_BOUNDARY":
			if inst.Value != "" {
				deferredTokens = append(deferredTokens, inst.Value)
			}

		case "BOUNDARY":
			if hasContentSinceBd {
				flushDeferred()
			} else {
				deferredTokens = nil
			}

			hasContentSinceBd = false

		default:
			// Ignore other ops for dry-run scope
		}
	}

	if hasContentSinceBd && len(deferredTokens) > 0 {
		flushDeferred()
	}

	return builder.String(), args, nil
}

func (q *QueryCmd) extractSlowQueryThreshold(defaultThreshold time.Duration) time.Duration {
	templateExt := strings.ToLower(filepath.Ext(q.TemplateFile))
	if templateExt == ".md" {
		file, err := os.Open(q.TemplateFile)
		if err != nil {
			return defaultThreshold
		}
		defer file.Close()

		doc, err := markdownparser.Parse(file)
		if err != nil {
			return defaultThreshold
		}

		if doc.Performance.SlowQueryThreshold > 0 {
			return doc.Performance.SlowQueryThreshold
		}
	}

	if templateExt == ".sql" {
		file, err := os.Open(q.TemplateFile)
		if err != nil {
			return defaultThreshold
		}
		defer file.Close()

		_, _, def, err := parser.ParseSQLFile(file, nil, q.TemplateFile, filepath.Dir(q.TemplateFile), parser.DefaultOptions)
		if err != nil || def == nil {
			return defaultThreshold
		}

		if def.SlowQueryThreshold > 0 {
			return def.SlowQueryThreshold
		}
	}

	return defaultThreshold
}

func (q *QueryCmd) analyzePerformance(db *sql.DB, options query.QueryOptions, result *query.QueryResult, threshold time.Duration, tables map[string]explain.TableMetadata) *explain.PerformanceEvaluation {
	if db == nil || result == nil || strings.TrimSpace(result.SQL) == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	timeout := time.Duration(options.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	analyze := shouldAnalyzeSQL(result.SQL)

	collector := explain.CollectorOptions{
		Runner:  db,
		Dialect: snapsql.Dialect(options.Driver),
		SQL:     result.SQL,
		Args:    result.Parameters,
		Analyze: analyze,
		Timeout: timeout,
	}

	structuredPlan := explain.SupportsStructuredPlan(snapsql.Dialect(options.Driver))

	doc, err := explain.Collect(ctx, collector)
	if err != nil {
		if errors.Is(err, explain.ErrNoPlanRows) && !structuredPlan {
			return nil
		}

		return &explain.PerformanceEvaluation{
			Warnings: []explain.Warning{{
				Kind:    explain.WarningParseError,
				Message: err.Error(),
			}},
		}
	}

	if structuredPlan && len(doc.RawJSON) == 0 && len(doc.Root) == 0 {
		return &explain.PerformanceEvaluation{
			Warnings: []explain.Warning{{
				Kind:    explain.WarningParseError,
				Message: "explain did not return JSON output",
			}},
		}
	}

	analyzerOpts := explain.AnalyzerOptions{
		Threshold: threshold,
		Tables:    tables,
	}

	evaluation, err := explain.Analyze(ctx, doc, analyzerOpts)
	if err != nil {
		return &explain.PerformanceEvaluation{
			Warnings: []explain.Warning{{
				Kind:    explain.WarningParseError,
				Message: err.Error(),
			}},
		}
	}

	if doc != nil && len(doc.Warnings) > 0 {
		for _, w := range doc.Warnings {
			evaluation.Warnings = append(evaluation.Warnings, explain.Warning{
				Kind:    explain.WarningParseError,
				Message: w.Error(),
			})
		}
	}

	return evaluation
}

func (q *QueryCmd) printPerformanceWarnings(ctx *Context, evaluation *explain.PerformanceEvaluation, refs []intermediate.TableReferenceInfo, tables map[string]explain.TableMetadata) {
	if evaluation == nil || len(evaluation.Warnings) == 0 {
		if ctx != nil && ctx.Verbose && evaluation != nil && len(evaluation.Estimates) > 0 {
			q.printPerformanceEstimates(ctx, evaluation)
		}

		return
	}

	if ctx != nil && ctx.Quiet {
		return
	}

	estMap := make(map[string]explain.QueryEstimate)
	for _, est := range evaluation.Estimates {
		estMap[est.QueryPath] = est
	}

	tableMap := intermediate.BuildTableReferenceMap(refs)
	physicalNames := physicalNameCandidatesFromMetadata(tables)
	warnLabel := color.New(color.Bold, color.FgYellow).Sprint("WARN")

	fmt.Fprintln(color.Output, "\nPerformance warnings:")

	for _, warn := range evaluation.Warnings {
		switch warn.Kind {
		case explain.WarningFullScan:
			debugDumpPlanTables("cli", warn.Tables, tableMap)

			targets := describeTablesForWarning(warn.Tables, tableMap, physicalNames)
			if len(targets) == 0 {
				targets = fallbackPlanTableDescriptions(warn.Tables)
			}

			for _, target := range targets {
				message := "full scan detected on " + target
				if warn.QueryPath != "" {
					message = fmt.Sprintf("%s [path=%s]", message, warn.QueryPath)
				}

				fmt.Fprintf(color.Output, "  %s %s\n", warnLabel, message)
			}
		default:
			message := warn.Message
			debugDumpPlanTables("cli", warn.Tables, tableMap)

			targets := describeTablesForWarning(warn.Tables, tableMap, physicalNames)
			if len(targets) == 0 {
				targets = fallbackPlanTableDescriptions(warn.Tables)
			}

			if warn.Kind == explain.WarningSlowQuery {
				if est, ok := estMap[warn.QueryPath]; ok {
					message = fmt.Sprintf("%s (actual=%s, estimated=%s, threshold=%s, scale=%.2f)",
						message,
						formatDuration(est.Actual),
						formatDuration(est.Estimated),
						formatDuration(est.Threshold),
						est.ScaleFactor)
				}
			}

			if len(targets) > 0 {
				message = fmt.Sprintf("%s (tables=%s)", message, strings.Join(targets, ", "))
			}

			if warn.QueryPath != "" {
				message = fmt.Sprintf("%s [path=%s]", message, warn.QueryPath)
			}

			fmt.Fprintf(color.Output, "  %s %s\n", warnLabel, message)
		}
	}

	if ctx != nil && ctx.Verbose {
		q.printPerformanceEstimates(ctx, evaluation)
	}
}

func describeTablesForWarning(keys []string, mapping map[string]intermediate.TableReferenceInfo, physicalNames []string) []string {
	return intermediate.DescribePlanTables(keys, mapping, physicalNames)
}

func physicalNameCandidatesFromMetadata(tables map[string]explain.TableMetadata) []string {
	if len(tables) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(tables)*2)
	for key := range tables {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}

		unique[trimmed] = struct{}{}
		if idx := strings.Index(trimmed, "."); idx > 0 && idx < len(trimmed)-1 {
			alias := trimmed[idx+1:]
			if alias != "" {
				unique[alias] = struct{}{}
			}
		}
	}

	results := make([]string, 0, len(unique))
	for name := range unique {
		results = append(results, name)
	}

	sort.Strings(results)

	return results
}

func debugDumpPlanTables(stage string, raw []string, mapping map[string]intermediate.TableReferenceInfo) {
	if os.Getenv("SNAPSQL_DEBUG_TABLES") == "" {
		return
	}

	fmt.Fprintf(os.Stderr, "[SNAPSQL][%s] EXPLAIN tables: %v\n", stage, raw)

	if len(mapping) == 0 {
		fmt.Fprintf(os.Stderr, "[SNAPSQL][%s] TableReferenceMap: <empty>\n", stage)
		return
	}

	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		info := mapping[key]
		fmt.Fprintf(
			os.Stderr,
			"[SNAPSQL][%s] map[%s]={Name:%s Alias:%s Table:%s Query:%s Context:%s}\n",
			stage,
			key,
			info.Name,
			info.Alias,
			info.TableName,
			info.QueryName,
			info.Context,
		)
	}
}

func fallbackPlanTableDescriptions(raw []string) []string {
	if len(raw) == 0 {
		return []string{"table '<unknown>' (physical table unresolved)"}
	}

	out := make([]string, 0, len(raw))
	for _, name := range raw {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			trimmed = "<unknown>"
		}

		out = append(out, fmt.Sprintf("table '%s' (physical table unresolved)", trimmed))
	}

	return out
}

func shouldAnalyzeSQL(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if upper == "" {
		return true
	}

	keyword := upper
	if idx := strings.IndexAny(upper, " \n\t"); idx >= 0 {
		keyword = upper[:idx]
	}

	switch keyword {
	case "SELECT", "WITH":
		return true
	default:
		return false
	}
}

func (q *QueryCmd) printPerformanceEstimates(ctx *Context, evaluation *explain.PerformanceEvaluation) {
	if ctx != nil && ctx.Quiet {
		return
	}

	if len(evaluation.Estimates) == 0 {
		return
	}

	infoLabel := color.New(color.Bold, color.FgCyan).Sprint("INFO")

	fmt.Fprintln(color.Output, "\nPerformance estimates:")

	printed := make(map[string]struct{})

	for _, warn := range evaluation.Warnings {
		if warn.QueryPath != "" {
			printed[warn.QueryPath] = struct{}{}
		}
	}

	for _, est := range evaluation.Estimates {
		if _, exists := printed[est.QueryPath]; exists {
			continue
		}

		message := fmt.Sprintf("actual=%s, estimated=%s, threshold=%s, scale=%.2f",
			formatDuration(est.Actual),
			formatDuration(est.Estimated),
			formatDuration(est.Threshold),
			est.ScaleFactor,
		)

		if est.QueryPath != "" {
			fmt.Fprintf(color.Output, "  %s %s [path=%s]\n", infoLabel, message, est.QueryPath)
		} else {
			fmt.Fprintf(color.Output, "  %s %s\n", infoLabel, message)
		}
	}
}

func isOnlyWhitespace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
	}

	return len(s) > 0
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
