package fixtureexecutor

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_ExecuteTest(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Minimal table info (schema validation path tolerant of empty ColumnOrder)
	tableInfo := map[string]*snapsql.TableInfo{
		"users": {
			Name: "users",
			Columns: map[string]*snapsql.ColumnInfo{
				"id":    {Name: "id", IsPrimaryKey: true},
				"name":  {Name: "name"},
				"email": {Name: "email"},
			},
		},
	}
	executor := NewExecutor(db, "sqlite", tableInfo)

	// Create test case with fixtures
	testCase := &markdownparser.TestCase{
		Name: "Test User Insertion",
		Fixtures: []markdownparser.TableFixture{
			{
				TableName: "users",
				Strategy:  markdownparser.Upsert,
				Data: []map[string]any{
					{"id": 1, "name": "John Doe", "email": "john@example.com"},
					{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
				},
			},
		},
		Parameters: map[string]any{
			"limit": 10,
		},
		ExpectedResult: []map[string]any{
			{"id": 1, "name": "John Doe", "email": "john@example.com"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
		},
	}

	// Test fixture-only mode
	options := &ExecutionOptions{
		Mode:     FixtureOnly,
		Commit:   true,
		Parallel: 1,
		Timeout:  time.Minute,
	}

	result, trace, err := executor.ExecuteTest(testCase, "", map[string]any{}, options)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, trace)

	// Verify data was inserted
	rows, err := db.Query("SELECT id, name, email FROM users ORDER BY id")
	require.NoError(t, err)

	defer rows.Close()

	var users []map[string]any

	for rows.Next() {
		var (
			id          int
			name, email string
		)

		err := rows.Scan(&id, &name, &email)
		require.NoError(t, err)

		users = append(users, map[string]any{
			"id":    id,
			"name":  name,
			"email": email,
		})
	}

	require.NoError(t, rows.Err())

	assert.Equal(t, 2, len(users))
	assert.Equal(t, "John Doe", users[0]["name"])
	assert.Equal(t, "Jane Smith", users[1]["name"])
}

func TestExecutor_ExecuteTest_UsesPreparedSQL(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob')`)
	require.NoError(t, err)

	tableInfo := map[string]*snapsql.TableInfo{
		"users": {
			Name: "users",
			Columns: map[string]*snapsql.ColumnInfo{
				"id":   {Name: "id", IsPrimaryKey: true},
				"name": {Name: "name"},
			},
		},
	}

	executor := NewExecutor(db, "sqlite", tableInfo)

	testCase := &markdownparser.TestCase{
		Name:        "Prepared SQL fetch",
		PreparedSQL: "SELECT id, name FROM users WHERE id = ?",
		SQLArgs:     []any{2},
		ExpectedResult: []map[string]any{
			{"id": 2, "name": "Bob"},
		},
	}

	options := &ExecutionOptions{
		Mode:     FullTest,
		Commit:   false,
		Parallel: 1,
		Timeout:  time.Minute,
		Verbose:  true,
	}

	result, trace, err := executor.ExecuteTest(
		testCase,
		"SELECT id, name FROM users WHERE id = /*= id */1",
		map[string]any{"id": 2},
		options,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, trace, 1)

	require.Len(t, result.Data, 1)
	row := result.Data[0]
	assert.EqualValues(t, 2, row["id"])
	assert.Equal(t, "Bob", row["name"])
	assert.EqualValues(t, 1, result.RowsAffected)

	assert.Equal(t, "SELECT id, name FROM users WHERE id = ?", trace[0].Statement)
	assert.Equal(t, []any{2}, trace[0].Args)
	assert.Equal(t, map[string]any{"id": 2}, trace[0].Parameters)
}

func TestExecutor_FixtureCurrentDateValue(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE logs (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP NOT NULL
		)
	`)
	require.NoError(t, err)

	executor := NewExecutor(db, "sqlite", map[string]*snapsql.TableInfo{
		"logs": {
			Name: "logs",
			Columns: map[string]*snapsql.ColumnInfo{
				"id":         {Name: "id", IsPrimaryKey: true},
				"created_at": {Name: "created_at"},
			},
			ColumnOrder: []string{"id", "created_at"},
		},
	})

	testCase := &markdownparser.TestCase{
		Name: "Fixture currentdate",
		Fixtures: []markdownparser.TableFixture{
			{
				TableName: "logs",
				Strategy:  markdownparser.Upsert,
				Data: []map[string]any{
					{"id": 1, "created_at": []any{"currentdate", "+2h"}},
				},
			},
		},
	}

	options := &ExecutionOptions{
		Mode:     FixtureOnly,
		Commit:   true,
		Parallel: 1,
		Timeout:  time.Minute,
	}

	result, trace, err := executor.ExecuteTest(testCase, "", map[string]any{}, options)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, trace)

	var created time.Time
	err = db.QueryRow("SELECT created_at FROM logs WHERE id = 1").Scan(&created)
	require.NoError(t, err)

	assert.True(t, durationAbs(time.Since(created)) <= 3*time.Hour, "expected timestamp within tolerance: %v", created)
}

func TestExecutor_ClearInsertStrategy(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert initial data
	_, err = db.Exec("INSERT INTO users (id, name, email) VALUES (999, 'Old User', 'old@example.com')")
	require.NoError(t, err)

	// Minimal table info for clear-insert test
	executor := NewExecutor(db, "sqlite", map[string]*snapsql.TableInfo{
		"users": {
			Name: "users",
			Columns: map[string]*snapsql.ColumnInfo{
				"id":    {Name: "id", IsPrimaryKey: true},
				"name":  {Name: "name"},
				"email": {Name: "email"},
			},
		},
	})

	// Create test case with clear-insert strategy
	testCase := &markdownparser.TestCase{
		Name: "Test Clear Insert",
		Fixtures: []markdownparser.TableFixture{
			{
				TableName: "users",
				Strategy:  markdownparser.ClearInsert,
				Data: []map[string]any{
					{"id": 1, "name": "New User", "email": "new@example.com"},
				},
			},
		},
	}

	options := &ExecutionOptions{
		Mode:     FixtureOnly,
		Commit:   true,
		Parallel: 1,
		Timeout:  time.Minute,
	}

	result, trace, err := executor.ExecuteTest(testCase, "", map[string]any{}, options)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, trace)

	// Verify old data was cleared and new data was inserted
	rows, err := db.Query("SELECT id, name, email FROM users ORDER BY id")
	require.NoError(t, err)

	defer rows.Close()

	var users []map[string]any

	for rows.Next() {
		var (
			id          int
			name, email string
		)

		err := rows.Scan(&id, &name, &email)
		require.NoError(t, err)

		users = append(users, map[string]any{
			"id":    id,
			"name":  name,
			"email": email,
		})
	}

	require.NoError(t, rows.Err())

	// Should only have the new user, old user should be cleared
	assert.Equal(t, 1, len(users))
	assert.Equal(t, "New User", users[0]["name"])
}

func TestTestRunner_RunTests(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create test cases
	testCases := []*markdownparser.TestCase{
		{
			Name: "Test 1",
			Fixtures: []markdownparser.TableFixture{
				{
					TableName: "users",
					Strategy:  markdownparser.Upsert,
					Data: []map[string]any{
						{"id": 1, "name": "User 1", "email": "user1@example.com"},
					},
				},
			},
		},
		{
			Name: "Test 2",
			Fixtures: []markdownparser.TableFixture{
				{
					TableName: "users",
					Strategy:  markdownparser.Upsert,
					Data: []map[string]any{
						{"id": 2, "name": "User 2", "email": "user2@example.com"},
					},
				},
			},
		},
	}

	// Create test runner with sequential execution for SQLite
	options := &ExecutionOptions{
		Mode:     FixtureOnly,
		Commit:   false, // Use rollback for isolation
		Parallel: 1,     // Sequential execution for SQLite
		Timeout:  time.Minute,
	}

	runner := NewTestRunner(db, "sqlite", options)
	runner.SetTableInfo(map[string]*snapsql.TableInfo{
		"users": {
			Name: "users",
			Columns: map[string]*snapsql.ColumnInfo{
				"id":    {Name: "id", IsPrimaryKey: true},
				"name":  {Name: "name"},
				"email": {Name: "email"},
			},
		},
	})

	// Run tests
	ctx := context.Background()
	summary, err := runner.RunTests(ctx, testCases)
	require.NoError(t, err)
	require.NotNil(t, summary)

	// Verify summary
	assert.Equal(t, 2, summary.TotalTests)
	assert.Equal(t, 2, summary.PassedTests)
	assert.Equal(t, 0, summary.FailedTests)
	assert.Equal(t, 2, len(summary.Results))

	// Verify all tests succeeded
	for _, result := range summary.Results {
		assert.True(t, result.Success, "Test %s should succeed", result.TestCase.Name)
		assert.NoError(t, result.Error)
	}

	// Verify data was rolled back (table should be empty)
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Data should be rolled back")
}

func TestCompareRowsWithMatchersCurrentDate(t *testing.T) {
	now := time.Now().UTC()
	rowExpected := map[string]any{
		"created_at": []any{"currentdate"},
	}
	rowActual := map[string]any{
		"created_at": now,
	}

	require.NoError(t, compareRowsWithMatchers(rowExpected, rowActual))

	rowExpected = map[string]any{
		"created_at": []any{"currentdate", "1s"},
	}
	rowActual = map[string]any{
		"created_at": now.Add(-2 * time.Minute),
	}

	err := compareRowsWithMatchers(rowExpected, rowActual)
	assert.Error(t, err)
}
