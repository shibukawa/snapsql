package snapsql

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/goccy/go-yaml"
)

func TestConfig_IsSystemField(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"explicit created_at", "created_at", true},
		{"explicit updated_at", "updated_at", true},
		{"explicit created_by", "created_by", true},
		{"explicit updated_by", "updated_by", true},
		{"non-system field", "name", false},
		{"non-system field", "email", false},
		{"non-system field", "status", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsSystemField(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetSystemField(t *testing.T) {
	config := getDefaultConfig()

	// Test existing field
	field, exists := config.GetSystemField("created_at")
	assert.True(t, exists)
	assert.Equal(t, "created_at", field.Name)
	assert.False(t, field.ExcludeFromSelect)
	assert.True(t, field.OnInsert.Default != nil)
	assert.Equal(t, "NOW()", field.OnInsert.Default)
	assert.Equal(t, ParameterError, field.OnUpdate.Parameter)

	// Test non-existing field
	_, exists = config.GetSystemField("non_existent")
	assert.False(t, exists)
}

func TestConfig_HasDefaultForInsert(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"created_at has default", "created_at", true},
		{"updated_at has default", "updated_at", true},
		{"created_by no default", "created_by", false},
		{"updated_by no default", "updated_by", false},
		{"non-existent field", "non_existent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.HasDefaultForInsert(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_HasDefaultForUpdate(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"created_at no default on update", "created_at", false},
		{"updated_at has default on update", "updated_at", true},
		{"created_by no default on update", "created_by", false},
		{"updated_by no default on update", "updated_by", false},
		{"non-existent field", "non_existent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.HasDefaultForUpdate(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetParameterHandlingForInsert(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  SystemFieldParameter
	}{
		{"created_at no parameter", "created_at", ParameterNone},
		{"updated_at no parameter", "updated_at", ParameterNone},
		{"created_by implicit parameter", "created_by", ParameterImplicit},
		{"updated_by implicit parameter", "updated_by", ParameterImplicit},
		{"non-existent field", "non_existent", ParameterNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetParameterHandlingForInsert(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetParameterHandlingForUpdate(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  SystemFieldParameter
	}{
		{"created_at error parameter", "created_at", ParameterError},
		{"updated_at no parameter", "updated_at", ParameterNone},
		{"created_by error parameter", "created_by", ParameterError},
		{"updated_by implicit parameter", "updated_by", ParameterImplicit},
		{"non-existent field", "non_existent", ParameterNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetParameterHandlingForUpdate(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_GetSystemFieldsForInsert(t *testing.T) {
	config := getDefaultConfig()

	fields := config.GetSystemFieldsForInsert()

	// All default system fields should be included for INSERT
	assert.Equal(t, 4, len(fields))

	fieldNames := make(map[string]bool)
	for _, field := range fields {
		fieldNames[field.Name] = true
	}

	assert.True(t, fieldNames["created_at"])
	assert.True(t, fieldNames["updated_at"])
	assert.True(t, fieldNames["created_by"])
	assert.True(t, fieldNames["updated_by"])
}

func TestConfig_GetSystemFieldsForUpdate(t *testing.T) {
	config := getDefaultConfig()

	fields := config.GetSystemFieldsForUpdate()

	// All fields have UPDATE configuration (either default or parameter)
	assert.Equal(t, 4, len(fields))

	fieldNames := make(map[string]bool)
	for _, field := range fields {
		fieldNames[field.Name] = true
	}

	assert.True(t, fieldNames["created_at"]) // Has error parameter
	assert.True(t, fieldNames["updated_at"]) // Has default
	assert.True(t, fieldNames["created_by"]) // Has error parameter
	assert.True(t, fieldNames["updated_by"]) // Has implicit parameter
}

func TestConfig_ShouldExcludeFromSelect(t *testing.T) {
	config := getDefaultConfig()

	// Default configuration should not exclude any fields from SELECT
	assert.False(t, config.ShouldExcludeFromSelect("created_at"))
	assert.False(t, config.ShouldExcludeFromSelect("updated_at"))
	assert.False(t, config.ShouldExcludeFromSelect("created_by"))
	assert.False(t, config.ShouldExcludeFromSelect("updated_by"))
	assert.False(t, config.ShouldExcludeFromSelect("non_existent"))
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Test loading config with non-existent file (should return defaults)
	config, err := LoadConfig("non-existent-file.yaml")
	assert.NoError(t, err)
	assert.True(t, config != nil)

	// Verify system field defaults are applied
	assert.Equal(t, 4, len(config.System.Fields))

	// Verify created_at configuration
	createdAt, exists := config.GetSystemField("created_at")
	assert.True(t, exists)
	assert.True(t, createdAt.OnInsert.Default != nil)
	assert.Equal(t, "NOW()", createdAt.OnInsert.Default)
	assert.Equal(t, ParameterError, createdAt.OnUpdate.Parameter)

	// Verify created_by configuration
	createdBy, exists := config.GetSystemField("created_by")
	assert.True(t, exists)
	assert.Equal(t, ParameterImplicit, createdBy.OnInsert.Parameter)
	assert.Equal(t, ParameterError, createdBy.OnUpdate.Parameter)
}

func TestSystemFieldParameter_Constants(t *testing.T) {
	// Test that parameter constants are defined correctly
	assert.Equal(t, SystemFieldParameter("explicit"), ParameterExplicit)
	assert.Equal(t, SystemFieldParameter("implicit"), ParameterImplicit)
	assert.Equal(t, SystemFieldParameter("error"), ParameterError)
	assert.Equal(t, SystemFieldParameter(""), ParameterNone)
}

func TestConfig_GetDefaultValueForInsert(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  any
		exists    bool
	}{
		{"created_at has default", "created_at", "NOW()", true},
		{"updated_at has default", "updated_at", "NOW()", true},
		{"created_by no default", "created_by", nil, false},
		{"updated_by no default", "updated_by", nil, false},
		{"non-existent field", "non_existent", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := config.GetDefaultValueForInsert(tt.fieldName)
			assert.Equal(t, tt.exists, exists)
			if exists {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConfig_GetDefaultValueForUpdate(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name      string
		fieldName string
		expected  any
		exists    bool
	}{
		{"created_at no default on update", "created_at", nil, false},
		{"updated_at has default on update", "updated_at", "NOW()", true},
		{"created_by no default on update", "created_by", nil, false},
		{"updated_by no default on update", "updated_by", nil, false},
		{"non-existent field", "non_existent", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := config.GetDefaultValueForUpdate(tt.fieldName)
			assert.Equal(t, tt.exists, exists)
			if exists {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConfig_YAMLNullHandling(t *testing.T) {
	// Test YAML parsing with null values
	yamlContent := `
system:
  fields:
    - name: "test_field"
      on_insert:
        default: null
      on_update:
        parameter: "explicit"
    - name: "string_field"
      on_insert:
        default: "NOW()"
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	assert.NoError(t, err)

	// Verify that YAML null is properly handled
	_, exists := config.GetSystemField("test_field")
	assert.True(t, exists)

	// YAML null should result in nil pointer, so HasDefaultForInsert should return false
	assert.False(t, config.HasDefaultForInsert("test_field"))

	// String field should have default
	assert.True(t, config.HasDefaultForInsert("string_field"))
	value, exists := config.GetDefaultValueForInsert("string_field")
	assert.True(t, exists)
	assert.Equal(t, "NOW()", value)
}

func TestConfig_DifferentDefaultTypes(t *testing.T) {
	// Test YAML parsing with different default value types
	yamlContent := `
system:
  fields:
    - name: "string_field"
      on_insert:
        default: "NOW()"
    - name: "int_field"
      on_insert:
        default: 42
    - name: "bool_field"
      on_insert:
        default: true
    - name: "float_field"
      on_insert:
        default: 3.14
    - name: "null_field"
      on_insert:
        default: null
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	assert.NoError(t, err)

	// Test string default
	value, exists := config.GetDefaultValueForInsert("string_field")
	assert.True(t, exists)
	assert.Equal(t, "NOW()", value)

	// Test integer default
	value, exists = config.GetDefaultValueForInsert("int_field")
	assert.True(t, exists)
	assert.Equal(t, 42, value)

	// Test boolean default
	value, exists = config.GetDefaultValueForInsert("bool_field")
	assert.True(t, exists)
	assert.Equal(t, true, value)

	// Test float default
	value, exists = config.GetDefaultValueForInsert("float_field")
	assert.True(t, exists)
	assert.Equal(t, 3.14, value)

	// Test null default (should not have default)
	assert.False(t, config.HasDefaultForInsert("null_field"))
}
