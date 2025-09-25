package intermediate

import (
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/typeinference"
)

// determineResponseType analyzes the statement and determines the response type
func determineResponseType(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) []Response {
	// First attempt with provided schema
	schemas := convertTableInfoToSchemas(tableInfo)
	inferredFields, err := typeinference.InferFieldTypes(schemas, stmt, nil)

	// Inline schema fallbackは開発用に残されていたが、正式運用では schema YAML か tableInfo が必須。
	// schema 未提供で推論できない場合は即エラー扱い (上位で検出) するため空 slice を返す。
	if (err != nil || len(inferredFields) == 0) && len(tableInfo) == 0 {
		return []Response{}
	}

	if err == nil && len(inferredFields) > 0 {
		var fields []Response
		for _, field := range inferredFields {
			response := Response{Name: field.Name}
			if field.Type != nil {
				response.Type = field.Type.BaseType
				response.BaseType = field.Type.BaseType
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
