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

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// setupPostgreSQLContainer starts a PostgreSQL container and returns the database connection
func setupPostgreSQLContainer(ctx context.Context, t *testing.T) (*sql.DB, func()) {
	// Start PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get PostgreSQL connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Create tables
	schema := `
	CREATE TABLE users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE,
		age INTEGER,
		status VARCHAR(50) DEFAULT 'inactive',
		department_id INTEGER
	);

	CREATE TABLE departments (
		id INTEGER PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT
	);

	INSERT INTO departments (id, name, description) VALUES 
		(1, 'Engineering', 'Software development'),
		(2, 'Design', 'UI/UX design'),
		(3, 'Marketing', 'Product marketing');
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create PostgreSQL schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate PostgreSQL container: %v", err)
		}
	}

	return db, cleanup
}

// setupMySQLContainer starts a MySQL container and returns the database connection
func setupMySQLContainer(ctx context.Context, t *testing.T) (*sql.DB, func()) {
	// Start MySQL container
	mysqlContainer, err := mysql.RunContainer(ctx,
		testcontainers.WithImage("mysql:8.0"),
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start MySQL container: %v", err)
	}

	// Get connection string
	connStr, err := mysqlContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get MySQL connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Wait for connection to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
		if i == 29 {
			t.Fatalf("Failed to ping MySQL after 30 seconds")
		}
	}

	// Create tables
	schemas := []string{
		`CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			age INT,
			status VARCHAR(50) DEFAULT 'inactive',
			department_id INT
		)`,
		`CREATE TABLE departments (
			id INT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT
		)`,
		`INSERT INTO departments (id, name, description) VALUES 
			(1, 'Engineering', 'Software development'),
			(2, 'Design', 'UI/UX design'),
			(3, 'Marketing', 'Product marketing')`,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			t.Fatalf("Failed to create MySQL schema: %v", err)
		}
	}

	cleanup := func() {
		db.Close()
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate MySQL container: %v", err)
		}
	}

	return db, cleanup
}

// runTestCaseWithDB executes a test case with a specific database connection
func runTestCaseWithDB(t *testing.T, db *sql.DB, dialect, driver, connStr, testCaseName string) {
	t.Helper()

	// Find the test file (look in testdata/testrunner/markdown directory)
	testFile := filepath.Join("..", "testdata", "testrunner", "markdown", fmt.Sprintf("%s.md", testCaseName))
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Test file not found: %s", testFile)
	}

	// Parse the test file with database override
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file %s: %v", testFile, err)
	}
	defer file.Close()

	options := &markdownparser.ParseOptions{
		DatabaseOverride: &markdownparser.DatabaseConfig{
			Driver:     driver,
			Connection: connStr,
			Dialect:    dialect,
		},
	}

	doc, err := markdownparser.ParseWithOptions(file, options)
	if err != nil {
		t.Fatalf("Failed to parse test file %s: %v", testFile, err)
	}

	if len(doc.TestCases) == 0 {
		t.Fatalf("No test cases found in %s", testFile)
	}

	// Create test runner
	executionOptions := &fixtureexecutor.ExecutionOptions{
		Mode:     fixtureexecutor.FullTest,
		Commit:   false, // Always rollback for tests
		Parallel: 1,     // Sequential execution for tests
		Timeout:  time.Minute * 5,
	}

	runner := fixtureexecutor.NewTestRunner(db, dialect, executionOptions)
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
		t.Logf("✅ %s (%s): All %d test(s) passed", testCaseName, dialect, summary.PassedTests)
	}
}

func TestPostgreSQLBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL container test in short mode")
	}

	ctx := context.Background()
	db, cleanup := setupPostgreSQLContainer(ctx, t)
	defer cleanup()

	// Get connection string for override
	connStr := fmt.Sprintf("postgres://testuser:testpass@%s/testdb?sslmode=disable", 
		"localhost") // This will be overridden by the actual container connection

	runTestCaseWithDB(t, db, "postgres", "pgx", connStr, "test_postgres_basic")
}

func TestMySQLBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL container test in short mode")
	}

	ctx := context.Background()
	db, cleanup := setupMySQLContainer(ctx, t)
	defer cleanup()

	// Get connection string for override
	connStr := "testuser:testpass@tcp(localhost:3306)/testdb"

	runTestCaseWithDB(t, db, "mysql", "mysql", connStr, "test_mysql_basic")
}

// Comprehensive test that runs all database types
func TestMultiDatabaseSupport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-database test in short mode")
	}

	ctx := context.Background()

	t.Run("PostgreSQL", func(t *testing.T) {
		db, cleanup := setupPostgreSQLContainer(ctx, t)
		defer cleanup()

		connStr := "postgres://testuser:testpass@localhost/testdb?sslmode=disable"
		runTestCaseWithDB(t, db, "postgres", "pgx", connStr, "test_postgres_basic")
	})

	t.Run("MySQL", func(t *testing.T) {
		db, cleanup := setupMySQLContainer(ctx, t)
		defer cleanup()

		connStr := "testuser:testpass@tcp(localhost:3306)/testdb"
		runTestCaseWithDB(t, db, "mysql", "mysql", connStr, "test_mysql_basic")
	})

	t.Run("SQLite", func(t *testing.T) {
		// Use the existing SQLite test setup
		db := setupTestDatabase(t)
		defer db.Close()
		defer cleanupTestDatabase(t)

		runTestCase(t, db, "test_select")
	})

	t.Logf("🎉 Multi-database support test completed successfully!")
}
