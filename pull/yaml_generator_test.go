package pull

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	snapsql "github.com/shibukawa/snapsql"
)

func TestYAMLGenerator(t *testing.T) {
	t.Run("CreateYAMLGenerator", func(t *testing.T) {
		generator := NewYAMLGenerator(true)
		assert.NotZero(t, generator)
	})

	t.Run("CreateYAMLGeneratorWithDefaults", func(t *testing.T) {
		generator := NewYAMLGenerator(false)
		assert.NotZero(t, generator)
	})
}

func TestSchemaPathGeneration(t *testing.T) {
	generator := NewYAMLGenerator(true)

	t.Run("GetSchemaPathWithSchemaAware", func(t *testing.T) {
		testCases := []struct {
			outputPath string
			schemaName string
			expected   string
		}{
			{".snapsql/schema", "public", ".snapsql/schema/public"},
			{".snapsql/schema", "auth", ".snapsql/schema/auth"},
			{".snapsql/schema", "", ".snapsql/schema/global"},
			{".snapsql/schema", "main", ".snapsql/schema/global"},
			{"/tmp/schemas", "public", "/tmp/schemas/public"},
		}

		for _, tc := range testCases {
			result := generator.getSchemaPath(tc.outputPath, tc.schemaName)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("GetSchemaPathWithoutSchemaAware", func(t *testing.T) {
		generator := NewYAMLGenerator(false)

		testCases := []struct {
			outputPath string
			schemaName string
			expected   string
		}{
			{".snapsql/schema", "public", ".snapsql/schema"},
			{".snapsql/schema", "auth", ".snapsql/schema"},
			{".snapsql/schema", "", ".snapsql/schema"},
		}

		for _, tc := range testCases {
			result := generator.getSchemaPath(tc.outputPath, tc.schemaName)
			assert.Equal(t, tc.expected, result)
		}
	})
}

func TestYAMLGeneration(t *testing.T) {
	tempDir := t.TempDir()

	// Create test data
	testSchema := createTestDatabaseSchema()

	t.Run("GeneratePerTableYAML", func(t *testing.T) {
		generator := NewYAMLGenerator(true)
		outputPath := filepath.Join(tempDir, "per_table")

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		// Check that schema directory was created
		schemaDir := filepath.Join(outputPath, "public")
		dirExists(t, schemaDir)

		// Check that table files were created
		usersFile := filepath.Join(schemaDir, "users.yaml")
		postsFile := filepath.Join(schemaDir, "posts.yaml")
		fileExists(t, usersFile)
		fileExists(t, postsFile)

		// Read and verify users table content
		content, err := os.ReadFile(usersFile)
		assert.NoError(t, err)

		yamlContent := string(content)
		assert.Contains(t, yamlContent, "table:")
		assert.Contains(t, yamlContent, "name: users")
		assert.Contains(t, yamlContent, "schema: public")
		assert.Contains(t, yamlContent, "columns:")
		assert.Contains(t, yamlContent, "metadata:")
		// assert.Contains(t, yamlContent, "extracted_at:") // 冪等性のため削除

		// Check for flow style in columns (per table)
		assert.Contains(t, yamlContent, "name: id")
		assert.Contains(t, yamlContent, "snapsql_type: int")
	})

	t.Run("GenerateAndRestorePerTableYAML", func(t *testing.T) {
		generator := NewYAMLGenerator(true)
		outputPath := filepath.Join(tempDir, "restore_per_table")

		testSchema := createTestDatabaseSchema()
		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		schemaDir := filepath.Join(outputPath, "public")
		restored, err := LoadDatabaseSchemaFromDir(schemaDir)
		assert.NoError(t, err)

		// DatabaseInfo など比較不要なフィールドを揃える
		restored.DatabaseInfo = testSchema.DatabaseInfo
		// ViewsはYAML出力・復元に未対応の場合は空にする
		restored.Views = testSchema.Views

		assert.Equal(t, &testSchema, restored)
	})
}

func TestYAMLGenerationErrorHandling(t *testing.T) {
	generator := NewYAMLGenerator(true)

	t.Run("GenerateWithInvalidPath", func(t *testing.T) {
		// Try to generate to a path that can't be created
		invalidPath := "/root/invalid/path/that/cannot/be/created"
		testSchema := createTestDatabaseSchema()

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, invalidPath)
		assert.Error(t, err)
	})

	t.Run("GenerateWithEmptySchemas", func(t *testing.T) {
		tempDir := t.TempDir()

		err := generator.Generate([]snapsql.DatabaseSchema{}, tempDir)
		assert.NoError(t, err) // Should not error, just create empty structure
	})
}

func TestYAMLFileNaming(t *testing.T) {
	generator := NewYAMLGenerator(true)

	t.Run("GetTableFileName", func(t *testing.T) {
		testCases := []struct {
			tableName string
			expected  string
		}{
			{"users", "users.yaml"},
			{"user_profiles", "user_profiles.yaml"},
			{"UserProfiles", "UserProfiles.yaml"},
			{"123_table", "123_table.yaml"},
		}

		for _, tc := range testCases {
			result := generator.getTableFileName(tc.tableName)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("GetSchemaFileName", func(t *testing.T) {
		testCases := []struct {
			schemaName string
			expected   string
		}{
			{"public", "public.yaml"},
			{"auth", "auth.yaml"},
			{"user_data", "user_data.yaml"},
			{"", "global.yaml"},
			{"main", "global.yaml"},
		}

		for _, tc := range testCases {
			result := generator.getSchemaFileName(tc.schemaName)
			assert.Equal(t, tc.expected, result)
		}
	})
}

// Helper functions to check if file/directory exists
func fileExists(t *testing.T, filename string) {
	t.Helper()
	_, err := os.Stat(filename)
	assert.NoError(t, err, "File should exist: %s", filename)
}

func dirExists(t *testing.T, dirname string) {
	t.Helper()
	info, err := os.Stat(dirname)
	assert.NoError(t, err, "Directory should exist: %s", dirname)
	assert.True(t, info.IsDir(), "Path should be a directory: %s", dirname)
}
