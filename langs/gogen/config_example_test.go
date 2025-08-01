// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gogen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestConfigurationExamples(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "find_user",
		ResponseAffinity: "one",
		Parameters: []intermediate.Parameter{
			{Name: "id", Type: "int"},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT * FROM users WHERE id = "},
			{Op: "EMIT_EVAL", ExprIndex: &[]int{0}[0]},
		},
		CELEnvironments: []intermediate.CELEnvironment{
			{Index: 0},
		},
		CELExpressions: []intermediate.CELExpression{
			{ID: "expr_001", Expression: "id", EnvironmentIndex: 0},
		},
	}

	tests := []struct {
		name        string
		config      Config
		outputPath  string
		expectedPkg string
		description string
	}{
		{
			name: "auto-inferred from simple path",
			config: Config{
				Package: "", // Auto-infer
			},
			outputPath:  "./internal/queries",
			expectedPkg: "queries",
			description: "Package name inferred from directory name",
		},
		{
			name: "auto-inferred from hyphenated path",
			config: Config{
				Package: "",
			},
			outputPath:  "./pkg/go-sqlgen",
			expectedPkg: "sqlgen",
			description: "Package name inferred from longest part after splitting by '-'",
		},
		{
			name: "explicit package name",
			config: Config{
				Package: "myqueries",
			},
			outputPath:  "./internal/queries",
			expectedPkg: "myqueries",
			description: "Explicit package name overrides auto-inference",
		},
		{
			name: "complex hyphenated path",
			config: Config{
				Package: "",
			},
			outputPath:  "./generated/user-management-go",
			expectedPkg: "management", // "user", "management", "go" -> "management" is longest
			description: "Complex hyphenated path with multiple parts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			generator := New(format,
				WithConfig(tt.config, tt.outputPath),
				WithDialect("postgresql"),
			)

			err := generator.Generate(&output)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			generatedCode := output.String()
			expectedPackageDecl := "package " + tt.expectedPkg

			if !strings.Contains(generatedCode, expectedPackageDecl) {
				t.Errorf("Expected package declaration '%s', but not found in generated code", expectedPackageDecl)
				t.Logf("Generated code:\n%s", generatedCode)
			}

			t.Logf("%s: %s -> package %s", tt.description, tt.outputPath, tt.expectedPkg)
		})
	}
}

// Demonstrate the new simplified configuration structure
func TestSimplifiedConfigStructure(t *testing.T) {
	// Old structure (with settings hierarchy):
	// generators:
	//   go:
	//     output: "./internal/queries"
	//     enabled: true
	//     settings:
	//       package: "queries"
	//       generate_tests: true
	//       mock_path: "./testdata/mocks"

	// New structure (flat):
	// generators:
	//   go:
	//     output: "./internal/queries"
	//     enabled: true
	//     package: "queries"              # Optional: auto-inferred if omitted
	//     preserve_hierarchy: true        # Optional: default true
	//     mock_path: "./testdata/mocks"   # Optional
	//     generate_tests: false           # Optional: default false

	config := Config{
		Package:           "",   // Auto-infer from output path
		PreserveHierarchy: true, // Maintain directory structure
		MockPath:          "./testdata/mocks",
		GenerateTests:     false,
	}

	outputPath := "./internal/queries"
	expectedPackage := "queries" // Auto-inferred from "queries" directory

	// Test the configuration
	generator := &Generator{PackageName: "default"}
	option := WithConfig(config, outputPath)
	option(generator)

	if generator.PackageName != expectedPackage {
		t.Errorf("Expected package name '%s', got '%s'", expectedPackage, generator.PackageName)
	}

	t.Logf("Configuration applied successfully:")
	t.Logf("  Output path: %s", outputPath)
	t.Logf("  Package name: %s (auto-inferred)", generator.PackageName)
	t.Logf("  Mock path: %s", generator.MockPath)
}
