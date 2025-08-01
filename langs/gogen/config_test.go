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
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestInferPackageNameFromPath(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		expected   string
	}{
		{
			name:       "simple directory",
			outputPath: "./internal/queries",
			expected:   "queries",
		},
		{
			name:       "hyphenated directory - queries longest",
			outputPath: "./pkg/db-queries",
			expected:   "queries",
		},
		{
			name:       "hyphenated directory - models longest",
			outputPath: "./generated/go-models",
			expected:   "models",
		},
		{
			name:       "multiple hyphens - user longest",
			outputPath: "./src/user-go-api",
			expected:   "user",
		},
		{
			name:       "equal length parts - first one",
			outputPath: "./pkg/go-db",
			expected:   "go", // Both "go" and "db" are 2 chars, takes first longest found
		},
		{
			name:       "single character parts",
			outputPath: "./a-b-c-longer",
			expected:   "longer",
		},
		{
			name:       "no hyphens",
			outputPath: "./generated",
			expected:   "generated",
		},
		{
			name:       "empty path",
			outputPath: "",
			expected:   "generated",
		},
		{
			name:       "absolute path",
			outputPath: "/home/user/project/internal/queries",
			expected:   "queries",
		},
		{
			name:       "with special characters",
			outputPath: "./pkg/user_management-go-api",
			expected:   "user_management", // Longest part
		},
		{
			name:       "common go prefix pattern",
			outputPath: "./pkg/go-sqlgen",
			expected:   "sqlgen",
		},
		{
			name:       "common go suffix pattern",
			outputPath: "./pkg/queries-go",
			expected:   "queries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferPackageNameFromPath(tt.outputPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		outputPath string
		expected   string
	}{
		{
			name: "explicit package name",
			config: Config{
				Package: "mypackage",
			},
			outputPath: "./internal/queries",
			expected:   "mypackage",
		},
		{
			name: "auto-inferred package name",
			config: Config{
				Package: "", // Empty means auto-infer
			},
			outputPath: "./internal/queries",
			expected:   "queries",
		},
		{
			name: "auto-inferred with hyphens",
			config: Config{
				Package: "",
			},
			outputPath: "./pkg/go-sqlgen",
			expected:   "sqlgen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := &Generator{
				PackageName: "default", // Should be overridden
			}

			option := WithConfig(tt.config, tt.outputPath)
			option(generator)

			assert.Equal(t, tt.expected, generator.PackageName)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "", config.Package) // Should be empty for auto-inference
	assert.Equal(t, true, config.PreserveHierarchy)
	assert.Equal(t, "", config.MockPath)
	assert.Equal(t, false, config.GenerateTests)
}
