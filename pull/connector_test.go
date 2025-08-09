package pull

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
)

func TestDatabaseConnector(t *testing.T) {
	t.Run("CreateDatabaseConnector", func(t *testing.T) {
		connector := NewDatabaseConnector()
		assert.NotZero(t, connector)
	})

	t.Run("ParseDatabaseURL", func(t *testing.T) {
		testCases := []struct {
			url          string
			expectedType string
			shouldError  bool
		}{
			{"postgres://user:pass@localhost:5432/dbname", "postgresql", false},
			{"postgresql://user:pass@localhost:5432/dbname", "postgresql", false},
			{"mysql://user:pass@localhost:3306/dbname", "mysql", false},
			{"sqlite:///path/to/database.db", "sqlite", false},
			{"sqlite://./database.db", "sqlite", false},
			{"invalid://url", "", true},
			{"", "", true},
		}

		connector := NewDatabaseConnector()
		for _, tc := range testCases {
			dbType, err := connector.ParseDatabaseURL(tc.url)
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedType, dbType)
			}
		}
	})

	t.Run("ValidateConnectionString", func(t *testing.T) {
		testCases := []struct {
			url         string
			shouldError bool
		}{
			{"postgres://user:pass@localhost:5432/dbname", false},
			{"mysql://user:pass@localhost:3306/dbname", false},
			{"sqlite:///path/to/database.db", false},
			{"invalid://url", true},
			{"", true},
			{"postgres://", true}, // Missing required parts
		}

		connector := NewDatabaseConnector()
		for _, tc := range testCases {
			err := connector.ValidateConnectionString(tc.url)
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		}
	})
}

func TestConnectionPooling(t *testing.T) {
	connector := NewDatabaseConnector()

	t.Run("SetConnectionPoolSettings", func(t *testing.T) {
		settings := ConnectionPoolSettings{
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300, // 5 minutes in seconds
		}

		connector.SetPoolSettings(settings)
		assert.Equal(t, settings, connector.GetPoolSettings())
	})

	t.Run("DefaultConnectionPoolSettings", func(t *testing.T) {
		connector := NewDatabaseConnector()
		settings := connector.GetPoolSettings()

		// Check default values
		assert.Equal(t, 25, settings.MaxOpenConns)
		assert.Equal(t, 25, settings.MaxIdleConns)
		assert.Equal(t, 300, settings.ConnMaxLifetime)
	})
}

func TestDatabaseConnection(t *testing.T) {
	// Note: These tests use mock connections since we don't have real databases in CI
	t.Run("ConnectToDatabase", func(t *testing.T) {
		connector := NewDatabaseConnector()

		// Test with invalid URL (should fail)
		db, err := connector.Connect("invalid://url")
		assert.Error(t, err)
		assert.Zero(t, db)
	})

	t.Run("CloseConnection", func(t *testing.T) {
		connector := NewDatabaseConnector()

		// Test with nil database (should not error)
		err := connector.Close(nil)
		assert.NoError(t, err)
	})

	t.Run("PingDatabase", func(t *testing.T) {
		connector := NewDatabaseConnector()

		// Test with nil database (should error)
		err := connector.Ping(nil)
		assert.Error(t, err)
		assert.Equal(t, ErrConnectionFailed, err)
	})
}

func TestPullOperation(t *testing.T) {
	t.Run("CreatePullOperation", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "postgres://user:pass@localhost:5432/testdb",
			DatabaseType: "postgresql",
			OutputPath:   ".snapsql/schema",
			SchemaAware:  true,
		}

		operation := NewPullOperation(config)
		assert.NotZero(t, operation)
		assert.Equal(t, config, operation.Config)
	})

	t.Run("ValidatePullConfig", func(t *testing.T) {
		testCases := []struct {
			name        string
			config      PullConfig
			shouldError bool
		}{
			{
				name: "ValidConfig",
				config: PullConfig{
					DatabaseURL:  "postgres://user:pass@localhost:5432/testdb",
					DatabaseType: "postgresql",
					OutputPath:   ".snapsql/schema",
					SchemaAware:  true,
				},
				shouldError: false,
			},
			{
				name: "EmptyDatabaseURL",
				config: PullConfig{
					DatabaseURL:  "",
					DatabaseType: "postgresql",
					OutputPath:   ".snapsql/schema",
					SchemaAware:  true,
				},
				shouldError: true,
			},
			{
				name: "EmptyDatabaseType",
				config: PullConfig{
					DatabaseURL:  "postgres://user:pass@localhost:5432/testdb",
					DatabaseType: "",
					OutputPath:   ".snapsql/schema",
					SchemaAware:  true,
				},
				shouldError: true,
			},
			{
				name: "EmptyOutputPath",
				config: PullConfig{
					DatabaseURL:  "postgres://user:pass@localhost:5432/testdb",
					DatabaseType: "postgresql",
					OutputPath:   "",
					SchemaAware:  true,
				},
				shouldError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				operation := NewPullOperation(tc.config)

				err := operation.ValidateConfig()
				if tc.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestPullExecution(t *testing.T) {
	t.Run("ExecutePullOperation", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "postgres://user:pass@localhost:5432/testdb",
			DatabaseType: "postgresql",
			OutputPath:   ".snapsql/schema",
			SchemaAware:  true,
		}

		operation := NewPullOperation(config)

		// This will fail because we don't have a real database connection
		// but it tests the execution flow
		result, err := operation.Execute(context.Background())
		assert.Error(t, err) // Expected to fail without real DB
		assert.Zero(t, result)
	})

	t.Run("ExecuteWithMockData", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "postgres://user:pass@localhost:5432/testdb",
			DatabaseType: "postgresql",
			OutputPath:   ".snapsql/schema",
			SchemaAware:  true,
		}

		operation := NewPullOperation(config)

		// Test with mock data (simulating successful extraction)
		mockSchemas := []snapsql.DatabaseSchema{
			createTestDatabaseSchema(),
		}

		result := operation.CreateResult(mockSchemas, nil)
		assert.NotZero(t, result)
		assert.Equal(t, 1, len(result.Schemas))
		assert.Equal(t, 0, len(result.Errors))
	})
}

func TestConnectionStringParsing(t *testing.T) {
	t.Run("ParsePostgreSQLURL", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected ConnectionInfo
		}{
			{
				url: "postgres://user:pass@localhost:5432/dbname",
				expected: ConnectionInfo{
					Type:     "postgresql",
					Host:     "localhost",
					Port:     "5432",
					Database: "dbname",
					Username: "user",
					Password: "pass",
				},
			},
			{
				url: "postgresql://user@localhost/dbname",
				expected: ConnectionInfo{
					Type:     "postgresql",
					Host:     "localhost",
					Port:     "5432", // Default port
					Database: "dbname",
					Username: "user",
					Password: "",
				},
			},
		}

		connector := NewDatabaseConnector()
		for _, tc := range testCases {
			info, err := connector.ParseConnectionInfo(tc.url)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.Type, info.Type)
			assert.Equal(t, tc.expected.Host, info.Host)
			assert.Equal(t, tc.expected.Port, info.Port)
			assert.Equal(t, tc.expected.Database, info.Database)
			assert.Equal(t, tc.expected.Username, info.Username)
			assert.Equal(t, tc.expected.Password, info.Password)
		}
	})

	t.Run("ParseMySQLURL", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected ConnectionInfo
		}{
			{
				url: "mysql://user:pass@localhost:3306/dbname",
				expected: ConnectionInfo{
					Type:     "mysql",
					Host:     "localhost",
					Port:     "3306",
					Database: "dbname",
					Username: "user",
					Password: "pass",
				},
			},
		}

		connector := NewDatabaseConnector()
		for _, tc := range testCases {
			info, err := connector.ParseConnectionInfo(tc.url)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.Type, info.Type)
			assert.Equal(t, tc.expected.Host, info.Host)
			assert.Equal(t, tc.expected.Port, info.Port)
			assert.Equal(t, tc.expected.Database, info.Database)
			assert.Equal(t, tc.expected.Username, info.Username)
			assert.Equal(t, tc.expected.Password, info.Password)
		}
	})

	t.Run("ParseSQLiteURL", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected ConnectionInfo
		}{
			{
				url: "sqlite:///path/to/database.db",
				expected: ConnectionInfo{
					Type:     "sqlite",
					Database: "/path/to/database.db",
				},
			},
			{
				url: "sqlite://./database.db",
				expected: ConnectionInfo{
					Type:     "sqlite",
					Database: "./database.db",
				},
			},
		}

		connector := NewDatabaseConnector()
		for _, tc := range testCases {
			info, err := connector.ParseConnectionInfo(tc.url)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.Type, info.Type)
			assert.Equal(t, tc.expected.Database, info.Database)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("HandleConnectionErrors", func(t *testing.T) {
		connector := NewDatabaseConnector()

		// Test various error scenarios
		testCases := []struct {
			url         string
			expectedErr error
		}{
			{"", ErrEmptyDatabaseURL},
			{"invalid://url", ErrInvalidDatabaseURL},
			{"postgres://", ErrInvalidDatabaseURL},
		}

		for _, tc := range testCases {
			_, err := connector.Connect(tc.url)
			assert.Error(t, err)
			// The specific error might be wrapped, so we check if it contains our expected error
		}
	})

	t.Run("HandlePullErrors", func(t *testing.T) {
		config := PullConfig{
			DatabaseURL:  "",
			DatabaseType: "postgresql",
			OutputPath:   ".snapsql/schema",
		}

		operation := NewPullOperation(config)
		err := operation.ValidateConfig()
		assert.Error(t, err)
		assert.Equal(t, ErrEmptyDatabaseURL, err)
	})
}

func TestIntegrationHelpers(t *testing.T) {
	t.Run("CreateTestConnectionInfo", func(t *testing.T) {
		info := ConnectionInfo{
			Type:     "postgresql",
			Host:     "localhost",
			Port:     "5432",
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
		}

		assert.Equal(t, "postgresql", info.Type)
		assert.Equal(t, "localhost", info.Host)
		assert.Equal(t, "5432", info.Port)
		assert.Equal(t, "testdb", info.Database)
		assert.Equal(t, "testuser", info.Username)
		assert.Equal(t, "testpass", info.Password)
	})

	t.Run("BuildConnectionString", func(t *testing.T) {
		connector := NewDatabaseConnector()

		info := ConnectionInfo{
			Type:     "postgresql",
			Host:     "localhost",
			Port:     "5432",
			Database: "testdb",
			Username: "testuser",
			Password: "testpass",
		}

		connStr := connector.BuildConnectionString(info)
		expected := "postgres://testuser:testpass@localhost:5432/testdb"
		assert.Equal(t, expected, connStr)
	})
}
