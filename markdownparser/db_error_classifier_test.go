package markdownparser

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// setupPostgres initializes a PostgreSQL container for testing.
// Returns the database connection and a cleanup function.
func setupPostgres(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping postgres container in short mode")
	}

	ctx := context.Background()

	cont, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	connStr, err := cont.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = cont.Terminate(ctx)

		t.Fatalf("postgres conn string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		_ = cont.Terminate(ctx)

		t.Fatalf("open postgres: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	// wait until ready
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Ping(); err == nil {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	cleanup := func() {
		_ = db.Close()
		_ = cont.Terminate(context.Background())
	}

	return db, cleanup
}

// setupMySQL initializes a MySQL container for testing.
// Returns the database connection and a cleanup function.
func setupMySQL(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping mysql container in short mode")
	}

	ctx := context.Background()

	cont, err := mysql.Run(ctx,
		"mysql:8.4",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("start mysql: %v", err)
	}

	connStr, err := cont.ConnectionString(ctx)
	if err != nil {
		_ = cont.Terminate(ctx)

		t.Fatalf("mysql conn string: %v", err)
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		_ = cont.Terminate(ctx)

		t.Fatalf("open mysql: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	// wait until ready
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Ping(); err == nil {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	cleanup := func() {
		_ = db.Close()
		_ = cont.Terminate(context.Background())
	}

	return db, cleanup
}

// setupSQLite initializes an in-memory SQLite database for testing.
// Returns the database connection and a cleanup function.
func setupSQLite(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup
}

// setupTestTable creates a test table with various constraints
func setupTestTable(t *testing.T, db *sql.DB, dialect string) {
	t.Helper()

	// Drop table if exists
	_, _ = db.Exec("DROP TABLE IF EXISTS test_orders")
	_, _ = db.Exec("DROP TABLE IF EXISTS test_users")

	// Create users table with various constraints
	var createUserSQL string

	switch dialect {
	case "postgres":
		createUserSQL = `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(100) UNIQUE NOT NULL,
				age INTEGER CHECK (age >= 0),
				name VARCHAR(50) NOT NULL
			)`
	case "mysql":
		createUserSQL = `
			CREATE TABLE test_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				email VARCHAR(100) UNIQUE NOT NULL,
				age INT CHECK (age >= 0),
				name VARCHAR(50) NOT NULL
			)`
	case "sqlite":
		createUserSQL = `
			CREATE TABLE test_users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				email TEXT UNIQUE NOT NULL,
				age INTEGER CHECK (age >= 0),
				name TEXT NOT NULL
			)`
	}

	if _, err := db.Exec(createUserSQL); err != nil {
		t.Fatalf("create users table: %v", err)
	}

	// Create orders table for foreign key tests
	var createOrderSQL string

	switch dialect {
	case "postgres":
		createOrderSQL = `
			CREATE TABLE test_orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES test_users(id),
				amount DECIMAL(10, 2) NOT NULL
			)`
	case "mysql":
		createOrderSQL = `
			CREATE TABLE test_orders (
				id INT AUTO_INCREMENT PRIMARY KEY,
				user_id INT NOT NULL,
				amount DECIMAL(10, 2) NOT NULL,
				FOREIGN KEY (user_id) REFERENCES test_users(id)
			)`
	case "sqlite":
		// SQLite requires foreign keys to be enabled
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			t.Fatalf("enable foreign keys: %v", err)
		}

		createOrderSQL = `
			CREATE TABLE test_orders (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL REFERENCES test_users(id),
				amount REAL NOT NULL
			)`
	}

	if _, err := db.Exec(createOrderSQL); err != nil {
		t.Fatalf("create orders table: %v", err)
	}
}

func TestClassifyDatabaseError_SQLite(t *testing.T) {
	db, cleanup := setupSQLite(t)
	defer cleanup()

	setupTestTable(t, db, "sqlite")

	t.Run("unique violation", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM test_users")
		_, _ = db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Alice', 25)")

		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Bob', 30)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeUniqueViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeUniqueViolation, errorType, err)
		}
	})

	t.Run("foreign key violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_orders (user_id, amount) VALUES (999, 100.00)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeForeignKeyViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeForeignKeyViolation, errorType, err)
		}
	})

	t.Run("not null violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_users (email, age) VALUES ('test2@example.com', 25)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeNotNullViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeNotNullViolation, errorType, err)
		}
	})

	t.Run("check violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test3@example.com', 'Charlie', -5)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeCheckViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeCheckViolation, errorType, err)
		}
	})
}

func TestClassifyDatabaseError_PostgreSQL(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	setupTestTable(t, db, "postgres")

	t.Run("unique violation", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM test_orders")
		_, _ = db.Exec("DELETE FROM test_users WHERE email = 'test@example.com'")
		_, _ = db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Alice', 25)")

		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Bob', 30)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeUniqueViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeUniqueViolation, errorType, err)
		}
	})

	t.Run("foreign key violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_orders (user_id, amount) VALUES (999, 100.00)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeForeignKeyViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeForeignKeyViolation, errorType, err)
		}
	})

	t.Run("not null violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_users (email, age) VALUES ('test2@example.com', 25)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeNotNullViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeNotNullViolation, errorType, err)
		}
	})

	t.Run("check violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test3@example.com', 'Charlie', -5)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeCheckViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeCheckViolation, errorType, err)
		}
	})
}

func TestClassifyDatabaseError_MySQL(t *testing.T) {
	db, cleanup := setupMySQL(t)
	defer cleanup()

	setupTestTable(t, db, "mysql")

	t.Run("unique violation", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM test_orders")
		_, _ = db.Exec("DELETE FROM test_users WHERE email = 'test@example.com'")
		_, _ = db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Alice', 25)")

		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Bob', 30)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeUniqueViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeUniqueViolation, errorType, err)
		}
	})

	t.Run("foreign key violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_orders (user_id, amount) VALUES (999, 100.00)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeForeignKeyViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeForeignKeyViolation, errorType, err)
		}
	})

	t.Run("not null violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_users (email, age) VALUES ('test2@example.com', 25)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeNotNullViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeNotNullViolation, errorType, err)
		}
	})

	t.Run("check violation", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test3@example.com', 'Charlie', -5)")
		if err == nil {
			t.Fatal("expected error but got none")
		}

		errorType := ClassifyDatabaseError(err)
		if errorType != ErrorTypeCheckViolation {
			t.Errorf("expected error type %q, got %q (original error: %v)", ErrorTypeCheckViolation, errorType, err)
		}
	})
}

func TestMatchesExpectedError(t *testing.T) {
	db, cleanup := setupSQLite(t)
	defer cleanup()

	setupTestTable(t, db, "sqlite")

	t.Run("match unique violation with space notation", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM test_users")
		_, _ = db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Alice', 25)")
		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test@example.com', 'Bob', 30)")

		matches, msg := MatchesExpectedError(err, "unique violation")
		if !matches {
			t.Errorf("expected match=true, got match=false (message: %s)", msg)
		}
	})

	t.Run("match unique violation with underscore notation", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM test_users")
		_, _ = db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test2@example.com', 'Alice', 25)")
		_, err := db.Exec("INSERT INTO test_users (email, name, age) VALUES ('test2@example.com', 'Bob', 30)")

		matches, msg := MatchesExpectedError(err, "unique_violation")
		if !matches {
			t.Errorf("expected match=true, got match=false (message: %s)", msg)
		}
	})

	t.Run("mismatch - expected unique but got foreign key", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO test_orders (user_id, amount) VALUES (999, 100.00)")

		matches, msg := MatchesExpectedError(err, "unique violation")
		if matches {
			t.Errorf("expected match=false, got match=true")
		}

		if msg == "" || !containsIgnoreCase(msg, "error type mismatch") {
			t.Errorf("expected error message to contain 'error type mismatch', got %q", msg)
		}
	})

	t.Run("no error when error expected", func(t *testing.T) {
		matches, msg := MatchesExpectedError(nil, "unique violation")
		if matches {
			t.Errorf("expected match=false, got match=true")
		}

		if msg == "" || !containsIgnoreCase(msg, "expected error but got no error") {
			t.Errorf("expected error message to contain 'expected error but got no error', got %q", msg)
		}
	})
}

func containsIgnoreCase(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if matchesAtIgnoreCase(s[i:], substr) {
			return true
		}
	}

	return false
}

func matchesAtIgnoreCase(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	for i := range len(substr) {
		if toLower(s[i]) != toLower(substr[i]) {
			return false
		}
	}

	return true
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}

	return b
}
