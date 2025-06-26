package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestPullCmd(t *testing.T) {
	// Create temporary directory for tests
	tempDir := t.TempDir()

	t.Run("CreatePullCmd", func(t *testing.T) {
		cmd := &PullCmd{
			DB:     "sqlite://test.db",
			Type:   "sqlite",
			Output: tempDir,
			Format: "per_table",
		}

		assert.Equal(t, "sqlite://test.db", cmd.DB)
		assert.Equal(t, "sqlite", cmd.Type)
		assert.Equal(t, tempDir, cmd.Output)
		assert.Equal(t, "per_table", cmd.Format)
	})

	t.Run("ResolveDatabaseConnection", func(t *testing.T) {
		cmd := &PullCmd{
			DB:   "sqlite://test.db",
			Type: "sqlite",
		}

		config := &Config{
			Databases: make(map[string]Database),
		}

		dbURL, dbType, err := cmd.resolveDatabaseConnection(config)
		assert.NoError(t, err)
		assert.Equal(t, "sqlite://test.db", dbURL)
		assert.Equal(t, "sqlite", dbType)
	})

	t.Run("ResolveDatabaseConnectionFromConfig", func(t *testing.T) {
		cmd := &PullCmd{
			Env: "test_env",
		}

		config := &Config{
			Databases: map[string]Database{
				"test_env": {
					Driver:     "sqlite",
					Connection: "sqlite://config_test.db",
				},
			},
		}

		dbURL, dbType, err := cmd.resolveDatabaseConnection(config)
		assert.NoError(t, err)
		assert.Equal(t, "sqlite://config_test.db", dbURL)
		assert.Equal(t, "sqlite", dbType)
	})

	t.Run("ResolveDatabaseConnectionError", func(t *testing.T) {
		cmd := &PullCmd{}

		config := &Config{
			Databases: make(map[string]Database),
		}

		_, _, err := cmd.resolveDatabaseConnection(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either --db or --env must be specified")
	})

	t.Run("CreatePullConfig", func(t *testing.T) {
		cmd := &PullCmd{
			Output:         tempDir,
			Format:         "single_file",
			SchemaAware:    true,
			IncludeSchemas: []string{"public"},
			ExcludeSchemas: []string{"temp"},
			IncludeTables:  []string{"users", "posts"},
			ExcludeTables:  []string{"*_temp"},
			IncludeViews:   true,
			IncludeIndexes: true,
		}

		config := cmd.createPullConfig("sqlite://test.db", "sqlite")

		assert.Equal(t, "sqlite://test.db", config.DatabaseURL)
		assert.Equal(t, "sqlite", config.DatabaseType)
		assert.Equal(t, tempDir, config.OutputPath)
		assert.True(t, config.SchemaAware)
		assert.Equal(t, []string{"public"}, config.IncludeSchemas)
		assert.Equal(t, []string{"temp"}, config.ExcludeSchemas)
		assert.Equal(t, []string{"users", "posts"}, config.IncludeTables)
		assert.Equal(t, []string{"*_temp"}, config.ExcludeTables)
		assert.True(t, config.IncludeViews)
		assert.True(t, config.IncludeIndexes)
	})
}

func TestConfigLoading(t *testing.T) {
	t.Run("LoadDefaultConfig", func(t *testing.T) {
		// Test loading default config when file doesn't exist
		config, err := LoadConfig("nonexistent.yaml")
		assert.NoError(t, err)
		assert.NotZero(t, config)
		assert.Equal(t, "postgres", config.Dialect)
	})

	t.Run("LoadConfigFromFile", func(t *testing.T) {
		// Create temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test_config.yaml")

		configContent := `
dialect: mysql
databases:
  test:
    driver: mysql
    connection: mysql://user:pass@localhost/testdb
generation:
  input_dir: ./test_queries
`

		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		config, err := LoadConfig(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "mysql", config.Dialect)
		assert.Equal(t, "./test_queries", config.Generation.InputDir)
		assert.True(t, len(config.Databases) > 0)
		testDB, exists := config.Databases["test"]
		assert.True(t, exists)
		assert.Equal(t, "mysql", testDB.Driver)
	})
}

func TestEnvironmentVariableExpansion(t *testing.T) {
	t.Run("ExpandEnvVars", func(t *testing.T) {
		// Set test environment variables
		t.Setenv("TEST_USER", "testuser")
		t.Setenv("TEST_PASS", "testpass")

		// Test ${VAR} format
		result := expandEnvVars("postgres://${TEST_USER}:${TEST_PASS}@localhost/db")
		expected := "postgres://testuser:testpass@localhost/db"
		assert.Equal(t, expected, result)

		// Test $VAR format
		result = expandEnvVars("postgres://$TEST_USER:$TEST_PASS@localhost/db")
		assert.Equal(t, expected, result)
	})

	t.Run("ExpandConfigEnvVars", func(t *testing.T) {
		// Set test environment variables
		t.Setenv("DB_HOST", "testhost")
		t.Setenv("DB_PORT", "5432")

		config := &Config{
			Databases: map[string]Database{
				"test": {
					Connection: "postgres://user:pass@${DB_HOST}:${DB_PORT}/db",
				},
			},
		}

		expandConfigEnvVars(config)

		expected := "postgres://user:pass@testhost:5432/db"
		assert.Equal(t, expected, config.Databases["test"].Connection)
	})
}

func TestCLIHelpers(t *testing.T) {
	t.Run("EnsureDir", func(t *testing.T) {
		tempDir := t.TempDir()
		testPath := filepath.Join(tempDir, "test", "nested", "dir")

		err := ensureDir(testPath)
		assert.NoError(t, err)

		// Check that directory was created
		info, err := os.Stat(testPath)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("WriteFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testPath := filepath.Join(tempDir, "nested", "test.txt")
		content := "test content"

		err := writeFile(testPath, content)
		assert.NoError(t, err)

		// Check that file was created with correct content
		data, err := os.ReadFile(testPath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("FileExists", func(t *testing.T) {
		tempDir := t.TempDir()
		existingFile := filepath.Join(tempDir, "existing.txt")
		nonExistingFile := filepath.Join(tempDir, "nonexisting.txt")

		// Create existing file
		err := os.WriteFile(existingFile, []byte("test"), 0644)
		assert.NoError(t, err)

		assert.True(t, fileExists(existingFile))
		assert.False(t, fileExists(nonExistingFile))
	})
}

func TestInitCmd(t *testing.T) {
	t.Run("CreateInitCmd", func(t *testing.T) {
		cmd := &InitCmd{}
		assert.NotZero(t, cmd)
	})
}

func TestValidateCmd(t *testing.T) {
	t.Run("CreateValidateCmd", func(t *testing.T) {
		cmd := &ValidateCmd{
			Input: "./test_queries",
		}
		assert.Equal(t, "./test_queries", cmd.Input)
	})
}

func TestGenerateCmd(t *testing.T) {
	t.Run("CreateGenerateCmd", func(t *testing.T) {
		cmd := &GenerateCmd{
			Input:   "./test_queries",
			Lang:    "go",
			Package: "queries",
		}
		assert.Equal(t, "./test_queries", cmd.Input)
		assert.Equal(t, "go", cmd.Lang)
		assert.Equal(t, "queries", cmd.Package)
	})
}
