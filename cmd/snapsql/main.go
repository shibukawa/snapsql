package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/inspect"
	"github.com/shibukawa/snapsql/pull"
	"github.com/shibukawa/snapsql/testrunner"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	ErrFixtureOnlyRequiresRunPattern            = errors.New("--fixture-only mode requires --run pattern to specify which test case to execute")
	ErrFixtureOnlyAndQueryOnlyMutuallyExclusive = errors.New("--fixture-only and --query-only are mutually exclusive")
	// ErrPathOutsideProjectRoot indicates a provided path escapes the project root.
	ErrPathOutsideProjectRoot      = errors.New("path is outside the project root")
	ErrUnsupportedPathType         = errors.New("unsupported path type")
	ErrDialectNotConfigured        = errors.New("dialect not configured in snapsql.yaml; please run 'snapsql init' or set the dialect explicitly")
	ErrUnsupportedEphemeralDialect = errors.New("unsupported dialect for ephemeral database mode")
	ErrSchemaOutputNotDirectory    = errors.New("schema output path is not a directory")
)

// Context represents the global context for commands
type Context struct {
	Config  string
	Verbose bool
	Quiet   bool
}

// TestCmd represents the test command
type TestCmd struct {
	RunPattern   string   `help:"Run only tests matching the regular expression" short:"r"`
	Timeout      string   `help:"Test timeout duration" default:"10m"`
	Parallel     int      `help:"Number of parallel workers" default:"0"` // 0 means use CPU count
	FixtureOnly  bool     `help:"Execute only fixture insertion and commit (requires --run pattern)"`
	QueryOnly    bool     `help:"Execute only queries without fixtures"`
	Commit       bool     `help:"Commit transactions instead of rollback"`
	Environment  string   `help:"Database environment to use from config" default:"development"`
	Schema       []string `help:"SQL files or directories to initialize an ephemeral database (repeatable)" short:"s"`
	UseExisting  bool     `help:"Use configured database environment instead of provisioning an ephemeral database"`
	SchemaOutput string   `help:"Directory to emit schema YAML snapshots before running tests" default:"./schema"`
	Paths        []string `arg:"" optional:"" name:"path" help:"Optional file or directory paths to limit executed tests"`
}

type provisionedDatabase struct {
	DB          *sql.DB
	DriverName  string
	DatabaseURL string
	cleanup     func(context.Context) error
}

func (p *provisionedDatabase) Close(ctx context.Context) error {
	var firstErr error

	if p == nil {
		return nil
	}

	if p.DB != nil {
		if err := p.DB.Close(); err != nil {
			firstErr = err
		}
	}

	if p.cleanup != nil {
		if err := p.cleanup(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Run executes the test command
func (cmd *TestCmd) Run(ctx *Context) error {
	// Validate fixture-only mode requirements
	if cmd.FixtureOnly && cmd.RunPattern == "" {
		return ErrFixtureOnlyRequiresRunPattern
	}

	// Validate mutually exclusive options
	if cmd.FixtureOnly && cmd.QueryOnly {
		return ErrFixtureOnlyAndQueryOnlyMutuallyExclusive
	}

	// Get current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load configuration
	config, err := snapsql.LoadConfig(ctx.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Parse timeout
	timeout, err := time.ParseDuration(cmd.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout duration: %w", err)
	}

	// Determine execution mode
	var mode = fixtureexecutor.FullTest
	if cmd.FixtureOnly {
		mode = fixtureexecutor.FixtureOnly
	} else if cmd.QueryOnly {
		mode = fixtureexecutor.QueryOnly
	}

	// Set parallel count
	parallel := cmd.Parallel
	if parallel <= 0 {
		parallel = runtime.NumCPU()
	}

	// Create execution options
	options := &fixtureexecutor.ExecutionOptions{
		Mode:     mode,
		Commit:   cmd.Commit,
		Parallel: parallel,
		Timeout:  timeout,
	}

	verbose := ctx.Verbose
	options.Verbose = verbose

	includePaths, err := cmd.resolveTargetPaths(projectRoot)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Starting test execution in: %s\n", projectRoot)
		fmt.Printf("Execution mode: %s\n", mode)
		fmt.Printf("Timeout: %s\n", timeout)
		fmt.Printf("Parallel workers: %d\n", parallel)
		fmt.Printf("Commit after test: %t\n", cmd.Commit)
		fmt.Printf("Environment: %s\n", cmd.Environment)

		if cmd.RunPattern != "" {
			fmt.Printf("Test pattern: %s\n", cmd.RunPattern)
		}

		if len(includePaths) > 0 {
			fmt.Printf("Target paths:\n")

			for _, p := range includePaths {
				rel, err := filepath.Rel(projectRoot, p)
				if err != nil {
					rel = p
				}

				fmt.Printf("  - %s\n", filepath.ToSlash(rel))
			}
		}

		fmt.Println()
	}

	if cmd.UseExisting {
		return cmd.runWithExistingDatabase(projectRoot, config, includePaths, options, verbose)
	}

	return cmd.runWithEphemeralDatabase(projectRoot, config, includePaths, options, verbose)
}

func (cmd *TestCmd) resolveTargetPaths(projectRoot string) ([]string, error) {
	if len(cmd.Paths) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})

	var resolved []string

	for _, p := range cmd.Paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		absPath := p
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(projectRoot, p)
		}

		absPath = filepath.Clean(absPath)

		rel, err := filepath.Rel(projectRoot, absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path '%s': %w", p, err)
		}

		if strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("%w: %s", ErrPathOutsideProjectRoot, p)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat path '%s': %w", p, err)
		}

		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedPathType, p)
		}

		if _, ok := seen[absPath]; ok {
			continue
		}

		seen[absPath] = struct{}{}
		resolved = append(resolved, absPath)
	}

	return resolved, nil
}

func (cmd *TestCmd) runWithExistingDatabase(projectRoot string, config *snapsql.Config, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
	if config.Databases == nil {
		if verbose {
			fmt.Printf("No database configuration present in snapsql.yaml, falling back to Go tests\n")
		}

		return cmd.runGoTests(projectRoot, includePaths, options, verbose)
	}

	dbConfig, exists := config.Databases[cmd.Environment]
	if !exists {
		if verbose {
			fmt.Printf("No database configuration found for environment '%s', falling back to Go tests\n", cmd.Environment)
		}

		return cmd.runGoTests(projectRoot, includePaths, options, verbose)
	}

	return cmd.runFixtureTests(projectRoot, config, dbConfig, includePaths, options, verbose)
}

func (cmd *TestCmd) runWithEphemeralDatabase(projectRoot string, config *snapsql.Config, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
	if strings.TrimSpace(config.Dialect) == "" {
		return ErrDialectNotConfigured
	}

	ctx := context.Background()

	provisioned, err := cmd.provisionEphemeralDatabase(ctx, config, verbose)
	if err != nil {
		return err
	}
	defer provisioned.Close(ctx)

	if err := cmd.applySchema(ctx, provisioned.DB, cmd.Schema, verbose); err != nil {
		return err
	}

	if err := cmd.executeSchemaPull(ctx, config, provisioned, verbose); err != nil {
		return err
	}

	tableInfo, err := cmd.loadTableInfo(cmd.SchemaOutput, verbose)
	if err != nil {
		return err
	}

	return cmd.executeFixtureTests(projectRoot, config, provisioned.DB, tableInfo, includePaths, options, verbose)
}

// runFixtureTests runs fixture-based tests
func (cmd *TestCmd) runFixtureTests(projectRoot string, config *snapsql.Config, dbConfig snapsql.Database, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
	// Open database connection
	db, err := sql.Open(dbConfig.Driver, dbConfig.Connection)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if verbose {
		fmt.Printf("Connected to database: %s\n", dbConfig.Driver)
		fmt.Printf("Schema: %s\n", dbConfig.Schema)
		fmt.Println()
	}

	return cmd.executeFixtureTests(projectRoot, config, db, nil, includePaths, options, verbose)
}

func (cmd *TestCmd) executeFixtureTests(projectRoot string, config *snapsql.Config, db *sql.DB, tableInfo map[string]*snapsql.TableInfo, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
	// Create fixture test runner
	runner := testrunner.NewFixtureTestRunner(projectRoot, db, config.Dialect)
	runner.SetVerbose(verbose)
	runner.SetExecutionOptions(options)

	if len(tableInfo) > 0 {
		runner.SetTableInfo(tableInfo)
	}

	if len(includePaths) > 0 {
		runner.SetIncludePaths(includePaths)
	}

	if cmd.RunPattern != "" {
		runner.SetRunPattern(cmd.RunPattern)

		if verbose {
			fmt.Printf("Running tests matching pattern: %s\n", cmd.RunPattern)
		}
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// Run fixture tests
	summary, err := runner.RunAllFixtureTests(testCtx)
	if err != nil {
		return fmt.Errorf("fixture test execution failed: %w", err)
	}

	// Print summary
	runner.PrintSummary(summary)

	// Exit with non-zero code if any tests failed
	if summary.FailedTests > 0 {
		os.Exit(1)
	}

	return nil
}

func (cmd *TestCmd) provisionEphemeralDatabase(ctx context.Context, config *snapsql.Config, verbose bool) (*provisionedDatabase, error) {
	dialect := normalizeDialect(config.Dialect)

	switch dialect {
	case "sqlite":
		return cmd.provisionSQLiteDatabase(ctx, verbose)
	case "postgresql":
		return cmd.provisionPostgresDatabase(ctx, verbose)
	case "mysql":
		return cmd.provisionMySQLDatabase(ctx, verbose)
	default:
		return nil, fmt.Errorf("%w: %s; specify --use-existing-db to use a configured connection", ErrUnsupportedEphemeralDialect, config.Dialect)
	}
}

func (cmd *TestCmd) provisionSQLiteDatabase(ctx context.Context, verbose bool) (*provisionedDatabase, error) {
	tempDir, err := os.MkdirTemp("", "snapsql-sqlite-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory for sqlite database: %w", err)
	}

	dbPath := filepath.Join(tempDir, "snapsql-test.db")

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to resolve sqlite database path: %w", err)
	}

	dsnURL := &url.URL{
		Scheme:   "file",
		Path:     absPath,
		RawQuery: "_busy_timeout=5000&_foreign_keys=1",
	}

	db, err := sql.Open("sqlite3", dsnURL.String())
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		os.RemoveAll(tempDir)

		return nil, fmt.Errorf("failed to initialize sqlite database: %w", err)
	}

	// Limit connections to avoid locking issues when multiple connections are opened.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	sqliteURL := (&url.URL{Scheme: "sqlite", Path: absPath}).String()

	if verbose {
		fmt.Printf("Provisioned SQLite database at %s\n", absPath)
	}

	cleanup := func(context.Context) error {
		return os.RemoveAll(tempDir)
	}

	return &provisionedDatabase{
		DB:          db,
		DriverName:  "sqlite3",
		DatabaseURL: sqliteURL,
		cleanup:     cleanup,
	}, nil
}

func (cmd *TestCmd) provisionPostgresDatabase(ctx context.Context, verbose bool) (*provisionedDatabase, error) {
	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("snapsql"),
		postgres.WithUsername("snapsql"),
		postgres.WithPassword("snapsql"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to obtain postgres connection string: %w", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		container.Terminate(ctx)

		return nil, fmt.Errorf("failed to ping postgres container: %w", err)
	}

	if verbose {
		fmt.Printf("Provisioned PostgreSQL container with DSN %s\n", connStr)
	}

	cleanup := func(c context.Context) error {
		return container.Terminate(c)
	}

	return &provisionedDatabase{
		DB:          db,
		DriverName:  "pgx",
		DatabaseURL: connStr,
		cleanup:     cleanup,
	}, nil
}

func (cmd *TestCmd) provisionMySQLDatabase(ctx context.Context, verbose bool) (*provisionedDatabase, error) {
	container, err := mysql.Run(ctx,
		"mysql:8.4",
		mysql.WithDatabase("snapsql"),
		mysql.WithUsername("snapsql"),
		mysql.WithPassword("snapsql"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start mysql container: %w", err)
	}

	rawConnStr, err := container.ConnectionString(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to obtain mysql connection string: %w", err)
	}

	dsn := ensureMySQLMultiStatements(rawConnStr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Give the server a brief moment to finish booting before pinging.
	time.Sleep(2 * time.Second)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		container.Terminate(ctx)

		return nil, fmt.Errorf("failed to ping mysql container: %w", err)
	}

	if verbose {
		fmt.Printf("Provisioned MySQL container with DSN %s\n", rawConnStr)
	}

	cleanup := func(c context.Context) error {
		return container.Terminate(c)
	}

	return &provisionedDatabase{
		DB:          db,
		DriverName:  "mysql",
		DatabaseURL: convertMySQLConnStrToURL(rawConnStr),
		cleanup:     cleanup,
	}, nil
}

func (cmd *TestCmd) applySchema(ctx context.Context, db *sql.DB, schemaPaths []string, verbose bool) error {
	if len(schemaPaths) == 0 {
		if verbose {
			fmt.Println("No schema paths provided; skipping schema initialization")
		}

		return nil
	}

	for _, rawPath := range schemaPaths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to access schema path %s: %w", path, err)
		}

		if info.IsDir() {
			files, err := collectSQLFiles(path)
			if err != nil {
				return fmt.Errorf("failed to collect SQL files under %s: %w", path, err)
			}

			for _, file := range files {
				if verbose {
					fmt.Printf("Applying schema file: %s\n", file)
				}

				if err := executeSQLFile(ctx, db, file); err != nil {
					return err
				}
			}
		} else {
			if verbose {
				fmt.Printf("Applying schema file: %s\n", path)
			}

			if err := executeSQLFile(ctx, db, path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cmd *TestCmd) executeSchemaPull(ctx context.Context, config *snapsql.Config, provisioned *provisionedDatabase, verbose bool) error {
	outputDir := cmd.SchemaOutput
	if outputDir == "" {
		outputDir = "./schema"
	}

	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to clean schema output directory %s: %w", outputDir, err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create schema output directory %s: %w", outputDir, err)
	}

	pullConfig := pull.PullConfig{
		DatabaseURL:    provisioned.DatabaseURL,
		DatabaseType:   normalizeDialect(config.Dialect),
		OutputPath:     outputDir,
		SchemaAware:    true,
		IncludeViews:   config.Schema.IncludeViews,
		IncludeIndexes: config.Schema.IncludeIndexes,
		IncludeTables:  config.Schema.TablePatterns.Include,
		ExcludeTables:  config.Schema.TablePatterns.Exclude,
	}

	result, err := pull.ExecutePull(ctx, pullConfig)
	if err != nil {
		return fmt.Errorf("schema pull failed: %w", err)
	}

	if verbose {
		fmt.Printf("Schema pull complete: %d schema(s) processed\n", len(result.Schemas))
	}

	return nil
}

func (cmd *TestCmd) loadTableInfo(root string, verbose bool) (map[string]*snapsql.TableInfo, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			if verbose {
				fmt.Printf("Schema directory %s does not exist; continuing without table metadata\n", root)
			}

			return map[string]*snapsql.TableInfo{}, nil
		}

		return nil, fmt.Errorf("failed to inspect schema directory %s: %w", root, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrSchemaOutputNotDirectory, root)
	}

	tableInfo := make(map[string]*snapsql.TableInfo)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		table, err := pull.LoadTableFromYAMLFile(path)
		if err != nil {
			return fmt.Errorf("failed to load schema YAML %s: %w", path, err)
		}

		tableInfo[table.Name] = table
		if table.Schema != "" {
			qualified := table.Schema + "." + table.Name
			if _, exists := tableInfo[qualified]; !exists {
				tableInfo[qualified] = table
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("Loaded %d table definition(s) from %s\n", len(tableInfo), root)
	}

	return tableInfo, nil
}

func collectSQLFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".sql" {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	return files, nil
}

func executeSQLFile(ctx context.Context, db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", path, err)
	}

	query := strings.TrimSpace(string(content))
	if query == "" {
		return nil
	}

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to execute schema file %s: %w", path, err)
	}

	return nil
}

func ensureMySQLMultiStatements(dsn string) string {
	if strings.Contains(dsn, "multiStatements=") {
		return dsn
	}

	if strings.Contains(dsn, "?") {
		return dsn + "&multiStatements=true"
	}

	return dsn + "?multiStatements=true"
}

func convertMySQLConnStrToURL(connStr string) string {
	if strings.Contains(connStr, "@tcp(") {
		parts := strings.Split(connStr, "@tcp(")
		if len(parts) == 2 {
			userPass := parts[0]
			hostPortDb := parts[1]
			hostPortDb = strings.Replace(hostPortDb, ")", "", 1)

			return fmt.Sprintf("mysql://%s@%s", userPass, hostPortDb)
		}
	}

	return "mysql://" + connStr
}

func normalizeDialect(dialect string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql", "pgx":
		return "postgresql"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return strings.ToLower(strings.TrimSpace(dialect))
	}
}

// runGoTests runs regular Go tests (fallback)
func (cmd *TestCmd) runGoTests(projectRoot string, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
	// Create regular test runner
	runner := testrunner.NewTestRunner(projectRoot)
	runner.SetVerbose(verbose)

	if len(includePaths) > 0 {
		runner.SetIncludePaths(includePaths)
	}

	// Set run pattern if specified
	if cmd.RunPattern != "" {
		err := runner.SetRunPattern(cmd.RunPattern)
		if err != nil {
			return fmt.Errorf("invalid run pattern: %w", err)
		}

		if verbose {
			fmt.Printf("Running tests matching pattern: %s\n", cmd.RunPattern)
		}
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// Run all tests
	summary, err := runner.RunAllTests(testCtx)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Print summary
	runner.PrintSummary(summary)

	// Exit with non-zero code if any tests failed
	if summary.FailedPackages > 0 {
		os.Exit(1)
	}

	return nil
}

// CLI represents the command-line interface
var CLI struct {
	Config    string       `help:"Configuration file path" default:"snapsql.yaml"`
	Verbose   bool         `help:"Enable verbose output" short:"v"`
	Quiet     bool         `help:"Suppress output" short:"q"`
	Generate  GenerateCmd  `cmd:"" help:"Generate intermediate files from SQL templates"`
	Validate  ValidateCmd  `cmd:"" help:"Validate SQL templates"`
	Init      InitCmd      `cmd:"" help:"Initialize a new SnapSQL project"`
	Pull      PullCmd      `cmd:"" help:"Pull schema information from database"`
	Query     QueryCmd     `cmd:"" help:"Execute SQL queries"`
	Test      TestCmd      `cmd:"" help:"Run tests"`
	Format    FormatCmd    `cmd:"" help:"Format SnapSQL template files"`
	HelpTypes HelpTypesCmd `cmd:"help-types" help:"Show detailed information about supported types"`
	Inspect   InspectCmd   `cmd:"" help:"Inspect an SQL and print JSON summary"`
	Version   VersionCmd   `cmd:"" help:"Show version information"`
}

// InspectCmd represents the inspect command
type InspectCmd struct {
	Stdin  bool   `help:"Read SQL from stdin"`
	Pretty bool   `help:"Pretty-print JSON output"`
	Strict bool   `help:"Strict mode: fail on partial/unsupported constructs"`
	Format string `help:"Output format: json|csv" default:"json"`
	Path   string `arg:"" optional:"" help:"Path to SQL file (omit or '-' to use --stdin)"`
}

var ErrUnsupportedFormat = errors.New("unsupported format")

// Run executes the inspect command
func (cmd *InspectCmd) Run(ctx *Context) error {
	var r *os.File
	if cmd.Stdin || cmd.Path == "-" || cmd.Path == "" {
		r = os.Stdin
	} else {
		f, err := os.Open(cmd.Path)
		if err != nil {
			return fmt.Errorf("failed to open SQL file: %w", err)
		}
		defer f.Close()

		r = f
	}

	// Execute inspect
	res, err := inspect.Inspect(r, inspect.InspectOptions{InspectMode: true, Strict: cmd.Strict, Pretty: cmd.Pretty})
	if err != nil {
		return err
	}

	switch cmd.Format {
	case "json", "":
		var b []byte
		if cmd.Pretty {
			b, err = json.MarshalIndent(res, "", "  ")
		} else {
			b, err = json.Marshal(res)
		}

		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}

		os.Stdout.Write(b)
		os.Stdout.WriteString("\n")
	case "csv":
		b, err := inspect.TablesCSV(res, true)
		if err != nil {
			return err
		}

		os.Stdout.Write(b)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, cmd.Format)
	}

	return nil
}

// HelpTypesCmd represents the help-types command
type HelpTypesCmd struct{}

// Run executes the help-types command
func (cmd *HelpTypesCmd) Run() error {
	fmt.Println("SnapSQL Supported Types")
	fmt.Println("=======================")
	fmt.Println()

	fmt.Println("Basic Types:")
	fmt.Println("  int, int32, int64    - Integer numbers")
	fmt.Println("  string               - Text strings")
	fmt.Println("  bool                 - Boolean values (true/false)")
	fmt.Println("  float, float32, float64 - Floating point numbers")
	fmt.Println("  decimal              - High-precision decimal numbers")
	fmt.Println("  timestamp, date, time - Date and time values")
	fmt.Println("  bytes                - Binary data")
	fmt.Println("  any                  - Any type (interface{})")
	fmt.Println()

	fmt.Println("Array Types:")
	fmt.Println("  string[]             - Array of strings")
	fmt.Println("  int[]                - Array of integers")
	fmt.Println("  any[]                - Array of any type")
	fmt.Println("  CustomType[]         - Array of custom types")
	fmt.Println()

	fmt.Println("Pointer Types:")
	fmt.Println("  *string              - Pointer to string (nullable)")
	fmt.Println("  *int                 - Pointer to integer (nullable)")
	fmt.Println("  *CustomType          - Pointer to custom type")
	fmt.Println()

	fmt.Println("Package-Qualified Types:")
	fmt.Println("  time.Time            - Go standard library time")
	fmt.Println("  decimal.Decimal      - Decimal library type")
	fmt.Println("  mypackage.MyType     - Custom package types")
	fmt.Println()

	fmt.Println("Custom Types:")
	fmt.Println("  MyType               - Custom struct types")
	fmt.Println("  UserModel            - Domain model types")
	fmt.Println("  ./User               - Relative path types")
	fmt.Println("  ./models/User        - Nested path types")
	fmt.Println()

	fmt.Println("System Column Types (for implicit parameters):")
	fmt.Println("  int                  - For user IDs, version numbers")
	fmt.Println("  string               - For user names, reasons")
	fmt.Println("  timestamp            - For created_at, updated_at")
	fmt.Println("  bool                 - For flags and status")
	fmt.Println()

	fmt.Println("Examples:")
	fmt.Println("  parameters:")
	fmt.Println("    user_id: int")
	fmt.Println("    name: string")
	fmt.Println("    tags: string[]")
	fmt.Println("    profile: ./UserProfile")
	fmt.Println("    created_at: timestamp")
	fmt.Println()

	return nil
}

// VersionCmd represents the version command
type VersionCmd struct{}

// Run executes the version command
func (cmd *VersionCmd) Run() error {
	fmt.Println("SnapSQL v0.1.0")
	return nil
}

func main() {
	ctx := kong.Parse(&CLI)

	// Create context with config path
	appCtx := &Context{
		Config:  CLI.Config,
		Verbose: CLI.Verbose,
		Quiet:   CLI.Quiet,
	}

	err := ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
