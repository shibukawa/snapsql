package testrunner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/explain"
	"github.com/shibukawa/snapsql/intermediate"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/query"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
)

const verboseListLimit = 200

// Static errors for err113 compliance
var (
	ErrMissingMarkdownDocumentContext = errors.New("missing markdown document context")
)

// FixtureTestRunner manages fixture-based test execution
type FixtureTestRunner struct {
	projectRoot  string
	db           *sql.DB
	dialect      snapsql.Dialect
	verbose      bool
	runPattern   string
	options      *fixtureexecutor.ExecutionOptions
	tableInfo    map[string]*snapsql.TableInfo
	includePaths []string
	testCaseMeta map[*markdownparser.TestCase]*testCaseMetadata
}

type preparationIssue struct {
	testCase *markdownparser.TestCase
	filePath string
	line     int
	name     string
	err      error
}

type fileTestSummary struct {
	path  string
	cases []*markdownparser.TestCase
	doc   *markdownparser.SnapSQLDocument
}

// NewFixtureTestRunner creates a new fixture test runner

func NewFixtureTestRunner(projectRoot string, db *sql.DB, dialect snapsql.Dialect) *FixtureTestRunner {
	runner := &FixtureTestRunner{
		projectRoot:  projectRoot,
		db:           db,
		dialect:      dialect,
		verbose:      false,
		options:      fixtureexecutor.DefaultExecutionOptions(),
		tableInfo:    nil,
		testCaseMeta: make(map[*markdownparser.TestCase]*testCaseMetadata),
	}

	return runner
}

// SetVerbose enables or disables verbose output
func (ftr *FixtureTestRunner) SetVerbose(verbose bool) {
	ftr.verbose = verbose
	if ftr.options != nil {
		ftr.options.Verbose = verbose
	}
}

// SetRunPattern sets the test name filter pattern
func (ftr *FixtureTestRunner) SetRunPattern(pattern string) {
	ftr.runPattern = pattern
}

// SetExecutionOptions sets the fixture execution options
func (ftr *FixtureTestRunner) SetExecutionOptions(options *fixtureexecutor.ExecutionOptions) {
	ftr.options = options
}

// SetTableInfo injects table metadata used for fixture validation.
func (ftr *FixtureTestRunner) SetTableInfo(tableInfo map[string]*snapsql.TableInfo) {
	ftr.tableInfo = tableInfo
}

// SetIncludePaths restricts discovery to specific paths (absolute or relative to project root).
func (ftr *FixtureTestRunner) SetIncludePaths(paths []string) {
	ftr.includePaths = ftr.includePaths[:0]

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		clean := p
		if !filepath.IsAbs(clean) {
			clean = filepath.Join(ftr.projectRoot, clean)
		}

		clean = filepath.Clean(clean)
		ftr.includePaths = append(ftr.includePaths, clean)
	}
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
	var allTestCases []*markdownparser.TestCase

	fileSummaries := make([]fileTestSummary, 0, len(testFiles))
	parseIssues := make([]preparationIssue, 0)

	for _, file := range testFiles {
		fileInfo, err := ftr.parseTestFile(file)
		if err != nil {
			var displayPath string
			if rel, relErr := filepath.Rel(ftr.projectRoot, file); relErr == nil && !strings.HasPrefix(rel, "..") {
				displayPath = filepath.ToSlash(rel)
			} else {
				displayPath = filepath.ToSlash(file)
			}

			issue := preparationIssue{
				filePath: displayPath,
				name:     "Parse " + filepath.Base(displayPath),
				err:      fmt.Errorf("failed to parse markdown: %w", err),
			}

			parseIssues = append(parseIssues, issue)

			if ftr.verbose {
				fmt.Printf("Warning: failed to parse %s: %v\n", displayPath, err)
			}

			continue
		}

		// Use SQL and parameters from the first successfully parsed file
		casesForFile := make([]*markdownparser.TestCase, 0, len(fileInfo.TestCases))
		for _, tc := range fileInfo.TestCases {
			if tc == nil {
				continue
			}

			if tc.SQL == "" {
				tc.SQL = fileInfo.SQL
			}

			if len(fileInfo.Parameters) > 0 {
				if tc.Parameters == nil {
					tc.Parameters = make(map[string]any, len(fileInfo.Parameters))
				}

				for k, v := range fileInfo.Parameters {
					if _, exists := tc.Parameters[k]; !exists {
						tc.Parameters[k] = v
					}
				}
			}

			allTestCases = append(allTestCases, tc)
			casesForFile = append(casesForFile, tc)
		}

		fileSummaries = append(fileSummaries, fileTestSummary{path: file, cases: casesForFile, doc: fileInfo.Document})
	}

	if ftr.verbose {
		ftr.printVerboseDiscovery(fileSummaries)
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

	runnableCases, prepIssues := ftr.prepareTestCases(fileSummaries)
	additionalIssues := append(parseIssues, prepIssues...)

	if ftr.verbose {
		fmt.Printf("Executing %d test cases\n", len(runnableCases))
		fmt.Printf("Execution mode: %s\n", ftr.options.Mode)
		fmt.Printf("Parallel workers: %d\n", ftr.options.Parallel)
		fmt.Printf("Commit after test: %t\n", ftr.options.Commit)
		fmt.Println()
	}

	var summary *fixtureexecutor.TestSummary

	if len(runnableCases) > 0 {
		// Create test runner and execute tests
		runner := fixtureexecutor.NewTestRunner(ftr.db, ftr.dialect, ftr.options)
		runner.SetVerbose(ftr.verbose)

		if len(ftr.tableInfo) > 0 {
			runner.SetTableInfo(ftr.tableInfo)
		}

		runner.SetTableReferences(ftr.collectTableReferences(runnableCases))

		summary, err = runner.RunTests(ctx, runnableCases)
		if err != nil {
			return nil, fmt.Errorf("failed to run tests: %w", err)
		}
	} else {
		summary = &fixtureexecutor.TestSummary{}
	}

	// Convert to FixtureTestSummary
	fixtureSummary := &FixtureTestSummary{
		TotalTests:    summary.TotalTests + len(additionalIssues),
		PassedTests:   summary.PassedTests,
		FailedTests:   summary.FailedTests + len(additionalIssues),
		TotalDuration: summary.TotalDuration,
		Results:       make([]FixtureTestResult, 0, len(summary.Results)+len(additionalIssues)),
	}

	for _, result := range summary.Results {
		kind := fixtureexecutor.ClassifyFailure(result.Error)
		sourceFile := ""
		sourceLine := 0

		if result.TestCase != nil {
			sourceFile = result.TestCase.SourceFile
			sourceLine = result.TestCase.Line
		}

		testName := "<unknown test>"
		if result.TestCase != nil && strings.TrimSpace(result.TestCase.Name) != "" {
			testName = result.TestCase.Name
		}

		fixtureSummary.Results = append(fixtureSummary.Results, FixtureTestResult{
			TestName:    testName,
			TestCase:    result.TestCase,
			Success:     result.Success,
			Duration:    result.Duration,
			Error:       result.Error,
			FailureKind: kind,
			SourceFile:  sourceFile,
			SourceLine:  sourceLine,
			ExecutedSQL: result.Trace,
			Performance: result.Performance,
		})

		if !result.Success {
			switch kind {
			case fixtureexecutor.FailureKindAssertion:
				fixtureSummary.AssertionFailures++
			case fixtureexecutor.FailureKindDefinition:
				fixtureSummary.DefinitionFailures++
			default:
				fixtureSummary.UnknownFailures++
			}
		}
	}

	for _, issue := range additionalIssues {
		fixtureSummary.Results = append(fixtureSummary.Results, issue.toFixtureResult())
		fixtureSummary.DefinitionFailures++
	}

	return fixtureSummary, nil
}

// findMarkdownTestFiles finds all markdown test files in the project
func (ftr *FixtureTestRunner) findMarkdownTestFiles() ([]string, error) {
	var files []string

	seen := make(map[string]struct{})

	if len(ftr.includePaths) == 0 {
		if err := ftr.collectMarkdownFiles(ftr.projectRoot, seen, &files, true); err != nil {
			return nil, err
		}

		return files, nil
	}

	for _, p := range ftr.includePaths {
		if err := ftr.collectMarkdownFiles(p, seen, &files, false); err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (ftr *FixtureTestRunner) collectMarkdownFiles(path string, seen map[string]struct{}, files *[]string, allowSkipRoot bool) error {
	return walkAndProcessFiles(path, allowSkipRoot, func(p string, info os.FileInfo) {
		ftr.maybeAddMarkdownFile(p, info, seen, files)
	})
}

func (ftr *FixtureTestRunner) maybeAddMarkdownFile(path string, info os.FileInfo, seen map[string]struct{}, files *[]string) {
	if info.IsDir() {
		return
	}

	name := info.Name()

	if strings.HasSuffix(name, ".snap.md") || (strings.HasSuffix(name, ".md") && (strings.Contains(name, "test") || strings.Contains(name, "spec"))) {
		if _, ok := seen[path]; ok {
			return
		}

		seen[path] = struct{}{}
		*files = append(*files, path)
	}
}

// TestFileInfo represents parsed test file information
type TestFileInfo struct {
	TestCases  []*markdownparser.TestCase
	SQL        string
	Parameters map[string]any
	Document   *markdownparser.SnapSQLDocument
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
		testCases[i].SQL = doc.SQL
	}

	relPath := filepath.ToSlash(filePath)
	if rel, err := filepath.Rel(ftr.projectRoot, filePath); err == nil && !strings.HasPrefix(rel, "..") {
		relPath = filepath.ToSlash(rel)
	}

	for _, tc := range testCases {
		if tc == nil {
			continue
		}

		tc.SourceFile = relPath
	}

	// Parse parameters from parameter block if present
	parameters := make(map[string]any)

	if doc.ParametersText != "" {
		// YAML parameter parsing is not yet implemented
		// Parameters are currently handled through individual test cases
		_ = doc.ParametersText // Explicitly acknowledge the unused parameter text
	}

	return &TestFileInfo{
		TestCases:  testCases,
		SQL:        doc.SQL,
		Parameters: parameters,
		Document:   doc,
	}, nil
}

func (ftr *FixtureTestRunner) prepareTestCases(summaries []fileTestSummary) ([]*markdownparser.TestCase, []preparationIssue) {
	if len(summaries) == 0 {
		return nil, nil
	}

	config := &snapsql.Config{Dialect: ftr.dialect}
	valid := make([]*markdownparser.TestCase, 0)
	issues := make([]preparationIssue, 0)

	for _, summary := range summaries {
		if len(summary.cases) == 0 {
			continue
		}

		if summary.doc == nil {
			for _, tc := range summary.cases {
				issues = append(issues, preparationIssue{
					testCase: tc,
					err:      fmt.Errorf("%w: %s", ErrMissingMarkdownDocumentContext, tc.Name),
				})
			}

			continue
		}

		format, err := intermediate.GenerateFromMarkdown(summary.doc, summary.path, ftr.projectRoot, nil, ftr.tableInfo, config)
		if err != nil {
			for _, tc := range summary.cases {
				issues = append(issues, preparationIssue{
					testCase: tc,
					err:      fmt.Errorf("failed to compile SQL for %s: %w", tc.Name, err),
				})
			}

			continue
		}

		tableMap := intermediate.BuildTableReferenceMap(format.TableReferences)
		ftr.registerTableReferences(summary.cases, tableMap)

		generator := query.NewSQLGenerator(format, config.Dialect)
		ordered := format.HasOrderedResult

		for _, tc := range summary.cases {
			if tc == nil {
				continue
			}

			if err := fixtureexecutor.NormalizeParameters(tc.Parameters); err != nil {
				issues = append(issues, preparationIssue{
					testCase: tc,
					err:      fmt.Errorf("failed to normalize parameters for %s: %w", tc.Name, err),
				})

				continue
			}

			finalSQL, args, err := generator.Generate(tc.Parameters)
			if err != nil {
				issues = append(issues, preparationIssue{
					testCase: tc,
					err:      fmt.Errorf("failed to render SQL for %s: %w", tc.Name, err),
				})

				continue
			}

			tc.PreparedSQL = finalSQL
			tc.SQLArgs = args
			tc.ResultOrdered = ordered
			valid = append(valid, tc)
		}
	}

	return valid, issues
}

func (pi preparationIssue) toFixtureResult() FixtureTestResult {
	name := strings.TrimSpace(pi.name)
	if name == "" {
		name = "<unknown test>"
	}

	file := pi.filePath
	line := pi.line

	if pi.testCase != nil {
		if trimmed := strings.TrimSpace(pi.testCase.Name); trimmed != "" {
			name = trimmed
		}

		if tcFile := strings.TrimSpace(pi.testCase.SourceFile); tcFile != "" {
			file = tcFile
		}

		if pi.testCase.Line > 0 {
			line = pi.testCase.Line
		}
	}

	return FixtureTestResult{
		TestName:    name,
		TestCase:    pi.testCase,
		Success:     false,
		Error:       pi.err,
		FailureKind: fixtureexecutor.FailureKindDefinition,
		SourceFile:  file,
		SourceLine:  line,
	}
}

func (ftr *FixtureTestRunner) printVerboseDiscovery(summaries []fileTestSummary) {
	if len(summaries) == 0 {
		fmt.Println("Discovered markdown tests (files: 0, cases: 0)")
		fmt.Println()

		return
	}

	totalCases := 0
	for _, summary := range summaries {
		totalCases += len(summary.cases)
	}

	fmt.Printf("Discovered markdown tests (files: %d, cases: %d):\n", len(summaries), totalCases)

	fileLimit := min(len(summaries), verboseListLimit)

	for i := range fileLimit {
		summary := summaries[i]

		path := summary.path
		if rel, err := filepath.Rel(ftr.projectRoot, path); err == nil {
			path = rel
		}

		fmt.Printf("  %s\n", filepath.ToSlash(path))

		if len(summary.cases) == 0 {
			fmt.Println("    (no test cases)")
			continue
		}

		caseLimit := min(len(summary.cases), verboseListLimit)

		for j := range caseLimit {
			tc := summary.cases[j]
			if tc == nil {
				continue
			}

			name := strings.TrimSpace(tc.Name)
			if name == "" {
				name = "<unnamed>"
			}

			fmt.Printf("    - %s\n", name)
		}

		if len(summary.cases) > verboseListLimit {
			fmt.Printf("    ... (%d more)\n", len(summary.cases)-verboseListLimit)
		}
	}

	if len(summaries) > verboseListLimit {
		fmt.Printf("  ... (%d more files)\n", len(summaries)-verboseListLimit)
	}

	fmt.Println()
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
	TestName    string
	TestCase    *markdownparser.TestCase
	Success     bool
	Duration    time.Duration
	Error       error
	FailureKind fixtureexecutor.FailureKind
	SourceFile  string
	SourceLine  int
	ExecutedSQL []fixtureexecutor.SQLTrace
	Performance *explain.PerformanceEvaluation
}

// FixtureTestSummary represents the summary of fixture test execution
type FixtureTestSummary struct {
	TotalTests         int
	PassedTests        int
	FailedTests        int
	TotalDuration      time.Duration
	Results            []FixtureTestResult
	AssertionFailures  int
	DefinitionFailures int
	UnknownFailures    int
}

// PrintSummary prints the fixture test execution summary
func (ftr *FixtureTestRunner) PrintSummary(summary *FixtureTestSummary) {
	fmt.Fprintln(color.Output)
	fmt.Fprintln(color.Output, "=== Fixture Test Summary ===")
	fmt.Fprintf(color.Output, "Tests: %d total, %d passed, %d failed\n",
		summary.TotalTests, summary.PassedTests, summary.FailedTests)

	if summary.FailedTests > 0 {
		fmt.Fprintf(color.Output, "Assertions Failed: %d, Definition Failures: %d, Unknown Failures: %d\n",
			summary.AssertionFailures, summary.DefinitionFailures, summary.UnknownFailures)
	}

	fmt.Fprintf(color.Output, "Duration: %.3fs\n", summary.TotalDuration.Seconds())

	fileOrder, fileGroups := groupResultsByFile(summary.Results)

	if ftr.verbose {
		ftr.printVerboseFileReports(fileOrder, fileGroups)
	}

	if summary.FailedTests > 0 {
		fmt.Fprintln(color.Output, "\nFailed tests:")

		assertionLabel := color.New(color.Bold, color.FgYellow).SprintFunc()
		definitionLabel := color.New(color.Bold, color.FgRed).SprintFunc()
		unknownLabel := color.New(color.Bold, color.FgMagenta).SprintFunc()

		sortedResults := make([]FixtureTestResult, len(summary.Results))
		copy(sortedResults, summary.Results)
		sort.Slice(sortedResults, func(i, j int) bool {
			a := sortedResults[i]

			b := sortedResults[j]
			if a.SourceFile != b.SourceFile {
				return a.SourceFile < b.SourceFile
			}

			if a.SourceLine != b.SourceLine {
				if a.SourceLine == 0 {
					return false
				}

				if b.SourceLine == 0 {
					return true
				}

				return a.SourceLine < b.SourceLine
			}

			return a.TestName < b.TestName
		})

		for _, result := range sortedResults {
			if result.Success {
				continue
			}

			emoji := "❔"
			labelText := "[Unknown]"
			styledLabel := unknownLabel(labelText)
			marker := "?"

			switch result.FailureKind {
			case fixtureexecutor.FailureKindAssertion:
				emoji = "⚠️"
				labelText = "[Failure]"
				styledLabel = assertionLabel(labelText)
				marker = "⚠"
			case fixtureexecutor.FailureKindDefinition:
				emoji = "❌"
				labelText = "[Error]"
				styledLabel = definitionLabel(labelText)
				marker = "✖"
			}

			location := result.SourceFile
			if location != "" && result.SourceLine > 0 {
				location = fmt.Sprintf("%s:%d", location, result.SourceLine)
			}

			if location != "" {
				fmt.Fprintf(color.Output, "  %s %s %s %s (%s)\n", marker, emoji, styledLabel, result.TestName, location)
			} else {
				fmt.Fprintf(color.Output, "  %s %s %s %s\n", marker, emoji, styledLabel, result.TestName)
			}

			if result.Error != nil {
				fmt.Fprintf(color.Output, "    Error: %v\n", result.Error)

				if ff, ok := fixtureexecutor.AsFixtureFailure(result.Error); ok {
					ctx := ff.Context()
					if table := ctx["table"]; table != "" {
						fmt.Fprintf(color.Output, "    Table: %s\n", table)
					}

					if line := ctx["line"]; line != "" {
						fmt.Fprintf(color.Output, "    Line: %s\n", line)
					}

					if strategy := ctx["strategy"]; strategy != "" {
						fmt.Fprintf(color.Output, "    Strategy: %s\n", strategy)
					}

					if op := ctx["operation"]; op != "" {
						fmt.Fprintf(color.Output, "    Operation: %s\n", op)
					}

					if ftr.verbose {
						if sql := ctx["sql"]; sql != "" {
							fmt.Fprintf(color.Output, "    SQL: %s\n", sql)
						}

						if args := ctx["args"]; args != "" {
							fmt.Fprintf(color.Output, "    Args: %s\n", args)
						}
					}
				}

				if diff, ok := fixtureexecutor.AsDiffError(result.Error); ok {
					diffText := fixtureexecutor.FormatDiffUnifiedYAML(diff)
					if diffText != "" {
						fmt.Fprintln(color.Output)
						printColoredDiff(diffText)
					}
				}

				if ftr.verbose && len(result.ExecutedSQL) > 0 {
					fmt.Fprintln(color.Output, "    SQL Trace:")
					printSQLTrace(result.ExecutedSQL)
				}

				fmt.Fprintln(color.Output)
				fmt.Fprintln(color.Output)
			}
		}
	}

	if summary.FailedTests == 0 {
		fmt.Fprintln(color.Output, "\nAll fixture tests passed! ✅")
	} else {
		fmt.Fprintln(color.Output, "\nSome fixture tests failed! ❌")
	}
}

type testCaseMetadata struct {
	tableMap map[string]intermediate.TableReferenceInfo
}

func (ftr *FixtureTestRunner) registerTableReferences(cases []*markdownparser.TestCase, mapping map[string]intermediate.TableReferenceInfo) {
	if len(cases) == 0 {
		return
	}

	for _, tc := range cases {
		if tc == nil {
			continue
		}

		meta, ok := ftr.testCaseMeta[tc]
		if !ok {
			meta = &testCaseMetadata{}
			ftr.testCaseMeta[tc] = meta
		}

		if len(mapping) > 0 {
			meta.tableMap = mapping
		}
	}
}

func (ftr *FixtureTestRunner) tableMapFor(tc *markdownparser.TestCase) map[string]intermediate.TableReferenceInfo {
	if tc == nil {
		return nil
	}

	if meta, ok := ftr.testCaseMeta[tc]; ok {
		return meta.tableMap
	}

	return nil
}

func (ftr *FixtureTestRunner) collectTableReferences(cases []*markdownparser.TestCase) map[*markdownparser.TestCase]map[string]intermediate.TableReferenceInfo {
	if len(cases) == 0 {
		return make(map[*markdownparser.TestCase]map[string]intermediate.TableReferenceInfo)
	}

	refs := make(map[*markdownparser.TestCase]map[string]intermediate.TableReferenceInfo, len(cases))
	for _, tc := range cases {
		if tc == nil {
			continue
		}

		if mapping := ftr.tableMapFor(tc); len(mapping) > 0 {
			refs[tc] = mapping
		}
	}

	return refs
}

func groupResultsByFile(results []FixtureTestResult) ([]string, map[string][]FixtureTestResult) {
	groups := make(map[string][]FixtureTestResult)
	order := make([]string, 0)

	for _, res := range results {
		path := res.SourceFile
		if path == "" && res.TestCase != nil {
			path = res.TestCase.SourceFile
		}

		if path == "" {
			path = "<unknown>"
		}

		if _, exists := groups[path]; !exists {
			order = append(order, path)
			groups[path] = make([]FixtureTestResult, 0, 1)
		}

		groups[path] = append(groups[path], res)
	}

	return order, groups
}

func (ftr *FixtureTestRunner) printVerboseFileReports(fileOrder []string, groups map[string][]FixtureTestResult) {
	if len(fileOrder) == 0 {
		return
	}

	passLabel := color.New(color.Bold, color.FgGreen).Sprint("PASS")
	failLabel := color.New(color.Bold, color.FgRed).Sprint("FAIL")
	infoLabel := color.New(color.Bold, color.FgCyan).Sprint("INFO")
	warnLabel := color.New(color.Bold, color.FgYellow).Sprint("WARN")

	fmt.Fprintln(color.Output, "\nDetailed results by file:")

	for _, path := range fileOrder {
		results := groups[path]
		fmt.Fprintf(color.Output, "%s\n", path)

		for _, res := range results {
			statusLabel := passLabel
			if !res.Success {
				statusLabel = failLabel
			}

			name := res.TestName
			if strings.TrimSpace(name) == "" && res.TestCase != nil {
				name = res.TestCase.Name
			}

			fmt.Fprintf(color.Output, "  %s %s (%s)\n", statusLabel, strings.TrimSpace(name), formatDuration(res.Duration))

			if !res.Success && res.Error != nil {
				fmt.Fprintf(color.Output, "    error: %v\n", res.Error)
			}
		}

		perfs := ftr.aggregatePerformance(results, true)
		if len(perfs) > 0 {
			fmt.Fprintln(color.Output, "  Performance:")

			for _, agg := range perfs {
				rel := relativeLocation(path, agg.Location)
				for _, msg := range agg.WarnMessages {
					fmt.Fprintf(color.Output, "    - %s %s: %s\n", warnLabel, rel, msg)
				}

				for _, msg := range agg.InfoMessages {
					fmt.Fprintf(color.Output, "    - %s %s: %s\n", infoLabel, rel, msg)
				}
			}
		}

		fmt.Fprintln(color.Output)
	}
}

func (ftr *FixtureTestRunner) aggregatePerformance(results []FixtureTestResult, includeInfo bool) []performanceAggregate {
	order := make([]performanceAggregate, 0)
	index := make(map[string]int)

	for _, res := range results {
		if res.Performance == nil {
			continue
		}

		mapping := ftr.tableMapFor(res.TestCase)
		location := buildLocation(res)

		estMap := make(map[string]explain.QueryEstimate)
		for _, est := range res.Performance.Estimates {
			estMap[est.QueryPath] = est
		}

		warnedPaths := make(map[string]struct{})

		for _, warn := range res.Performance.Warnings {
			targets := ftr.describeTables(warn.Tables, mapping)
			if len(targets) == 0 {
				targets = []string{"table '<unknown>'"}
			}

			switch warn.Kind {
			case explain.WarningFullScan:
				for _, target := range targets {
					message := "full scan detected on " + target
					if warn.QueryPath != "" {
						message = fmt.Sprintf("%s [path=%s]", message, warn.QueryPath)
					}

					agg := ensureAggregate(&order, index, location)
					agg.WarnMessages = append(agg.WarnMessages, message)
				}
			default:
				message := warn.Message
				if warn.Kind == explain.WarningSlowQuery {
					if est, ok := estMap[warn.QueryPath]; ok {
						message = fmt.Sprintf("%s (actual=%s, estimated=%s, threshold=%s, scale=%.2f)",
							message,
							formatDuration(est.Actual),
							formatDuration(est.Estimated),
							formatDuration(est.Threshold),
							est.ScaleFactor,
						)
					}
				}

				if len(targets) > 0 {
					message = fmt.Sprintf("%s (tables=%s)", message, strings.Join(targets, ", "))
				}

				if warn.QueryPath != "" {
					message = fmt.Sprintf("%s [path=%s]", message, warn.QueryPath)
				}

				agg := ensureAggregate(&order, index, location)
				agg.WarnMessages = append(agg.WarnMessages, message)
			}

			if warn.QueryPath != "" {
				warnedPaths[warn.QueryPath] = struct{}{}
			}
		}

		if includeInfo {
			for _, est := range res.Performance.Estimates {
				if _, exists := warnedPaths[est.QueryPath]; exists {
					continue
				}

				message := fmt.Sprintf("estimated runtime (actual=%s, estimated=%s, threshold=%s, scale=%.2f)",
					formatDuration(est.Actual),
					formatDuration(est.Estimated),
					formatDuration(est.Threshold),
					est.ScaleFactor,
				)
				if est.QueryPath != "" {
					message = fmt.Sprintf("%s [path=%s]", message, est.QueryPath)
				}

				agg := ensureAggregate(&order, index, location)
				agg.InfoMessages = append(agg.InfoMessages, message)
			}
		}
	}

	filtered := make([]performanceAggregate, 0, len(order))
	for _, agg := range order {
		if len(agg.WarnMessages) == 0 && len(agg.InfoMessages) == 0 {
			continue
		}

		filtered = append(filtered, agg)
	}

	return filtered
}

func (ftr *FixtureTestRunner) describeTables(keys []string, mapping map[string]intermediate.TableReferenceInfo) []string {
	if len(keys) == 0 {
		return nil
	}

	physical := ftr.physicalNameCandidates()
	descriptions := make([]string, 0, len(keys))

	for _, key := range keys {
		trimmed := strings.TrimSpace(key)

		lower := strings.ToLower(trimmed)
		if trimmed != "" && strings.HasPrefix(lower, "table ") {
			descriptions = append(descriptions, trimmed)
			continue
		}

		described := intermediate.DescribePlanTables([]string{key}, mapping, physical)
		if len(described) == 0 {
			fallback := fallbackPlanTableDescriptions([]string{key})
			descriptions = append(descriptions, fallback...)

			continue
		}

		descriptions = append(descriptions, described...)
	}

	return descriptions
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

func ensureAggregate(order *[]performanceAggregate, index map[string]int, location string) *performanceAggregate {
	if location == "" {
		location = "<unknown>"
	}

	if idx, ok := index[location]; ok {
		return &(*order)[idx]
	}

	*order = append(*order, performanceAggregate{Location: location})
	idx := len(*order) - 1
	index[location] = idx

	return &(*order)[idx]
}

type performanceAggregate struct {
	Location     string
	WarnMessages []string
	InfoMessages []string
}

func buildLocation(res FixtureTestResult) string {
	path := res.SourceFile
	if path == "" && res.TestCase != nil {
		path = res.TestCase.SourceFile
	}

	if path == "" {
		path = "<unknown>"
	}

	line := res.SourceLine
	if line == 0 && res.TestCase != nil {
		line = res.TestCase.Line
	}

	if line > 0 {
		return fmt.Sprintf("%s:%d", path, line)
	}

	return path
}

func relativeLocation(path, location string) string {
	if location == "" {
		return "<unknown>"
	}

	if after, ok := strings.CutPrefix(location, path); ok {
		rest := after

		rest = strings.TrimPrefix(rest, ":")
		if rest == "" {
			return path
		}

		return "L" + rest
	}

	return location
}

func (ftr *FixtureTestRunner) physicalNameCandidates() []string {
	unique := make(map[string]struct{})

	add := func(name string) {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return
		}

		unique[trimmed] = struct{}{}
	}

	for _, info := range ftr.tableInfo {
		if info == nil {
			continue
		}

		if info.Schema != "" {
			add(info.Schema + "." + info.Name)
		}

		add(info.Name)
	}

	if ftr.options != nil {
		for key := range ftr.options.TableMetadata {
			add(key)
		}
	}

	if len(unique) == 0 {
		return nil
	}

	results := make([]string, 0, len(unique))
	for name := range unique {
		results = append(results, name)
	}

	sort.Strings(results)

	return results
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d/time.Microsecond)
	}

	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	}

	return d.Round(time.Millisecond).String()
}

func printColoredDiff(diffText string) {
	lines := strings.Split(diffText, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		if line == "" {
			fmt.Fprintln(color.Output)
			continue
		}

		fmt.Fprintln(color.Output, formatDiffLine(line))
	}
}

func formatDiffLine(line string) string {
	if line == "" {
		return line
	}

	if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
		return color.New(color.Bold, color.FgCyan).Sprint(line)
	}

	if strings.HasPrefix(line, "@@") {
		return color.New(color.FgCyan).Sprint(line)
	}

	attrs := make([]color.Attribute, 0, 3)

	switch {
	case strings.HasPrefix(line, "+"):
		attrs = append(attrs, color.FgGreen)
	case strings.HasPrefix(line, "-"):
		attrs = append(attrs, color.FgRed)
	}

	content := line
	if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
		content = line[1:]
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		if len(attrs) == 0 {
			return line
		}

		return color.New(attrs...).Sprint(line)
	}

	if strings.HasPrefix(trimmed, "row ") {
		attrs = append(attrs, color.Bold, color.FgBlue)
	} else if strings.HasPrefix(trimmed, "table:") || strings.HasPrefix(trimmed, "primary_keys:") || strings.HasPrefix(trimmed, "rows:") {
		attrs = append(attrs, color.Bold)
	}

	if len(attrs) == 0 {
		return line
	}

	return color.New(attrs...).Sprint(line)
}

func printSQLTrace(traces []fixtureexecutor.SQLTrace) {
	for i, trace := range traces {
		label := trace.Label
		if label == "" {
			label = "query"
		}

		fmt.Fprintf(color.Output, "      (%s)\n", label)
		fmt.Fprintf(color.Output, "        Statement: %s\n", trace.Statement)

		if len(trace.Parameters) > 0 {
			fmt.Fprintln(color.Output, "        Params:")

			keys := make([]string, 0, len(trace.Parameters))
			for k := range trace.Parameters {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			for _, k := range keys {
				fmt.Fprintf(color.Output, "          %s: %v\n", k, trace.Parameters[k])
			}
		}

		if len(trace.Args) > 0 {
			fmt.Fprintln(color.Output, "        Args:")

			for idx, arg := range trace.Args {
				fmt.Fprintf(color.Output, "          [%d]: %v\n", idx+1, arg)
			}
		}

		fmt.Fprintf(color.Output, "        Query Type: %s\n", trace.QueryType.String())

		if len(trace.Rows) > 0 {
			fmt.Fprintln(color.Output, "        Rows:")

			for _, row := range trace.Rows {
				fmt.Fprintf(color.Output, "          - %s\n", formatTraceRow(row))
			}

			if trace.RowsTruncated && trace.TotalRows > len(trace.Rows) {
				fmt.Fprintf(color.Output, "          ... (%d more rows)\n", trace.TotalRows-len(trace.Rows))
			}
		} else if trace.TotalRows > 0 {
			fmt.Fprintln(color.Output, "        Rows: (empty)")
		}

		if i < len(traces)-1 {
			fmt.Fprintln(color.Output)
		}
	}
}

func formatTraceRow(row map[string]any) string {
	if len(row) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, row[k]))
	}

	return strings.Join(parts, ", ")
}
