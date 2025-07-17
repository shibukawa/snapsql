package typeinference

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Type aliases for parser types
type (
	StatementNode       = parser.StatementNode
	SelectStatement     = parser.SelectStatement
	InsertIntoStatement = parser.InsertIntoStatement
	UpdateStatement     = parser.UpdateStatement
	DeleteFromStatement = parser.DeleteFromStatement
	SelectField         = parser.SelectField
)

// TypeInfo represents inferred type information
type TypeInfo struct {
	BaseType   string // Normalized type (string, int, float, decimal, bool, timestamp, date, time, json, any)
	IsNullable bool   // Whether the field can be null
	MaxLength  *int   // For string types
	Precision  *int   // For numeric types
	Scale      *int   // For numeric types
}

// FieldSource represents the source of a field
type FieldSource struct {
	Type         string // "column", "expression", "function", "literal", "subquery", "case", "cast"
	Table        string // Table name for column sources
	Column       string // Column name for column sources
	Expression   string // Expression text for expression sources
	FunctionName string // Function name for function sources
}

// InferredFieldInfo represents a field with inferred type information
type InferredFieldInfo struct {
	Name         string      // Field name (auto-generated if necessary)
	OriginalName string      // Original field name (may be empty)
	Alias        string      // Field alias
	Type         *TypeInfo   // Inferred type information
	Source       FieldSource // Field source information
	IsGenerated  bool        // Whether the name was auto-generated
	CastType     string      // Explicit type from CAST expression
}

// InferenceContext provides context for type inference
type InferenceContext struct {
	Dialect       snapsql.Dialect   // Database dialect
	TableAliases  map[string]string // Table alias mappings
	CurrentTables []string          // Currently available tables
	SubqueryDepth int               // Current subquery nesting depth
}

// FieldNameGenerator handles automatic field name generation
type FieldNameGenerator struct {
	usedNames     map[string]int    // Used names and their sequence numbers
	functionNames map[string]string // Function name mappings
}

// NewFieldNameGenerator creates a new field name generator
func NewFieldNameGenerator() *FieldNameGenerator {
	return &FieldNameGenerator{
		usedNames:     make(map[string]int),
		functionNames: functionFieldNameMappings(),
	}
}

// GenerateFieldName generates a unique field name
func (g *FieldNameGenerator) GenerateFieldName(originalName, functionName, expressionType string) string {
	var baseName string

	// Determine base name
	if originalName != "" {
		baseName = originalName
	} else if functionName != "" {
		if mapped, ok := g.functionNames[functionName]; ok {
			baseName = mapped
		} else {
			baseName = functionName
		}
	} else if expressionType != "" {
		baseName = expressionType
	} else {
		baseName = "field"
	}

	// Make name unique
	if count, exists := g.usedNames[baseName]; exists {
		g.usedNames[baseName] = count + 1
		return fmt.Sprintf("%s%d", baseName, count+1)
	} else {
		g.usedNames[baseName] = 1
		return baseName
	}
}

// ReserveFieldName reserves a field name to prevent conflicts
func (g *FieldNameGenerator) ReserveFieldName(name string) {
	if name != "" {
		g.usedNames[name] = 1
	}
}

// TypeInferenceEngine2 performs type inference on StatementNode
type TypeInferenceEngine2 struct {
	databaseSchemas  []snapsql.DatabaseSchema        // Database schemas from pull functionality
	schemaResolver   *SchemaResolver                 // Schema resolver for type lookup
	statementNode    StatementNode                   // Parsed SQL AST
	enhancedResolver *EnhancedSubqueryResolver       // Enhanced subquery type resolver (Phase 5)
	dmlEngine        *DMLInferenceEngine             // DML inference engine (Phase 6)
	context          *InferenceContext               // Inference context
	fieldNameGen     *FieldNameGenerator             // Basic field name generator
	enhancedGen      *EnhancedFieldNameGenerator     // Enhanced field name generator for complex expressions (Phase 4)
	typeCache        map[string][]*InferredFieldInfo // Type inference cache
}

// NewTypeInferenceEngine2 creates a new type inference engine.
// Subquery analysis information is automatically extracted from the StatementNode.
func NewTypeInferenceEngine2(
	databaseSchemas []snapsql.DatabaseSchema,
	statementNode StatementNode,
) *TypeInferenceEngine2 {
	// Extract subquery analysis from statement node
	var subqueryInfo *SubqueryAnalysisResult
	if statementNode != nil && statementNode.HasSubqueryAnalysis() {
		subqueryInfo = statementNode.GetSubqueryAnalysis()
	}

	// Create schema resolver from database schemas
	schemaResolver := NewSchemaResolver(databaseSchemas)

	// Determine dialect from first schema (if available)
	dialect := snapsql.DialectPostgres // default
	if len(databaseSchemas) > 0 {
		switch databaseSchemas[0].DatabaseInfo.Type {
		case "postgres", "postgresql":
			dialect = snapsql.DialectPostgres
		case "mysql":
			dialect = snapsql.DialectMySQL
		case "sqlite":
			dialect = snapsql.DialectSQLite
		default:
			dialect = snapsql.DialectPostgres
		}
	}

	context := &InferenceContext{
		Dialect:       dialect,
		TableAliases:  make(map[string]string),
		CurrentTables: []string{},
		SubqueryDepth: 0,
	}

	// Create subquery resolver if subquery information is available
	var enhancedResolver *EnhancedSubqueryResolver
	if subqueryInfo != nil && subqueryInfo.HasSubqueries {
		// Use enhanced subquery resolver for complete type inference
		enhancedResolver = NewEnhancedSubqueryResolver(schemaResolver, statementNode, dialect)
	}

	engine := &TypeInferenceEngine2{
		databaseSchemas:  databaseSchemas,
		schemaResolver:   schemaResolver,
		statementNode:    statementNode,
		enhancedResolver: enhancedResolver,
		context:          context,
		fieldNameGen:     NewFieldNameGenerator(),
		enhancedGen:      NewEnhancedFieldNameGenerator(),
		typeCache:        make(map[string][]*InferredFieldInfo),
	}

	// Create DML inference engine (Phase 6)
	engine.dmlEngine = NewDMLInferenceEngine(engine)

	return engine
}

// functionFieldNameMappings returns mappings from function names to field names
func functionFieldNameMappings() map[string]string {
	return map[string]string{
		"COUNT":             "count",
		"SUM":               "sum",
		"AVG":               "avg",
		"MIN":               "min",
		"MAX":               "max",
		"ROW_NUMBER":        "row_number",
		"RANK":              "rank",
		"DENSE_RANK":        "dense_rank",
		"LAG":               "lag",
		"LEAD":              "lead",
		"FIRST_VALUE":       "first_value",
		"LAST_VALUE":        "last_value",
		"LENGTH":            "length",
		"CHAR_LENGTH":       "char_length",
		"UPPER":             "upper",
		"LOWER":             "lower",
		"TRIM":              "trim",
		"LTRIM":             "ltrim",
		"RTRIM":             "rtrim",
		"CONCAT":            "concat",
		"SUBSTRING":         "substring",
		"REPLACE":           "replace",
		"COALESCE":          "coalesce",
		"NULLIF":            "nullif",
		"CASE":              "case",
		"CAST":              "cast",
		"EXTRACT":           "extract",
		"DATE_PART":         "date_part",
		"NOW":               "now",
		"CURRENT_TIMESTAMP": "current_timestamp",
		"CURRENT_DATE":      "current_date",
		"CURRENT_TIME":      "current_time",
	}
}

// InferSelectTypes performs type inference on SELECT statement fields
func (e *TypeInferenceEngine2) InferSelectTypes() ([]*InferredFieldInfo, error) {
	// Get SELECT statement
	selectStmt, ok := e.statementNode.(*parser.SelectStatement)
	if !ok {
		return nil, fmt.Errorf("statement is not a SELECT statement")
	}

	// Phase 5: Resolve subquery types first (if enhanced resolver is available)
	if e.enhancedResolver != nil {
		if err := e.enhancedResolver.ResolveSubqueryTypesComplete(); err != nil {
			// Don't fail completely - log warning and continue with degraded mode
			fmt.Printf("Warning: enhanced subquery type resolution failed: %v\n", err)
		}
	}

	// Extract table aliases from FROM clause
	e.extractTableAliases(selectStmt)

	// Add available subquery tables to current tables
	if e.enhancedResolver != nil {
		subqueryTables := e.enhancedResolver.GetAvailableSubqueryTables()
		e.context.CurrentTables = append(e.context.CurrentTables, subqueryTables...)
	}

	return e.inferSelectStatement(selectStmt)
}

// InferTypes performs unified type inference for any statement type (Phase 6)
func (e *TypeInferenceEngine2) InferTypes() ([]*InferredFieldInfo, error) {
	switch stmt := e.statementNode.(type) {
	case *parser.SelectStatement:
		return e.InferSelectTypes()
	case *parser.InsertIntoStatement, *parser.UpdateStatement, *parser.DeleteFromStatement:
		// Phase 6: DML statement inference
		return e.dmlEngine.InferDMLStatementType(e.statementNode)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// performSchemaValidation is a helper method that performs schema validation
// and returns all validation errors as a slice of error
func (e *TypeInferenceEngine2) performSchemaValidation() []error {
	var allErrors []error

	// For SELECT statements, validate fields
	if selectStmt, ok := e.statementNode.(*parser.SelectStatement); ok {
		// Check if SELECT statement structure is valid
		if selectStmt == nil || selectStmt.Select == nil {
			return allErrors // Return early if structure is invalid
		}

		// Extract table aliases
		e.extractTableAliases(selectStmt)

		// Create schema validator only if schema resolver is available
		if e.schemaResolver != nil {
			validator := NewSchemaValidator(e.schemaResolver)
			if validator != nil {
				validator.SetTableAliases(e.context.TableAliases)
				validator.SetAvailableTables(e.context.CurrentTables)

				// Validate SELECT fields only if they exist
				if selectStmt.Select.Fields != nil {
					validationErrors := validator.ValidateSelectFields(selectStmt.Select.Fields)

					// Convert ValidationError to error and add to allErrors
					for _, vErr := range validationErrors {
						allErrors = append(allErrors, &vErr)
					}
				}
			}
		}

		// Add subquery validation errors if enhanced resolver is available
		if e.enhancedResolver != nil {
			subqueryErrors := e.enhancedResolver.ValidateSubqueryReferences()
			// Convert ValidationError to error and add to allErrors
			for _, vErr := range subqueryErrors {
				allErrors = append(allErrors, &vErr)
			}
		}

		return allErrors
	}

	// For DML statements, validation is handled by DML inference engine
	if e.dmlEngine != nil {
		// DML validation is performed during type inference
		// For now, return empty validation errors for DML statements
		return allErrors
	}

	return allErrors
}

// performTypeInference is a helper method that performs type inference
func (e *TypeInferenceEngine2) performTypeInference() ([]*InferredFieldInfo, error) {
	// Perform unified type inference
	return e.InferTypes()
}

// inferSelectStatement handles the core SELECT statement inference logic
func (e *TypeInferenceEngine2) inferSelectStatement(selectStmt *parser.SelectStatement) ([]*InferredFieldInfo, error) {

	// Create schema validator
	validator := NewSchemaValidator(e.schemaResolver)
	validator.SetTableAliases(e.context.TableAliases)
	validator.SetAvailableTables(e.context.CurrentTables)

	// Validate schema first
	validationErrors := validator.ValidateSelectFields(selectStmt.Select.Fields)

	// Add subquery validation errors
	if e.enhancedResolver != nil {
		subqueryErrors := e.enhancedResolver.ValidateSubqueryReferences()
		validationErrors = append(validationErrors, subqueryErrors...)
	}

	for _, err := range validationErrors {
		// For critical validation errors, return early
		if strings.Contains(err.ErrorType, "not_found") {
			return nil, fmt.Errorf("schema validation failed: %s", err.Message)
		}
	}

	// Perform type inference on each field
	var inferredFields []*InferredFieldInfo
	for i, field := range selectStmt.Select.Fields {
		inferredField, err := e.inferFieldType(&field, i)
		if err != nil {
			// For inference errors, continue with 'any' type
			inferredField = &InferredFieldInfo{
				Name:         e.fieldNameGen.GenerateFieldName("", "", "unknown"),
				OriginalName: field.OriginalField,
				Type:         &TypeInfo{BaseType: "any", IsNullable: true},
				Source:       FieldSource{Type: "unknown"},
				IsGenerated:  true,
			}
		}
		inferredFields = append(inferredFields, inferredField)
	}

	return inferredFields, nil
}

// extractTableAliases extracts table aliases from the FROM clause
func (e *TypeInferenceEngine2) extractTableAliases(selectStmt *parser.SelectStatement) {
	e.context.TableAliases = make(map[string]string)
	e.context.CurrentTables = []string{}

	if selectStmt == nil || selectStmt.From == nil || selectStmt.From.Tables == nil {
		return
	}

	// Extract table names and aliases from FROM clause
	for _, table := range selectStmt.From.Tables {
		tableName := table.TableName
		aliasName := table.Name // Use Name field for alias

		if table.ExplicitName && aliasName != tableName {
			// This is an alias
			e.context.TableAliases[aliasName] = tableName
			e.context.CurrentTables = append(e.context.CurrentTables, aliasName)
		} else {
			// No alias, use table name directly
			e.context.CurrentTables = append(e.context.CurrentTables, tableName)
		}
	}
}

// inferFieldType performs type inference on a single SELECT field
func (e *TypeInferenceEngine2) inferFieldType(field *parser.SelectField, fieldIndex int) (*InferredFieldInfo, error) {
	var fieldType *TypeInfo
	var fieldSource FieldSource
	var err error

	// Check if explicit type is available (from CAST)
	if field.ExplicitType && field.TypeName != "" {
		fieldType = &TypeInfo{
			BaseType:   normalizeType(field.TypeName),
			IsNullable: true, // CAST results can generally be null
		}
		fieldSource = FieldSource{
			Type:       "cast",
			Expression: field.TypeName,
		}
	} else {
		// Infer type based on field kind
		switch field.FieldKind {
		case parser.TableField:
			fieldType, fieldSource, err = e.inferTableFieldType(field)
		case parser.SingleField:
			fieldType, fieldSource, err = e.inferSingleFieldType(field)
		case parser.FunctionField:
			fieldType, fieldSource, err = e.inferFunctionFieldType(field)
		case parser.LiteralField:
			fieldType, fieldSource, err = e.inferLiteralFieldType(field)
		case parser.ComplexField:
			fieldType, fieldSource, err = e.inferComplexFieldType(field)
		default:
			fieldType = &TypeInfo{BaseType: "any", IsNullable: true}
			fieldSource = FieldSource{Type: "unknown"}
		}
	}

	if err != nil {
		return nil, err
	}

	// Generate field name using enhanced generator for complex expressions
	var fieldName string
	if field.ExplicitName && field.FieldName != "" {
		fieldName = field.FieldName
		e.fieldNameGen.ReserveFieldName(fieldName)
	} else {
		// Use enhanced generator for complex fields, fallback to basic for simple fields
		if field.FieldKind == parser.ComplexField || field.FieldKind == parser.FunctionField {
			fieldName = e.enhancedGen.GenerateComplexFieldName(field)
		} else {
			// Generate field name based on field kind for improved naming (Phase 4 enhancement)
			if field.FieldKind == parser.ComplexField || field.FieldKind == parser.FunctionField {
				// Use enhanced generator for complex expressions and functions
				fieldName = e.enhancedGen.GenerateComplexFieldName(field)
			} else {
				// Use basic generator for simple fields
				var fieldKindString string
				switch field.FieldKind {
				case parser.FunctionField:
					fieldKindString = "function"
				case parser.LiteralField:
					fieldKindString = "literal"
				case parser.ComplexField:
					fieldKindString = "expr"
				default:
					fieldKindString = "field"
				}

				fieldName = e.fieldNameGen.GenerateFieldName(
					field.OriginalField,
					e.extractFunctionName(field),
					fieldKindString,
				)
			}
		}
	}

	return &InferredFieldInfo{
		Name:         fieldName,
		OriginalName: field.OriginalField,
		Alias:        field.FieldName,
		Type:         fieldType,
		Source:       fieldSource,
		IsGenerated:  !field.ExplicitName,
		CastType:     field.TypeName,
	}, nil
}

// Type inference methods for different field kinds

// inferTableFieldType infers type for table.column references
func (e *TypeInferenceEngine2) inferTableFieldType(field *parser.SelectField) (*TypeInfo, FieldSource, error) {
	// Resolve table alias
	realTableName := field.TableName
	if alias, exists := e.context.TableAliases[field.TableName]; exists {
		realTableName = alias
	}

	// Extract column name from table.column format if needed
	realColumnName := field.OriginalField
	if strings.Contains(field.OriginalField, ".") {
		parts := strings.Split(field.OriginalField, ".")
		if len(parts) == 2 {
			realColumnName = parts[1] // Take the column part
		}
	}

	// Phase 5: Check if this is a subquery reference (CTE or derived table)
	if e.enhancedResolver != nil {
		if subqueryFields, found := e.enhancedResolver.ResolveSubqueryReference(realTableName); found {
			// Find the field in the subquery results
			for _, subField := range subqueryFields {
				if subField.OriginalName == realColumnName ||
					subField.Name == realColumnName ||
					subField.Alias == realColumnName {
					fieldSource := FieldSource{
						Type:   "subquery",
						Table:  realTableName,
						Column: realColumnName,
					}
					return subField.Type, fieldSource, nil
				}
			}
			return nil, FieldSource{}, fmt.Errorf("column '%s' not found in subquery '%s'", realColumnName, realTableName)
		}
	}

	// Find schema for table
	schemaName := e.findSchemaForTable(realTableName)
	if schemaName == "" {
		return nil, FieldSource{}, fmt.Errorf("table '%s' not found in schema", realTableName)
	}

	// Get column information
	column, err := e.schemaResolver.ResolveTableColumn(schemaName, realTableName, realColumnName)
	if err != nil {
		return nil, FieldSource{}, err
	}

	// Convert to TypeInfo
	fieldType := e.schemaResolver.ConvertToFieldType(column)
	fieldSource := FieldSource{
		Type:   "column",
		Table:  realTableName,
		Column: realColumnName,
	}

	return fieldType, fieldSource, nil
}

// inferSingleFieldType infers type for unqualified column references
func (e *TypeInferenceEngine2) inferSingleFieldType(field *parser.SelectField) (*TypeInfo, FieldSource, error) {
	// Phase 5: First check subqueries for this column
	if e.enhancedResolver != nil {
		var subqueryMatches []string
		var subqueryFieldInfo *InferredFieldInfo

		// Check all resolved subqueries
		subqueryTables := e.enhancedResolver.GetAvailableSubqueryTables()
		for _, tableName := range subqueryTables {
			if subqueryFields, found := e.enhancedResolver.ResolveSubqueryReference(tableName); found {
				for _, subField := range subqueryFields {
					if subField.OriginalName == field.OriginalField ||
						subField.Name == field.OriginalField ||
						subField.Alias == field.OriginalField {
						subqueryMatches = append(subqueryMatches, tableName)
						subqueryFieldInfo = subField
						break
					}
				}
			}
		}

		// If found in subqueries, use that (prioritize subqueries over base tables)
		if len(subqueryMatches) == 1 && subqueryFieldInfo != nil {
			fieldSource := FieldSource{
				Type:   "subquery",
				Table:  subqueryMatches[0],
				Column: field.OriginalField,
			}
			return subqueryFieldInfo.Type, fieldSource, nil
		} else if len(subqueryMatches) > 1 {
			return nil, FieldSource{}, fmt.Errorf("column '%s' is ambiguous in subqueries, found in: %v", field.OriginalField, subqueryMatches)
		}
	}

	// Find all tables that contain this column
	matches := e.schemaResolver.FindColumnInTables(field.OriginalField, e.context.CurrentTables)

	if len(matches) == 0 {
		return nil, FieldSource{}, fmt.Errorf("column '%s' not found in any available table", field.OriginalField)
	}

	if len(matches) > 1 {
		return nil, FieldSource{}, fmt.Errorf("column '%s' is ambiguous, found in tables: %v", field.OriginalField, matches)
	}

	// Use the single match
	parts := strings.Split(matches[0], ".")
	if len(parts) != 2 {
		return nil, FieldSource{}, fmt.Errorf("invalid table reference: %s", matches[0])
	}

	schemaName, tableName := parts[0], parts[1]
	column, err := e.schemaResolver.ResolveTableColumn(schemaName, tableName, field.OriginalField)
	if err != nil {
		return nil, FieldSource{}, err
	}

	fieldType := e.schemaResolver.ConvertToFieldType(column)
	fieldSource := FieldSource{
		Type:   "column",
		Table:  tableName,
		Column: field.OriginalField,
	}

	return fieldType, fieldSource, nil
}

// inferFunctionFieldType infers type for function calls
func (e *TypeInferenceEngine2) inferFunctionFieldType(field *parser.SelectField) (*TypeInfo, FieldSource, error) {
	functionName := e.extractFunctionName(field)
	if functionName == "" {
		return &TypeInfo{BaseType: "any", IsNullable: true},
			FieldSource{Type: "function", FunctionName: "unknown"}, nil
	}

	// Parse function arguments for advanced type inference
	argTokens := e.extractFunctionArguments(field.Expression)

	// Use advanced function inference with CAST tracking
	fieldType, err := e.InferFunctionWithCasts(functionName, argTokens)
	if err != nil {
		// Fallback to basic function rules
		fieldType = e.applyFunctionTypeRule(functionName)
	}

	fieldSource := FieldSource{
		Type:         "function",
		FunctionName: functionName,
		Expression:   field.OriginalField,
	}

	return fieldType, fieldSource, nil
}

// inferLiteralFieldType infers type for literal values
func (e *TypeInferenceEngine2) inferLiteralFieldType(field *parser.SelectField) (*TypeInfo, FieldSource, error) {
	// Basic literal type inference
	literalType := e.inferLiteralType(field.OriginalField)
	fieldSource := FieldSource{
		Type:       "literal",
		Expression: field.OriginalField,
	}

	return literalType, fieldSource, nil
}

// inferComplexFieldType infers type for complex expressions
func (e *TypeInferenceEngine2) inferComplexFieldType(field *parser.SelectField) (*TypeInfo, FieldSource, error) {
	// Use ExpressionCastAnalyzer for advanced type inference
	analyzer := NewExpressionCastAnalyzer(field.Expression, e)

	// Check if this is a CASE expression
	if len(field.Expression) > 0 &&
		field.Expression[0].Type == tokenizer.IDENTIFIER &&
		strings.ToUpper(field.Expression[0].Value) == "CASE" {

		fieldType, err := e.InferCaseExpression(field.Expression)
		if err != nil {
			fieldType = &TypeInfo{BaseType: "any", IsNullable: true}
		}

		fieldSource := FieldSource{
			Type:       "case",
			Expression: field.OriginalField,
		}

		return fieldType, fieldSource, nil
	}

	// Use expression analyzer for other complex expressions
	fieldType, err := analyzer.InferExpressionType()
	if err != nil {
		// Fallback to any type on error
		fieldType = &TypeInfo{BaseType: "any", IsNullable: true}
	}

	fieldSource := FieldSource{
		Type:       "expression",
		Expression: field.OriginalField,
	}

	return fieldType, fieldSource, nil
}

// Helper methods

// findSchemaForTable finds the schema name that contains the given table
func (e *TypeInferenceEngine2) findSchemaForTable(tableName string) string {
	for _, schemaName := range e.schemaResolver.GetAllSchemas() {
		tables := e.schemaResolver.GetTablesInSchema(schemaName)
		for _, table := range tables {
			if table == tableName {
				return schemaName
			}
		}
	}
	return ""
}

// extractFunctionName extracts function name from a function field
func (e *TypeInferenceEngine2) extractFunctionName(field *parser.SelectField) string {
	if field.FieldKind != parser.FunctionField || len(field.Expression) == 0 {
		return ""
	}

	// Look for the first identifier token which should be the function name
	for _, token := range field.Expression {
		if token.Type == tokenizer.IDENTIFIER {
			return strings.ToUpper(token.Value)
		}
	}

	return ""
}

// extractFunctionArguments extracts function arguments from expression tokens
func (e *TypeInferenceEngine2) extractFunctionArguments(tokens []tokenizer.Token) [][]tokenizer.Token {
	var args [][]tokenizer.Token

	// Find opening parenthesis
	openParenPos := -1
	for i, token := range tokens {
		if token.Type == tokenizer.OPENED_PARENS {
			openParenPos = i
			break
		}
	}

	if openParenPos == -1 {
		return args // No arguments found
	}

	// Find matching closing parenthesis
	closeParenPos := -1
	parenLevel := 0
	for i := openParenPos; i < len(tokens); i++ {
		if tokens[i].Type == tokenizer.OPENED_PARENS {
			parenLevel++
		} else if tokens[i].Type == tokenizer.CLOSED_PARENS {
			parenLevel--
			if parenLevel == 0 {
				closeParenPos = i
				break
			}
		}
	}

	if closeParenPos == -1 {
		return args // No matching closing parenthesis
	}

	// Extract arguments between parentheses
	argTokens := tokens[openParenPos+1 : closeParenPos]
	if len(argTokens) == 0 {
		return args // No arguments
	}

	// Split arguments by commas (at top level)
	var currentArg []tokenizer.Token
	parenLevel = 0

	for _, token := range argTokens {
		if token.Type == tokenizer.OPENED_PARENS {
			parenLevel++
		} else if token.Type == tokenizer.CLOSED_PARENS {
			parenLevel--
		} else if token.Type == tokenizer.COMMA && parenLevel == 0 {
			// Top-level comma - end of current argument
			if len(currentArg) > 0 {
				args = append(args, currentArg)
				currentArg = nil
			}
			continue
		}

		currentArg = append(currentArg, token)
	}

	// Add the last argument
	if len(currentArg) > 0 {
		args = append(args, currentArg)
	}

	return args
}

// applyFunctionTypeRule applies type inference rules for functions
func (e *TypeInferenceEngine2) applyFunctionTypeRule(functionName string) *TypeInfo {
	// Apply function type rules from rules.go
	switch functionName {
	case "COUNT":
		return &TypeInfo{BaseType: "int", IsNullable: false}
	case "SUM", "AVG":
		return &TypeInfo{BaseType: "decimal", IsNullable: true}
	case "MIN", "MAX":
		return &TypeInfo{BaseType: "any", IsNullable: true} // Depends on argument type
	case "LENGTH", "CHAR_LENGTH":
		return &TypeInfo{BaseType: "int", IsNullable: true}
	case "UPPER", "LOWER", "TRIM", "CONCAT":
		return &TypeInfo{BaseType: "string", IsNullable: true}
	case "NOW", "CURRENT_TIMESTAMP":
		return &TypeInfo{BaseType: "timestamp", IsNullable: false}
	case "CURRENT_DATE":
		return &TypeInfo{BaseType: "date", IsNullable: false}
	case "CURRENT_TIME":
		return &TypeInfo{BaseType: "time", IsNullable: false}
	default:
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}
}

// inferLiteralType infers type from literal value
func (e *TypeInferenceEngine2) inferLiteralType(literal string) *TypeInfo {
	literal = strings.TrimSpace(literal)

	// String literals (quoted)
	if (strings.HasPrefix(literal, "'") && strings.HasSuffix(literal, "'")) ||
		(strings.HasPrefix(literal, "\"") && strings.HasSuffix(literal, "\"")) {
		return &TypeInfo{BaseType: "string", IsNullable: false}
	}

	// NULL
	if strings.ToUpper(literal) == "NULL" {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	// Boolean
	upper := strings.ToUpper(literal)
	if upper == "TRUE" || upper == "FALSE" {
		return &TypeInfo{BaseType: "bool", IsNullable: false}
	}

	// Integer
	if _, err := fmt.Sscanf(literal, "%d", new(int)); err == nil {
		return &TypeInfo{BaseType: "int", IsNullable: false}
	}

	// Decimal/Float
	if _, err := fmt.Sscanf(literal, "%f", new(float64)); err == nil {
		if strings.Contains(literal, ".") {
			return &TypeInfo{BaseType: "decimal", IsNullable: false}
		}
		return &TypeInfo{BaseType: "float", IsNullable: false}
	}

	// Default to string
	return &TypeInfo{BaseType: "string", IsNullable: false}
}

// normalizeType normalizes a type name to standard form (moved from validator.go)
func normalizeType(typeName string) string {
	upper := strings.ToUpper(typeName)
	if normalized, exists := TypeMappings[upper]; exists {
		return normalized
	}
	return strings.ToLower(typeName)
}

// InferFunctionWithCasts infers type for function calls with CAST tracking
func (e *TypeInferenceEngine2) InferFunctionWithCasts(
	funcName string,
	argTokens [][]tokenizer.Token,
) (*TypeInfo, error) {
	var argTypes []*TypeInfo

	for _, tokens := range argTokens {
		// Create analyzer for each argument
		analyzer := NewExpressionCastAnalyzer(tokens, e)

		// Check for CAST expressions first
		casts := analyzer.DetectCasts()

		if len(casts) > 0 {
			// Use the target type of the outermost cast
			argTypes = append(argTypes, casts[len(casts)-1].TargetType)
		} else {
			// Perform regular type inference
			argType, err := analyzer.InferExpressionType()
			if err != nil {
				// Default to any type on error
				argType = &TypeInfo{BaseType: "any", IsNullable: true}
			}
			argTypes = append(argTypes, argType)
		}
	}

	// Apply function type rules with inferred argument types
	return e.applyAdvancedFunctionRule(funcName, argTypes)
}

// applyAdvancedFunctionRule applies advanced function type inference rules
func (e *TypeInferenceEngine2) applyAdvancedFunctionRule(funcName string, argTypes []*TypeInfo) (*TypeInfo, error) {
	funcName = strings.ToUpper(funcName)

	switch funcName {
	case "COUNT":
		return &TypeInfo{BaseType: "int", IsNullable: false}, nil

	case "SUM":
		if len(argTypes) > 0 && argTypes[0] != nil {
			// SUM returns the same type as input for exact types, or promoted type
			baseType := argTypes[0].BaseType
			if baseType == "int" {
				return &TypeInfo{BaseType: "decimal", IsNullable: true}, nil // Prevent overflow
			}
			return &TypeInfo{BaseType: baseType, IsNullable: true}, nil
		}
		return &TypeInfo{BaseType: "decimal", IsNullable: true}, nil

	case "AVG":
		return &TypeInfo{BaseType: "float", IsNullable: true}, nil

	case "MIN", "MAX":
		if len(argTypes) > 0 && argTypes[0] != nil {
			// MIN/MAX return the same type as their argument
			return &TypeInfo{
				BaseType:   argTypes[0].BaseType,
				IsNullable: true, // MIN/MAX can return NULL on empty sets
			}, nil
		}
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil

	case "LENGTH", "CHAR_LENGTH", "CHARACTER_LENGTH":
		return &TypeInfo{BaseType: "int", IsNullable: true}, nil

	case "UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM":
		return &TypeInfo{BaseType: "string", IsNullable: true}, nil

	case "CONCAT":
		return &TypeInfo{BaseType: "string", IsNullable: true}, nil

	case "SUBSTRING", "SUBSTR":
		return &TypeInfo{BaseType: "string", IsNullable: true}, nil

	case "COALESCE":
		// COALESCE returns the type of its first non-null argument
		if len(argTypes) > 0 {
			// Use the first argument's type but make it non-nullable
			baseType := argTypes[0].BaseType
			if baseType == "any" && len(argTypes) > 1 {
				// Try to use a more specific type from other arguments
				for _, argType := range argTypes[1:] {
					if argType.BaseType != "any" {
						baseType = argType.BaseType
						break
					}
				}
			}
			return &TypeInfo{BaseType: baseType, IsNullable: false}, nil
		}
		return &TypeInfo{BaseType: "any", IsNullable: false}, nil

	case "CAST":
		// CAST should have been handled by expression analyzer, but fallback
		if len(argTypes) > 0 {
			return argTypes[0], nil
		}
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil

	// Window functions
	case "ROW_NUMBER", "RANK", "DENSE_RANK":
		return &TypeInfo{BaseType: "int", IsNullable: false}, nil

	case "FIRST_VALUE", "LAST_VALUE", "LAG", "LEAD":
		if len(argTypes) > 0 && argTypes[0] != nil {
			return &TypeInfo{
				BaseType:   argTypes[0].BaseType,
				IsNullable: true, // Window functions can return NULL
			}, nil
		}
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil

	// Date/Time functions
	case "NOW", "CURRENT_TIMESTAMP":
		return &TypeInfo{BaseType: "timestamp", IsNullable: false}, nil

	case "CURRENT_DATE":
		return &TypeInfo{BaseType: "date", IsNullable: false}, nil

	case "CURRENT_TIME":
		return &TypeInfo{BaseType: "time", IsNullable: false}, nil

	case "DATE_TRUNC", "DATE_PART", "EXTRACT":
		return &TypeInfo{BaseType: "int", IsNullable: true}, nil

	// Math functions
	case "ABS":
		if len(argTypes) > 0 && argTypes[0] != nil {
			return &TypeInfo{
				BaseType:   argTypes[0].BaseType,
				IsNullable: argTypes[0].IsNullable,
			}, nil
		}
		return &TypeInfo{BaseType: "decimal", IsNullable: true}, nil

	case "ROUND", "CEIL", "CEILING", "FLOOR":
		if len(argTypes) > 0 && argTypes[0] != nil {
			baseType := argTypes[0].BaseType
			if baseType == "float" || baseType == "decimal" {
				return &TypeInfo{BaseType: baseType, IsNullable: argTypes[0].IsNullable}, nil
			}
		}
		return &TypeInfo{BaseType: "decimal", IsNullable: true}, nil

	case "SQRT", "POWER", "POW", "EXP", "LN", "LOG":
		return &TypeInfo{BaseType: "float", IsNullable: true}, nil

	// JSON functions (PostgreSQL)
	case "JSON_EXTRACT_PATH", "JSON_EXTRACT_PATH_TEXT":
		return &TypeInfo{BaseType: "json", IsNullable: true}, nil

	case "JSON_OBJECT", "JSON_ARRAY":
		return &TypeInfo{BaseType: "json", IsNullable: false}, nil

	default:
		// Unknown function - default to any type
		return &TypeInfo{BaseType: "any", IsNullable: true}, nil
	}
}

// InferCaseExpression infers type for CASE expressions
func (e *TypeInferenceEngine2) InferCaseExpression(tokens []tokenizer.Token) (*TypeInfo, error) {
	analyzer := NewCaseExpressionAnalyzer(tokens, e)
	result, err := analyzer.AnalyzeCaseExpression()
	if err != nil {
		return &TypeInfo{BaseType: "any", IsNullable: true}, err
	}

	if result != nil && result.InferredType != nil {
		return result.InferredType, nil
	}

	return &TypeInfo{BaseType: "any", IsNullable: true}, nil
}
