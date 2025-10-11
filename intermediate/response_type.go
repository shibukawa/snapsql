package intermediate

import (
	"fmt"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/typeinference"
)

// determineResponseType analyzes the statement and determines the response type
func determineResponseType(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) []Response {
	// Augment tableInfo with CTE/subquery derived tables
	augmentedTableInfo := augmentTableInfoWithDerivedTables(stmt, tableInfo)

	// First attempt with provided schema (now including derived tables)
	schemas := convertTableInfoToSchemas(augmentedTableInfo)
	inferredFields, _ := typeinference.InferFieldTypes(schemas, stmt, nil)

	// Inline schema fallbackは開発用に残されていたが、正式運用では schema YAML か tableInfo が必須。
	// schema 未提供で推論できない場合は即エラー扱い (上位で検出) するため空 slice を返す。
	if len(inferredFields) == 0 && len(tableInfo) == 0 {
		return []Response{}
	}

	// If we have inferred fields, return them even if there's an error
	// (e.g., subquery type information warnings)
	if len(inferredFields) > 0 {
		var fields []Response
		for _, field := range inferredFields {
			response := Response{Name: field.Name}
			if field.Type != nil {
				response.Type = field.Type.BaseType
				response.IsNullable = field.Type.IsNullable
				response.Precision = field.Type.Precision
				response.Scale = field.Type.Scale
				response.MaxLength = field.Type.MaxLength
			} else {
				response.Type = "any"
			}

			if field.Source.Type == "column" {
				response.SourceTable = field.Source.Table
				response.SourceColumn = field.Source.Column
			}

			fields = append(fields, response)
		}

		return fields
	}

	// 最終手段の alias ベース fallback は廃止 (型解像しないまま進めると階層キー判定が壊れるため)
	return []Response{}
}

// augmentTableInfoWithDerivedTables adds CTE/subquery derived tables to tableInfo
func augmentTableInfoWithDerivedTables(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) map[string]*snapsql.TableInfo {
	// Create a copy of tableInfo to avoid modifying the original
	augmented := make(map[string]*snapsql.TableInfo, len(tableInfo))
	for k, v := range tableInfo {
		augmented[k] = v
	}

	// Get SubqueryAnalysisResult
	if !stmt.HasSubqueryAnalysis() {
		return augmented
	}

	analysis := stmt.GetSubqueryAnalysis()
	if analysis == nil {
		return augmented
	}

	// Convert each DerivedTableInfo to TableInfo
	for _, dt := range analysis.DerivedTables {
		fmt.Printf("DEBUG: Processing derived table '%s' with %d SelectFields\n", dt.Name, len(dt.SelectFields))

		// Convert SelectFields to ColumnInfo map
		columns := make(map[string]*snapsql.ColumnInfo)
		columnOrder := make([]string, 0, len(dt.SelectFields))

		for _, sf := range dt.SelectFields {
			fmt.Printf("DEBUG:   - Field: %s (type: %s)\n", sf.FieldName, sf.TypeName)

			col := &snapsql.ColumnInfo{
				Name:     sf.FieldName,
				DataType: sf.TypeName, // May be empty if not explicitly cast
				Nullable: true,        // Default to nullable for derived fields
			}

			// If no explicit type, try to infer from field kind
			if col.DataType == "" {
				col.DataType = inferTypeFromFieldKind(sf)
			}

			columns[sf.FieldName] = col
			columnOrder = append(columnOrder, sf.FieldName)
		}

		// Create virtual TableInfo for this derived table
		virtualTable := &snapsql.TableInfo{
			Name:        dt.Name,
			Columns:     columns,
			ColumnOrder: columnOrder,
		}

		fmt.Printf("DEBUG: Created virtual table '%s' with %d columns\n", dt.Name, len(columns))
		augmented[dt.Name] = virtualTable
	}

	return augmented
}

// inferTypeFromFieldKind attempts to infer a basic type from SelectField kind
func inferTypeFromFieldKind(sf cmn.SelectField) string {
	// For simple fields, type needs to come from schema lookup (not done here)
	// For function fields, we might have hints
	// For literal fields, we can try to detect type
	// Default to "any" for unknown types
	return "any"
}

// convertTableInfoToSchemas converts intermediate.TableInfo to typeinference.DatabaseSchema
func convertTableInfoToSchemas(tableInfo map[string]*snapsql.TableInfo) []snapsql.DatabaseSchema {
	if len(tableInfo) == 0 {
		return nil
	}

	// Create a single database schema with all tables
	schema := snapsql.DatabaseSchema{
		Name:   "default", // Default database name
		Tables: make([]*snapsql.TableInfo, 0, len(tableInfo)),
	}

	// TableInfo is already in the correct format, just add to schema
	for _, info := range tableInfo {
		schema.Tables = append(schema.Tables, info)
	}

	return []snapsql.DatabaseSchema{schema}
}
