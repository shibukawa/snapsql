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

func TestParseFileHierarchy(t *testing.T) {
	tests := []struct {
		name       string
		inputPath  string
		inputRoot  string
		outputRoot string
		expected   FileHierarchy
	}{
		{
			name:       "root level SQL file",
			inputPath:  "/project/queries/users.snap.sql",
			inputRoot:  "/project/queries",
			outputRoot: "/project/generated",
			expected: FileHierarchy{
				InputPath:    "/project/queries/users.snap.sql",
				OutputPath:   "/project/generated/users.go",
				RelativeDir:  ".",
				FileName:     "users.snap.sql",
				IsCommonType: false,
			},
		},
		{
			name:       "nested SQL file",
			inputPath:  "/project/queries/orders/reports/monthly.snap.sql",
			inputRoot:  "/project/queries",
			outputRoot: "/project/generated",
			expected: FileHierarchy{
				InputPath:    "/project/queries/orders/reports/monthly.snap.sql",
				OutputPath:   "/project/generated/orders/reports/monthly.go",
				RelativeDir:  "orders/reports",
				FileName:     "monthly.snap.sql",
				IsCommonType: false,
			},
		},
		{
			name:       "common types file",
			inputPath:  "/project/queries/orders/_common.yaml",
			inputRoot:  "/project/queries",
			outputRoot: "/project/generated",
			expected: FileHierarchy{
				InputPath:    "/project/queries/orders/_common.yaml",
				OutputPath:   "/project/generated/orders/common_types.go",
				RelativeDir:  "orders",
				FileName:     "_common.yaml",
				IsCommonType: true,
			},
		},
		{
			name:       "markdown file",
			inputPath:  "/project/queries/admin/stats.snap.md",
			inputRoot:  "/project/queries",
			outputRoot: "/project/generated",
			expected: FileHierarchy{
				InputPath:    "/project/queries/admin/stats.snap.md",
				OutputPath:   "/project/generated/admin/stats.go",
				RelativeDir:  "admin",
				FileName:     "stats.snap.md",
				IsCommonType: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFileHierarchy(tt.inputPath, tt.inputRoot, tt.outputRoot)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPackageNameFromHierarchy(t *testing.T) {
	tests := []struct {
		name        string
		hierarchy   FileHierarchy
		basePackage string
		expected    string
	}{
		{
			name: "root level",
			hierarchy: FileHierarchy{
				RelativeDir: ".",
			},
			basePackage: "generated",
			expected:    "generated",
		},
		{
			name: "single level",
			hierarchy: FileHierarchy{
				RelativeDir: "orders",
			},
			basePackage: "generated",
			expected:    "orders",
		},
		{
			name: "nested level",
			hierarchy: FileHierarchy{
				RelativeDir: "orders/reports",
			},
			basePackage: "generated",
			expected:    "reports",
		},
		{
			name: "with special characters",
			hierarchy: FileHierarchy{
				RelativeDir: "user-management/api-v2",
			},
			basePackage: "generated",
			expected:    "api_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPackageNameFromHierarchy(tt.hierarchy, tt.basePackage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateImportPath(t *testing.T) {
	tests := []struct {
		name        string
		baseImport  string
		relativeDir string
		expected    string
	}{
		{
			name:        "root level",
			baseImport:  "github.com/example/project/generated",
			relativeDir: ".",
			expected:    "github.com/example/project/generated",
		},
		{
			name:        "single level",
			baseImport:  "github.com/example/project/generated",
			relativeDir: "orders",
			expected:    "github.com/example/project/generated/orders",
		},
		{
			name:        "nested level",
			baseImport:  "github.com/example/project/generated",
			relativeDir: "orders/reports",
			expected:    "github.com/example/project/generated/orders/reports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateImportPath(tt.baseImport, tt.relativeDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}
