package testhelper

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql/testdata"
)

// GetAcceptanceTestDirs returns a list of acceptance test directories
func GetAcceptanceTestDirs() ([]string, error) {
	var dirs []string
	
	// Pattern for test directories: 3 digits followed by a name
	pattern := regexp.MustCompile(`^[0-9]{3}.*$`)
	
	// Get the embedded filesystem
	embedFS := testdata.GetFS()
	
	// List directories in acceptancetests
	entries, err := fs.ReadDir(embedFS, "acceptancetests")
	if err != nil {
		return nil, fmt.Errorf("failed to read acceptancetests directory: %v", err)
	}
	
	// Filter directories that match the pattern
	for _, entry := range entries {
		if entry.IsDir() && pattern.MatchString(entry.Name()) {
			dirs = append(dirs, filepath.Join("acceptancetests", entry.Name()))
		}
	}
	
	return dirs, nil
}

// ReadTestFile reads a file from the embedded test data
func ReadTestFile(filePath string) ([]byte, error) {
	// Get the embedded filesystem
	embedFS := testdata.GetFS()
	
	// Read the file
	return fs.ReadFile(embedFS, filePath)
}

// IsErrorTest checks if a test is an error test
func IsErrorTest(testPath string) bool {
	return strings.HasSuffix(path.Base(testPath), "_err")
}

// WriteTestFile writes a file to the test directory using relative path
// Note: This uses the real filesystem, not the embedded one
func WriteTestFile(filePath string, data []byte) error {
	// Convert embedded path to relative path
	// For example, "acceptancetests/001_simple_case_sql_ok/actual.json" -> "../testdata/acceptancetests/001_simple_case_sql_ok/actual.json"
	relPath := filepath.Join("..", "testdata", filePath)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(relPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Write the file
	return os.WriteFile(relPath, data, 0644)
}
