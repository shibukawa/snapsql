package testrunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TestRunner manages test execution across the project
type TestRunner struct {
	projectRoot  string
	verbose      bool
	runPattern   *regexp.Regexp
	includePaths []string
}

// TestResult represents the result of a single test package
type TestResult struct {
	Package  string
	Success  bool
	Duration time.Duration
	Output   string
	Error    error
}

// TestSummary represents the overall test execution summary
type TestSummary struct {
	TotalPackages  int
	PassedPackages int
	FailedPackages int
	TotalDuration  time.Duration
	Results        []TestResult
}

// NewTestRunner creates a new test runner instance
func NewTestRunner(projectRoot string) *TestRunner {
	return &TestRunner{
		projectRoot: projectRoot,
		verbose:     false,
		runPattern:  nil,
	}
}

// SetVerbose enables or disables verbose output
func (tr *TestRunner) SetVerbose(verbose bool) {
	tr.verbose = verbose
}

// SetRunPattern sets the test name filter pattern
func (tr *TestRunner) SetRunPattern(pattern string) error {
	if pattern == "" {
		tr.runPattern = nil
		return nil
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid run pattern: %w", err)
	}

	tr.runPattern = regex

	return nil
}

// SetIncludePaths restricts package discovery to the provided paths.
func (tr *TestRunner) SetIncludePaths(paths []string) {
	tr.includePaths = tr.includePaths[:0]

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		clean := p
		if !filepath.IsAbs(clean) {
			clean = filepath.Join(tr.projectRoot, clean)
		}

		clean = filepath.Clean(clean)
		tr.includePaths = append(tr.includePaths, clean)
	}
}

// RunAllTests executes all tests in the project
func (tr *TestRunner) RunAllTests(ctx context.Context) (*TestSummary, error) {
	packages, err := tr.findTestPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to find test packages: %w", err)
	}

	if tr.verbose {
		fmt.Printf("Found %d test packages\n", len(packages))
	}

	summary := &TestSummary{
		TotalPackages: len(packages),
		Results:       make([]TestResult, 0, len(packages)),
	}

	startTime := time.Now()

	for _, pkg := range packages {
		if tr.verbose {
			fmt.Printf("=== RUN   %s\n", pkg)
		}

		result := tr.runPackageTests(ctx, pkg)
		summary.Results = append(summary.Results, result)

		if result.Success {
			summary.PassedPackages++

			if tr.verbose {
				fmt.Printf("--- PASS: %s (%.3fs)\n", pkg, result.Duration.Seconds())
			}
		} else {
			summary.FailedPackages++

			if tr.verbose {
				fmt.Printf("--- FAIL: %s (%.3fs)\n", pkg, result.Duration.Seconds())

				if result.Error != nil {
					fmt.Printf("    Error: %v\n", result.Error)
				}
			}
		}
	}

	summary.TotalDuration = time.Since(startTime)

	return summary, nil
}

// findTestPackages discovers all packages with test files
func (tr *TestRunner) findTestPackages() ([]string, error) {
	packages := make([]string, 0)
	seen := make(map[string]struct{})

	if len(tr.includePaths) == 0 {
		if err := tr.collectTestPackages(tr.projectRoot, seen, &packages, true); err != nil {
			return nil, err
		}

		return packages, nil
	}

	for _, p := range tr.includePaths {
		if err := tr.collectTestPackages(p, seen, &packages, false); err != nil {
			return nil, err
		}
	}

	return packages, nil
}

func (tr *TestRunner) collectTestPackages(path string, seen map[string]struct{}, packages *[]string, allowSkipRoot bool) error {
	return walkAndProcessFiles(path, allowSkipRoot, func(p string, info os.FileInfo) {
		tr.maybeAddTestPackage(p, info, seen, packages)
	})
}

func (tr *TestRunner) maybeAddTestPackage(path string, info os.FileInfo, seen map[string]struct{}, packages *[]string) {
	if info.IsDir() {
		return
	}

	if !strings.HasSuffix(info.Name(), "_test.go") {
		return
	}

	dir := filepath.Dir(path)

	relDir, err := filepath.Rel(tr.projectRoot, dir)
	if err != nil || strings.HasPrefix(relDir, "..") {
		return
	}

	pkgPath := "./" + filepath.ToSlash(relDir)
	if _, ok := seen[pkgPath]; ok {
		return
	}

	seen[pkgPath] = struct{}{}
	*packages = append(*packages, pkgPath)
}

// runPackageTests executes tests for a single package
func (tr *TestRunner) runPackageTests(ctx context.Context, pkg string) TestResult {
	startTime := time.Now()

	args := []string{"test"}

	if tr.verbose {
		args = append(args, "-v")
	}

	if tr.runPattern != nil {
		args = append(args, "-run", tr.runPattern.String())
	}

	args = append(args, pkg)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = tr.projectRoot

	output, err := cmd.CombinedOutput()

	result := TestResult{
		Package:  pkg,
		Success:  err == nil,
		Duration: time.Since(startTime),
		Output:   string(output),
		Error:    err,
	}

	return result
}

// PrintSummary prints the test execution summary
func (tr *TestRunner) PrintSummary(summary *TestSummary) {
	fmt.Printf("\n")
	fmt.Printf("=== Test Summary ===\n")
	fmt.Printf("Packages: %d total, %d passed, %d failed\n",
		summary.TotalPackages, summary.PassedPackages, summary.FailedPackages)
	fmt.Printf("Duration: %.3fs\n", summary.TotalDuration.Seconds())

	if summary.FailedPackages > 0 {
		fmt.Printf("\nFailed packages:\n")

		for _, result := range summary.Results {
			if !result.Success {
				fmt.Printf("  %s\n", result.Package)

				if result.Error != nil {
					fmt.Printf("    Error: %v\n", result.Error)
				}
			}
		}
	}

	if summary.FailedPackages == 0 {
		fmt.Printf("\nAll tests passed! ✅\n")
	} else {
		fmt.Printf("\nSome tests failed! ❌\n")
	}
}
