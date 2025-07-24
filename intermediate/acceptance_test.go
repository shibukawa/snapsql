package intermediate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestAcceptance(t *testing.T) {
	// Get test data directory
	testDataDir := "../testdata/acceptancetests"

	// Get all test directories
	entries, err := os.ReadDir(testDataDir)
	if err != nil {
		t.Fatalf("Failed to read test data directory: %v", err)
	}

	// Run each test
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testName := entry.Name()
		testDir := filepath.Join(testDataDir, testName)

		t.Run(testName, func(t *testing.T) {
			// Read the input SQL file
			sqlPath := filepath.Join(testDir, "input.snap.sql")
			sqlContent, err := os.ReadFile(sqlPath)
			if err != nil {
				t.Fatalf("Failed to read input SQL file: %v", err)
			}

			// Generate intermediate format using the new function
			reader := strings.NewReader(string(sqlContent))
			format, err := GenerateFromSQL(reader, nil, sqlPath, "")

			// Check if this is an error test
			isErrorTest := strings.HasSuffix(testName, "_err")

			if isErrorTest {
				// For error tests, we expect an error
				assert.Error(t, err)
				return
			}

			// For success tests, we expect no error
			if err != nil {
				t.Fatalf("Did not expect an error but got: %v", err)
			}

			// Convert to JSON
			actualJSON, err := format.ToJSON()
			if err != nil {
				t.Fatalf("Failed to convert to JSON: %v", err)
			}

			// Write actual JSON for debugging
			actualPath := filepath.Join(testDir, "actual.json")
			err = os.WriteFile(actualPath, actualJSON, 0644)
			if err != nil {
				t.Fatalf("Failed to write actual JSON: %v", err)
			}

			// Read expected JSON
			expectedPath := filepath.Join(testDir, "expected.json")
			expectedJSON, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected JSON file: %v", err)
			}

			// Parse both JSON for comparison
			var actual, expected interface{}
			err = json.Unmarshal(actualJSON, &actual)
			if err != nil {
				t.Fatalf("Failed to parse actual JSON: %v", err)
			}

			err = json.Unmarshal(expectedJSON, &expected)
			if err != nil {
				t.Fatalf("Failed to parse expected JSON: %v", err)
			}

			// Compare the results
			assert.Equal(t, expected, actual)
		})
	}
}
