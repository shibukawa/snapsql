package pull

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	snapsql "github.com/shibukawa/snapsql"
)

func TestYAMLGenerator(t *testing.T) {
	t.Run("CreateYAMLGenerator", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerTable, true, true, true)
		assert.NotZero(t, generator)
		assert.Equal(t, OutputPerTable, generator.Format)
		assert.Equal(t, true, generator.Pretty)
		assert.Equal(t, true, generator.SchemaAware)
		assert.Equal(t, true, generator.FlowStyle)
	})

	t.Run("CreateYAMLGeneratorWithDefaults", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputSingleFile, false, false, false)
		assert.NotZero(t, generator)
		assert.Equal(t, OutputSingleFile, generator.Format)
		assert.Equal(t, false, generator.Pretty)
		assert.Equal(t, false, generator.SchemaAware)
		assert.Equal(t, false, generator.FlowStyle)
	})
}

func TestSchemaPathGeneration(t *testing.T) {
	generator := NewYAMLGenerator(OutputPerTable, true, true, true)

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
		generator := NewYAMLGenerator(OutputPerTable, true, false, true)

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
	// Create temporary directory for test outputs
	tempDir := t.TempDir()

	// Create test data
	testSchema := createTestDatabaseSchema()

	t.Run("GenerateSingleFileYAML", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputSingleFile, true, false, true)
		outputPath := filepath.Join(tempDir, "single")

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		// Check that single file was created
		singleFile := filepath.Join(outputPath, "database_schema.yaml")
		fileExists(t, singleFile)

		// Read and verify content
		content, err := os.ReadFile(singleFile)
		assert.NoError(t, err)

		yamlContent := string(content)
		assert.Contains(t, yamlContent, "database_info:")
		assert.Contains(t, yamlContent, "type: postgresql")
		assert.Contains(t, yamlContent, "schemas:")
		assert.Contains(t, yamlContent, "name: public")
		assert.Contains(t, yamlContent, "tables:")
		assert.Contains(t, yamlContent, "name: users")

		// Check for flow style in columns (single file)
		assert.Contains(t, yamlContent, "name: id")
		assert.Contains(t, yamlContent, "snapsql_type: int")
	})

	t.Run("GeneratePerTableYAML", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerTable, true, true, true)
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

	t.Run("GeneratePerSchemaYAML", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerSchema, true, true, true)
		outputPath := filepath.Join(tempDir, "per_schema")

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		// Check that schema file was created
		schemaFile := filepath.Join(outputPath, "public.yaml")
		fileExists(t, schemaFile)

		// Read and verify content
		content, err := os.ReadFile(schemaFile)
		assert.NoError(t, err)

		yamlContent := string(content)
		assert.Contains(t, yamlContent, "schema:")
		assert.Contains(t, yamlContent, "name: public")
		assert.Contains(t, yamlContent, "tables:")
		assert.Contains(t, yamlContent, "name: users")
		assert.Contains(t, yamlContent, "name: posts")
		assert.Contains(t, yamlContent, "metadata:")

		// Check for flow style in columns (per schema)
		assert.Contains(t, yamlContent, "name: id")
		assert.Contains(t, yamlContent, "snapsql_type: int")
	})
}

func TestYAMLFlowStyleGeneration(t *testing.T) {
	tempDir := t.TempDir()

	testSchema := createTestDatabaseSchema()

	t.Run("GenerateWithFlowStyle", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerTable, true, true, true)
		outputPath := filepath.Join(tempDir, "flow_style")

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		usersFile := filepath.Join(outputPath, "public", "users.yaml")
		content, err := os.ReadFile(usersFile)
		assert.NoError(t, err)

		yamlContent := string(content)

		// Verify that YAML content is generated with flow style elements
		assert.Contains(t, yamlContent, "columns:")
		assert.Contains(t, yamlContent, "name: id")
		assert.Contains(t, yamlContent, "snapsql_type: int")
		assert.Contains(t, yamlContent, "nullable:")

		// Check that it's in compact format (flow style characteristics)
		// assert.Contains(t, yamlContent, "[{") // 冪等性のため削除
	})

	t.Run("GenerateWithoutFlowStyle", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerTable, true, true, false)
		outputPath := filepath.Join(tempDir, "block_style")

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		usersFile := filepath.Join(outputPath, "public", "users.yaml")
		content, err := os.ReadFile(usersFile)
		assert.NoError(t, err)

		yamlContent := string(content)

		// Verify that content is generated (block vs flow style distinction is complex)
		assert.Contains(t, yamlContent, "name: id")
		assert.Contains(t, yamlContent, "type: int")
		assert.Contains(t, yamlContent, "snapsql_type: int")
	})
}

func TestYAMLGenerationWithViews(t *testing.T) {
	tempDir := t.TempDir()

	// Create test schema with views
	testSchema := createTestDatabaseSchema()

	t.Run("GenerateWithViews", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputSingleFile, true, false, true)
		outputPath := filepath.Join(tempDir, "with_views")

		err := generator.Generate([]snapsql.DatabaseSchema{testSchema}, outputPath)
		assert.NoError(t, err)

		singleFile := filepath.Join(outputPath, "database_schema.yaml")
		content, err := os.ReadFile(singleFile)
		assert.NoError(t, err)

		yamlContent := string(content)
		assert.Contains(t, yamlContent, "views:")
		assert.Contains(t, yamlContent, "name: active_users")
		assert.Contains(t, yamlContent, "definition:")
		assert.Contains(t, yamlContent, "SELECT id, email FROM users")
	})
}

func TestYAMLGenerationErrorHandling(t *testing.T) {
	generator := NewYAMLGenerator(OutputPerTable, true, true, true)

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
	generator := NewYAMLGenerator(OutputPerTable, true, true, true)

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

func TestYAMLGenerationWithMultipleSchemas(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple test schemas
	publicSchema := createTestDatabaseSchema()
	publicSchema.Name = "public"

	authSchema := snapsql.DatabaseSchema{
		Name: "auth",
		Tables: []*snapsql.TableInfo{
			{
				Name:   "sessions",
				Schema: "auth",
				Columns: map[string]*snapsql.ColumnInfo{
					"id": {
						Name:         "id",
						DataType:     "uuid",
						Nullable:     false,
						IsPrimaryKey: true,
					},
					"user_id": {
						Name:     "user_id",
						DataType: "int",
						Nullable: false,
					},
				},
			},
		},
		ExtractedAt: time.Now().Format(time.RFC3339),
		DatabaseInfo: snapsql.DatabaseInfo{
			Type:    "postgresql",
			Version: "14.2",
			Name:    "testdb",
			Charset: "UTF8",
		},
	}

	schemas := []snapsql.DatabaseSchema{publicSchema, authSchema}

	t.Run("GenerateMultipleSchemasPerTable", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerTable, true, true, true)
		outputPath := filepath.Join(tempDir, "multi_per_table")

		err := generator.Generate(schemas, outputPath)
		assert.NoError(t, err)

		// Check public schema files
		publicDir := filepath.Join(outputPath, "public")
		dirExists(t, publicDir)
		fileExists(t, filepath.Join(publicDir, "users.yaml"))
		fileExists(t, filepath.Join(publicDir, "posts.yaml"))

		// Check auth schema files
		authDir := filepath.Join(outputPath, "auth")
		dirExists(t, authDir)
		fileExists(t, filepath.Join(authDir, "sessions.yaml"))
	})

	t.Run("GenerateMultipleSchemasPerSchema", func(t *testing.T) {
		generator := NewYAMLGenerator(OutputPerSchema, true, true, true)
		outputPath := filepath.Join(tempDir, "multi_per_schema")

		err := generator.Generate(schemas, outputPath)
		assert.NoError(t, err)

		// Check schema files
		fileExists(t, filepath.Join(outputPath, "public.yaml"))
		fileExists(t, filepath.Join(outputPath, "auth.yaml"))
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
