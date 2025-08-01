package gogen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			// Read expected JSON
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
			goGen := New(format, WithPackageName("generated"))
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
