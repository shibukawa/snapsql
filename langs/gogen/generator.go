package gogen

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/shibukawa/snapsql/intermediate"
)

// Generator generates Go code from intermediate format
type Generator struct {
	PackageName       string
	OutputPath        string
	Format            *intermediate.IntermediateFormat
	MockPath          string
	Dialect           string                  // Target database dialect (postgres, mysql, sqlite)
	Hierarchy         *FileHierarchy          // File hierarchy information (optional)
	BaseImport        string                  // Base import path for hierarchical packages
	hierarchicalMetas []*hierarchicalNodeMeta // internal: prepared metas for hierarchical aggregation
}

// Option is a function that configures Generator
type Option func(*Generator)

// WithPackageName sets the package name for generated code
func WithPackageName(name string) Option {
	return func(g *Generator) {
		g.PackageName = name
	}
}

// WithDialect sets the target database dialect
func WithDialect(dialect string) Option {
	return func(g *Generator) {
		g.Dialect = dialect
	}
}

// WithOutputPath sets the output path for generated code
func WithOutputPath(path string) Option {
	return func(g *Generator) {
		g.OutputPath = path
	}
}

// WithMockPath sets the mock data path
func WithMockPath(path string) Option {
	return func(g *Generator) {
		g.MockPath = path
	}
}

// WithHierarchy sets the file hierarchy information
func WithHierarchy(hierarchy FileHierarchy) Option {
	return func(g *Generator) {
		g.Hierarchy = &hierarchy
		// Auto-adjust package name based on hierarchy
		if hierarchy.RelativeDir != "." && hierarchy.RelativeDir != "" {
			g.PackageName = GetPackageNameFromHierarchy(hierarchy, g.PackageName)
		}
	}
}

// WithBaseImport sets the base import path for hierarchical packages
func WithBaseImport(baseImport string) Option {
	return func(g *Generator) {
		g.BaseImport = baseImport
	}
}

// New creates a new Generator
func New(format *intermediate.IntermediateFormat, opts ...Option) *Generator {
	g := &Generator{
		PackageName: "generated", // Default package name
		Format:      format,
		Dialect:     "", // Must be specified via WithDialect or WithConfig
	}
	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Generate generates Go code and writes it to the writer
func (g *Generator) Generate(w io.Writer) error {
	// Reset per-file state to avoid leaking hierarchical metas across files
	g.hierarchicalMetas = nil

	// Process CEL environments
	celEnvs, err := processCELEnvironments(g.Format)
	if err != nil {
		return fmt.Errorf("failed to process CEL environments: %w", err)
	}

	// Generate CEL programs
	celPrograms, err := generateCELPrograms(g.Format, celEnvs)
	if err != nil {
		return fmt.Errorf("failed to generate CEL programs: %w", err)
	}

	// Process parameters
	parameters, structDefinitions, err := processParameters(g.Format.Parameters, g.Format.FunctionName)
	if err != nil {
		return fmt.Errorf("failed to process parameters: %w", err)
	}

	// Extract additional parameters from CEL environments
	additionalParams := extractCELParameters(g.Format.CELEnvironments, parameters)
	parameters = append(parameters, additionalParams...)

	// Process response type
	responseType, err := processResponseType(g.Format)
	if err != nil {
		return fmt.Errorf("failed to process response type: %w", err)
	}

	// Process response struct
	responseStruct, err := processResponseStruct(g.Format)
	if err != nil && !errors.Is(err, ErrNoResponseFields) {
		return fmt.Errorf("failed to process response struct: %w", err)
	}

	// Generate hierarchical structs if needed
	hierarchicalGroups, _, err := detectHierarchicalStructure(g.Format.Responses)
	if err != nil {
		return fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	if len(hierarchicalGroups) > 0 {
		hierarchicalStructs, _, err := generateHierarchicalStructs(g.Format.FunctionName, hierarchicalGroups, nil)
		if err != nil {
			return fmt.Errorf("failed to generate hierarchical structs: %w", err)
		}

		structDefinitions = append(structDefinitions, hierarchicalStructs...)

		// Build metadata for future hierarchical aggregation code generation (now injected into template for static expansion later)
		metas, metaErr := buildHierarchicalNodeMetas(g.Format.FunctionName, g.Format.Responses)
		if metaErr != nil {
			fmt.Printf("[warn] hierarchical meta build skipped: %v\n", metaErr)
		} else {
			// Attach metas to generator for template usage
			g.hierarchicalMetas = metas
		}
	}

	// Generate type registrations for custom types
	typeRegistrations, typeDefinitions := generateTypeRegistrations(g.Format, structDefinitions)

	// Process SQL builder
	sqlBuilder, err := processSQLBuilderWithDialect(g.Format, g.Dialect)
	if err != nil {
		return fmt.Errorf("failed to process SQL builder: %w", err)
	}

	// Process query execution
	queryExecution, err := generateQueryExecution(g.Format, responseStruct, g.hierarchicalMetas)
	if err != nil {
		return fmt.Errorf("failed to generate query execution: %w", err)
	}

	// Process implicit parameters (system columns)
	implicitParams, err := processImplicitParameters(g.Format)
	if err != nil {
		return fmt.Errorf("failed to process implicit parameters: %w", err)
	}

	// Convert function name to CamelCase
	funcName := snakeToCamel(g.Format.FunctionName)

	functionReturnType := fmt.Sprintf("(%s, error)", responseType)
	declareResult := true
	iteratorYieldType := ""

	if queryExecution.IsIterator && responseStruct != nil {
		functionReturnType = fmt.Sprintf("iter.Seq2[*%s, error]", responseStruct.Name)
		declareResult = false
		iteratorYieldType = queryExecution.IteratorYieldType
	}

	data := struct {
		Timestamp          time.Time
		PackageName        string
		FunctionName       string
		LowerFuncName      string
		Description        string
		MockPath           string
		CELEnvironments    []celEnvironmentData
		CELPrograms        []celProgramData
		Instructions       []instruction
		ResponseType       string
		FunctionReturnType string
		ResponseStruct     *responseStructData
		SQLBuilder         *sqlBuilderData
		QueryExecution     *queryExecutionData
		Parameters         []parameterData
		StructDefinitions  []string
		TypeRegistrations  []string
		TypeDefinitions    map[string]map[string]string
		ImplicitParams     []implicitParam
		Imports            map[string]struct{}
		ImportSlice        []string
		NumCELEnvs         int
		NumCELPrograms     int
		HierarchicalMetas  []*hierarchicalNodeMeta
		IteratorYieldType  string
		DeclareResult      bool
	}{
		Timestamp:          time.Now(),
		PackageName:        g.PackageName,
		FunctionName:       funcName,
		LowerFuncName:      strings.ToLower(funcName),
		Description:        g.Format.Description,
		MockPath:           g.MockPath,
		CELEnvironments:    celEnvs,
		CELPrograms:        celPrograms,
		Parameters:         parameters,
		ResponseType:       responseType,
		ResponseStruct:     responseStruct,
		SQLBuilder:         sqlBuilder,
		QueryExecution:     queryExecution,
		StructDefinitions:  structDefinitions,
		TypeRegistrations:  typeRegistrations,
		TypeDefinitions:    typeDefinitions,
		ImplicitParams:     implicitParams,
		NumCELEnvs:         len(g.Format.CELEnvironments),
		NumCELPrograms:     len(g.Format.CELExpressions),
		Imports:            make(map[string]struct{}),
		HierarchicalMetas:  g.hierarchicalMetas,
		FunctionReturnType: functionReturnType,
		IteratorYieldType:  iteratorYieldType,
		DeclareResult:      declareResult,
	}

	if queryExecution.IsIterator && responseStruct != nil {
		data.Imports["iter"] = struct{}{}
	}

	// Collect imports from all environments
	for _, env := range celEnvs {
		for imp := range env.Imports {
			data.Imports[imp] = struct{}{}
		}
	}

	// Add time import if any implicit parameter uses time.Now() as default
	for _, param := range implicitParams {
		if param.DefaultValueLiteral == "time.Now()" {
			data.Imports["time"] = struct{}{}
			break
		}
	}

	// Add time import if any struct field uses time.Time
	if data.ResponseStruct != nil {
		for _, f := range data.ResponseStruct.Fields {
			if strings.Contains(f.Type, "time.Time") {
				data.Imports["time"] = struct{}{}
				break
			}
		}
	}

	// Add time/decimal imports if appear in struct definitions
	for _, def := range structDefinitions {
		if strings.Contains(def, "time.Time") {
			data.Imports["time"] = struct{}{}
		}

		if strings.Contains(def, "decimal.Decimal") {
			data.Imports["github.com/shopspring/decimal"] = struct{}{}
		}
	}

	// Convert imports map to slice for template
	var importSlice []string
	for imp := range data.Imports {
		importSlice = append(importSlice, imp)
	}

	sort.Strings(importSlice)

	data.ImportSlice = importSlice

	// Execute template
	tmpl, err := template.New("go").Funcs(template.FuncMap{
		"toLower":  strings.ToLower,
		"backtick": func() string { return "`" },
		"title":    cases.Title(language.English).String,
		"needStringsImport": func(isStatic bool, metas []*hierarchicalNodeMeta) bool {
			// strings is only necessary for dynamic SQL builder (non-static).
			// Hierarchical metas do not require strings import on their own.
			return !isStatic
		},
		"isSystemColumn": func(paramName string) bool {
			systemColumns := []string{"created_at", "updated_at", "created_by", "updated_by", "version"}
			for _, col := range systemColumns {
				if paramName == col {
					return true
				}
			}

			return false
		},
		"hasAnySystemParam": func(names []string) bool {
			systemColumns := map[string]struct{}{"created_at": {}, "updated_at": {}, "created_by": {}, "updated_by": {}, "version": {}}
			for _, n := range names {
				if _, ok := systemColumns[n]; ok {
					return true
				}
			}

			return false
		},
		"celTypeConvert": func(typeName string) string {
			// Handle array types
			if strings.HasPrefix(typeName, "[]") {
				elementType := strings.TrimPrefix(typeName, "[]")
				// Drop pointer for element types in CEL object representation
				if strings.HasPrefix(elementType, "*") {
					elementType = strings.TrimPrefix(elementType, "*")
				}

				elementCELType := convertSingleType(elementType)

				return fmt.Sprintf("types.NewListType(%s)", elementCELType)
			}

			// Handle pointer types
			if strings.HasPrefix(typeName, "*") {
				baseType := strings.TrimPrefix(typeName, "*")
				// For nullable types, we still use the base type in CEL
				return convertSingleType(baseType)
			}

			return convertSingleType(typeName)
		},
		"convertSingleType": func(typeName string) string {
			switch typeName {
			case "int":
				return "types.IntType"
			case "string":
				return "types.StringType"
			case "bool":
				return "types.BoolType"
			case "double":
				return "types.DoubleType"
			case "decimal":
				return "snapsqlgo.DecimalType"
			case "time.Time":
				return "types.TimestampType"
			case "any":
				return "types.AnyType"
			default:
				// Custom struct type
				return fmt.Sprintf("types.NewObjectType(\"%s\")", typeName)
			}
		},
		// celNameToGoName はテンプレート内で Raw なフィールド名 (snake_case) を単回変換するときのみ使用。
		// responseStruct.Fields には既に PascalCase 済み Name が入っているため再適用しないこと。
		"celNameToGoName": func(celName string) string {
			if strings.Contains(celName, "__") { // 階層用は末端のみ変換
				segs := strings.Split(celName, "__")
				last := segs[len(segs)-1]
				celName = last
			}

			parts := strings.Split(celName, "_")
			caser := cases.Title(language.English)

			for i, part := range parts {
				if part == "id" {
					parts[i] = "ID"
					continue
				}

				if part == "url" {
					parts[i] = "URL"
					continue
				}

				parts[i] = caser.String(part)
			}

			return strings.Join(parts, "")
		},
	}).Parse(goTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder

	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	_, err = fmt.Fprint(w, buf.String())

	return err
}

// snakeToCamel converts a snake_case string to CamelCase
func snakeToCamel(s string) string {
	// If the string doesn't contain underscores, it might already be camelCase
	if !strings.Contains(s, "_") {
		// Convert first letter to uppercase for PascalCase
		if len(s) == 0 {
			return s
		}

		return strings.ToUpper(string(s[0])) + s[1:]
	}

	words := strings.Split(s, "_")
	for i := range words {
		words[i] = capitalizeWord(words[i])
	}

	return strings.Join(words, "")
}

// capitalizeWord capitalizes a word with special handling for common abbreviations
func capitalizeWord(word string) string {
	lower := strings.ToLower(word)
	switch lower {
	case "id":
		return "ID"
	case "url":
		return "URL"
	case "http":
		return "HTTP"
	case "api":
		return "API"
	case "json":
		return "JSON"
	case "xml":
		return "XML"
	case "sql":
		return "SQL"
	case "db":
		return "DB"
	default:
		if len(word) == 0 {
			return ""
		}

		return strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
	}
}

// snakeToCamelLower converts a snake_case string to camelCase (first letter lowercase)
func snakeToCamelLower(s string) string {
	var result strings.Builder

	capitalize := false

	for i, r := range s {
		if r == '_' {
			capitalize = true
			continue
		}

		if capitalize {
			// Special handling for "id" at the end of the string
			if i+2 == len(s) && strings.ToLower(s[i:]) == "id" {
				result.WriteString("ID")
				break
			}

			result.WriteString(strings.ToUpper(string(r)))

			capitalize = false
		} else {
			result.WriteString(strings.ToLower(string(r)))
		}
	}

	return result.String()
}

// processParameters converts intermediate parameters to Go parameter data
func processParameters(params []intermediate.Parameter, funcName string) ([]parameterData, []string, error) {
	result := make([]parameterData, len(params))

	var structDefinitions []string

	for i, param := range params {
		// Special handling for complex types based on function name and parameter name
		if funcName == "insert_all_sub_departments" && param.Name == "departments" {
			// Generate specific struct types for nested departments
			structDefs := []string{
				`type InsertAllSubDepartmentsSubDepartment struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}`,
				`type InsertAllSubDepartmentsDepartment struct {
	DepartmentName string                                      ` + "`json:\"department_name\"`" + `
	DepartmentCode string                                      ` + "`json:\"department_code\"`" + `
	SubDepartments []InsertAllSubDepartmentsSubDepartment     ` + "`json:\"sub_departments\"`" + `
}`,
			}
			structDefinitions = append(structDefinitions, structDefs...)

			result[i] = parameterData{
				Name:     snakeToCamelLower(param.Name),
				Type:     "[]InsertAllSubDepartmentsDepartment",
				Required: !param.Optional,
			}

			continue
		}

		// Default type conversion
		goType, err := convertToGoType(param.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert parameter %s type: %w", param.Name, err)
		}

		result[i] = parameterData{
			Name:     snakeToCamelLower(param.Name),
			Type:     goType,
			Required: !param.Optional,
		}
	}

	return result, structDefinitions, nil
}

// convertToGoType converts SnapSQL type to Go type
func convertToGoType(snapType string) (string, error) {
	// Handle arrays
	if strings.HasSuffix(snapType, "[]") {
		baseType := strings.TrimSuffix(snapType, "[]")

		goBaseType, err := convertToGoType(baseType)
		if err != nil {
			return "", err
		}

		return "[]" + goBaseType, nil
	}

	// Handle pointers
	if strings.HasSuffix(snapType, "*") {
		baseType := strings.TrimSuffix(snapType, "*")

		goBaseType, err := convertToGoType(baseType)
		if err != nil {
			return "", err
		}

		return "*" + goBaseType, nil
	}

	// Handle custom types (relative paths)
	if strings.HasPrefix(snapType, "../") || strings.HasPrefix(snapType, "./") {
		// Extract the type name from the path
		parts := strings.Split(snapType, "/")
		typeName := parts[len(parts)-1]

		return typeName, nil
	}

	// Handle basic types
	switch strings.ToLower(snapType) {
	case "int", "int32", "int64":
		return snapType, nil
	case "string":
		return "string", nil
	case "bool":
		return "bool", nil
	case "float", "float32", "float64":
		// Normalize all float variants to Go's float64
		return "float64", nil
	case "decimal":
		return "decimal.Decimal", nil
	case "*decimal.decimal":
		return "*decimal.Decimal", nil
	case "timestamp", "date", "time", "time.time":
		return "time.Time", nil
	case "datetime":
		return "time.Time", nil
	case "*time.time":
		return "*time.Time", nil
	case "bytes":
		return "[]byte", nil
	case "any":
		return "interface{}", nil
	default:
		// Handle custom types (valid Go type names)
		if isValidGoTypeName(snapType) {
			return snapType, nil
		}

		return "", newUnsupportedTypeError(snapType, "parameter")
	}
}

// processResponseType determines the response type based on response affinity and responses
func processResponseType(format *intermediate.IntermediateFormat) (string, error) {
	if len(format.Responses) == 0 {
		// No response fields -> plain write without RETURNING
		return "sql.Result", nil
	}

	// Rely solely on pipeline-determined ResponseAffinity (generator no longer mutates it)
	hierarchicalGroups, _, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return "", fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	structName := generateStructName(format.FunctionName)
	if len(hierarchicalGroups) > 0 {
		switch format.ResponseAffinity {
		case "one":
			return structName, nil
		case "many":
			return "[]" + structName, nil
		default:
			return structName, nil
		}
	}

	switch format.ResponseAffinity {
	case "one":
		return structName, nil
	case "many":
		return "[]" + structName, nil
	case "none":
		return "interface{}", nil
	default:
		return structName, nil
	}
}

// hasReturningClause performs a lightweight detection of RETURNING in the SQL build instructions.
// It checks emitted static fragments for the keyword. This mirrors logic in query_execution.go.
// hasReturningClause removed: pipeline decides affinity; local heuristic deleted.

// generateStructName generates a struct name from function name
func generateStructName(functionName string) string {
	// Convert function name to struct name
	// e.g., "get_user_by_id" -> "GetUserByIDResult"
	// e.g., "find_user" -> "FindUserResult"
	// e.g., "getFilteredData" -> "GetFilteredDataResult"
	camelName := snakeToCamel(functionName)

	// Add "Result" suffix if it doesn't already end with a noun-like suffix
	if !strings.HasSuffix(camelName, "Result") &&
		!strings.HasSuffix(camelName, "Response") &&
		!strings.HasSuffix(camelName, "Data") {
		return camelName + "Result"
	}

	// If it ends with "Data", keep it as is
	return camelName + "Result"
}

// responseStructData represents a response struct for code generation
type responseStructData struct {
	Name   string
	Fields []responseFieldData
	// RawResponses keeps original intermediate.Response slice for advanced generation (hierarchical, PK, etc.)
	RawResponses []intermediate.Response
}

// responseFieldData represents a field in a response struct
type responseFieldData struct {
	Name      string
	Type      string
	JSONTag   string
	IsPointer bool
}

// ErrNoResponseFields indicates that there are no response fields
var ErrNoResponseFields = errors.New("no response fields")

// processResponseStruct processes response fields and generates struct data
func processResponseStruct(format *intermediate.IntermediateFormat) (*responseStructData, error) {
	if len(format.Responses) == 0 {
		// No response fields - this is normal for INSERT/UPDATE/DELETE statements
		return nil, ErrNoResponseFields
	}

	// Check for hierarchical structure
	hierarchicalGroups, rootFields, err := detectHierarchicalStructure(format.Responses)
	if err != nil {
		return nil, fmt.Errorf("failed to detect hierarchical structure: %w", err)
	}

	if len(hierarchicalGroups) > 0 {
		// This is a hierarchical response - use hierarchical processing
		_, mainStruct, err := generateHierarchicalStructs(format.FunctionName, hierarchicalGroups, rootFields)
		if err != nil {
			return nil, fmt.Errorf("failed to generate hierarchical structs: %w", err)
		}

		mainStruct.RawResponses = format.Responses

		return mainStruct, nil
	}

	// Regular flat structure
	structName := generateStructName(format.FunctionName)

	fields := make([]responseFieldData, len(format.Responses))

	for i, response := range format.Responses {
		goType, err := convertToGoType(response.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to convert response field %s type: %w", response.Name, err)
		}

		forceNonNullable := response.HierarchyKeyLevel == 1 && !strings.Contains(response.Name, "__")

		// Handle nullable fields
		isPointer := response.IsNullable && !forceNonNullable
		if isPointer && !strings.HasPrefix(goType, "*") {
			goType = "*" + goType
		}

		if forceNonNullable && strings.HasPrefix(goType, "*") {
			goType = strings.TrimPrefix(goType, "*")
		}

		fields[i] = responseFieldData{
			Name:      celNameToGoName(response.Name), // 一度だけ変換
			Type:      goType,
			JSONTag:   response.Name,
			IsPointer: isPointer,
		}
	}

	return &responseStructData{
		Name:         structName,
		Fields:       fields,
		RawResponses: format.Responses,
	}, nil
}

type instruction struct {
	Op    string
	Value string
	Index int
}

type parameterData struct {
	Name     string
	Type     string
	Required bool
}

type parameter struct {
	Name     string
	Type     string
	Required bool
}

type implicitParam struct {
	Name                string
	Type                string
	Required            bool
	Default             any
	DefaultValueLiteral string
}

// UnsupportedTypeError represents an error for unsupported types with helpful hints.
type UnsupportedTypeError struct {
	Type    string
	Context string
	Message string
	Hints   []string
}

func (e *UnsupportedTypeError) Error() string {
	msg := e.Message
	if len(e.Hints) > 0 {
		msg += "\n\nHint: " + e.Hints[0]
		if len(e.Hints) > 1 {
			msg += "\nFor more information, run with --help-types flag"
		}
	}

	return msg
}

// newUnsupportedTypeError creates a new UnsupportedTypeError with context-appropriate hints
func newUnsupportedTypeError(typeName, context string) *UnsupportedTypeError {
	err := &UnsupportedTypeError{
		Type:    typeName,
		Context: context,
		Message: fmt.Sprintf("unsupported %s type '%s'", context, typeName),
	}

	// Add context-specific hints
	switch {
	case context == "parameter":
		err.Hints = []string{
			"Basic types: int, string, bool, float, decimal, timestamp, date, time, bytes, any",
			"Arrays: string[], int[], etc.",
			"Pointers: *string, *int, etc.",
			"Custom types: MyType, time.Time, ./CustomType",
		}
	case strings.Contains(context, "implicit parameter"):
		err.Hints = []string{
			"System column types: int, string, bool, timestamp, decimal",
			"Arrays: int[], string[], etc.",
		}
	case context == "type":
		err.Hints = []string{
			"Supported types: int, string, bool, float, double, decimal, timestamp, datetime, date, any",
			"Arrays: type[], custom Go types",
		}
	default:
		err.Hints = []string{
			"Check the documentation for supported type formats",
		}
	}

	return err
}

// isValidGoTypeName checks if a type name follows Go naming conventions
func isValidGoTypeName(typeName string) bool {
	if typeName == "" {
		return false
	}

	// Check for package qualified types (e.g., "time.Time", "decimal.Decimal")
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) != 2 {
			return false
		}
		// Both package and type name should be valid identifiers
		return isValidGoIdentifier(parts[0]) && isValidGoIdentifier(parts[1])
	}

	// Check for simple type names
	return isValidGoIdentifier(typeName)
}

// isValidGoIdentifier checks if a string is a valid Go identifier
func isValidGoIdentifier(name string) bool {
	if name == "" {
		return false
	}

	// First character must be a letter or underscore
	first := rune(name[0])
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for _, r := range name[1:] {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}

	return true
}

func convertTypeToGo(typeName string) (string, error) {
	switch typeName {
	case "int":
		return "int", nil
	case "string":
		return "string", nil
	case "bool":
		return "bool", nil
	case "float", "double":
		return "float64", nil
	case "decimal":
		return "decimal.Decimal", nil
	case "timestamp", "datetime":
		return "time.Time", nil
	case "date":
		return "time.Time", nil
	case "any":
		return "interface{}", nil
	default:
		if strings.HasSuffix(typeName, "[]") {
			elementType := strings.TrimSuffix(typeName, "[]")

			goElementType, err := convertTypeToGo(elementType)
			if err != nil {
				return "", err
			}

			return "[]" + goElementType, nil
		}
		// For custom types, we assume they are valid Go types
		// but we should validate that they follow Go naming conventions
		if isValidGoTypeName(typeName) {
			return typeName, nil
		}

		return "", newUnsupportedTypeError(typeName, "type")
	}
}

// processImplicitParameters processes implicit parameters from intermediate format
func processImplicitParameters(format *intermediate.IntermediateFormat) ([]implicitParam, error) {
	var implicitParams []implicitParam

	for _, param := range format.ImplicitParameters {
		ptype := param.Type
		if ptype == "" {
			// Fallback by convention for common system fields when type is missing
			switch param.Name {
			case "created_at", "updated_at":
				ptype = "timestamp"
			case "created_by", "updated_by":
				ptype = "string"
			default:
				ptype = "any"
			}
		}

		goType, err := convertTypeToGo(ptype)
		if err != nil {
			return nil, newUnsupportedTypeError(ptype, fmt.Sprintf("implicit parameter '%s'", param.Name))
		}

		// Determine if parameter is required (no default value and not nullable)
		required := param.Default == nil && !isNullableType(goType)

		// Generate default value literal for Go code
		var defaultValueLiteral string
		if param.Default != nil {
			defaultValueLiteral, err = generateDefaultValueLiteral(param.Default, goType)
			if err != nil {
				return nil, fmt.Errorf("failed to generate default value literal for %s: %w", param.Name, err)
			}
		}

		implicitParams = append(implicitParams, implicitParam{
			Name:                param.Name,
			Type:                goType,
			Required:            required,
			Default:             param.Default,
			DefaultValueLiteral: defaultValueLiteral,
		})
	}

	return implicitParams, nil
}

// generateDefaultValueLiteral generates Go code literal for default values
func generateDefaultValueLiteral(defaultValue any, goType string) (string, error) {
	switch v := defaultValue.(type) {
	case string:
		if v == "NOW()" {
			// For NOW() function, generate time.Now() call
			return "time.Now()", nil
		}
		// For other string values, quote them
		return fmt.Sprintf("%q", v), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return fmt.Sprintf("%g", v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "nil", nil
	default:
		// For complex types, use fmt.Sprintf with %v
		return fmt.Sprintf("%v", v), nil
	}
}

// isNullableType checks if a Go type is nullable (pointer type)
func isNullableType(goType string) bool {
	return strings.HasPrefix(goType, "*")
}

const goTemplate = `//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by snapsql. DO NOT EDIT.

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

package {{ .PackageName }}

import (
	"context"
	"fmt"
	{{- /* strings is needed only for dynamic SQL builder outputs or when join operations are emitted */}}
	{{- if not .SQLBuilder.IsStatic }}
	"strings"
	{{- end }}
	{{- /* database/sql only when used in response type */}}
	{{- if eq .ResponseType "sql.Result" }}
	"database/sql"
	{{- end }}
	{{- /* bring in snapsql root when hierarchical aggregation path or query execution requires it */}}
	{{- if or (gt (len .HierarchicalMetas) 0) (.QueryExecution.NeedsSnapsqlImport) }}
	"github.com/shibukawa/snapsql"
	{{- end }}
	{{- range .ImportSlice }}
	"{{ . }}"
	{{- end }}

	"github.com/google/cel-go/cel"
	{{- /* types/ref are needed when type definitions exist or CreateCELOptionsWithTypes is used */}}
	{{- if .TypeDefinitions }}
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	{{- end }}
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)
{{- range .StructDefinitions }}
{{ . }}

{{- end }}
{{- if .ResponseStruct }}
// {{ .ResponseStruct.Name }} represents the response structure for {{ .FunctionName }}
type {{ .ResponseStruct.Name }} struct {
	{{- range .ResponseStruct.Fields }}
	{{ .Name }} {{ .Type }} {{backtick}}json:"{{ .JSONTag }}"{{backtick}}
	{{- end }}
}
{{- end }}

// {{ .FunctionName }} specific CEL programs and mock path
var (
	{{ .LowerFuncName }}Programs []cel.Program
)

const {{ .LowerFuncName }}MockPath = "{{ .MockPath }}"

func init() {
	{{- if .TypeDefinitions }}
	// Static accessor functions for each type
	{{- range $typeName, $fields := .TypeDefinitions }}
	{{- range $fieldName, $fieldType := $fields }}
	{{ $typeName | toLower }}{{ $fieldName | celNameToGoName }}Accessor := func(value interface{}) ref.Val {
		v := value.(*{{ $typeName }})
		return snapsqlgo.ConvertGoValueToCEL(v.{{ $fieldName | celNameToGoName }})
	}
	{{- end }}
	{{- end }}

	// Create type definitions for local type store
	typeDefinitions := map[string]map[string]snapsqlgo.FieldInfo{
		{{- range $typeName, $fields := .TypeDefinitions }}
		"{{ $typeName }}": {
			{{- range $fieldName, $fieldType := $fields }}
			"{{ $fieldName }}": snapsqlgo.CreateFieldInfo(
				"{{ $fieldName }}", 
				{{ $fieldType | celTypeConvert }}, 
				{{ $typeName | toLower }}{{ $fieldName | celNameToGoName }}Accessor,
			),
			{{- end }}
		},
		{{- end }}
	}

	// Create and set up local type store
	registry := snapsqlgo.NewLocalTypeRegistry()
	for typeName, fields := range typeDefinitions {
		registry.RegisterStructWithFields(typeName, fields)
	}
    
	// Set global registry for nested type resolution
	snapsqlgo.SetGlobalRegistry(registry)
	{{- end }}

	// CEL environments based on intermediate format
	celEnvironments := make([]*cel.Env, {{ .NumCELEnvs }})
	
	{{- range .CELEnvironments }}
	// Environment {{ .Index }}: Base environment
	{
		// Build CEL env options then expand variadic at call-site to avoid type inference issues
		opts := []cel.EnvOption{
			cel.HomogeneousAggregateLiterals(),
			cel.EagerlyValidateDeclarations(true),
			snapsqlgo.DecimalLibrary,
			{{- range .Variables }}
			cel.Variable("{{ .Name }}", cel.{{ .CelType }}),
			{{- end }}
		}
		{{- if $.TypeDefinitions }}
		opts = append(opts, snapsqlgo.CreateCELOptionsWithTypes(typeDefinitions)...)
		{{- end }}
		env{{ .Index }}, err := cel.NewEnv(opts...)
		if err != nil {
			panic(fmt.Sprintf("failed to create {{ $.FunctionName }} CEL environment {{ .Index }}: %v", err))
		}
		celEnvironments[{{ .Index }}] = env{{ .Index }}
	}
	{{- end }}

	// Create programs for each expression using the corresponding environment
	{{ .LowerFuncName }}Programs = make([]cel.Program, {{ .NumCELPrograms }})
	
	{{- range .CELPrograms }}
	// {{ .ID }}: "{{ .Expression }}" using environment {{ .EnvironmentIdx }}
	{
		ast, issues := celEnvironments[{{ .EnvironmentIdx }}].Compile("{{ .Expression }}")
		if issues != nil && issues.Err() != nil {
			panic(fmt.Sprintf("failed to compile CEL expression '{{ .Expression }}': %v", issues.Err()))
		}
		program, err := celEnvironments[{{ .EnvironmentIdx }}].Program(ast)
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL program for '{{ .Expression }}': %v", err))
		}
		{{ $.LowerFuncName }}Programs[{{ .Index }}] = program
	}
	{{- end }}
}

{{- if .Description }}
// {{ .FunctionName }} {{ .Description }}
{{- else }}
// {{ .FunctionName }} - {{ .ResponseType }} Affinity
{{- end }}
func {{ .FunctionName }}(ctx context.Context, executor snapsqlgo.DBExecutor{{- range .Parameters }}, {{ .Name }} {{ .Type }}{{- end }}, opts ...snapsqlgo.FuncOpt) {{ .FunctionReturnType }} {
	{{- if .DeclareResult }}
	var result {{ .ResponseType }}

	// Hierarchical metas (for nested aggregation code generation - placeholder)
	// Count: {{ if .HierarchicalMetas }}{{ len .HierarchicalMetas }}{{ else }}0{{ end }}
	{{- end }}

	funcConfig := snapsqlgo.GetFunctionConfig(ctx, "{{ .LowerFuncName }}", "{{ .ResponseType | toLower }}")

	{{- if not .QueryExecution.IsIterator }}
	// Check for mock mode
	if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
		mockData, err := snapsqlgo.GetMockDataFromFiles({{ .LowerFuncName }}MockPath, funcConfig.MockDataNames)
		if err != nil {
			return result, fmt.Errorf("failed to get mock data: %w", err)
		}

		result, err = snapsqlgo.MapMockDataToStruct[{{ .ResponseType }}](mockData)
		if err != nil {
			return result, fmt.Errorf("failed to map mock data to {{ .ResponseType }} struct: %w", err)
		}

		return result, nil
	}
	{{- end }}

	{{- if and .ImplicitParams (hasAnySystemParam .SQLBuilder.ParameterNames) }}
	// Extract implicit parameters
	implicitSpecs := []snapsqlgo.ImplicitParamSpec{
		{{- range .ImplicitParams }}
		{Name: "{{ .Name }}", Type: "{{ .Type }}", Required: {{ .Required }}{{ if .Default }}, DefaultValue: {{ .DefaultValueLiteral }}{{ end }}},
		{{- end }}
	}
	systemValues := snapsqlgo.ExtractImplicitParams(ctx, implicitSpecs)
	_ = systemValues // avoid unused if not referenced in args
	{{- end }}

	// Build SQL
	{{- if .SQLBuilder.IsStatic }}
	query := {{ printf "%q" .SQLBuilder.StaticSQL }}
	{{- if or .SQLBuilder.HasArguments .ImplicitParams }}
	args := []any{
		{{- range .SQLBuilder.ParameterNames }}
		{{- if isSystemColumn . }}
		systemValues["{{ . }}"],
		{{- else }}
		{{ . }},
		{{- end }}
		{{- end }}
	}
	{{- else }}
	args := []any{}
	{{- end }}
	{{- else }}
	var builder strings.Builder
	args := make([]any, 0)
	
	{{- range .SQLBuilder.BuilderCode }}
	{{ . }}
	{{- end }}
	
	query := builder.String()
	{{- end }}

{{- if .QueryExecution.IsIterator }}
	return func(yield func({{ .IteratorYieldType }}, error) bool) {
		if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
			mockData, err := snapsqlgo.GetMockDataFromFiles({{ .LowerFuncName }}MockPath, funcConfig.MockDataNames)
			if err != nil {
				_ = yield(nil, fmt.Errorf("failed to get mock data: %w", err))
				return
			}

			rows, err := snapsqlgo.MapMockDataToStruct[{{ .ResponseType }}](mockData)
			if err != nil {
				_ = yield(nil, fmt.Errorf("failed to map mock data to {{ .ResponseType }} struct: %w", err))
				return
			}

			for i := range rows {
				item := rows[i]
				if !yield(&item, nil) {
					return
				}
			}

			return
		}

		{{- range .QueryExecution.IteratorBody }}
		{{ . }}
		{{- end }}
	}
		{{- else }}
		// Execute query
		stmt, err := executor.PrepareContext(ctx, query)
		if err != nil {
			return result, fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		{{- range .QueryExecution.Code }}
		{{ . }}
		{{- end }}

		return result, nil
		{{- end }}
}
`

// Helper function to convert snake_case to PascalCase for Go field names
// parseStructDefinition parses a Go struct definition and extracts type name and fields
func parseStructDefinition(structDef string) (string, map[string]string) {
	lines := strings.Split(structDef, "\n")
	fields := make(map[string]string)

	var typeName string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract type name from "type TypeName struct {"
		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				typeName = parts[1]
			}

			continue
		}

		// Skip empty lines and braces
		if line == "" || line == "{" || line == "}" {
			continue
		}

		// Parse field definition: "FieldName FieldType `json:\"field_name\"`"
		if strings.Contains(line, "`json:") {
			// Extract field name and type
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				_ = parts[0] // fieldName (not used in current implementation)
				fieldType := parts[1]

				// Extract JSON tag name
				jsonTagStart := strings.Index(line, "`json:\"")
				if jsonTagStart != -1 {
					jsonTagStart += 7 // len("`json:\"")

					jsonTagEnd := strings.Index(line[jsonTagStart:], "\"")
					if jsonTagEnd != -1 {
						jsonFieldName := line[jsonTagStart : jsonTagStart+jsonTagEnd]

						// Convert Go type to CEL type
						celType := goTypeToCELType(fieldType)
						fields[jsonFieldName] = celType
					}
				}
			}
		}
	}

	return typeName, fields
}

// goTypeToCELType converts Go types to CEL type names
func goTypeToCELType(goType string) string {
	// Handle array/slice types
	if strings.HasPrefix(goType, "[]") {
		elementType := strings.TrimPrefix(goType, "[]")
		elementCELType := goTypeToCELType(elementType)

		return "[]" + elementCELType
	}

	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		baseType := strings.TrimPrefix(goType, "*")
		return "*" + goTypeToCELType(baseType)
	}

	// Handle basic types
	switch goType {
	case "int", "int32", "int64":
		return "int"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "float32", "float64":
		return "double"
	case "time.Time":
		return "time.Time"
	case "decimal.Decimal":
		return "decimal"
	default:
		// Assume it's a custom struct type
		return goType
	}
}

// snapTypeToCELType converts SnapSQL types to CEL type names
func snapTypeToCELType(snapType string) string {
	// Handle array types
	if strings.HasPrefix(snapType, "[]") {
		elementType := strings.TrimPrefix(snapType, "[]")
		elementCELType := snapTypeToCELType(elementType)

		return "[]" + elementCELType
	}

	// Handle pointer types
	if strings.HasPrefix(snapType, "*") {
		baseType := strings.TrimPrefix(snapType, "*")
		return "*" + snapTypeToCELType(baseType)
	}

	// Handle basic types
	switch strings.ToLower(snapType) {
	case "int", "int32", "int64":
		return "int"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "float", "float32", "float64", "double":
		return "double"
	case "timestamp", "date", "time", "time.time":
		return "time.Time"
	case "*time.time":
		return "*time.Time"
	case "decimal":
		return "decimal"
	case "*decimal.decimal":
		return "*decimal"
	case "any", "interface{}":
		return "any"
	default:
		// Assume it's a custom struct type
		return snapType
	}
}

func generateTypeRegistrations(format *intermediate.IntermediateFormat, structDefinitions []string) ([]string, map[string]map[string]string) {
	var registrations []string

	typeDefinitions := make(map[string]map[string]string)

	// Parse struct definitions to extract nested types
	for _, structDef := range structDefinitions {
		typeName, fields := parseStructDefinition(structDef)
		if typeName != "" {
			registrations = append(registrations, fmt.Sprintf("// Register %s type", typeName))
			typeDefinitions[typeName] = fields
		}
	}

	// Add response struct if it exists
	if len(format.Responses) > 0 {
		// For now, skip response struct processing as it's handled differently
		// This will be implemented when we have proper response struct definitions
	}

	return registrations, typeDefinitions
}

// extractCELParameters extracts additional parameters from CEL environments
func extractCELParameters(celEnvs []intermediate.CELEnvironment, existingParams []parameterData) []parameterData {
	var additionalParams []parameterData

	existingNames := make(map[string]bool)

	// Create map of existing parameter names
	for _, param := range existingParams {
		existingNames[param.Name] = true
	}

	// Extract variables from CEL environments
	for _, env := range celEnvs {
		for _, variable := range env.AdditionalVariables {
			// Convert snake_case to camelCase for Go parameter name
			paramName := snakeToCamelLower(variable.Name)

			// Skip if parameter already exists
			if existingNames[paramName] {
				continue
			}

			// Convert type
			goType, err := convertToGoType(variable.Type)
			if err != nil {
				// Skip unsupported types
				continue
			}

			additionalParams = append(additionalParams, parameterData{
				Name: paramName,
				Type: goType,
			})

			existingNames[paramName] = true
		}
	}

	return additionalParams
}

// convertSingleType converts a single type name to CEL type
func convertSingleType(typeName string) string {
	switch typeName {
	case "int":
		return "types.IntType"
	case "string":
		return "types.StringType"
	case "bool":
		return "types.BoolType"
	case "double":
		return "types.DoubleType"
	case "decimal":
		return "snapsqlgo.DecimalType"
	case "time.Time":
		return "types.TimestampType"
	case "any":
		return "types.AnyType"
	default:
		// Custom struct type
		return fmt.Sprintf("types.NewObjectType(\"%s\")", typeName)
	}
}
