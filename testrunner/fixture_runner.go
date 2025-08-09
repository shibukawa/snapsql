package testrunner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
)

// FixtureTestRunner manages fixture-based test execution
type FixtureTestRunner struct {
	projectRoot string
	db          *sql.DB
	dialect     string
	verbose     bool
	runPattern  string
	options     *fixtureexecutor.ExecutionOptions
}

// NewFixtureTestRunner creates a new fixture test runner
func NewFixtureTestRunner(projectRoot string, db *sql.DB, dialect string) *FixtureTestRunner {
	return &FixtureTestRunner{
		projectRoot: projectRoot,
		db:          db,
		dialect:     dialect,
		verbose:     false,
		options:     fixtureexecutor.DefaultExecutionOptions(),
	}
}

// SetVerbose enables or disables verbose output
func (ftr *FixtureTestRunner) SetVerbose(verbose bool) {
	ftr.verbose = verbose
}

// SetRunPattern sets the test name filter pattern
func (ftr *FixtureTestRunner) SetRunPattern(pattern string) {
	ftr.runPattern = pattern
}

// SetExecutionOptions sets the fixture execution options
func (ftr *FixtureTestRunner) SetExecutionOptions(options *fixtureexecutor.ExecutionOptions) {
	ftr.options = options
}

// RunAllFixtureTests executes all fixture-based tests
func (ftr *FixtureTestRunner) RunAllFixtureTests(ctx context.Context) (*FixtureTestSummary, error) {
	// Find all markdown test files
	testFiles, err := ftr.findMarkdownTestFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find test files: %w", err)
	}

	// Filter test files by pattern if specified
	if ftr.runPattern != "" {
		testFiles = ftr.filterTestFiles(testFiles)
	}

	if ftr.verbose {
		fmt.Printf("Found %d markdown test files\n", len(testFiles))
	}

	// Parse test cases from filtered markdown files
	var (
		allTestCases       []*markdownparser.TestCase
		documentSQL        string
		documentParameters map[string]any
	)

	for _, file := range testFiles {
		fileInfo, err := ftr.parseTestFile(file)
		if err != nil {
			if ftr.verbose {
				fmt.Printf("Warning: failed to parse %s: %v\n", file, err)
			}

			continue
		}

		// Use SQL and parameters from the first successfully parsed file
		if documentSQL == "" {
			documentSQL = fileInfo.SQL
			documentParameters = fileInfo.Parameters
		}

		allTestCases = append(allTestCases, fileInfo.TestCases...)
	}

	// For fixture-only mode, ensure exactly one test case or one file with one test case
	if ftr.options.Mode == fixtureexecutor.FixtureOnly {
		if len(allTestCases) == 0 {
			return nil, fmt.Errorf("%w: '%s'", snapsql.ErrNoTestCasesFound, ftr.runPattern)
		}

		if len(allTestCases) > 1 {
			var names []string
			for _, tc := range allTestCases {
				names = append(names, tc.Name)
			}

			return nil, fmt.Errorf("%w, but found %d test cases: %s",
				snapsql.ErrFixtureOnlyModeRequiresOne, len(allTestCases), strings.Join(names, ", "))
		}

		if ftr.verbose {
			fmt.Printf("Selected test case for fixture-only mode: %s\n", allTestCases[0].Name)
		}
	}

	if ftr.verbose {
		fmt.Printf("Executing %d test cases\n", len(allTestCases))
		fmt.Printf("Execution mode: %s\n", ftr.options.Mode)
		fmt.Printf("Parallel workers: %d\n", ftr.options.Parallel)
		fmt.Printf("Commit after test: %t\n", ftr.options.Commit)
		fmt.Println()
	}

	// Create test runner and execute tests
	runner := fixtureexecutor.NewTestRunner(ftr.db, ftr.dialect, ftr.options)
	runner.SetSQL(documentSQL)
	runner.SetParameters(documentParameters)

	summary, err := runner.RunTests(ctx, allTestCases)
	if err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	// Convert to FixtureTestSummary
	fixtureSummary := &FixtureTestSummary{
		TotalTests:    summary.TotalTests,
		PassedTests:   summary.PassedTests,
		FailedTests:   summary.FailedTests,
		TotalDuration: summary.TotalDuration,
		Results:       make([]FixtureTestResult, len(summary.Results)),
	}

	for i, result := range summary.Results {
		fixtureSummary.Results[i] = FixtureTestResult{
			TestName: result.TestCase.Name,
			Success:  result.Success,
			Duration: result.Duration,
			Error:    result.Error,
		}
	}

	return fixtureSummary, nil
}

// findMarkdownTestFiles finds all markdown test files in the project
func (ftr *FixtureTestRunner) findMarkdownTestFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(ftr.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, .git, and other common directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" ||
				strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}

			return nil
		}

		// Look for markdown files with test cases
		if strings.HasSuffix(info.Name(), ".md") &&
			(strings.Contains(info.Name(), "test") ||
				strings.Contains(info.Name(), "spec")) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// TestFileInfo represents parsed test file information
type TestFileInfo struct {
	TestCases  []*markdownparser.TestCase
	SQL        string
	Parameters map[string]any
}

// parseTestFile parses test cases from a markdown file
func (ftr *FixtureTestRunner) parseTestFile(filePath string) (*TestFileInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc, err := markdownparser.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Convert to pointers
	testCases := make([]*markdownparser.TestCase, len(doc.TestCases))
	for i := range doc.TestCases {
		testCases[i] = &doc.TestCases[i]
	}

	// Parse parameters from parameter block if present
	parameters := make(map[string]any)

	if doc.ParametersText != "" {
		// TODO: Parse YAML parameters from ParametersText
		// For now, leave empty
	}

	return &TestFileInfo{
		TestCases:  testCases,
		SQL:        doc.SQL,
		Parameters: parameters,
	}, nil
}

// filterTestFiles filters test files by the run pattern (filename without extension)
func (ftr *FixtureTestRunner) filterTestFiles(testFiles []string) []string {
	var filtered []string

	for _, file := range testFiles {
		// Get filename without extension
		filename := filepath.Base(file)
		nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

		// Use prefix matching like Go's -run flag
		if strings.HasPrefix(nameWithoutExt, ftr.runPattern) {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// FixtureTestResult represents the result of a fixture test
type FixtureTestResult struct {
	TestName string
	Success  bool
	Duration time.Duration
	Error    error
}

// FixtureTestSummary represents the summary of fixture test execution
type FixtureTestSummary struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	TotalDuration time.Duration
	Results       []FixtureTestResult
}

// PrintSummary prints the fixture test execution summary
func (ftr *FixtureTestRunner) PrintSummary(summary *FixtureTestSummary) {
	fmt.Printf("\n")
	fmt.Printf("=== Fixture Test Summary ===\n")
	fmt.Printf("Tests: %d total, %d passed, %d failed\n",
		summary.TotalTests, summary.PassedTests, summary.FailedTests)
	fmt.Printf("Duration: %.3fs\n", summary.TotalDuration.Seconds())

	if summary.FailedTests > 0 {
		fmt.Printf("\nFailed tests:\n")

		for _, result := range summary.Results {
			if !result.Success {
				fmt.Printf("  %s\n", result.TestName)

				if result.Error != nil {
					fmt.Printf("    Error: %v\n", result.Error)
				}
			}
		}
	}

	if summary.FailedTests == 0 {
		fmt.Printf("\nAll fixture tests passed! ✅\n")
	} else {
		fmt.Printf("\nSome fixture tests failed! ❌\n")
	}
}
