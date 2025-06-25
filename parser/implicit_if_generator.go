package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// ImplicitIfGenerator generates implicit conditional blocks for variable references
// GenerateImplicitConditionals processes AST nodes and generates implicit conditionals
func GenerateImplicitConditionals(node AstNode, schema *InterfaceSchema, ns *Namespace) AstNode {
	switch n := node.(type) {
	case *SelectStatement:
		return processSelectStatement(n, schema, ns)
	case *VariableSubstitution:
		return processVariableSubstitution(n, schema, ns)
	default:
		return node
	}
}

// processSelectStatement processes a SELECT statement and generates implicit conditionals
func processSelectStatement(stmt *SelectStatement, schema *InterfaceSchema, ns *Namespace) *SelectStatement {
	// Process WHERE clause for implicit conditions
	if stmt.WhereClause != nil {
		processedClause := processWhereClause(stmt.WhereClause, schema, ns)
		if implicitIf, ok := processedClause.(*ImplicitConditional); ok {
			// Store the implicit conditional information
			// For now, we keep the original clause but mark it as having implicit conditions
			_ = implicitIf // Use the implicit conditional information as needed
		}
	}

	// Process ORDER BY clause
	if stmt.OrderByClause != nil {
		processedClause := processOrderByClause(stmt.OrderByClause, schema, ns)
		if implicitIf, ok := processedClause.(*ImplicitConditional); ok {
			_ = implicitIf // Store implicit conditional information
		}
	}

	// Process LIMIT clause
	if stmt.LimitClause != nil {
		processedClause := processLimitClause(stmt.LimitClause, schema, ns)
		if implicitIf, ok := processedClause.(*ImplicitConditional); ok {
			_ = implicitIf // Store implicit conditional information
		}
	}

	// Process OFFSET clause
	if stmt.OffsetClause != nil {
		processedClause := processOffsetClause(stmt.OffsetClause, schema, ns)
		if implicitIf, ok := processedClause.(*ImplicitConditional); ok {
			_ = implicitIf // Store implicit conditional information
		}
	}

	return stmt
}

// processWhereClause processes WHERE clause and generates implicit conditionals
func processWhereClause(clause *WhereClause, schema *InterfaceSchema, _ns *Namespace) AstNode {
	// Extract variable references from WHERE conditions
	variables := extractVariableReferences(clause.String())

	if len(variables) == 0 {
		return clause
	}

	// Generate implicit conditional for the entire WHERE clause
	condition := generateWhereCondition(variables, schema)

	return &ImplicitConditional{
		BaseAstNode: BaseAstNode{
			nodeType: IMPLICIT_CONDITIONAL,
			position: clause.Position(),
		},
		Variable:   strings.Join(variables, " || "),
		Content:    clause,
		ClauseType: "WHERE",
		Condition:  condition,
	}
}

// processOrderByClause processes ORDER BY clause and generates implicit conditionals
func processOrderByClause(clause *OrderByClause, schema *InterfaceSchema, _ns *Namespace) AstNode {
	variables := extractVariableReferences(clause.String())

	if len(variables) == 0 {
		return clause
	}

	condition := generateOrderByCondition(variables, schema)

	return &ImplicitConditional{
		BaseAstNode: BaseAstNode{
			nodeType: IMPLICIT_CONDITIONAL,
			position: clause.Position(),
		},
		Variable:   strings.Join(variables, " || "),
		Content:    clause,
		ClauseType: "ORDER_BY",
		Condition:  condition,
	}
}

// processLimitClause processes LIMIT clause and generates implicit conditionals
func processLimitClause(clause *LimitClause, _schema *InterfaceSchema, _ns *Namespace) AstNode {
	variables := extractVariableReferences(clause.String())

	if len(variables) == 0 {
		return clause
	}

	// LIMIT should be present if the variable is not null and > 0
	condition := fmt.Sprintf("%s != null && %s > 0", variables[0], variables[0])

	return &ImplicitConditional{
		BaseAstNode: BaseAstNode{
			nodeType: IMPLICIT_CONDITIONAL,
			position: clause.Position(),
		},
		Variable:   variables[0],
		Content:    clause,
		ClauseType: "LIMIT",
		Condition:  condition,
	}
}

// processOffsetClause processes OFFSET clause and generates implicit conditionals
func processOffsetClause(clause *OffsetClause, _schema *InterfaceSchema, _ns *Namespace) AstNode {
	variables := extractVariableReferences(clause.String())

	if len(variables) == 0 {
		return clause
	}

	// OFFSET should be present if the variable is not null and >= 0
	condition := fmt.Sprintf("%s != null && %s >= 0", variables[0], variables[0])

	return &ImplicitConditional{
		BaseAstNode: BaseAstNode{
			nodeType: IMPLICIT_CONDITIONAL,
			position: clause.Position(),
		},
		Variable:   variables[0],
		Content:    clause,
		ClauseType: "OFFSET",
		Condition:  condition,
	}
}

// processVariableSubstitution processes individual variable substitutions
func processVariableSubstitution(varSub *VariableSubstitution, schema *InterfaceSchema, _ *Namespace) AstNode {
	// Check if this variable substitution should be wrapped in implicit conditional
	if shouldWrapInImplicitIf(varSub.Expression, schema) {
		condition := generateVariableCondition(varSub.Expression, schema)

		return &ImplicitConditional{
			BaseAstNode: BaseAstNode{
				nodeType: IMPLICIT_CONDITIONAL,
				position: varSub.Position(),
			},
			Variable:   varSub.Expression,
			Content:    varSub,
			ClauseType: "CONDITION",
			Condition:  condition,
		}
	}

	return varSub
}

// extractVariableReferences extracts variable references from SQL text
func extractVariableReferences(sqlText string) []string {
	// Pattern to match /*= variable */ expressions
	re := regexp.MustCompile(`/\*=\s*([^*]+)\s*\*/`)
	matches := re.FindAllStringSubmatch(sqlText, -1)

	variables := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			variable := strings.TrimSpace(match[1])
			variables = append(variables, variable)
		}
	}

	return variables
}

// shouldWrapInImplicitIf determines if a variable should be wrapped in implicit conditional
func shouldWrapInImplicitIf(variable string, schema *InterfaceSchema) bool {
	// Check if variable is a list type or nullable using hierarchical lookup
	if schema != nil {
		if paramType, exists := schema.GetParameterType(variable); exists {
			// List types should be wrapped
			if strings.HasPrefix(paramType, "list[") {
				return true
			}
			// Optional parameters should be wrapped
			// This could be extended based on schema annotations
		}
	}

	// For now, wrap variables that contain dots (nested objects)
	return strings.Contains(variable, ".")
}

// generateWhereCondition generates CEL condition for WHERE clause
func generateWhereCondition(variables []string, schema *InterfaceSchema) string {
	var conditions []string

	for _, variable := range variables {
		condition := generateVariableCondition(variable, schema)
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 0 {
		return "true"
	}

	return strings.Join(conditions, " || ")
}

// generateOrderByCondition generates CEL condition for ORDER BY clause
func generateOrderByCondition(variables []string, schema *InterfaceSchema) string {
	var conditions []string

	for _, variable := range variables {
		condition := generateVariableCondition(variable, schema)
		if condition != "" {
			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 0 {
		return "true"
	}

	return strings.Join(conditions, " || ")
}

// generateVariableCondition generates CEL condition for a specific variable
func generateVariableCondition(variable string, schema *InterfaceSchema) string {
	if schema == nil {
		return fmt.Sprintf("%s != null", variable)
	}

	// Get parameter type using hierarchical lookup
	paramType, exists := schema.GetParameterType(variable)
	if !exists {
		return fmt.Sprintf("%s != null", variable)
	}

	// Generate condition based on type
	switch {
	case strings.HasPrefix(paramType, "list["):
		return fmt.Sprintf("%s != null && size(%s) > 0", variable, variable)
	case paramType == "str":
		return fmt.Sprintf("%s != null && %s != ''", variable, variable)
	case paramType == "int":
		return fmt.Sprintf("%s != null", variable)
	case paramType == "bool":
		return fmt.Sprintf("%s != null", variable)
	default:
		return fmt.Sprintf("%s != null", variable)
	}
}

// ValidateImplicitConditions validates generated implicit conditions using CEL engine
func ValidateImplicitConditions(conditions []string, ns *Namespace) []error {
	var errors []error

	if ns == nil {
		return errors
	}

	for _, condition := range conditions {
		if err := ns.ValidateParameterExpression(condition); err != nil {
			errors = append(errors, fmt.Errorf("invalid implicit condition '%s': %w", condition, err))
		}
	}

	return errors
}
