package pygen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMockData(t *testing.T) {
	// Create a temporary mock file
	tmpDir := t.TempDir()
	mockFile := filepath.Join(tmpDir, "mocks.json")

	mockContent := `{
		"get_user_by_id": {
			"rows": [
				{"user_id": 1, "username": "testuser", "email": "test@example.com"}
			]
		},
		"list_users": {
			"rows": [
				{"user_id": 1, "username": "user1"},
				{"user_id": 2, "username": "user2"}
			]
		},
		"insert_user": {
			"rows_affected": 1,
			"last_insert_id": 123
		},
		"delete_user": {
			"error": "User not found",
			"error_type": "not_found"
		}
	}`

	err := os.WriteFile(mockFile, []byte(mockContent), 0644)
	require.NoError(t, err)

	// Test loading mock data
	mockData, err := LoadMockData(mockFile)
	require.NoError(t, err)
	assert.NotNil(t, mockData)

	// Test get_user_by_id mock
	getUserMock, exists := GetMockForFunction(mockData, "get_user_by_id")
	assert.True(t, exists)
	assert.Len(t, getUserMock.Rows, 1)
	assert.Equal(t, float64(1), getUserMock.Rows[0]["user_id"]) // JSON numbers are float64
	assert.Equal(t, "testuser", getUserMock.Rows[0]["username"])

	// Test list_users mock
	listUsersMock, exists := GetMockForFunction(mockData, "list_users")
	assert.True(t, exists)
	assert.Len(t, listUsersMock.Rows, 2)

	// Test insert_user mock
	insertUserMock, exists := GetMockForFunction(mockData, "insert_user")
	assert.True(t, exists)
	assert.Equal(t, int64(1), insertUserMock.RowsAffected)
	assert.Equal(t, int64(123), insertUserMock.LastInsertID)

	// Test delete_user mock with error
	deleteUserMock, exists := GetMockForFunction(mockData, "delete_user")
	assert.True(t, exists)
	assert.Equal(t, "User not found", deleteUserMock.Error)
	assert.Equal(t, "not_found", deleteUserMock.ErrorType)

	// Test non-existent function
	_, exists = GetMockForFunction(mockData, "non_existent")
	assert.False(t, exists)
}

func TestLoadMockData_EmptyPath(t *testing.T) {
	_, err := LoadMockData("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock path is empty")
}

func TestLoadMockData_FileNotFound(t *testing.T) {
	_, err := LoadMockData("/nonexistent/path/mocks.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read mock file")
}

func TestLoadMockData_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mockFile := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(mockFile, []byte("invalid json {"), 0644)
	require.NoError(t, err)

	_, err = LoadMockData(mockFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse mock file")
}

func TestGenerateMockLoadCode(t *testing.T) {
	tests := []struct {
		name     string
		mockPath string
		want     string
	}{
		{
			name:     "with mock path",
			mockPath: "./testdata/mocks.json",
			want:     `MOCK_DATA_PATH = "./testdata/mocks.json"`,
		},
		{
			name:     "empty mock path",
			mockPath: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GenerateMockLoadCode(tt.mockPath)
			if tt.want == "" {
				assert.Empty(t, code)
			} else {
				assert.Contains(t, code, tt.want)
				assert.Contains(t, code, "def load_mock_data")
				assert.Contains(t, code, "json.load")
			}
		})
	}
}

func TestGenerateMockExecutionCode(t *testing.T) {
	tests := []struct {
		name             string
		functionName     string
		responseAffinity string
		returnType       string
		dialect          string
		queryType        string
		wantContains     []string
	}{
		{
			name:             "none affinity",
			functionName:     "delete_user",
			responseAffinity: "none",
			returnType:       "int",
			dialect:          "postgres",
			queryType:        "delete",
			wantContains: []string{
				"ctx.mock_mode",
				"ctx.mock_data.get(\"delete_user\")",
				"return mock.get('rows_affected', 0)",
				"ctx.query_logger",
				"dialect=\"postgres\"",
				"query_type=\"delete\"",
			},
		},
		{
			name:             "one affinity",
			functionName:     "get_user_by_id",
			responseAffinity: "one",
			returnType:       "GetUserByIdResult",
			dialect:          "mysql",
			queryType:        "select",
			wantContains: []string{
				"ctx.mock_mode",
				"ctx.mock_data.get(\"get_user_by_id\")",
				"return GetUserByIdResult(**rows[0])",
				"raise NotFoundError",
				"dialect=\"mysql\"",
				"query_type=\"select\"",
			},
		},
		{
			name:             "many affinity",
			functionName:     "list_users",
			responseAffinity: "many",
			returnType:       "ListUsersResult",
			dialect:          "sqlite",
			queryType:        "select",
			wantContains: []string{
				"ctx.mock_mode",
				"ctx.mock_data.get(\"list_users\")",
				"yield ListUsersResult(**row)",
				"for row in rows",
				"dialect=\"sqlite\"",
				"query_type=\"select\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GenerateMockExecutionCode(tt.functionName, tt.responseAffinity, tt.returnType, tt.dialect, tt.queryType)
			assert.NotEmpty(t, code)

			for _, want := range tt.wantContains {
				assert.Contains(t, code, want, "code should contain: %s", want)
			}

			// All should handle errors
			assert.Contains(t, code, "mock.get('error')")
			assert.Contains(t, code, "error_type")
			assert.Contains(t, code, "NotFoundError")
			assert.Contains(t, code, "ValidationError")
			assert.Contains(t, code, "DatabaseError")
		})
	}
}

func TestProcessMockData(t *testing.T) {
	tests := []struct {
		name             string
		mockPath         string
		functionName     string
		responseAffinity string
		returnType       string
		dialect          string
		queryType        string
		wantHasMock      bool
	}{
		{
			name:             "with mock path",
			mockPath:         "./testdata/mocks.json",
			functionName:     "get_user",
			responseAffinity: "one",
			returnType:       "GetUserResult",
			dialect:          "postgres",
			queryType:        "select",
			wantHasMock:      true,
		},
		{
			name:             "without mock path",
			mockPath:         "",
			functionName:     "get_user",
			responseAffinity: "one",
			returnType:       "GetUserResult",
			dialect:          "postgres",
			queryType:        "select",
			wantHasMock:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := processMockData(tt.mockPath, tt.functionName, tt.responseAffinity, tt.returnType, tt.dialect, tt.queryType)
			assert.Equal(t, tt.wantHasMock, data.HasMock)

			if tt.wantHasMock {
				assert.NotEmpty(t, data.MockPath)
				assert.NotEmpty(t, data.MockLoadCode)
				assert.NotEmpty(t, data.MockExecCode)
			} else {
				assert.Empty(t, data.MockPath)
				assert.Empty(t, data.MockLoadCode)
				assert.Empty(t, data.MockExecCode)
			}
		})
	}
}
