package gogen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	snapsql "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestAcceptanceGeneration(t *testing.T) {
	// Get all test cases from testdata/acceptancetests
	testDataDir := "../../testdata/acceptancetests"
	entries, err := os.ReadDir(testDataDir)
	if err != nil {
		t.Fatalf("Failed to read test data directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testName := entry.Name()
		testDir := filepath.Join(testDataDir, testName)

		t.Run(testName, func(t *testing.T) {
			// Check if this is an error test case
			isErrorTest := strings.Contains(testName, "_err")

			if isErrorTest {
				// For error test cases, we expect the intermediate generation to fail
				inputPath := filepath.Join(testDir, "input.snap.sql")
				inputSQL, err := os.ReadFile(inputPath)
				if err != nil {
					t.Fatalf("Failed to read input SQL file: %v", err)
				}

				// Try to load config if it exists
				var config *snapsql.Config
				configPath := filepath.Join(testDir, "snapsql.yaml")
				if _, err := os.Stat(configPath); err == nil {
					config, err = snapsql.LoadConfig(configPath)
					if err != nil {
						t.Fatalf("Failed to load config: %v", err)
					}
				}

				// Try to generate intermediate format - this should fail
				reader := strings.NewReader(string(inputSQL))
				_, err = intermediate.GenerateFromSQL(reader, nil, testDir, testDir, nil, config)
				if err == nil {
					t.Errorf("Expected error for test %s, but generation succeeded", testName)
				} else {
					t.Logf("Expected error occurred for %s: %v", testName, err)
				}
				return
			}

			// Read expected JSON for success test cases
			expectedPath := filepath.Join(testDir, "expected.json")
			expectedJSON, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Skipf("Skipping test %s: no expected.json found", testName)
				return
			}

			// Parse expected JSON
			format, err := intermediate.FromJSON(expectedJSON)
			if err != nil {
				t.Fatalf("Failed to parse expected JSON: %v", err)
			}

			// Generate Go code
			var goCode strings.Builder
			goGen := New(format, WithPackageName("generated"), WithDialect("postgresql"))
			err = goGen.Generate(&goCode)
			if err != nil {
				t.Logf("Failed to generate Go code for %s: %v", testName, err)
			} else {
				// Create a directory for generated code
				genDir := filepath.Join(testDir, "generated")
				err = os.MkdirAll(genDir, 0755)
				if err != nil {
					t.Logf("Failed to create directory for generated code: %v", err)
				} else {
					// Write generated code to file
					genPath := filepath.Join(genDir, "generated.go")
					err = os.WriteFile(genPath, []byte(goCode.String()), 0644)
					if err != nil {
						t.Logf("Failed to write generated code: %v", err)
					}
				}
				t.Logf("Generated Go code for %s:\n%s", testName, goCode.String())
			}
		})
	}
}
