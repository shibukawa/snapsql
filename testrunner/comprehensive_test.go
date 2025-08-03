package testrunner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"

	_ "github.com/mattn/go-sqlite3"
)

const testDBPath = "test_comprehensive_go.db"

// setupTestDatabase initializes a clean test database with required tables
func setupTestDatabase(t *testing.T) *sql.DB {
	// Remove existing database
	if _, err := os.Stat(testDBPath); err == nil {
		if err := os.Remove(testDBPath); err != nil {
			t.Fatalf("Failed to remove existing test database: %v", err)
		}
	}

	// Create new database
	db, err := sql.Open("sqlite3", testDBPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	schema := `
	-- Create users table with all required columns
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		age INTEGER,
		status TEXT DEFAULT 'inactive',
		department_id INTEGER
	);

	-- Create departments table
	CREATE TABLE departments (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT
	);

	-- Insert some initial reference data
	INSERT INTO departments (id, name, description) VALUES 
		(1, 'Engineering', 'Software development'),
		(2, 'Design', 'UI/UX design'),
		(3, 'Marketing', 'Product marketing');
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create database schema: %v", err)
	}

	return db
}

// cleanupTestDatabase removes the test database
func cleanupTestDatabase(t *testing.T) {
	if err := os.Remove(testDBPath); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: Failed to cleanup test database: %v", err)
	}
}

// runTestCase executes a single test case and returns the result
func runTestCase(t *testing.T, db *sql.DB, testCaseName string) {
	t.Helper()

	// Find the test file (look in testdata/testrunner/markdown directory)
	testFile := filepath.Join("..", "testdata", "testrunner", "markdown", fmt.Sprintf("%s.md", testCaseName))
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Test file not found: %s", testFile)
	}

	// Parse the test file
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file %s: %v", testFile, err)
	}
	defer file.Close()

	doc, err := markdownparser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse test file %s: %v", testFile, err)
	}

	if len(doc.TestCases) == 0 {
		t.Fatalf("No test cases found in %s", testFile)
	}

	// Create test runner
	options := &fixtureexecutor.ExecutionOptions{
		Mode:     fixtureexecutor.FullTest,
		Commit:   false, // Always rollback for tests
		Parallel: 1,     // Sequential execution for tests
		Timeout:  time.Minute * 5,
	}

	runner := fixtureexecutor.NewTestRunner(db, "sqlite", options)
	runner.SetSQL(doc.SQL)
	runner.SetParameters(make(map[string]any))

	// Execute all test cases in the file
	ctx := context.Background()
	testCases := make([]*markdownparser.TestCase, len(doc.TestCases))
	for i := range doc.TestCases {
		testCases[i] = &doc.TestCases[i]
	}

	summary, err := runner.RunTests(ctx, testCases)
	if err != nil {
		t.Fatalf("Failed to run tests for %s: %v", testCaseName, err)
	}

	// Check results
	if summary.FailedTests > 0 {
		t.Errorf("Test case %s failed:", testCaseName)
		for _, result := range summary.Results {
			if !result.Success {
				t.Errorf("  - %s: %v", result.TestCase.Name, result.Error)
			}
		}
	} else {
		t.Logf("✅ %s: All %d test(s) passed", testCaseName, summary.PassedTests)
	}
}

// Test cases for each SQL operation type

func TestSelectQuery(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_select")
}

func TestInsertNoReturning(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_insert_no_returning")
}

func TestInsertReturning(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_insert_returning")
}

func TestInsertVerify(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_insert_verify")
}

func TestUpdateNoReturning(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_update_no_returning")
}

func TestUpdateReturning(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_update_returning")
}

func TestDeleteNoReturning(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_delete_no_returning")
}

func TestDeleteReturning(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_delete_returning")
}

func TestDeleteVerify(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()
	defer cleanupTestDatabase(t)

	runTestCase(t, db, "test_delete_verify")
}

// Comprehensive test that runs all test cases
func TestComprehensiveSQLiteTestSuite(t *testing.T) {
	testCases := []string{
		"test_select",
		"test_insert_no_returning",
		"test_insert_returning",
		"test_insert_verify",
		"test_update_no_returning",
		"test_update_returning",
		"test_delete_no_returning",
		"test_delete_returning",
		"test_delete_verify",
	}

	totalTests := len(testCases)
	passedTests := 0

	t.Logf("🚀 Running Comprehensive SQLite Test Suite (%d test cases)", totalTests)

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			db := setupTestDatabase(t)
			defer db.Close()
			defer cleanupTestDatabase(t)

			runTestCase(t, db, testCase)
			passedTests++
		})
	}

	t.Logf("🎉 Comprehensive test suite completed: %d/%d tests passed", passedTests, totalTests)
}

// Benchmark test for performance measurement
func BenchmarkComprehensiveTestSuite(b *testing.B) {
	testCases := []string{
		"test_select",
		"test_insert_no_returning",
		"test_insert_returning",
		"test_insert_verify",
		"test_update_no_returning",
		"test_update_returning",
		"test_delete_no_returning",
		"test_delete_returning",
		"test_delete_verify",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, testCase := range testCases {
			func() {
				// Create a temporary test database for benchmarking
				tempDB := fmt.Sprintf("bench_%d_%s.db", i, testCase)
				defer os.Remove(tempDB)

				db, err := sql.Open("sqlite3", tempDB)
				if err != nil {
					b.Fatalf("Failed to open benchmark database: %v", err)
				}
				defer db.Close()

				// Create schema
				schema := `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					email TEXT NOT NULL UNIQUE,
					age INTEGER,
					status TEXT DEFAULT 'inactive',
					department_id INTEGER
				);
				CREATE TABLE departments (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					description TEXT
				);
				INSERT INTO departments (id, name, description) VALUES 
					(1, 'Engineering', 'Software development'),
					(2, 'Design', 'UI/UX design'),
					(3, 'Marketing', 'Product marketing');
				`
				if _, err := db.Exec(schema); err != nil {
					b.Fatalf("Failed to create benchmark schema: %v", err)
				}

				// Find the test file
				testFile := filepath.Join("..", "testdata", "testrunner", "markdown", fmt.Sprintf("%s.md", testCase))
				file, err := os.Open(testFile)
				if err != nil {
					b.Fatalf("Failed to open test file %s: %v", testFile, err)
				}
				defer file.Close()

				doc, err := markdownparser.Parse(file)
				if err != nil {
					b.Fatalf("Failed to parse test file %s: %v", testFile, err)
				}

				// Create test runner
				options := &fixtureexecutor.ExecutionOptions{
					Mode:     fixtureexecutor.FullTest,
					Commit:   false,
					Parallel: 1,
					Timeout:  time.Minute * 5,
				}

				runner := fixtureexecutor.NewTestRunner(db, "sqlite", options)
				runner.SetSQL(doc.SQL)
				runner.SetParameters(make(map[string]any))

				// Execute test cases
				ctx := context.Background()
				testCases := make([]*markdownparser.TestCase, len(doc.TestCases))
				for i := range doc.TestCases {
					testCases[i] = &doc.TestCases[i]
				}

				_, err = runner.RunTests(ctx, testCases)
				if err != nil {
					b.Fatalf("Failed to run tests for %s: %v", testCase, err)
				}
			}()
		}
	}
}
