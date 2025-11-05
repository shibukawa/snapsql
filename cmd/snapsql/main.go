package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/shibukawa/snapsql/testrunner"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
)

var (
	ErrFixtureOnlyRequiresRunPattern            = errors.New("--fixture-only mode requires --run pattern to specify which test case to execute")
	ErrFixtureOnlyAndQueryOnlyMutuallyExclusive = errors.New("--fixture-only and --query-only are mutually exclusive")
	// ErrPathOutsideProjectRoot indicates a provided path escapes the project root.
	ErrPathOutsideProjectRoot = errors.New("path is outside the project root")
	ErrUnsupportedPathType    = errors.New("unsupported path type")
)

// Context represents the global context for commands
type Context struct {
	Config     string
	Verbose    bool
	Quiet      bool
	TblsConfig string
}

// TestCmd represents the test command
type TestCmd struct {
	RunPattern  string `help:"Run only tests matching the regular expression" short:"r"`
	Timeout     string `help:"Test timeout duration" default:"10m"`
	Parallel    int    `help:"Number of parallel workers" default:"0"` // 0 means use CPU count
	FixtureOnly bool   `help:"Execute only fixture insertion and commit (requires --run pattern)"`
	QueryOnly   bool   `help:"Execute only queries without fixtures"`
	Commit      bool   `help:"Commit transactions instead of rollback"`
	// Environment flag removed; tbls uses single DSN and explicit tbls config path is preferred
	Schema []string `help:"SQL files or directories to initialize an ephemeral database (repeatable)" short:"s"`
	Paths  []string `arg:"" optional:"" name:"path" help:"Optional file or directory paths to limit executed tests"`
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
	options.PerformanceEnabled = true

	options.SlowQueryThreshold = config.Performance.SlowQueryThreshold
	if options.SlowQueryThreshold <= 0 {
		options.SlowQueryThreshold = 3 * time.Second
	}

	options.TableMetadata = buildTableMetadataFromConfig(config.Tables)

	verbose := ctx.Verbose
	options.Verbose = verbose

	includePaths, err := cmd.resolveTargetPaths(projectRoot)
	if err != nil {
		return err
	}

	runtimeTables := loadRuntimeTables(ctx)
	if len(runtimeTables) == 0 {
		return snapsql.ErrNoSchemaYAMLFound
	}

	if verbose {
		fmt.Printf("Starting test execution in: %s\n", projectRoot)
		fmt.Printf("Execution mode: %s\n", mode)
		fmt.Printf("Timeout: %s\n", timeout)
		fmt.Printf("Parallel workers: %d\n", parallel)
		fmt.Printf("Commit after test: %t\n", cmd.Commit)
		// Environment field removed; tbls config path is used instead when needed

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

	if len(cmd.Schema) > 0 {
		return cmd.runWithSchemaDatabase(projectRoot, config, includePaths, options, verbose, runtimeTables)
	}

	return cmd.runWithTblsDatabase(projectRoot, config, includePaths, options, verbose, runtimeTables, ctx)
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

func (cmd *TestCmd) runWithSchemaDatabase(projectRoot string, config *snapsql.Config, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool, tableCatalog map[string]*snapsql.TableInfo) error {
	ctx := context.Background()

	db, cleanup, err := openInMemorySQLite(ctx, verbose)
	if err != nil {
		return err
	}
	defer cleanup()

	config.Dialect = snapsql.DialectSQLite

	if err := cmd.applySchema(ctx, db, cmd.Schema, verbose); err != nil {
		return err
	}

	return cmd.executeFixtureTests(projectRoot, config, db, tableCatalog, includePaths, options, verbose)
}

func openInMemorySQLite(ctx context.Context, verbose bool) (*sql.DB, func(), error) {
	dsn := "file:snapsql-test?mode=memory&cache=shared&_foreign_keys=1"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open in-memory sqlite database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to initialize in-memory sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if verbose {
		fmt.Println("Provisioned in-memory SQLite database")
	}

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup, nil
}

func (cmd *TestCmd) runWithTblsDatabase(projectRoot string, config *snapsql.Config, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool, tableCatalog map[string]*snapsql.TableInfo, appCtx *Context) error {
	fallback, err := resolveDatabaseFromTbls(appCtx)
	if err != nil {
		return fmt.Errorf("%w: failed to resolve tbls database configuration: %s", ErrDatabaseConnection, err.Error())
	}

	driverName := normalizeSQLDriverName(fallback.Driver)
	if driverName == "" {
		return fmt.Errorf("%w: unsupported database driver: %s", ErrDatabaseConnection, fallback.Driver)
	}

	dialect := canonicalDialectFromDriver(fallback.Driver)
	config.Dialect = snapsql.Dialect(dialect)

	if verbose {
		fmt.Printf("Using database configuration from tbls config (%s)\n", dialect)
	}

	db, err := sql.Open(driverName, fallback.Connection)
	if err != nil {
		return fmt.Errorf("%w: failed to connect to database: %w", ErrDatabaseConnection, err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("%w: failed to ping database: %s", ErrDatabaseConnection, err.Error())
	}

	return cmd.executeFixtureTests(projectRoot, config, db, tableCatalog, includePaths, options, verbose)
}

func (cmd *TestCmd) executeFixtureTests(projectRoot string, config *snapsql.Config, db *sql.DB, tableInfo map[string]*snapsql.TableInfo, includePaths []string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
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

	testCtx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	summary, err := runner.RunAllFixtureTests(testCtx)
	if err != nil {
		return fmt.Errorf("fixture test execution failed: %w", err)
	}

	runner.PrintSummary(summary)

	if summary.FailedTests > 0 {
		os.Exit(1)
	}

	return nil
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

// CLI represents the command-line interface
var CLI struct {
	Config     string       `help:"Configuration file path" default:"snapsql.yaml"`
	Verbose    bool         `help:"Enable verbose output" short:"v"`
	Quiet      bool         `help:"Suppress output" short:"q"`
	TblsConfig string       `help:"Path to tbls config (.tbls.yaml); overrides --config"`
	Generate   GenerateCmd  `cmd:"" help:"Generate intermediate files from SQL templates"`
	Validate   ValidateCmd  `cmd:"" help:"Validate SQL templates"`
	Init       InitCmd      `cmd:"" help:"Initialize a new SnapSQL project"`
	Query      QueryCmd     `cmd:"" help:"Execute SQL queries"`
	Test       TestCmd      `cmd:"" help:"Run tests"`
	Format     FormatCmd    `cmd:"" help:"Format SnapSQL template files"`
	HelpTypes  HelpTypesCmd `cmd:"help-types" help:"Show detailed information about supported types"`
	Inspect    InspectCmd   `cmd:"" help:"Inspect an SQL and print JSON summary"`
	Version    VersionCmd   `cmd:"" help:"Show version information"`
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
		Config:     CLI.Config,
		Verbose:    CLI.Verbose,
		Quiet:      CLI.Quiet,
		TblsConfig: CLI.TblsConfig,
	}

	err := ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
