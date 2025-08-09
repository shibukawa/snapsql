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
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/shibukawa/snapsql/intermediate"
)

// hierarchicalField represents a field in a hierarchical structure
type hierarchicalField struct {
	Name      string
	Type      string
	JSONTag   string
	GoType    string
	IsPointer bool
}

// hierarchicalGroup represents a group of fields with the same prefix
type hierarchicalGroup struct {
	Prefix string
	Fields []hierarchicalField
}

// detectHierarchicalStructure analyzes response fields for hierarchical patterns
func detectHierarchicalStructure(responses []intermediate.Response) (map[string]hierarchicalGroup, []hierarchicalField, error) {
	groups := make(map[string]hierarchicalGroup)
	rootFields := make([]hierarchicalField, 0)

	for _, response := range responses {
		if strings.Contains(response.Name, "__") {
			// This is a hierarchical field
			parts := strings.SplitN(response.Name, "__", 2)
			if len(parts) != 2 {
				continue
			}

			prefix := parts[0]
			fieldName := parts[1]

			goType, err := convertToGoType(response.Type)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert type for field %s: %w", response.Name, err)
			}

			isPointer := response.IsNullable
			if isPointer && !strings.HasPrefix(goType, "*") {
				goType = "*" + goType
			}

			field := hierarchicalField{
				Name:      celNameToGoName(fieldName),
				Type:      response.Type,
				JSONTag:   fieldName,
				GoType:    goType,
				IsPointer: isPointer,
			}

			if group, exists := groups[prefix]; exists {
				group.Fields = append(group.Fields, field)
				groups[prefix] = group
			} else {
				groups[prefix] = hierarchicalGroup{
					Prefix: prefix,
					Fields: []hierarchicalField{field},
				}
			}
		} else {
			// This is a root field
			goType, err := convertToGoType(response.Type)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert type for field %s: %w", response.Name, err)
			}

			isPointer := response.IsNullable
			if isPointer && !strings.HasPrefix(goType, "*") {
				goType = "*" + goType
			}

			field := hierarchicalField{
				Name:      celNameToGoName(response.Name), // Use full name for root fields
				Type:      response.Type,
				JSONTag:   response.Name,
				GoType:    goType,
				IsPointer: isPointer,
			}

			rootFields = append(rootFields, field)
		}
	}

	return groups, rootFields, nil
}

// generateHierarchicalStructs generates struct definitions for hierarchical data
func generateHierarchicalStructs(functionName string, groups map[string]hierarchicalGroup, rootFields []hierarchicalField) ([]string, *responseStructData, error) {
	if len(groups) == 0 {
		// No hierarchical structure detected
		return nil, nil, nil
	}

	structs := make([]string, 0)
	mainStructFields := make([]responseFieldData, 0)

	// Generate nested structs for each group
	for prefix, group := range groups {
		structName := fmt.Sprintf("%s%s", generateStructName(functionName), celNameToGoName(prefix))

		var structDef strings.Builder
		structDef.WriteString(fmt.Sprintf("type %s struct {\n", structName))

		for _, field := range group.Fields {
			structDef.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
				field.Name, field.GoType, field.JSONTag))
		}

		structDef.WriteString("}")
		structs = append(structs, structDef.String())

		// Add this as a field in the main struct
		mainStructFields = append(mainStructFields, responseFieldData{
			Name:    celNameToGoName(prefix),
			Type:    "[]" + structName, // Array of nested structs
			JSONTag: prefix,
		})
	}

	// Add root fields to main struct
	for _, field := range rootFields {
		mainStructFields = append(mainStructFields, responseFieldData{
			Name:      field.Name,
			Type:      field.GoType,
			JSONTag:   field.JSONTag,
			IsPointer: field.IsPointer,
		})
	}

	mainStruct := &responseStructData{
		Name:   generateStructName(functionName), // Remove duplicate "Result"
		Fields: mainStructFields,
	}

	return structs, mainStruct, nil
}

// celNameToGoName converts CEL field names to Go field names
func celNameToGoName(celName string) string {
	// Handle dot notation (e.g., "u.id" -> "UId")
	if strings.Contains(celName, ".") {
		parts := strings.Split(celName, ".")

		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = celNameToGoName(part) // Recursive call for each part
		}

		return strings.Join(result, "")
	}

	// Handle underscore notation (e.g., "user_id" -> "UserID")
	parts := strings.Split(celName, "_")
	caser := cases.Title(language.English)

	for i, part := range parts {
		if part == "id" {
			parts[i] = "ID"
		} else {
			parts[i] = caser.String(part)
		}
	}

	return strings.Join(parts, "")
}
