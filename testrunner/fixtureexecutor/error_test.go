package fixtureexecutor

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shibukawa/snapsql/markdownparser"
)

func TestErrorTestExecution(t *testing.T) {
	// Setup database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create test tables with various constraints
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			age INTEGER CHECK (age >= 18 AND age <= 150),
			bio TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
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

	// Enable foreign keys for SQLite
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	runner := NewTestRunner(db, "sqlite", &ExecutionOptions{
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
			name:          "check violation - age too high",
			sql:           "INSERT INTO users (email, name, age) VALUES ('old@example.com', 'Bob', 200)",
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
			name:          "error mismatch - expected unique but got not null",
			sql:           "INSERT INTO users (email, name, age) VALUES ('test2@example.com', NULL, 30)",
			expectedError: "unique violation",
			wantSuccess:   false,
			wantErrorType: "not null violation",
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
				t.Errorf("Expected success=%v, got %v\n  ErrorMatchMessage: %s",
					tt.wantSuccess, result.Success, result.ErrorMatchMessage)
			}
			if result.ActualErrorType != tt.wantErrorType {
				t.Errorf("Expected ActualErrorType=%q, got %q",
					tt.wantErrorType, result.ActualErrorType)
			}
			if tt.wantSuccess && !result.ErrorMatch {
				t.Errorf("Expected ErrorMatch=true for matching error, got false")
			}
			if !tt.wantSuccess && result.ErrorMatch {
				t.Errorf("Expected ErrorMatch=false for non-matching error, got true")
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
