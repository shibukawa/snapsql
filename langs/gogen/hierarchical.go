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
	"sort"
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

// node represents a hierarchical group (path of prefixes) collecting its fields and children
type node struct {
	PathSegments []string            // e.g. ["board", "list"]
	Fields       []hierarchicalField // columns whose name == path + "__" + field
	Children     map[string]*node    // key = next segment
}

func newNode(path []string) *node {
	return &node{PathSegments: append([]string{}, path...), Children: map[string]*node{}}
}

// pathKey joins segments with "__" for stable map keys
func pathKey(segs []string) string { return strings.Join(segs, "__") }

// detectHierarchicalStructure analyzes response fields for hierarchical patterns
// detectHierarchicalStructure builds a multi-level tree of hierarchical fields.
// Field naming rule assumed: level1__level2__...__column
// Each path (without final column segment) becomes a node; columns attach to that node.
func detectHierarchicalStructure(responses []intermediate.Response) (map[string]*node, []hierarchicalField, error) {
	roots := map[string]*node{} // top-level nodes keyed by first segment
	allNodes := map[string]*node{}
	rootFields := make([]hierarchicalField, 0)

	var getOrCreate func(path []string) *node

	getOrCreate = func(path []string) *node {
		k := pathKey(path)
		if n, ok := allNodes[k]; ok {
			return n
		}

		n := newNode(path)

		allNodes[k] = n
		if len(path) == 1 {
			roots[path[0]] = n
		} else {
			parent := getOrCreate(path[:len(path)-1])
			parent.Children[path[len(path)-1]] = n
		}

		return n
	}

	for _, r := range responses {
		if strings.Contains(r.Name, "__") {
			segs := strings.Split(r.Name, "__")
			if len(segs) < 2 { // safety
				continue
			}

			fieldName := segs[len(segs)-1]
			groupPath := segs[:len(segs)-1]
			n := getOrCreate(groupPath)

			goType, err := convertToGoType(r.Type)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert type for field %s: %w", r.Name, err)
			}

			isPointer := r.IsNullable
			if isPointer && !strings.HasPrefix(goType, "*") {
				goType = "*" + goType
			}

			n.Fields = append(n.Fields, hierarchicalField{
				Name:      celNameToGoName(fieldName),
				Type:      r.Type,
				JSONTag:   fieldName,
				GoType:    goType,
				IsPointer: isPointer,
			})
		} else {
			goType, err := convertToGoType(r.Type)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert type for field %s: %w", r.Name, err)
			}

			isPointer := r.IsNullable
			if isPointer && !strings.HasPrefix(goType, "*") {
				goType = "*" + goType
			}

			rootFields = append(rootFields, hierarchicalField{
				Name:      celNameToGoName(r.Name),
				Type:      r.Type,
				JSONTag:   r.Name,
				GoType:    goType,
				IsPointer: isPointer,
			})
		}
	}

	return allNodes, rootFields, nil
}

// generateHierarchicalStructs generates struct definitions for hierarchical data
func generateHierarchicalStructs(functionName string, nodes map[string]*node, rootFields []hierarchicalField) ([]string, *responseStructData, error) {
	if len(nodes) == 0 {
		return nil, nil, nil
	}

	// Collect nodes sorted by path length (children before parents for slice pointer approach)
	nodeList := make([]*node, 0, len(nodes))
	for _, n := range nodes {
		nodeList = append(nodeList, n)
	}

	sort.Slice(nodeList, func(i, j int) bool {
		if len(nodeList[i].PathSegments) == len(nodeList[j].PathSegments) {
			return strings.Join(nodeList[i].PathSegments, "__") < strings.Join(nodeList[j].PathSegments, "__")
		}

		return len(nodeList[i].PathSegments) > len(nodeList[j].PathSegments) // deeper first
	})

	mainStructName := generateStructName(functionName)

	// Map pathKey -> structName for reference
	structNames := map[string]string{}

	for _, n := range nodeList {
		suffixParts := make([]string, len(n.PathSegments))
		for i, s := range n.PathSegments {
			suffixParts[i] = celNameToGoName(s)
		}

		structNames[pathKey(n.PathSegments)] = mainStructName + strings.Join(suffixParts, "")
	}

	var structs []string
	// Generate struct definitions (children first ensures availability)
	for _, n := range nodeList {
		structName := structNames[pathKey(n.PathSegments)]

		var b strings.Builder
		b.WriteString(fmt.Sprintf("type %s struct {\n", structName))
		// Fields
		for _, f := range n.Fields {
			b.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", f.Name, f.GoType, f.JSONTag))
		}
		// Child slice fields (pointer slices to allow later in-place mutation)
		// We add after fields for stability
		for childSeg := range n.Children {
			childPath := append(append([]string{}, n.PathSegments...), childSeg)
			childStruct := structNames[pathKey(childPath)]
			b.WriteString(fmt.Sprintf("\t%s []*%s `json:\"%s\"`\n", celNameToGoName(childSeg), childStruct, childSeg))
		}

		b.WriteString("}")
		structs = append(structs, b.String())
	}

	// Main struct fields: rootFields + top-level group slices
	mainFields := make([]responseFieldData, 0, len(rootFields)+8)
	for _, rf := range rootFields {
		mainFields = append(mainFields, responseFieldData{Name: rf.Name, Type: rf.GoType, JSONTag: rf.JSONTag, IsPointer: rf.IsPointer})
	}
	// top-level nodes have path length 1
	topKeys := make([]string, 0)

	for _, n := range nodes {
		if len(n.PathSegments) == 1 {
			topKeys = append(topKeys, pathKey(n.PathSegments))
		}
	}

	sort.Strings(topKeys)

	for _, k := range topKeys {
		n := nodes[k]
		structName := structNames[k]
		mainFields = append(mainFields, responseFieldData{
			Name:    celNameToGoName(n.PathSegments[0]),
			Type:    "[]*" + structName,
			JSONTag: n.PathSegments[0],
		})
	}

	mainStruct := &responseStructData{Name: mainStructName, Fields: mainFields}

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
