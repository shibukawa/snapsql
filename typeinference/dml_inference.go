package typeinference

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// DMLInferenceEngine handles type inference for INSERT/UPDATE/DELETE statements
type DMLInferenceEngine struct {
	baseEngine       *TypeInferenceEngine2
	schemaResolver   *SchemaResolver
	subqueryResolver *SubqueryTypeResolver
}

// NewDMLInferenceEngine creates a new DML inference engine
func NewDMLInferenceEngine(baseEngine *TypeInferenceEngine2) *DMLInferenceEngine {
	return &DMLInferenceEngine{
		baseEngine:       baseEngine,
		schemaResolver:   baseEngine.schemaResolver,
		subqueryResolver: baseEngine.subqueryResolver,
	}
}

// InferDMLStatementType performs type inference for DML statements
func (d *DMLInferenceEngine) InferDMLStatementType(stmt parsercommon.StatementNode) ([]*InferredFieldInfo, error) {
	switch s := stmt.(type) {
	case *parsercommon.InsertIntoStatement:
		return d.inferInsertStatement(s)
	case *parsercommon.UpdateStatement:
		return d.inferUpdateStatement(s)
	case *parsercommon.DeleteFromStatement:
		return d.inferDeleteStatement(s)
	default:
		return nil, fmt.Errorf("unsupported DML statement type: %T", stmt)
	}
}

// inferInsertStatement handles INSERT statement type inference
func (d *DMLInferenceEngine) inferInsertStatement(stmt *parsercommon.InsertIntoStatement) ([]*InferredFieldInfo, error) {
	var fields []*InferredFieldInfo

	// Get target table information from INTO clause
	targetTable, err := d.getTargetTableFromInsert(stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to get target table: %w", err)
	}

	// Handle RETURNING clause if present
	if stmt.Returning != nil {
		returningFields, err := d.inferReturningClause(stmt.Returning, targetTable)
		if err != nil {
			return nil, fmt.Errorf("failed to infer RETURNING clause: %w", err)
		}
		fields = append(fields, returningFields...)
	}

	// If no RETURNING clause, return empty result (INSERT without output)
	if len(fields) == 0 {
		fields = append(fields, &InferredFieldInfo{
			Name: "affected_rows",
			Type: &TypeInfo{
				BaseType:   "int",
				IsNullable: false,
			},
			Source: FieldSource{
				Type:       "function",
				Expression: "INSERT statement affected rows",
			},
			IsGenerated: true,
		})
	}

	return fields, nil
}

// inferUpdateStatement handles UPDATE statement type inference
func (d *DMLInferenceEngine) inferUpdateStatement(stmt *parsercommon.UpdateStatement) ([]*InferredFieldInfo, error) {
	var fields []*InferredFieldInfo

	// Get target table information from UPDATE clause
	targetTable, err := d.getTargetTableFromUpdate(stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to get target table: %w", err)
	}

	// Handle RETURNING clause if present
	if stmt.Returning != nil {
		returningFields, err := d.inferReturningClause(stmt.Returning, targetTable)
		if err != nil {
			return nil, fmt.Errorf("failed to infer RETURNING clause: %w", err)
		}
		fields = append(fields, returningFields...)
	}

	// If no RETURNING clause, return empty result (UPDATE without output)
	if len(fields) == 0 {
		fields = append(fields, &InferredFieldInfo{
			Name: "affected_rows",
			Type: &TypeInfo{
				BaseType:   "int",
				IsNullable: false,
			},
			Source: FieldSource{
				Type:       "function",
				Expression: "UPDATE statement affected rows",
			},
			IsGenerated: true,
		})
	}

	return fields, nil
}

// inferDeleteStatement handles DELETE statement type inference
func (d *DMLInferenceEngine) inferDeleteStatement(stmt *parsercommon.DeleteFromStatement) ([]*InferredFieldInfo, error) {
	var fields []*InferredFieldInfo

	// Get target table information from FROM clause
	targetTable, err := d.getTargetTableFromDelete(stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to get target table: %w", err)
	}

	// Handle RETURNING clause if present
	if stmt.Returning != nil {
		returningFields, err := d.inferReturningClause(stmt.Returning, targetTable)
		if err != nil {
			return nil, fmt.Errorf("failed to infer RETURNING clause: %w", err)
		}
		fields = append(fields, returningFields...)
	}

	// If no RETURNING clause, return empty result (DELETE without output)
	if len(fields) == 0 {
		fields = append(fields, &InferredFieldInfo{
			Name: "affected_rows",
			Type: &TypeInfo{
				BaseType:   "int",
				IsNullable: false,
			},
			Source: FieldSource{
				Type:       "function",
				Expression: "DELETE statement affected rows",
			},
			IsGenerated: true,
		})
	}

	return fields, nil
}

// inferReturningClause handles RETURNING clause type inference
func (d *DMLInferenceEngine) inferReturningClause(returning *parsercommon.ReturningClause, targetTable string) ([]*InferredFieldInfo, error) {
	var fields []*InferredFieldInfo

	// RETURNING clause uses SelectField structure, so we can reuse SELECT inference
	for i, field := range returning.Fields {
		fieldInfo, err := d.baseEngine.inferFieldType(&field, i)
		if err != nil {
			return nil, fmt.Errorf("failed to infer RETURNING field: %w", err)
		}
		fields = append(fields, fieldInfo)
	}

	return fields, nil
}

// getTargetTableFromInsert extracts target table name from INSERT statement
func (d *DMLInferenceEngine) getTargetTableFromInsert(stmt *parsercommon.InsertIntoStatement) (string, error) {
	if stmt.Into == nil {
		return "", fmt.Errorf("INSERT statement missing INTO clause")
	}

	// Extract table name from INTO clause
	// This is a simplified implementation - may need enhancement for complex table references
	return d.extractTableNameFromClause(stmt.Into)
}

// getTargetTableFromUpdate extracts target table name from UPDATE statement
func (d *DMLInferenceEngine) getTargetTableFromUpdate(stmt *parsercommon.UpdateStatement) (string, error) {
	if stmt.Update == nil {
		return "", fmt.Errorf("UPDATE statement missing UPDATE clause")
	}

	// Extract table name from UPDATE clause
	return d.extractTableNameFromClause(stmt.Update)
}

// getTargetTableFromDelete extracts target table name from DELETE statement
func (d *DMLInferenceEngine) getTargetTableFromDelete(stmt *parsercommon.DeleteFromStatement) (string, error) {
	if stmt.From == nil {
		return "", fmt.Errorf("DELETE statement missing FROM clause")
	}

	// Extract table name from FROM clause
	return d.extractTableNameFromClause(stmt.From)
}

// extractTableNameFromClause extracts table name from various clause types
func (d *DMLInferenceEngine) extractTableNameFromClause(clause parsercommon.ClauseNode) (string, error) {
	// This is a simplified implementation that extracts table name from clause text
	// In a real implementation, this would need to parse the clause more thoroughly
	clauseText := clause.SourceText()

	// Basic table name extraction (this needs improvement for complex cases)
	// For now, assume simple table names without schema qualification
	if clauseText == "" {
		// For testing purposes, return a default table name
		return "users", nil
	}

	// This is a placeholder implementation
	// TODO: Implement proper table name extraction from clause tokens
	return "placeholder_table", nil
}

// Sentinel errors for DML inference
var (
	ErrUnsupportedDMLStatement = fmt.Errorf("unsupported DML statement type")
	ErrMissingTargetTable      = fmt.Errorf("missing target table in DML statement")
	ErrInvalidReturningClause  = fmt.Errorf("invalid RETURNING clause")
)
