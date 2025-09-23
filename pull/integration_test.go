package pull

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver (pgx)
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestPostgreSQLIntegration tests the complete pull operation with a real PostgreSQL database
func TestPostgreSQLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := t.Context()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, postgresContainer.Terminate(ctx))
	}()

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	assert.NoError(t, err)

	// Connect and set up test data
	db, err := sql.Open("pgx", connStr)
	assert.NoError(t, err)

	defer db.Close()

	// Create test schema and tables
	err = setupPostgreSQLTestData(db)
	assert.NoError(t, err)

	// Test the pull operation
	t.Run("FullPullOperation", func(t *testing.T) {
		tempDir := t.TempDir()

		config := PullConfig{
			DatabaseURL:    connStr,
			DatabaseType:   "postgresql",
			OutputPath:     tempDir,
			SchemaAware:    true,
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		result, err := ExecutePull(t.Context(), config)
		if err != nil {
			t.Logf("Pull operation failed with error: %v", err)
			t.Logf("Connection string: %s", connStr)

			// Try to connect directly to debug
			testDB, dbErr := sql.Open("pgx", connStr)
			if dbErr != nil {
				t.Logf("Direct connection failed: %v", dbErr)
			} else {
				defer testDB.Close()

				var count int

				countErr := testDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
				if countErr != nil {
					t.Logf("Query test failed: %v", countErr)
				} else {
					t.Logf("Users table has %d rows", count)
				}
			}
		}

		assert.NoError(t, err)
		assert.NotZero(t, result)
		assert.Equal(t, 1, len(result.Schemas))
		assert.Equal(t, "public", result.Schemas[0].Name)
		assert.True(t, len(result.Schemas[0].Tables) >= 2) // users and posts tables

		// Verify files were created
		publicDir := filepath.Join(tempDir, "public")
		dirExists(t, publicDir)
		fileExists(t, filepath.Join(publicDir, "users.yaml"))
		fileExists(t, filepath.Join(publicDir, "posts.yaml"))

		// Verify YAML content
		usersContent, err := os.ReadFile(filepath.Join(publicDir, "users.yaml"))
		assert.NoError(t, err)

		usersYAML := string(usersContent)
		assert.Contains(t, usersYAML, "name: users")
		assert.Contains(t, usersYAML, "name: id")
		assert.Contains(t, usersYAML, "name: email")
		assert.Contains(t, usersYAML, "snapsql_type: int")
		assert.Contains(t, usersYAML, "snapsql_type: string")
	})

	t.Run("PullWithFilters", func(t *testing.T) {
		tempDir := t.TempDir()

		config := PullConfig{
			DatabaseURL:   connStr,
			DatabaseType:  "postgresql",
			OutputPath:    tempDir,
			SchemaAware:   true,
			IncludeTables: []string{"users"}, // Only include users table
		}

		result, err := ExecutePull(t.Context(), config)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(result.Schemas))

		// Should only have users table
		usersTables := 0

		for _, table := range result.Schemas[0].Tables {
			if table.Name == "users" {
				usersTables++
			}
		}

		assert.Equal(t, 1, usersTables)

		// Verify only users.yaml was created
		publicDir := filepath.Join(tempDir, "public")
		fileExists(t, filepath.Join(publicDir, "users.yaml"))

		// posts.yaml should not exist
		_, err = os.Stat(filepath.Join(publicDir, "posts.yaml"))
		assert.Error(t, err) // File should not exist
	})
}

// TestMySQLIntegration tests the complete pull operation with a real MySQL database
func TestMySQLIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := t.Context()

	// Start MySQL container
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.4",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
	)
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, mysqlContainer.Terminate(ctx))
	}()

	// Get connection string
	connStr, err := mysqlContainer.ConnectionString(ctx)
	assert.NoError(t, err)

	// Connect and set up test data
	db, err := sql.Open("mysql", connStr)
	assert.NoError(t, err)

	defer db.Close()

	// MySQL module waits for readiness; fixed sleep is unnecessary and slows tests

	// Create test schema and tables
	err = setupMySQLTestData(db)
	assert.NoError(t, err)

	// Test the pull operation
	t.Run("FullPullOperation", func(t *testing.T) {
		tempDir := t.TempDir()

		// Convert MySQL connection string to our URL format
		mysqlURL := convertMySQLConnStrToURL(connStr)
		t.Logf("Original connection string: %s", connStr)
		t.Logf("Converted MySQL URL: %s", mysqlURL)

		config := PullConfig{
			DatabaseURL:  mysqlURL,
			DatabaseType: "mysql",
			OutputPath:   tempDir,
			SchemaAware:  false, // MySQL doesn't use schema-aware structure in our test
		}

		result, err := ExecutePull(t.Context(), config)
		if err != nil {
			t.Logf("Pull operation failed with error: %v", err)
		}

		assert.NoError(t, err)
		assert.NotZero(t, result)
		assert.Equal(t, 1, len(result.Schemas))
		assert.True(t, len(result.Schemas[0].Tables) >= 2) // users and posts tables

		// Verify files were created
		fileExists(t, filepath.Join(tempDir, "users.yaml"))
		fileExists(t, filepath.Join(tempDir, "posts.yaml"))
	})
}

// TestSQLiteIntegration tests the complete pull operation with SQLite
func TestSQLiteIntegration(t *testing.T) {
	// Removed short guard to allow execution in short mode
	// Create temporary SQLite database
	tempDir := t.TempDir()

	dbPath := filepath.Join(tempDir, "test.db")

	// Connect and set up test data
	db, err := sql.Open("sqlite3", dbPath)
	assert.NoError(t, err)

	defer db.Close()

	// Create test schema and tables
	err = setupSQLiteTestData(db)
	assert.NoError(t, err)

	// Test the pull operation
	t.Run("FullPullOperation", func(t *testing.T) {
		outputDir := filepath.Join(tempDir, "output")

		config := PullConfig{
			DatabaseURL:  "sqlite://" + dbPath,
			DatabaseType: "sqlite",
			OutputPath:   outputDir,
			SchemaAware:  true,
		}

		result, err := ExecutePull(t.Context(), config)
		assert.NoError(t, err)
		assert.NotZero(t, result)
		assert.Equal(t, 1, len(result.Schemas))
		assert.Equal(t, "global", result.Schemas[0].Name)  // SQLite uses 'global' schema
		assert.True(t, len(result.Schemas[0].Tables) >= 2) // users and posts tables

		// Verify files were created
		globalDir := filepath.Join(outputDir, "global")
		dirExists(t, globalDir)
		fileExists(t, filepath.Join(globalDir, "users.yaml"))
		fileExists(t, filepath.Join(globalDir, "posts.yaml"))

		// Verify YAML content
		usersContent, err := os.ReadFile(filepath.Join(globalDir, "users.yaml"))
		assert.NoError(t, err)

		usersYAML := string(usersContent)
		assert.Contains(t, usersYAML, "name: users")
		assert.Contains(t, usersYAML, "name: id")
		assert.Contains(t, usersYAML, "name: email")
	})
}

// Helper functions to set up test data

func setupPostgreSQLTestData(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(200) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published)`,
		`INSERT INTO users (email, name) VALUES 
			('john@example.com', 'John Doe'),
			('jane@example.com', 'Jane Smith')
			ON CONFLICT (email) DO NOTHING`,
		`INSERT INTO posts (user_id, title, content, published) VALUES 
			(1, 'First Post', 'This is the first post', true),
			(2, 'Second Post', 'This is the second post', false)
			ON CONFLICT DO NOTHING`,
		`CREATE VIEW active_users AS 
			SELECT id, email, name FROM users WHERE created_at > NOW() - INTERVAL '30 days'`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}

func setupMySQLTestData(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT,
			title VARCHAR(200) NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE INDEX idx_users_email ON users(email)`,
		`CREATE INDEX idx_posts_user_id ON posts(user_id)`,
		`INSERT IGNORE INTO users (email, name) VALUES 
			('john@example.com', 'John Doe'),
			('jane@example.com', 'Jane Smith')`,
		`INSERT IGNORE INTO posts (user_id, title, content, published) VALUES 
			(1, 'First Post', 'This is the first post', true),
			(2, 'Second Post', 'This is the second post', false)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}

func setupSQLiteTestData(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER REFERENCES users(id),
			title TEXT NOT NULL,
			content TEXT,
			published BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id)`,
		`INSERT OR IGNORE INTO users (email, name) VALUES 
			('john@example.com', 'John Doe'),
			('jane@example.com', 'Jane Smith')`,
		`INSERT OR IGNORE INTO posts (user_id, title, content, published) VALUES 
			(1, 'First Post', 'This is the first post', 1),
			(2, 'Second Post', 'This is the second post', 0)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}

// convertMySQLConnStrToURL converts MySQL connection string to URL format
func convertMySQLConnStrToURL(connStr string) string {
	// connStr format: "user:password@tcp(host:port)/database"
	// Convert to: "mysql://user:password@host:port/database"
	if strings.Contains(connStr, "@tcp(") {
		parts := strings.Split(connStr, "@tcp(")
		if len(parts) == 2 {
			userPass := parts[0]
			hostPortDb := parts[1]

			// Remove the closing parenthesis
			hostPortDb = strings.Replace(hostPortDb, ")", "", 1)

			return fmt.Sprintf("mysql://%s@%s", userPass, hostPortDb)
		}
	}

	// If conversion fails, try to use the original connection string directly
	// This might work if the MySQL driver accepts it
	return "mysql://" + connStr
}

// TestDatabaseConnectorWithRealDatabases tests the connector with real database connections
func TestDatabaseConnectorWithRealDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("PostgreSQLConnection", func(t *testing.T) {
		ctx := t.Context()

		postgresContainer, err := postgres.Run(ctx,
			"postgres:17-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			postgres.BasicWaitStrategies(),
		)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, postgresContainer.Terminate(ctx))
		}()

		connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
		assert.NoError(t, err)

		connector := NewDatabaseConnector()

		// Test URL parsing
		dbType, err := connector.ParseDatabaseURL(connStr)
		assert.NoError(t, err)
		assert.Equal(t, "postgresql", dbType)

		// Test connection
		db, err := connector.Connect(connStr)
		assert.NoError(t, err)

		defer connector.Close(db)

		// Test ping
		err = connector.Ping(db)
		assert.NoError(t, err)

		// Test query execution
		var version string

		err = db.QueryRow("SELECT version()").Scan(&version)
		assert.NoError(t, err)
		assert.Contains(t, version, "PostgreSQL")
	})

	t.Run("SQLiteConnection", func(t *testing.T) {
		tempDir := t.TempDir()

		dbPath := filepath.Join(tempDir, "test.db")
		sqliteURL := "sqlite://" + dbPath

		connector := NewDatabaseConnector()

		// Test URL parsing
		dbType, err := connector.ParseDatabaseURL(sqliteURL)
		assert.NoError(t, err)
		assert.Equal(t, "sqlite", dbType)

		// Test connection
		db, err := connector.Connect(sqliteURL)
		assert.NoError(t, err)

		defer connector.Close(db)

		// Test ping
		err = connector.Ping(db)
		assert.NoError(t, err)

		// Test query execution
		var version string

		err = db.QueryRow("SELECT sqlite_version()").Scan(&version)
		assert.NoError(t, err)
		assert.True(t, len(version) > 0)
	})
}
