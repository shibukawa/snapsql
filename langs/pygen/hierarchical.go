package pygen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// hierarchicalField represents a field in a hierarchical structure
type hierarchicalField struct {
	Name       string
	Type       string
	JSONTag    string
	PyType     string
	IsOptional bool
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
// It builds a multi-level tree of hierarchical fields.
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

			pyType, err := ConvertToPythonType(r.Type, r.IsNullable)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert type for field %s: %w", r.Name, err)
			}

			n.Fields = append(n.Fields, hierarchicalField{
				Name:       toSnakeCase(fieldName),
				Type:       r.Type,
				JSONTag:    fieldName,
				PyType:     pyType,
				IsOptional: r.IsNullable,
			})
		} else {
			pyType, err := ConvertToPythonType(r.Type, r.IsNullable)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert type for field %s: %w", r.Name, err)
			}

			rootFields = append(rootFields, hierarchicalField{
				Name:       toSnakeCase(r.Name),
				Type:       r.Type,
				JSONTag:    r.Name,
				PyType:     pyType,
				IsOptional: r.IsNullable,
			})
		}
	}

	return allNodes, rootFields, nil
}

// generateHierarchicalStructs generates dataclass definitions for hierarchical data
func generateHierarchicalStructs(functionName string, nodes map[string]*node, rootFields []hierarchicalField) ([]responseStructData, *responseStructData, error) {
	if len(nodes) == 0 {
		return nil, nil, nil
	}

	// Collect nodes sorted by path length (children before parents)
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

	mainClassName := generateClassName(functionName)

	// Map pathKey -> className for reference
	classNames := map[string]string{}

	for _, n := range nodeList {
		suffixParts := make([]string, len(n.PathSegments))
		for i, s := range n.PathSegments {
			suffixParts[i] = toPascalCase(s)
		}

		classNames[pathKey(n.PathSegments)] = mainClassName + strings.Join(suffixParts, "")
	}

	var structs []responseStructData
	// Generate dataclass definitions (children first ensures availability)
	for _, n := range nodeList {
		className := classNames[pathKey(n.PathSegments)]

		fields := make([]responseFieldData, 0)

		// Add fields
		for _, f := range n.Fields {
			fields = append(fields, responseFieldData{
				Name:       f.Name,
				TypeHint:   f.PyType,
				HasDefault: f.IsOptional,
				Default:    "None",
			})
		}

		// Add child list fields
		childKeys := make([]string, 0, len(n.Children))
		for childSeg := range n.Children {
			childKeys = append(childKeys, childSeg)
		}

		sort.Strings(childKeys)

		for _, childSeg := range childKeys {
			childPath := append(append([]string{}, n.PathSegments...), childSeg)
			childClass := classNames[pathKey(childPath)]
			fields = append(fields, responseFieldData{
				Name:       toSnakeCase(childSeg),
				TypeHint:   fmt.Sprintf("List[%s]", childClass),
				HasDefault: true,
				Default:    "None",
			})
		}

		structs = append(structs, responseStructData{
			ClassName: className,
			Fields:    fields,
		})
	}

	// Main struct fields: rootFields + top-level group lists
	mainFields := make([]responseFieldData, 0, len(rootFields)+8)
	for _, rf := range rootFields {
		mainFields = append(mainFields, responseFieldData{
			Name:       rf.Name,
			TypeHint:   rf.PyType,
			HasDefault: rf.IsOptional,
			Default:    "None",
		})
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
		className := classNames[k]
		mainFields = append(mainFields, responseFieldData{
			Name:       toSnakeCase(n.PathSegments[0]),
			TypeHint:   fmt.Sprintf("List[%s]", className),
			HasDefault: true,
			Default:    "None",
		})
	}

	mainStruct := &responseStructData{
		ClassName: mainClassName,
		Fields:    mainFields,
	}

	return structs, mainStruct, nil
}

// toPascalCase converts a snake_case string to PascalCase
// Example: "user_id" -> "UserId", "board" -> "Board"
func toPascalCase(name string) string {
	parts := strings.Split(name, "_")
	result := make([]string, len(parts))

	for i, part := range parts {
		if part == "" {
			continue
		}
		// Special case for common abbreviations
		if strings.ToLower(part) == "id" {
			result[i] = "Id"
		} else {
			result[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	return strings.Join(result, "")
}
