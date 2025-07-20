package parsercommon

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// ParseFunctionDefinitionFromYAML parses a YAML string into a FunctionDefinition and calls Finalize.
func ParseFunctionDefinitionFromYAML(yamlStr string, basePath string, projectRootPath string) (*FunctionDefinition, error) {
	var def FunctionDefinition
	if err := yaml.Unmarshal([]byte(yamlStr), &def); err != nil {
		return nil, err
	}
	if err := def.Finalize(basePath, projectRootPath); err != nil {
		return nil, err
	}
	return &def, nil
}

// Sentinel errors
var (
	ErrParameterNotFound       = fmt.Errorf("parameter not found")
	ErrInvalidParameterName    = fmt.Errorf("invalid parameter name")
	ErrInvalidParameterValue   = fmt.Errorf("invalid parameter value for type")
	ErrInvalidNamingConvention = fmt.Errorf("parameter name does not follow naming convention")
	ErrDummyDataGeneration     = fmt.Errorf("failed to generate dummy data")
	ErrParameterValidation     = fmt.Errorf("parameter validation failed")
	ErrCommonTypeNotFound      = fmt.Errorf("common type not found")
	ErrCommonTypeFileNotFound  = fmt.Errorf("common type file not found")
)

// Regular expression for valid parameter names
var validParameterNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
// Regular expression for common type references
var commonTypeRefRegex = regexp.MustCompile(`^([\.\/]*)([A-Z][a-zA-Z0-9_]*)(\[\])?$`)

type FunctionDefinition struct {
	Name           string                    `yaml:"name"`
	Description    string                    `yaml:"description"`
	FunctionName   string                    `yaml:"function_name"`
	Parameters     map[string]any            `yaml:"-"` // normalized, checked
	ParameterOrder []string                  `yaml:"-"`
	RawParameters  yaml.MapSlice             `yaml:"parameters"`
	Generators     map[string]map[string]any `yaml:"generators"`
	dummyData      map[string]any
	
	// Common type related fields
	commonTypes     map[string]map[string]any  // Loaded common type definitions
	basePath        string                     // Base path for resolving relative paths (location of definition file)
	projectRootPath string                     // Project root path
}

// Finalize normalizes, validates, and caches dummy data for parameters
func (f *FunctionDefinition) Finalize(basePath string, projectRootPath string) error {
	f.Parameters = make(map[string]any)
	f.ParameterOrder = nil
	f.basePath = basePath
	f.projectRootPath = projectRootPath
	f.commonTypes = make(map[string]map[string]any)

	// Load common type definitions
	if err := f.loadCommonTypesFile(basePath); err != nil {
		return err
	}

	// Also load common type definitions from project root (if exists)
	if projectRootPath != "" && projectRootPath != basePath {
		if err := f.loadCommonTypesFile(projectRootPath); err != nil {
			return err
		}
	}

	// Normalize parameters and resolve common type references
	normalized, order, err := f.normalizeAndResolveParameters(f.RawParameters)
	if err != nil {
		f.dummyData = nil
		return fmt.Errorf("%w: %v", ErrParameterValidation, err)
	}
	f.Parameters = normalized
	f.ParameterOrder = order

	dummy, err := generateDummyData(f.Parameters)
	if err != nil {
		f.dummyData = nil
		return fmt.Errorf("%w: %v", ErrDummyDataGeneration, err)
	}
	f.dummyData = dummy
	return nil
}

// DummyData returns cached dummy data (call Finalize before).
// If path is specified, traverses the dummy data by keys (and 0th element for arrays).
func (f *FunctionDefinition) DummyData(path ...string) any {
	var current any = f.dummyData
	for _, p := range path {
		switch v := current.(type) {
		case map[string]any:
			current = v[p]
		case []any:
			if len(v) > 0 {
				current = v[0]
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	return current
}

// normalizeTypeString handles type aliases and array notations
func normalizeTypeString(typeStr string) string {
	t := strings.ToLower(strings.TrimSpace(typeStr))
	switch t {
	case "integer", "long", "int64":
		return "int"
	case "smallint":
		return "int16"
	case "tinyint":
		return "int8"
	case "text", "varchar", "str":
		return "string"
	case "double", "number":
		return "float"
	case "decimal", "numeric":
		return "decimal"
	case "boolean":
		return "bool"
	case "array":
		return "any[]"
	}
	// list type like int[]
	if strings.HasSuffix(t, "[]") {
		base := normalizeTypeString(t[:len(t)-2])
		return base + "[]"
	}
	return t
}

// inferTypeFromValue infers type string from Go value
func inferTypeFromValue(val any) string {
	switch v := val.(type) {
	case int, int64:
		return "int"
	case int32:
		return "int32"
	case int16:
		return "int16"
	case int8:
		return "int8"
	case float64:
		return "float"
	case float32:
		return "float32"
	case bool:
		return "bool"
	case string:
		return "string"
	case []any:
		if len(v) > 0 {
			return inferTypeFromValue(v[0]) + "[]"
		}
		return "any[]"
	case yaml.MapSlice, map[string]any:
		return "object"
	default:
		return "any"
	}
}

// generateDummyData creates dummy data tree from parameter definitions
func generateDummyData(params map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(params))
	for k, v := range params {
		switch val := v.(type) {
		case string:
			result[k] = generateDummyValueFromString(val)
		case map[string]any:
			d, err := generateDummyData(val)
			if err != nil {
				return nil, err
			}
			result[k] = d
		case []any:
			// Array type: [object] or [type name]
			if len(val) == 1 {
				switch elem := val[0].(type) {
				case string:
					result[k] = []any{generateDummyValueFromString(elem)}
				case map[string]any:
					d, err := generateDummyData(elem)
					if err != nil {
						return nil, err
					}
					result[k] = []any{d}
				default:
					result[k] = []any{elem}
				}
			} else {
				result[k] = []any{}
			}
		default:
			return nil, fmt.Errorf("unsupported parameter type: %T", v)
		}
	}
	return result, nil
}

// generateDummyValueFromString generates dummy value from string type definition
func generateDummyValueFromString(typeStr string) any {
	t := strings.TrimSpace(typeStr)
	switch t {
	case "string", "text", "varchar", "str":
		return "dummy"
	case "int":
		return int64(1)
	case "int32":
		return int32(2)
	case "int16":
		return int16(3)
	case "int8":
		return int8(4)
	case "float":
		return 1.1
	case "float32":
		return float32(2.2)
	case "decimal":
		return "1.0"
	case "bool":
		return true
	case "date":
		return "2024-01-01"
	case "datetime":
		return "2024-01-01 00:00:00"
	case "timestamp":
		return "2024-01-02 00:00:00"
	case "email":
		return "user@example.com"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "json":
		return map[string]any{"#": "json"}
	case "any":
		return map[string]any{"#": "any"}
	case "object":
		return map[string]any{"#": "object"}
	}
	// Array type: int[], string[], float32[] etc.
	if strings.HasSuffix(t, "[]") {
		base := t[:len(t)-2]
		return []any{generateDummyValueFromString(base)}
	}
	return ""
}

// inferTypeStringFromDummyValue infers type string from a dummy value generated by generateDummyValueFromString.
// Only primitive types, object, json, any are supported. No array support.
func inferTypeStringFromDummyValue(val any) string {
	switch v := val.(type) {
	case int64:
		if v == 1 {
			return "int"
		}
	case int32:
		if v == 2 {
			return "int32"
		}
	case int16:
		if v == 3 {
			return "int16"
		}
	case int8:
		if v == 4 {
			return "int8"
		}
	case float64:
		if v == 1.1 {
			return "float"
		}
	case float32:
		if v == float32(2.2) {
			return "float32"
		}
	case bool:
		if v {
			return "bool"
		}
	case *snapsqlgo.Decimal:
		return "decimal"
	case string:
		switch v {
		case "dummy":
			return "string"
		case "1.0":
			return "decimal"
		case "2024-01-01":
			return "date"
		case "2024-01-01 00:00:00":
			return "datetime"
		case "2024-01-02 00:00:00":
			return "timestamp"
		case "user@example.com":
			return "email"
		case "00000000-0000-0000-0000-000000000000":
			return "uuid"
		default:
			return "string"
		}
	case map[string]any:
		if tag, ok := v["#"]; ok {
			switch tag {
			case "json":
				return "json"
			case "any":
				return "any"
			case "object":
				return "object"
			}
		}
	}
	return "any"
}

// loadCommonTypesFile loads common type definitions from _common.yaml file
func (f *FunctionDefinition) loadCommonTypesFile(dirPath string) error {
	filePath := filepath.Join(dirPath, "_common.yaml")
	
	// Check if file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// If file doesn't exist, ignore (not an error)
		return nil
	} else if err != nil {
		return err
	}
	
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	// Parse YAML
	var commonTypes map[string]any
	if err := yaml.Unmarshal(data, &commonTypes); err != nil {
		return err
	}
	
	// Extract only type definitions that start with uppercase letter
	for typeName, typeDef := range commonTypes {
		if len(typeName) > 0 && unicode.IsUpper(rune(typeName[0])) {
			// Ensure type definition is a map
			if typeDefMap, ok := typeDef.(map[string]any); ok {
				// Create type name with relative path (e.g., ./User)
				relPath := "." + string(filepath.Separator)
				fullTypeName := relPath + typeName
				f.commonTypes[fullTypeName] = typeDefMap
				
				// Also register without path (e.g., User)
				f.commonTypes[typeName] = typeDefMap
				
				// Register with the directory path as prefix
				dirTypeName := dirPath + string(filepath.Separator) + typeName
				f.commonTypes[dirTypeName] = typeDefMap
			}
		}
	}
	
	return nil
}

// normalizeAndResolveParameters recursively normalizes parameters and resolves common type references
func (f *FunctionDefinition) normalizeAndResolveParameters(params yaml.MapSlice) (map[string]any, []string, error) {
	result := make(map[string]any, len(params))
	order := make([]string, 0, len(params))
	errs := &ParseError{}
	
	for _, item := range params {
		key, ok := item.Key.(string)
		if !ok {
			errs.Add(fmt.Errorf("%w: parameter key is not string: %v", ErrInvalidForSnapSQL, item.Key))
			continue
		}
		// Validate parameter name
		if !validParameterNameRegex.MatchString(key) {
			errs.Add(fmt.Errorf("%w: invalid parameter name: %s", ErrInvalidForSnapSQL, key))
			continue
		}
		order = append(order, key)
		
		result[key] = f.normalizeAndResolveAny(item.Value, key, errs)
	}
	
	if len(errs.Errors) > 0 {
		return result, order, errs
	}
	return result, order, nil
}

// normalizeAndResolveAny normalizes any value and resolves common type references
func (f *FunctionDefinition) normalizeAndResolveAny(v any, fullName string, errs *ParseError) any {
	switch val := v.(type) {
	case []any:
		// Array literal ([int], [string], etc.)
		if len(val) == 1 {
			// [int] → "int[]" etc.
			elem := val[0]
			switch elemType := elem.(type) {
			case string:
				// If array element is a string, check if it's a common type reference
				resolvedType := f.resolveCommonTypeRef(elemType)
				if resolvedType != nil {
					// Return array of common type
					return []any{resolvedType}
				}
				return elemType + "[]"
			case map[string]any:
				return []any{f.normalizeAndResolveAny(elemType, fullName+"[]", errs)}
			default:
				return []any{f.normalizeAndResolveAny(elemType, fullName+"[]", errs)}
			}
		} else {
			// Recursively normalize each element of the array
			result := make([]any, len(val))
			for i, e := range val {
				result[i] = f.normalizeAndResolveAny(e, fullName+fmt.Sprintf("[%d]", i), errs)
			}
			return result
		}
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			if !validParameterNameRegex.MatchString(k) {
				errs.Add(fmt.Errorf("%w: invalid parameter name: %s", ErrInvalidForSnapSQL, fullName+"."+k))
				continue
			}
			result[k] = f.normalizeAndResolveAny(v, fullName+"."+k, errs)
		}
		return result
	case string:
		// If string, check if it's a common type reference
		resolvedType := f.resolveCommonTypeRef(val)
		if resolvedType != nil {
			return resolvedType
		}
		return normalizeTypeString(val)
	default:
		return inferTypeFromValue(val)
	}
}

// resolveCommonTypeRef resolves a common type reference
func (f *FunctionDefinition) resolveCommonTypeRef(typeStr string) any {
	// Check if it's a common type reference
	matches := commonTypeRefRegex.FindStringSubmatch(typeStr)
	if matches == nil {
		return nil
	}
	
	path := matches[1]    // Path part (e.g., "../", "./", "")
	typeName := matches[2] // Type name part (e.g., "User")
	isArray := matches[3] != "" // Whether it's an array (has "[]" or not)
	
	// If path is specified, load _common.yaml from the corresponding directory
	if path != "" {
		var targetPath string
		if path == "." {
			targetPath = f.basePath
		} else {
			// First try path relative to basePath
			targetPath = filepath.Clean(filepath.Join(f.basePath, path))
		}
		
		if err := f.loadCommonTypesFile(targetPath); err == nil {
			// Loading successful
		} else if f.projectRootPath != "" {
			// If loading from basePath fails, try path relative to projectRootPath
			targetPath = filepath.Clean(filepath.Join(f.projectRootPath, path))
			if err := f.loadCommonTypesFile(targetPath); err != nil {
				// Ignore errors and continue
				// Will error later if type is not found
			}
		}
		
		// Try to find the type with the full path
		if path != "." {
			// For relative paths like "../roles/Role", we need to construct the key correctly
			if strings.Contains(path, "../") || strings.Contains(path, "./") {
				// Try with the full path
				fullPath := filepath.Clean(filepath.Join(f.basePath, path))
				fullTypeName := fullPath + string(filepath.Separator) + typeName
				
				typeDef, found := f.commonTypes[fullTypeName]
				if found {
					// Make a deep copy of the type definition
					typeCopy := deepCopyMap(typeDef)
					
					// If it's an array
					if isArray {
						return []any{typeCopy}
					}
					
					return typeCopy
				}
				
				// Try with just the directory name + type name
				dirName := filepath.Base(fullPath)
				dirTypeName := dirName + string(filepath.Separator) + typeName
				
				typeDef, found = f.commonTypes[dirTypeName]
				if found {
					// Make a deep copy of the type definition
					typeCopy := deepCopyMap(typeDef)
					
					// If it's an array
					if isArray {
						return []any{typeCopy}
					}
					
					return typeCopy
				}
				
				// Try with the target path + type name
				targetTypeName := targetPath + string(filepath.Separator) + typeName
				
				typeDef, found = f.commonTypes[targetTypeName]
				if found {
					// Make a deep copy of the type definition
					typeCopy := deepCopyMap(typeDef)
					
					// If it's an array
					if isArray {
						return []any{typeCopy}
					}
					
					return typeCopy
				}
			}
		}
	}
	
	// Convert type name to full form (including path)
	fullTypeName := path + typeName
	
	// Search for common type
	typeDef, found := f.commonTypes[fullTypeName]
	if !found {
		// Search by type name only (without path)
		typeDef, found = f.commonTypes[typeName]
		if !found {
			return nil
		}
	}
	
	// Make a deep copy of the type definition
	typeCopy := deepCopyMap(typeDef)
	
	// If it's an array
	if isArray {
		return []any{typeCopy}
	}
	
	return typeCopy
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = deepCopyMap(val)
		case []any:
			result[k] = deepCopySlice(val)
		default:
			result[k] = val
		}
	}
	return result
}

// deepCopySlice creates a deep copy of a slice
func deepCopySlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			result[i] = deepCopyMap(val)
		case []any:
			result[i] = deepCopySlice(val)
		default:
			result[i] = val
		}
	}
	return result
}
