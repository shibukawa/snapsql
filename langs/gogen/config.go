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
	"path/filepath"
	"strings"
)

// Config represents Go generator configuration from snapsql.yaml
type Config struct {
	Package           string `yaml:"package"`            // Package name for generated code (auto-inferred if empty)
	PreserveHierarchy bool   `yaml:"preserve_hierarchy"` // Whether to preserve directory hierarchy
	MockPath          string `yaml:"mock_path"`          // Base path for mock data files
	GenerateTests     bool   `yaml:"generate_tests"`     // Whether to generate test files
}

// DefaultConfig returns default configuration for Go generator
func DefaultConfig() Config {
	return Config{
		Package:           "", // Will be auto-inferred from output directory
		PreserveHierarchy: true,
		MockPath:          "",
		GenerateTests:     false,
	}
}

// InferPackageNameFromPath infers package name from output directory path
func InferPackageNameFromPath(outputPath string) string {
	if outputPath == "" {
		return "generated"
	}

	// Get the last directory component
	dir := filepath.Base(outputPath)

	// Handle hyphens: split by '-' and take the longest part
	if strings.Contains(dir, "-") {
		parts := strings.Split(dir, "-")

		longest := ""
		for _, part := range parts {
			if len(part) > len(longest) {
				longest = part
			}
		}

		if longest != "" {
			dir = longest
		}
	}

	return sanitizePackageName(dir)
}

// WithConfig creates a Generator option from configuration
func WithConfig(config Config, outputPath string) Option {
	return func(g *Generator) {
		// Auto-infer package name if not specified
		packageName := config.Package
		if packageName == "" {
			packageName = InferPackageNameFromPath(outputPath)
		}

		g.PackageName = packageName
		if config.MockPath != "" {
			g.MockPath = config.MockPath
		}
		// TODO: Add GenerateTests and PreserveHierarchy support when implemented
	}
}

// Example usage:
//
// From snapsql.yaml:
// generators:
//   go:
//     output: "./internal/queries"
//     enabled: true
//     package: "queries"              # Optional: auto-inferred from output path if omitted
//     preserve_hierarchy: true        # Optional: default true
//     mock_path: "./testdata/mocks"   # Optional
//     generate_tests: true            # Optional: default false
//
// Auto-inference examples:
// output: "./internal/queries"     -> package: "queries"
// output: "./pkg/db-queries"       -> package: "queries" (longest part after splitting by '-')
// output: "./generated/go-models"  -> package: "models"
// output: "./src/user-go-api"      -> package: "user" (longest part)
//
// In code:
// config := gogen.Config{
//     Package:           "", // Will be auto-inferred as "queries"
//     PreserveHierarchy: true,
//     MockPath:          "./testdata/mocks",
// }
// generator := gogen.New(format,
//     gogen.WithConfig(config, "./internal/queries"),
//     gogen.WithDialect("postgresql"),
// )
