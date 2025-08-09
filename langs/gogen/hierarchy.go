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

// FileHierarchy represents a hierarchical file structure
type FileHierarchy struct {
	InputPath    string // Original input file path (relative to input directory)
	OutputPath   string // Generated output file path (relative to output directory)
	RelativeDir  string // Relative directory path from input root
	FileName     string // Base file name without extension
	IsCommonType bool   // Whether this is a _common.yaml file
}

// ParseFileHierarchy extracts hierarchy information from file paths
func ParseFileHierarchy(inputPath, inputRoot, outputRoot string) FileHierarchy {
	// Normalize paths
	relPath, _ := filepath.Rel(inputRoot, inputPath)
	dir := filepath.Dir(relPath)
	fileName := filepath.Base(relPath)

	// Remove file extension and convert to Go file
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	baseName = strings.TrimSuffix(baseName, ".snap")

	// Handle special cases
	isCommonType := fileName == "_common.yaml"
	if isCommonType {
		baseName = "common_types"
	}

	// Generate output path
	var outputPath string
	if dir == "." {
		outputPath = filepath.Join(outputRoot, baseName+".go")
	} else {
		outputPath = filepath.Join(outputRoot, dir, baseName+".go")
	}

	return FileHierarchy{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		RelativeDir:  dir,
		FileName:     fileName,
		IsCommonType: isCommonType,
	}
}

// GetPackageNameFromHierarchy determines the package name based on hierarchy
func GetPackageNameFromHierarchy(hierarchy FileHierarchy, basePackage string) string {
	if hierarchy.RelativeDir == "." || hierarchy.RelativeDir == "" {
		return basePackage
	}

	// Convert directory path to package name
	// e.g., "orders/reports" -> "reports" (use deepest directory)
	parts := strings.Split(hierarchy.RelativeDir, string(filepath.Separator))
	if len(parts) > 0 {
		return sanitizePackageName(parts[len(parts)-1])
	}

	return basePackage
}

// sanitizePackageName ensures the package name is valid for Go
func sanitizePackageName(name string) string {
	// Replace invalid characters with underscores
	result := strings.ReplaceAll(name, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")

	// Ensure it starts with a letter
	if len(result) > 0 && (result[0] >= '0' && result[0] <= '9') {
		result = "pkg_" + result
	}

	return result
}

// GenerateImportPath generates import path for hierarchical packages
func GenerateImportPath(baseImport, relativeDir string) string {
	if relativeDir == "." || relativeDir == "" {
		return baseImport
	}

	return baseImport + "/" + strings.ReplaceAll(relativeDir, string(filepath.Separator), "/")
}
