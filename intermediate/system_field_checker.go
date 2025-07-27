package intermediate

import (
	"fmt"
	"strings"

	. "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// StatementTypeProvider is a minimal interface for getting statement type
type StatementTypeProvider interface {
	Type() parsercommon.NodeType
}

// CheckSystemFields validates system fields configuration and returns implicit parameters
// Returns error if validation fails (e.g., explicit parameter missing, error parameter provided)
func CheckSystemFields(stmt StatementTypeProvider, config *Config, parameters []Parameter) ([]ImplicitParameter, error) {
	if config == nil {
		return nil, nil
	}

	// Create a map of existing parameters for quick lookup
	paramMap := make(map[string]bool)
	for _, param := range parameters {
		paramMap[param.Name] = true
	}

	// Determine statement type
	stmtType := getStatementType(stmt)

	var implicitParams []ImplicitParameter
	var errors []string

	// Check each system field configured for the current operation
	for _, field := range config.System.Fields {
		var operation *SystemFieldOperation
		var operationName string

		switch stmtType {
		case "INSERT":
			if field.OnInsert.Default != nil || field.OnInsert.Parameter != "" {
				operation = &field.OnInsert
				operationName = "INSERT"
			}
		case "UPDATE":
			if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != "" {
				operation = &field.OnUpdate
				operationName = "UPDATE"
			}
		default:
			// SELECT, DELETE, etc. don't need system field validation
			continue
		}

		if operation == nil {
			continue
		}

		// Perform the same validation logic for both INSERT and UPDATE
		implicitParam, err := checkSystemField(field, operation, operationName, paramMap)
		if err != nil {
			errors = append(errors, err.Error())
		}
		if implicitParam != nil {
			implicitParams = append(implicitParams, *implicitParam)
		}
	}

	// Return error if validation failed
	if len(errors) > 0 {
		return nil, fmt.Errorf("System field validation errors:\n- %s",
			strings.Join(errors, "\n- "))
	}

	return implicitParams, nil
}

// checkSystemField performs validation for a single system field
func checkSystemField(field SystemField, operation *SystemFieldOperation, operationName string, paramMap map[string]bool) (*ImplicitParameter, error) {
	// Handle parameter configuration
	switch operation.Parameter {
	case ParameterExplicit:
		// Check if explicit parameter is provided
		if !paramMap[field.Name] {
			return nil, fmt.Errorf("%s statement requires explicit parameter '%s' but it was not provided", operationName, field.Name)
		}
		// Explicit parameter provided, no implicit parameter needed
		return nil, nil

	case ParameterImplicit:
		// Add to implicit parameters list
		implicitParam := &ImplicitParameter{
			Name: field.Name,
			Type: field.Type,
		}

		// Add default value if specified
		if operation.Default != nil {
			implicitParam.Default = operation.Default
		}

		return implicitParam, nil

	case ParameterError:
		// Check if parameter is provided (should cause error)
		if paramMap[field.Name] {
			return nil, fmt.Errorf("%s statement should not include parameter '%s' (configured as error)", operationName, field.Name)
		}
		// No parameter provided, no implicit parameter needed
		return nil, nil

	default:
		// For fields without explicit parameter configuration,
		// if there's a default value and the parameter is not provided,
		// it should be added as implicit parameter
		if operation.Default != nil && !paramMap[field.Name] {
			return &ImplicitParameter{
				Name:    field.Name,
				Type:    field.Type,
				Default: operation.Default,
			}, nil
		}
	}

	return nil, nil
}

// getStatementType determines the type of SQL statement
func getStatementType(stmt StatementTypeProvider) string {
	// Use the same approach as response_affinity.go
	switch stmt.Type() {
	case parsercommon.SELECT_STATEMENT:
		return "SELECT"
	case parsercommon.INSERT_INTO_STATEMENT:
		return "INSERT"
	case parsercommon.UPDATE_STATEMENT:
		return "UPDATE"
	case parsercommon.DELETE_FROM_STATEMENT:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}
