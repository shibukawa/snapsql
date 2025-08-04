package parsercommon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/tokenizer"
)

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
	FunctionName       string                    `yaml:"function_name"`
	Description        string                    `yaml:"description"`
	Parameters         map[string]any            `yaml:"-"` // normalized, checked
	OriginalParameters map[string]any            `yaml:"-"` // original from YAML
	ParameterOrder     []string                  `yaml:"-"`
	RawParameters      yaml.MapSlice             `yaml:"parameters"`
	Generators         map[string]map[string]any `yaml:"generators"`
	dummyData          map[string]any

	// Common type related fields
	commonTypes     map[string]map[string]map[string]any // Loaded common type definitions
	basePath        string                               // Base path for resolving relative paths (location of definition file)
	projectRootPath string                               // Project root path
}

func ParseFunctionDefinitionFromSQLComment(tokens []tokenizer.Token, basePath string, projectRootPath string) (*FunctionDefinition, error) {
	// Extract from comment tokens (inline extractCommentDefinitionFromTokens)
	for _, token := range tokens {
		if token.Type == tokenizer.BLOCK_COMMENT {
			content := strings.TrimSpace(token.Value)

			// Only process comments that start with /*# (def marker)
			if strings.HasPrefix(content, "/*#") && strings.HasSuffix(content, "*/") {
				// Remove /*# and */ markers
				content = strings.TrimSpace(content[3 : len(content)-2])

				// This should be YAML content
				if content != "" {
					return parseFunctionDefinitionFromYAML(content, basePath, projectRootPath)
				}
			}
		}
	}
	return nil, errors.New("no function definition found in SQL comment")
}

// ParseFunctionDefinitionFromSnapSQLDocument creates a FunctionDefinition from a SnapSQLDocument.
// It extracts metadata, description, and parameters from the document and calls Finalize.
func ParseFunctionDefinitionFromSnapSQLDocument(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string) (*FunctionDefinition, error) {
	// Create a new FunctionDefinition
	def := &FunctionDefinition{
		// Copy metadata fields
		FunctionName: getStringFromMap(doc.Metadata, "function_name", ""),
		Description:  getStringFromMap(doc.Metadata, "description", ""),
	}

	// Copy generators if present
	if generators, ok := doc.Metadata["generators"]; ok {
		def.Generators = make(map[string]map[string]any)

		// Handle different types of generators data structure
		switch g := generators.(type) {
		case map[string]any:
			// Direct map[string]any format
			for lang, config := range g {
				if langConfig, ok := config.(map[string]any); ok {
					def.Generators[lang] = langConfig
				}
			}
		case map[string]map[string]any:
			// Already in the right format
			def.Generators = g
		case map[any]any:
			// Convert from map[any]any
			for langAny, configAny := range g {
				if lang, ok := langAny.(string); ok {
					if config, ok := configAny.(map[any]any); ok {
						langConfig := make(map[string]any)
						for keyAny, valAny := range config {
							if key, ok := keyAny.(string); ok {
								langConfig[key] = valAny
							}
						}
						def.Generators[lang] = langConfig
					} else if config, ok := configAny.(map[string]any); ok {
						def.Generators[lang] = config
					}
				}
			}
		}
	}

	// Parse parameters from the parameter text
	if doc.ParametersText != "" {
		// Parse the raw parameter text to yaml.MapSlice to preserve order
		var rawParams yaml.MapSlice
		var err error

		switch doc.ParametersType {
		case "yaml", "yml":
			err = yaml.Unmarshal([]byte(doc.ParametersText), &rawParams)
			if err != nil {
				return nil, fmt.Errorf("failed to parse YAML parameters: %w", err)
			}

		case "json":
			// Parse JSON while preserving order using yaml parser (which preserves order)
			// Convert JSON to YAML first, then parse with yaml parser
			var jsonData interface{}
			if err := json.Unmarshal([]byte(doc.ParametersText), &jsonData); err != nil {
				return nil, fmt.Errorf("failed to parse JSON parameters: %w", err)
			}

			// Convert back to YAML to preserve order
			yamlBytes, err := yaml.Marshal(jsonData)
			if err != nil {
				return nil, fmt.Errorf("failed to convert JSON to YAML: %w", err)
			}

			err = yaml.Unmarshal(yamlBytes, &rawParams)
			if err != nil {
				return nil, fmt.Errorf("failed to parse converted YAML parameters: %w", err)
			}

		case "list":
			// Parse list format (e.g., "param1: type1\nparam2: type2")
			rawParams, err = parseListFormatParameters(doc.ParametersText)
			if err != nil {
				return nil, fmt.Errorf("failed to parse list parameters: %w", err)
			}

		default:
			return nil, fmt.Errorf("unsupported parameter type: %s", doc.ParametersType)
		}

		def.RawParameters = rawParams
	}

	// Set base path and project root path for common type resolution
	def.basePath = basePath
	def.projectRootPath = projectRootPath

	// Finalize the function definition to process parameters and resolve common types
	if err := def.Finalize(basePath, projectRootPath); err != nil {
		return nil, fmt.Errorf("failed to finalize function definition: %w", err)
	}

	return def, nil
}

// parseListFormatParameters parses list format parameters (e.g., "param1: type1\nparam2: type2")
func parseListFormatParameters(text string) (yaml.MapSlice, error) {
	var rawParams yaml.MapSlice

	lines := strings.Split(strings.TrimSpace(text), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse "name: type" format
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid list parameter format: %s", line)
		}

		name := strings.TrimSpace(parts[0])
		typeStr := strings.TrimSpace(parts[1])

		rawParams = append(rawParams, yaml.MapItem{
			Key:   name,
			Value: typeStr,
		})
	}

	return rawParams, nil
}

// parseFunctionDefinitionFromYAML parses a YAML string into a FunctionDefinition and calls Finalize.
func parseFunctionDefinitionFromYAML(yamlStr string, basePath string, projectRootPath string) (*FunctionDefinition, error) {
	var def FunctionDefinition
	if err := yaml.Unmarshal([]byte(yamlStr), &def); err != nil {
		return nil, err
	}
	if err := def.Finalize(basePath, projectRootPath); err != nil {
		return nil, err
	}
	return &def, nil
}

// Finalize normalizes, validates, and caches dummy data for parameters
func (f *FunctionDefinition) Finalize(basePath string, projectRootPath string) error {
	f.Parameters = make(map[string]any)
	f.ParameterOrder = nil
	f.basePath = basePath
	f.projectRootPath = projectRootPath
	f.commonTypes = make(map[string]map[string]map[string]any)

	// Normalize parameters and resolve common type references
	normalized, order, original, err := f.normalizeAndResolveParameters(f.RawParameters)
	if err != nil {
		f.dummyData = nil
		return fmt.Errorf("%w: %v", ErrParameterValidation, err)
	}
	f.Parameters = normalized
	f.ParameterOrder = order
	f.OriginalParameters = original

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
			// Check if this is a parameter definition with a "type" field
			if typeVal, hasType := val["type"]; hasType {
				if typeStr, ok := typeVal.(string); ok {
					result[k] = generateDummyValueFromString(typeStr)
				} else {
					result[k] = generateDummyValueFromString("string")
				}
			} else {
				// This is a nested object, recurse
				d, err := generateDummyData(val)
				if err != nil {
					return nil, err
				}
				result[k] = d
			}
		case []any:
			// Array type: [object] or [type name]
			if len(val) == 1 {
				switch elem := val[0].(type) {
				case string:
					// Check if it's a common type reference that should be kept as string
					if strings.HasPrefix(elem, "./") {
						// For common types, keep the reference as is for now
						// The actual object structure should be resolved at the parameter resolution stage
						result[k] = []any{elem}
					} else {
						result[k] = []any{generateDummyValueFromString(elem)}
					}
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
	// Common Type reference: ./User, ./Product etc.
	if strings.HasPrefix(t, "./") {
		// For Common Types, return a placeholder string that represents the type
		// This will be handled by the type system later
		return t
	}
	return ""
}

// InferTypeStringFromDummyValue infers type string from a dummy value generated by generateDummyValueFromString.
// Only primitive types, object, json, any are supported. No array support.
func InferTypeStringFromDummyValue(val any) string {
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
			// Check if it's a Common Type reference
			if strings.HasPrefix(v, "./") {
				return v
			}
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
func (f *FunctionDefinition) loadCommonTypesFile(absTargetDirPath string, targetDirKey string) error {
	if _, ok := f.commonTypes[targetDirKey]; ok {
		return nil
	}
	filePath := filepath.Join(absTargetDirPath, "_common.yaml")

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
	f.commonTypes[targetDirKey] = make(map[string]map[string]any)

	// Extract only type definitions that start with uppercase letter
	for typeName, typeDef := range commonTypes {
		if len(typeName) > 0 && unicode.IsUpper(rune(typeName[0])) {
			// Ensure type definition is a map
			if typeDefMap, ok := typeDef.(map[string]any); ok {
				// Create type name with relative path (e.g., ./User)
				f.commonTypes[targetDirKey][typeName] = typeDefMap
			}
		}
	}

	return nil
}

// normalizeAndResolveParameters recursively normalizes parameters and resolves common type references
func (f *FunctionDefinition) normalizeAndResolveParameters(params yaml.MapSlice) (map[string]any, []string, map[string]any, error) {
	resultParams := make(map[string]any, len(params))
	originalParams := make(map[string]any, len(params))
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
		detail, original := f.normalizeAndResolveAny(item.Value, key, errs)
		resultParams[key] = detail
		originalParams[key] = original
	}

	if len(errs.Errors) > 0 {
		return resultParams, order, originalParams, errs
	}
	return resultParams, order, originalParams, nil
}

// normalizeAndResolveAny normalizes any value and resolves common type references
func (f *FunctionDefinition) normalizeAndResolveAny(v any, fullName string, errs *ParseError) (any, any) {
	switch val := v.(type) {
	case []any:
		// Array literal ([int], [string], etc.)
		if len(val) == 1 {
			// [int] → "int[]" etc.
			elem := val[0]
			switch elemType := elem.(type) {
			case string:
				// If array element is a string, check if it's a common type reference
				resolvedType, commonTypeName := f.resolveCommonTypeRef(elemType)
				if resolvedType != nil {
					// Return array of common type
					return []any{resolvedType}, commonTypeName + "[]"
				}
				return elemType + "[]", elemType + "[]"
			case map[string]any:
				detail, original := f.normalizeAndResolveAny(elemType, fullName+"[]", errs)
				return []any{detail}, []any{original}
			default:
				detail, original := f.normalizeAndResolveAny(elemType, fullName+"[]", errs)
				return []any{detail}, []any{original}
			}
		} else {
			// Recursively normalize each element of the array
			detailResult := make([]any, len(val))
			originalResult := make([]any, len(val))
			for i, e := range val {
				detail, original := f.normalizeAndResolveAny(e, fullName+fmt.Sprintf("[%d]", i), errs)
				detailResult[i] = detail
				originalResult[i] = original
			}
			return detailResult, originalResult
		}
	case map[string]any:
		// Check if this is a JSON parameter structure (has "type" field)
		if typeVal, hasType := val["type"]; hasType {
			// This is a JSON parameter definition like {"type": "int", "description": "...", "optional": true}
			typeStr, ok := typeVal.(string)
			if !ok {
				errs.Add(fmt.Errorf("%w: parameter type is not string: %v", ErrInvalidForSnapSQL, typeVal))
				return "string", val // Return original structure for OriginalParameters
			}

			// Check if it's a common type reference
			resolvedType, _ := f.resolveCommonTypeRef(typeStr)
			if resolvedType != nil {
				return resolvedType, val // Return original structure for OriginalParameters
			}

			// Return the normalized type string, but preserve original structure
			normalizedType := normalizeTypeString(typeStr)
			return normalizedType, val // Return original structure for OriginalParameters
		}

		// Regular map processing (for nested objects)
		detailResult := make(map[string]any)
		originalResult := make(map[string]any)
		for k, v := range val {
			if !validParameterNameRegex.MatchString(k) {
				errs.Add(fmt.Errorf("%w: invalid parameter name: %s", ErrInvalidForSnapSQL, fullName+"."+k))
				continue
			}
			detail, original := f.normalizeAndResolveAny(v, fullName+"."+k, errs)
			detailResult[k] = detail
			originalResult[k] = original
		}
		return detailResult, originalResult
	case string:
		// If string, check if it's a common type reference
		resolvedType, commonTypeName := f.resolveCommonTypeRef(val)
		if resolvedType != nil {
			return resolvedType, commonTypeName
		}
		v := normalizeTypeString(val)
		return v, v
	default:
		v := inferTypeFromValue(val)
		return v, v
	}
}

// resolveCommonTypeRef resolves a common type reference
func (f *FunctionDefinition) resolveCommonTypeRef(typeStr string) (any, string) {
	// Check if it's a common type reference
	matches := commonTypeRefRegex.FindStringSubmatch(typeStr)
	if matches == nil {
		return nil, ""
	}

	path := matches[1]          // Path part (e.g., "../", "./", "")
	typeName := matches[2]      // Type name part (e.g., "User")
	isArray := matches[3] != "" // Whether it's an array (has "[]" or not)

	// If basePath is a file path, get its directory
	baseDir := f.basePath
	if filepath.Ext(baseDir) != "" {
		baseDir = filepath.Dir(baseDir)
	}

	// If no path is specified (e.g., "User" instead of "./User"), search from basePath up to projectRootPath
	if path == "" {
		return f.searchCommonTypeInAncestors(typeName, isArray, baseDir)
	}

	// If path is specified, load _common.yaml from the corresponding directory
	var targetPath string
	var absTargetPath string

	if strings.HasPrefix(path, ".") {
		absTargetPath = filepath.Clean(filepath.Join(baseDir, path))
		targetPath, _ = filepath.Rel(f.projectRootPath, absTargetPath)
	} else if strings.HasPrefix(path, "/") {
		targetPath = strings.TrimPrefix(path, "/")
		absTargetPath = filepath.Clean(filepath.Join(f.projectRootPath, targetPath))
	} else {
		targetPath = filepath.Clean(filepath.Join(f.projectRootPath, path))
		absTargetPath = filepath.Clean(filepath.Join(f.projectRootPath, targetPath))
	}
	targetPathKey := filepath.ToSlash(targetPath)
	f.loadCommonTypesFile(absTargetPath, targetPathKey)

	typeDef, found := f.commonTypes[targetPathKey][typeName]
	var typeKey string
	if targetPathKey == "" {
		typeKey = typeName
	} else {
		typeKey = targetPathKey + "/" + typeName
	}
	if found {
		if isArray {
			return []any{typeDef}, typeKey + "[]"
		}
		return typeDef, typeKey
	}

	return nil, ""
}

// searchCommonTypeInAncestors searches for a common type by traversing from basePath up to projectRootPath
func (f *FunctionDefinition) searchCommonTypeInAncestors(typeName string, isArray bool, startDir string) (any, string) {
	currentDir := startDir
	projectRootAbs, err := filepath.Abs(f.projectRootPath)
	if err != nil {
		return nil, ""
	}

	for {
		// Make currentDir absolute for comparison
		currentDirAbs, err := filepath.Abs(currentDir)
		if err != nil {
			break
		}

		// Calculate relative path from project root for the key
		targetPath, err := filepath.Rel(projectRootAbs, currentDirAbs)
		if err != nil {
			break
		}

		// Normalize path separators for consistent key format
		targetPathKey := filepath.ToSlash(targetPath)
		if targetPathKey == "." {
			targetPathKey = ""
		}

		// Try to load common types from this directory
		err = f.loadCommonTypesFile(currentDirAbs, targetPathKey)
		if err == nil {
			// Check if the type exists in this directory
			if typeMap, exists := f.commonTypes[targetPathKey]; exists {
				if typeDef, found := typeMap[typeName]; found {
					var typeKey string
					if targetPathKey == "" {
						typeKey = typeName
					} else {
						typeKey = targetPathKey + "/" + typeName
					}
					if isArray {
						return []any{typeDef}, typeKey + "[]"
					}
					return typeDef, typeKey
				}
			}
		}

		// Check if we've reached the project root
		if currentDirAbs == projectRootAbs {
			break
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}

	return nil, ""
}

// getStringFromMap safely extracts a string value from a map with a default fallback
func getStringFromMap(m map[string]any, key string, defaultValue string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}
