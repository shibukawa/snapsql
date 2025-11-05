package pygen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MockData represents mock data for a function
type MockData struct {
	Rows         []map[string]any `json:"rows,omitempty"`
	RowsAffected int64            `json:"rows_affected,omitempty"`
	LastInsertID int64            `json:"last_insert_id,omitempty"`
	Error        string           `json:"error,omitempty"`
	ErrorType    string           `json:"error_type,omitempty"` // "not_found", "database", "validation"
}

// MockDataFile represents the structure of a mock data JSON file
type MockDataFile map[string]MockData

// LoadMockData loads mock data from a JSON file
func LoadMockData(mockPath string) (MockDataFile, error) {
	if mockPath == "" {
		return nil, errors.New("mock path is empty")
	}

	// Read the file
	data, err := os.ReadFile(mockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock file %s: %w", mockPath, err)
	}

	// Parse JSON
	var mockData MockDataFile
	if err := json.Unmarshal(data, &mockData); err != nil {
		return nil, fmt.Errorf("failed to parse mock file %s: %w", mockPath, err)
	}

	return mockData, nil
}

// GetMockForFunction retrieves mock data for a specific function
func GetMockForFunction(mockData MockDataFile, functionName string) (MockData, bool) {
	data, exists := mockData[functionName]
	return data, exists
}

// GenerateMockLoadCode generates Python code to load mock data from JSON
func GenerateMockLoadCode(mockPath string) string {
	if mockPath == "" {
		return ""
	}

	// Escape the path for Python string
	escapedPath := filepath.ToSlash(mockPath)

	return fmt.Sprintf(`
# ============================================================================
# Mock Data Loading
# ============================================================================

import json
from pathlib import Path

def load_mock_data(mock_path: str) -> Dict[str, Any]:
    """Load mock data from JSON file"""
    try:
        with open(mock_path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except FileNotFoundError:
        print(f"Warning: Mock file not found: {mock_path}")
        return {}
    except json.JSONDecodeError as e:
        print(f"Warning: Failed to parse mock file {mock_path}: {e}")
        return {}

# Mock data path for this module
MOCK_DATA_PATH = "%s"
`, escapedPath)
}

// GenerateMockExecutionCode generates Python code to execute mock data
func GenerateMockExecutionCode(functionName string, responseAffinity string, returnType string, dialect string, queryType string) string {
	code := `
    # Check for mock execution
    ctx = get_snapsql_context()
    if ctx and ctx.mock_mode and ctx.mock_data:
        mock = ctx.mock_data.get("` + functionName + `")
        if mock:
            # Log mock execution
            if ctx.query_logger:
                ctx.query_logger.set_query(sql, args)
                await ctx.query_logger.write(
                    QueryLogMetadata(
                        func_name="` + functionName + `",
                        source_file="` + functionName + `",
                        dialect="` + dialect + `",
                        query_type="` + queryType + `"
                    ),
                    duration_ms=0.0,
                    row_count=len(mock.get('rows', [])) if 'rows' in mock else mock.get('rows_affected', 0),
                    error=None
                )
            
            # Handle mock error
            if mock.get('error'):
                error_type = mock.get('error_type', 'database')
                error_msg = mock['error']
                if error_type == 'not_found':
                    raise NotFoundError(
                        message=error_msg,
                        func_name="` + functionName + `",
                        query=sql
                    )
                elif error_type == 'validation':
                    raise ValidationError(
                        message=error_msg,
                        func_name="` + functionName + `"
                    )
                else:
                    raise DatabaseError(
                        message=error_msg,
                        func_name="` + functionName + `",
                        query=sql
                    )
            
`

	switch responseAffinity {
	case "none":
		code += `            # Return rows affected for none affinity
            return mock.get('rows_affected', 0)
`
	case "one":
		code += `            # Return single row for one affinity
            rows = mock.get('rows', [])
            if not rows:
                raise NotFoundError(
                    message="Record not found in mock data",
                    func_name="` + functionName + `",
                    query=sql
                )
            return ` + returnType + `(**rows[0])
`
	case "many":
		code += `            # Yield rows for many affinity (async generator)
            rows = mock.get('rows', [])
            for row in rows:
                yield ` + returnType + `(**row)
            return
`
	}

	return code
}

// mockTemplateData represents mock-related data for templates
type mockTemplateData struct {
	HasMock      bool
	MockPath     string
	MockLoadCode string
	MockExecCode string
}

// processMockData processes mock configuration and generates template data
func processMockData(mockPath string, functionName string, responseAffinity string, returnType string, dialect string, queryType string) mockTemplateData {
	if mockPath == "" {
		return mockTemplateData{
			HasMock: false,
		}
	}

	return mockTemplateData{
		HasMock:      true,
		MockPath:     mockPath,
		MockLoadCode: GenerateMockLoadCode(mockPath),
		MockExecCode: GenerateMockExecutionCode(functionName, responseAffinity, returnType, dialect, queryType),
	}
}
