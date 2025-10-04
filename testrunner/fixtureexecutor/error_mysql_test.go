package fixtureexecutor

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMySQLErrorClassification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MySQL integration test in short mode")
	}

	ctx := context.Background()

	// Start MySQL container
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.4",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	defer func() {
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := mysqlContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Wait for connection to be ready
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			age INT CHECK (age >= 18 AND age <= 150),
			bio VARCHAR(100)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create posts table: %v", err)
	}

	// Insert test data
	_, err = db.Exec("INSERT INTO users (email, name, age) VALUES ('existing@example.com', 'Alice', 25)")
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	runner := NewTestRunner(db, "mysql", &ExecutionOptions{
		Mode:     FullTest,
		Timeout:  30 * time.Second,
		Verbose:  true,
		Commit:   false,
		Parallel: 1,
	})

	tests := []struct {
		name          string
		sql           string
		expectedError string
		wantSuccess   bool
		wantErrorType string
	}{
		{
			name:          "unique violation",
			sql:           "INSERT INTO users (email, name, age) VALUES ('existing@example.com', 'Bob', 30)",
			expectedError: "unique violation",
			wantSuccess:   true,
			wantErrorType: "unique violation",
		},
		{
			name:          "not null violation",
			sql:           "INSERT INTO users (email, name, age) VALUES ('test@example.com', NULL, 30)",
			expectedError: "not null violation",
			wantSuccess:   true,
			wantErrorType: "not null violation",
		},
		{
			name:          "check violation - age too low",
			sql:           "INSERT INTO users (email, name, age) VALUES ('minor@example.com', 'Bob', 15)",
			expectedError: "check violation",
			wantSuccess:   true,
			wantErrorType: "check violation",
		},
		{
			name:          "foreign key violation",
			sql:           "INSERT INTO posts (user_id, title, content) VALUES (9999, 'Test Post', 'Content')",
			expectedError: "foreign key violation",
			wantSuccess:   true,
			wantErrorType: "foreign key violation",
		},
		{
			name:          "data too long",
			sql:           "INSERT INTO users (email, name, age, bio) VALUES ('long@example.com', 'Bob', 30, '" + strings.Repeat("x", 150) + "')",
			expectedError: "data too long",
			wantSuccess:   true,
			wantErrorType: "data too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCase := &markdownparser.TestCase{
				Name:          tt.name,
				SQL:           tt.sql,
				Parameters:    map[string]any{},
				ExpectedError: &tt.expectedError,
			}

			ctx := context.Background()

			result, err := runner.RunSingleTest(ctx, testCase)
			if err != nil {
				t.Fatalf("RunSingleTest failed: %v", err)
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v\n  ErrorMatchMessage: %s\n  ActualError: %v",
					tt.wantSuccess, result.Success, result.ErrorMatchMessage, result.Error)
			}

			if result.ActualErrorType != tt.wantErrorType {
				t.Errorf("Expected ActualErrorType=%q, got %q\n  ActualError: %v",
					tt.wantErrorType, result.ActualErrorType, result.Error)
			}
		})
	}
}
