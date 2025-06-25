package intermediate

import (
	"encoding/json"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql/parser"
)

func TestRecursiveParameterStructure(t *testing.T) {
	// Create interface schema with complex nested parameters
	schema := &parser.InterfaceSchema{
		Name:          "ComplexQuery",
		Description:   "Query with nested parameters",
		FunctionName:  "complexQuery",
		OrderedParams: parser.NewOrderedParameters(),
	}

	// Add simple parameter
	schema.OrderedParams.Add("active", "bool")

	// Add nested object parameter
	filtersMap := map[string]any{
		"name":       "string",
		"department": []any{"string"},
		"permissions": map[string]any{
			"read":  "bool",
			"write": "bool",
		},
	}
	schema.OrderedParams.Add("filters", filtersMap)

	// Add array parameter
	schema.OrderedParams.Add("sort_fields", []any{"string"})

	// Create intermediate format
	format := NewFormat()
	format.SetSource("test.sql", "SELECT * FROM users")
	format.SetInterfaceSchema(schema)

	// Convert to JSON
	jsonData, err := format.ToJSONPretty()
	assert.NoError(t, err)

	// Parse JSON to verify structure
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	assert.NoError(t, err)

	// Check interface schema
	interfaceSchema := result["interface_schema"].(map[string]interface{})
	assert.Equal(t, "ComplexQuery", interfaceSchema["name"])
	assert.Equal(t, "complexQuery", interfaceSchema["function_name"])

	// Check parameters structure
	parameters := interfaceSchema["parameters"].([]interface{})
	assert.Equal(t, 3, len(parameters))

	// Check simple parameter
	activeParam := parameters[0].(map[string]interface{})
	assert.Equal(t, "active", activeParam["name"])
	assert.Equal(t, "bool", activeParam["type"])
	assert.Zero(t, activeParam["children"]) // No children for simple type

	// Check nested object parameter
	filtersParam := parameters[1].(map[string]interface{})
	assert.Equal(t, "filters", filtersParam["name"])
	assert.Equal(t, "object", filtersParam["type"])

	// Check nested object's children (order may vary for map)
	filtersChildren := filtersParam["children"].([]interface{})
	assert.Equal(t, 3, len(filtersChildren)) // name, department, permissions

	// Find children by name (since map order is not guaranteed)
	childrenByName := make(map[string]map[string]interface{})
	for _, child := range filtersChildren {
		childMap := child.(map[string]interface{})
		childrenByName[childMap["name"].(string)] = childMap
	}

	// Check name child
	nameChild, nameExists := childrenByName["name"]
	assert.True(t, nameExists)
	assert.Equal(t, "name", nameChild["name"])
	assert.Equal(t, "string", nameChild["type"])

	// Check department child
	departmentChild, deptExists := childrenByName["department"]
	assert.True(t, deptExists)
	assert.Equal(t, "department", departmentChild["name"])
	assert.Equal(t, "array", departmentChild["type"])

	departmentChildren := departmentChild["children"].([]interface{})
	assert.Equal(t, 1, len(departmentChildren))
	assert.Equal(t, "o1", departmentChildren[0].(map[string]interface{})["name"])
	assert.Equal(t, "string", departmentChildren[0].(map[string]interface{})["type"])

	// Check permissions child
	permissionsChild, permExists := childrenByName["permissions"]
	assert.True(t, permExists)
	assert.Equal(t, "permissions", permissionsChild["name"])
	assert.Equal(t, "object", permissionsChild["type"])

	permissionsChildren := permissionsChild["children"].([]interface{})
	assert.Equal(t, 2, len(permissionsChildren)) // read, write

	// Check array parameter
	sortFieldsParam := parameters[2].(map[string]interface{})
	assert.Equal(t, "sort_fields", sortFieldsParam["name"])
	assert.Equal(t, "array", sortFieldsParam["type"])

	sortFieldsChildren := sortFieldsParam["children"].([]interface{})
	assert.Equal(t, 1, len(sortFieldsChildren))
	assert.Equal(t, "o1", sortFieldsChildren[0].(map[string]interface{})["name"])
	assert.Equal(t, "string", sortFieldsChildren[0].(map[string]interface{})["type"])
}

func TestParameterTypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string type",
			input:    "string",
			expected: "string",
		},
		{
			name:     "bool type",
			input:    "bool",
			expected: "bool",
		},
		{
			name:     "int type",
			input:    "int",
			expected: "int",
		},
		{
			name:     "array type",
			input:    []any{"string", "int"},
			expected: "array",
		},
		{
			name:     "object type",
			input:    map[string]any{"field": "string"},
			expected: "object",
		},
		{
			name:     "unknown type",
			input:    123,
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractChildrenRecursive(t *testing.T) {
	// Test deeply nested structure
	deeplyNested := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": "string",
			},
		},
		"array_field": []any{
			map[string]any{
				"nested_in_array": "bool",
			},
		},
	}

	children := extractChildren(deeplyNested)
	assert.Equal(t, 2, len(children))

	// Check level1 child
	level1Child := children[0]
	assert.Equal(t, "level1", level1Child.Name)
	assert.Equal(t, "object", level1Child.Type)
	assert.Equal(t, 1, len(level1Child.Children))

	// Check level2 nested in level1
	level2Child := level1Child.Children[0]
	assert.Equal(t, "level2", level2Child.Name)
	assert.Equal(t, "object", level2Child.Type)
	assert.Equal(t, 1, len(level2Child.Children))

	// Check level3 nested in level2
	level3Child := level2Child.Children[0]
	assert.Equal(t, "level3", level3Child.Name)
	assert.Equal(t, "string", level3Child.Type)
	assert.Equal(t, 0, len(level3Child.Children))

	// Check array_field child
	arrayChild := children[1]
	assert.Equal(t, "array_field", arrayChild.Name)
	assert.Equal(t, "array", arrayChild.Type)
	assert.Equal(t, 1, len(arrayChild.Children))

	// Check object nested in array
	nestedInArrayChild := arrayChild.Children[0]
	assert.Equal(t, "o1", nestedInArrayChild.Name)
	assert.Equal(t, "object", nestedInArrayChild.Type)
	assert.Equal(t, 1, len(nestedInArrayChild.Children))

	// Check field nested in object nested in array
	deepestChild := nestedInArrayChild.Children[0]
	assert.Equal(t, "nested_in_array", deepestChild.Name)
	assert.Equal(t, "bool", deepestChild.Type)
	assert.Equal(t, 0, len(deepestChild.Children))
}

func TestGenerateObjectNameSequence(t *testing.T) {
	counter := 1

	names := []string{
		generateObjectName(&counter),
		generateObjectName(&counter),
		generateObjectName(&counter),
	}

	assert.Equal(t, []string{"o1", "o2", "o3"}, names)
	assert.Equal(t, 4, counter) // Counter should be incremented
}
