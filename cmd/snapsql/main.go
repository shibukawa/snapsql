package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/alecthomas/kong"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/testrunner"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
)

var (
	ErrFixtureOnlyRequiresRunPattern            = errors.New("--fixture-only mode requires --run pattern to specify which test case to execute")
	ErrFixtureOnlyAndQueryOnlyMutuallyExclusive = errors.New("--fixture-only and --query-only are mutually exclusive")
)

// Context represents the global context for commands
type Context struct {
	Config  string
	Verbose bool
	Quiet   bool
}

// TestCmd represents the test command
type TestCmd struct {
	RunPattern  string `help:"Run only tests matching the regular expression" short:"r"`
	Timeout     string `help:"Test timeout duration" default:"10m"`
	Parallel    int    `help:"Number of parallel workers" default:"0"` // 0 means use CPU count
	FixtureOnly bool   `help:"Execute only fixture insertion and commit (requires --run pattern)"`
	QueryOnly   bool   `help:"Execute only queries without fixtures"`
	Commit      bool   `help:"Commit transactions instead of rollback"`
	Environment string `help:"Database environment to use from config" default:"development"`
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

		fmt.Println()
	}

	// Check if we have database configuration for fixture tests
	dbConfig, exists := config.Databases[cmd.Environment]
	if exists {
		return cmd.runFixtureTests(projectRoot, config, dbConfig, options, verbose)
	} else {
		if verbose {
			fmt.Printf("No database configuration found for environment '%s', falling back to Go tests\n", cmd.Environment)
		}
		// Fall back to regular Go tests
		return cmd.runGoTests(projectRoot, options, verbose)
	}
}

// runFixtureTests runs fixture-based tests
func (cmd *TestCmd) runFixtureTests(projectRoot string, config *snapsql.Config, dbConfig snapsql.Database, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
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

	// Create fixture test runner
	runner := testrunner.NewFixtureTestRunner(projectRoot, db, config.Dialect)
	runner.SetVerbose(verbose)
	runner.SetExecutionOptions(options)

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

// runGoTests runs regular Go tests (fallback)
func (cmd *TestCmd) runGoTests(projectRoot string, options *fixtureexecutor.ExecutionOptions, verbose bool) error {
	// Create regular test runner
	runner := testrunner.NewTestRunner(projectRoot)
	runner.SetVerbose(verbose)

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
	Version   VersionCmd   `cmd:"" help:"Show version information"`
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
