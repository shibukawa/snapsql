package testrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestRunner(t *testing.T) {
	projectRoot := "/test/project"
	runner := NewTestRunner(projectRoot)
	
	assert.Equal(t, projectRoot, runner.projectRoot)
	assert.False(t, runner.verbose)
	assert.Nil(t, runner.runPattern)
}

func TestSetVerbose(t *testing.T) {
	runner := NewTestRunner("/test")
	
	runner.SetVerbose(true)
	assert.True(t, runner.verbose)
	
	runner.SetVerbose(false)
	assert.False(t, runner.verbose)
}

func TestSetRunPattern(t *testing.T) {
	runner := NewTestRunner("/test")
	
	// Valid pattern
	err := runner.SetRunPattern("TestExample")
	assert.NoError(t, err)
	assert.NotNil(t, runner.runPattern)
	assert.Equal(t, "TestExample", runner.runPattern.String())
	
	// Empty pattern should clear the filter
	err = runner.SetRunPattern("")
	assert.NoError(t, err)
	assert.Nil(t, runner.runPattern)
	
	// Invalid pattern
	err = runner.SetRunPattern("[invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid run pattern")
}

func TestFindTestPackages(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "testrunner_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create test directory structure
	testDirs := []string{
		"pkg1",
		"pkg2/subpkg",
		"pkg3",
		"vendor/external", // Should be skipped
		".git/hooks",      // Should be skipped
	}
	
	for _, dir := range testDirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}
	
	// Create test files
	testFiles := []string{
		"pkg1/main_test.go",
		"pkg1/helper_test.go",
		"pkg2/subpkg/sub_test.go",
		"pkg3/another_test.go",
		"vendor/external/vendor_test.go", // Should be skipped
		".git/hooks/hook_test.go",        // Should be skipped
	}
	
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("package test"), 0644)
		require.NoError(t, err)
	}
	
	// Create non-test files (should be ignored)
	nonTestFiles := []string{
		"pkg1/main.go",
		"pkg2/subpkg/sub.go",
	}
	
	for _, file := range nonTestFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("package main"), 0644)
		require.NoError(t, err)
	}
	
	runner := NewTestRunner(tempDir)
	packages, err := runner.findTestPackages()
	require.NoError(t, err)
	
	expected := []string{
		"./pkg1",
		"./pkg2/subpkg",
		"./pkg3",
	}
	
	assert.ElementsMatch(t, expected, packages)
}

func TestRunPackageTests(t *testing.T) {
	// This test requires a real Go environment
	// We'll test with the current package
	wd, err := os.Getwd()
	require.NoError(t, err)
	
	projectRoot := filepath.Dir(wd) // Go up one level from testrunner
	runner := NewTestRunner(projectRoot)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Test running tests for the testrunner package itself
	result := runner.runPackageTests(ctx, "./testrunner")
	
	// The test should complete (success or failure doesn't matter for this test)
	assert.NotEmpty(t, result.Package)
	assert.Equal(t, "./testrunner", result.Package)
	assert.Greater(t, result.Duration, time.Duration(0))
	// Don't assert on output being non-empty as it might be empty in some cases
}

func TestTestSummary(t *testing.T) {
	summary := &TestSummary{
		TotalPackages:  3,
		PassedPackages: 2,
		FailedPackages: 1,
		TotalDuration:  time.Second * 5,
		Results: []TestResult{
			{Package: "./pkg1", Success: true, Duration: time.Second},
			{Package: "./pkg2", Success: true, Duration: time.Second * 2},
			{Package: "./pkg3", Success: false, Duration: time.Second * 2},
		},
	}
	
	assert.Equal(t, 3, summary.TotalPackages)
	assert.Equal(t, 2, summary.PassedPackages)
	assert.Equal(t, 1, summary.FailedPackages)
	assert.Equal(t, time.Second*5, summary.TotalDuration)
	assert.Len(t, summary.Results, 3)
}

func TestRunPatternMatching(t *testing.T) {
	runner := NewTestRunner("/test")
	
	// Test pattern compilation
	err := runner.SetRunPattern("TestExample.*")
	require.NoError(t, err)
	
	// Test that the pattern was compiled correctly
	assert.True(t, runner.runPattern.MatchString("TestExampleOne"))
	assert.True(t, runner.runPattern.MatchString("TestExampleTwo"))
	assert.False(t, runner.runPattern.MatchString("TestOther"))
}
