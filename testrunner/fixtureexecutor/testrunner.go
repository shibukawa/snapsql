package fixtureexecutor

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
)

// TestResult represents the result of a single test execution
type TestResult struct {
	TestCase          *markdownparser.TestCase
	Success           bool
	Duration          time.Duration
	Result            *ValidationResult
	Trace             []SQLTrace
	Error             error
	ExpectedError     *string // Expected error type from test case
	ActualErrorType   string  // Classified error type
	ErrorMatch        bool    // Whether error matched expected
	ErrorMatchMessage string  // Detailed error match message
}

// TestSummary represents the overall test execution summary
type TestSummary struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	TotalDuration time.Duration
	Results       []TestResult
}

// TestRunner manages parallel test execution
type TestRunner struct {
	executor   *Executor
	workerPool chan struct{} // セマフォ
	options    *ExecutionOptions
	sql        string         // SQL query from document
	parameters map[string]any // Default parameters from document
}

// NewTestRunner creates a new test runner
func NewTestRunner(db *sql.DB, dialect string, options *ExecutionOptions) *TestRunner {
	if options == nil {
		options = DefaultExecutionOptions()
	}

	return &TestRunner{
		executor:   NewExecutor(db, dialect, make(map[string]*snapsql.TableInfo)), // schema info can be injected later via SetTableInfo
		workerPool: make(chan struct{}, options.Parallel),
		options:    options,
		parameters: make(map[string]any),
	}
}

// SetTableInfo injects or replaces the schema information used during fixture execution.
// This is primarily used by unit tests that construct an in-memory database schema on the fly.
// It is safe to call multiple times; the reference is replaced atomically without locking because
// TestRunner / Executor instances are not expected to mutate tableInfo concurrently with active executions.
func (tr *TestRunner) SetTableInfo(tableInfo map[string]*snapsql.TableInfo) {
	if tr.executor != nil {
		tr.executor.tableInfo = tableInfo
	}
}

// SetBaseDir sets the base directory used to resolve external file references during execution.
func (tr *TestRunner) SetBaseDir(dir string) {
	if tr.executor != nil {
		tr.executor.SetBaseDir(dir)
	}
}

// SetSQL sets the SQL query for test execution
func (tr *TestRunner) SetSQL(sql string) {
	tr.sql = sql
}

// SetParameters sets the default parameters for test execution
func (tr *TestRunner) SetParameters(parameters map[string]any) {
	tr.parameters = parameters
}

// SetVerbose toggles verbose SQL tracing.
func (tr *TestRunner) SetVerbose(verbose bool) {
	if tr.options != nil {
		tr.options.Verbose = verbose
	}
}

// RunTests executes multiple test cases in parallel
func (tr *TestRunner) RunTests(ctx context.Context, testCases []*markdownparser.TestCase) (*TestSummary, error) {
	summary := &TestSummary{
		TotalTests: len(testCases),
		Results:    make([]TestResult, 0, len(testCases)),
	}

	startTime := time.Now()

	// Results channel
	results := make(chan TestResult, len(testCases))

	var wg sync.WaitGroup

	// Execute tests in parallel
	for _, testCase := range testCases {
		wg.Add(1)

		go func(tc *markdownparser.TestCase) {
			defer wg.Done()

			result := tr.executeTestWithTimeout(ctx, tc)
			results <- result
		}(testCase)
	}

	// Wait for all tests to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		summary.Results = append(summary.Results, result)
		if result.Success {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}
	}

	summary.TotalDuration = time.Since(startTime)

	return summary, nil
}

// executeTestWithTimeout executes a single test with timeout and semaphore
func (tr *TestRunner) executeTestWithTimeout(ctx context.Context, testCase *markdownparser.TestCase) TestResult {
	// Acquire semaphore
	select {
	case tr.workerPool <- struct{}{}:
		defer func() { <-tr.workerPool }()
	case <-ctx.Done():
		return TestResult{
			TestCase: testCase,
			Success:  false,
			Error:    ctx.Err(),
		}
	}

	// Create timeout context
	testCtx, cancel := context.WithTimeout(ctx, tr.options.Timeout)
	defer cancel()

	startTime := time.Now()

	// Execute test
	result, trace, err := tr.executeTestWithContext(testCtx, testCase)

	// Handle error test cases
	if testCase.ExpectedError != nil {
		return tr.handleErrorTest(testCase, result, trace, err, time.Since(startTime))
	}

	// Handle normal test cases
	return TestResult{
		TestCase: testCase,
		Success:  err == nil,
		Duration: time.Since(startTime),
		Result:   result,
		Trace:    trace,
		Error:    err,
	}
}

// handleErrorTest handles test cases that expect an error
func (tr *TestRunner) handleErrorTest(testCase *markdownparser.TestCase, result *ValidationResult, trace []SQLTrace, err error, duration time.Duration) TestResult {
	testResult := TestResult{
		TestCase:      testCase,
		Duration:      duration,
		Result:        result,
		Trace:         trace,
		Error:         err,
		ExpectedError: testCase.ExpectedError,
	}

	if err == nil {
		// Expected error but got success
		testResult.Success = false
		testResult.ErrorMatch = false
		testResult.ErrorMatchMessage = "expected error but operation succeeded"
		return testResult
	}

	// Classify the actual error
	actualErrorType := markdownparser.ClassifyDatabaseError(err)
	testResult.ActualErrorType = string(actualErrorType)

	// Check if error matches expected type
	matches, message := markdownparser.MatchesExpectedError(err, *testCase.ExpectedError)
	testResult.ErrorMatch = matches
	testResult.ErrorMatchMessage = message
	testResult.Success = matches

	return testResult
}

// executeTestWithContext executes a test within a context
func (tr *TestRunner) executeTestWithContext(ctx context.Context, testCase *markdownparser.TestCase) (*ValidationResult, []SQLTrace, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	// Merge default parameters with test case parameters
	parameters := make(map[string]any)
	for k, v := range tr.parameters {
		parameters[k] = v
	}

	for k, v := range testCase.Parameters {
		parameters[k] = v
	}

	sql := testCase.SQL
	if sql == "" {
		sql = tr.sql
	}

	// Execute the test
	return tr.executor.ExecuteTest(testCase, sql, parameters, tr.options)
}

// RunSingleTest executes a single test case
func (tr *TestRunner) RunSingleTest(ctx context.Context, testCase *markdownparser.TestCase) (*TestResult, error) {
	result := tr.executeTestWithTimeout(ctx, testCase)
	return &result, nil
}

// PrintSummary prints the test execution summary
func (tr *TestRunner) PrintSummary(summary *TestSummary) {
	fmt.Printf("\n")
	fmt.Printf("=== Test Summary ===\n")
	fmt.Printf("Tests: %d total, %d passed, %d failed\n",
		summary.TotalTests, summary.PassedTests, summary.FailedTests)
	fmt.Printf("Duration: %.3fs\n", summary.TotalDuration.Seconds())

	if summary.FailedTests > 0 {
		fmt.Printf("\nFailed tests:\n")

		for _, result := range summary.Results {
			if !result.Success {
				fmt.Printf("  %s\n", result.TestCase.Name)

				// Error test specific output
				if result.ExpectedError != nil {
					fmt.Printf("    Expected Error: %s\n", *result.ExpectedError)
					if result.Error != nil {
						fmt.Printf("    Actual Error:   %s\n", result.ActualErrorType)
						if tr.options.Verbose {
							fmt.Printf("    Error Details:  %v\n", result.Error)
						}
					} else {
						fmt.Printf("    Actual Error:   <none>\n")
					}
					if result.ErrorMatchMessage != "" {
						fmt.Printf("    Reason: %s\n", result.ErrorMatchMessage)
					}
				} else if result.Error != nil {
					// Normal test with unexpected error
					fmt.Printf("    Error: %v\n", result.Error)
					if tr.options.Verbose && len(result.Trace) > 0 {
						fmt.Printf("    SQL Trace:\n")
						for _, trace := range result.Trace {
							fmt.Printf("      %s: %s\n", trace.Label, trace.Statement)
						}
					}
				}
			}
		}
	}

	// Verbose mode: Show error details for all error tests
	if tr.options.Verbose && summary.PassedTests > 0 {
		hasErrorTests := false
		for _, result := range summary.Results {
			if result.Success && result.ExpectedError != nil {
				if !hasErrorTests {
					fmt.Printf("\nPassed error tests (verbose):\n")
					hasErrorTests = true
				}
				fmt.Printf("  %s\n", result.TestCase.Name)
				fmt.Printf("    Expected Error: %s\n", *result.ExpectedError)
				fmt.Printf("    Actual Error:   %s\n", result.ActualErrorType)
				fmt.Printf("    Error Details:  %v\n", result.Error)
			}
		}
	}

	if summary.FailedTests == 0 {
		fmt.Printf("\nAll tests passed! ✅\n")
	} else {
		fmt.Printf("\nSome tests failed! ❌\n")
	}
}

// SetOptions updates the execution options

func (tr *TestRunner) SetOptions(options *ExecutionOptions) {
	if options == nil {
		options = DefaultExecutionOptions()
	}

	tr.options = options
	// Recreate worker pool if parallel count changed
	if tr.workerPool == nil || cap(tr.workerPool) != options.Parallel {
		tr.workerPool = make(chan struct{}, options.Parallel)
	}
}
