package intermediate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/typeinference"
)

// determineResponseType analyzes the statement and determines the response type.
// It returns the inferred responses along with warning messages.
func determineResponseType(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) ([]Response, []string) {
	collector := newWarningCollector()

	// Augment tableInfo with CTE/subquery derived tables
	augmentedTableInfo := augmentTableInfoWithDerivedTables(stmt, tableInfo)

	// First attempt with provided schema (now including derived tables)
	schemas := convertTableInfoToSchemas(augmentedTableInfo)
	inferredFields, inferWarnings, inferErr := typeinference.InferFieldTypesWithWarnings(schemas, stmt, nil)
	collector.AddAll(inferWarnings)
	if inferErr != nil {
		collector.Add(fmt.Sprintf("type inference failed: %v", inferErr))
	}

	if len(inferredFields) > 0 {
		fields := make([]Response, 0, len(inferredFields))
		for idx, field := range inferredFields {
			name := field.Name
			if name == "" {
				name = field.Alias
			}
			if name == "" {
				name = field.OriginalName
			}
			name = cleanIdentifier(name)
			if name == "" {
				name = fmt.Sprintf("field_%d", idx+1)
			}

			response := Response{Name: name}
			if field.Type != nil {
				response.Type = field.Type.BaseType
				response.IsNullable = field.Type.IsNullable
				response.Precision = field.Type.Precision
				response.Scale = field.Type.Scale
				response.MaxLength = field.Type.MaxLength
			} else {
				response.Type = "any"
				response.IsNullable = true
			}

			if field.Source.Type == "column" {
				response.SourceTable = cleanIdentifier(field.Source.Table)
				response.SourceColumn = cleanIdentifier(field.Source.Column)
			}

			fields = append(fields, response)
		}

		return fields, collector.List()
	}

	// Fallback: synthesize responses from SELECT clause when inference failed (e.g., no schema)
	fallback := buildFallbackResponses(stmt)
	if len(fallback) > 0 {
		if len(inferredFields) == 0 {
			collector.Add("type inference unavailable: generated fallback responses with any type")
		}
		return fallback, collector.List()
	}

	return []Response{}, collector.List()
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
		// Convert SelectFields to ColumnInfo map
		columns := make(map[string]*snapsql.ColumnInfo)
		columnOrder := make([]string, 0, len(dt.SelectFields))

		for _, sf := range dt.SelectFields {
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
		augmented[dt.Name] = virtualTable
	}

	return augmented
}

// buildFallbackResponses synthesizes Response metadata based on SELECT clause when type inference fails.
func buildFallbackResponses(stmt parser.StatementNode) []Response {
	selectStmt, ok := stmt.(*parser.SelectStatement)
	if !ok || selectStmt.Select == nil {
		return nil
	}

	fields := selectStmt.Select.Fields
	if len(fields) == 0 {
		return nil
	}

	responses := make([]Response, 0, len(fields))
	nameCounter := make(map[string]int)
	fieldSources := stmt.GetFieldSources()

	for idx, field := range fields {
		name := fallbackResponseName(field, idx, nameCounter)
		sourceTable, sourceColumn := resolveFallbackSource(stmt, field, fieldSources)

		responses = append(responses, Response{
			Name:         name,
			Type:         "any",
			IsNullable:   true,
			SourceTable:  sourceTable,
			SourceColumn: sourceColumn,
		})
	}

	return responses
}

// fallbackResponseName generates a stable, unique response name for fallback mode.
func fallbackResponseName(field parser.SelectField, index int, counter map[string]int) string {
	var candidates []string
	if field.FieldName != "" {
		candidates = append(candidates, field.FieldName)
	}
	if field.OriginalField != "" {
		candidates = append(candidates, field.OriginalField)
	}

	for _, cand := range candidates {
		if cleaned := cleanIdentifier(cand); cleaned != "" {
			return uniquifyName(cleaned, counter)
		}
	}

	// Default fallback name
	return uniquifyName(fmt.Sprintf("field_%d", index+1), counter)
}

// resolveFallbackSource attempts to determine source table/column information for fallback responses.
func resolveFallbackSource(stmt parser.StatementNode, field parser.SelectField, sources map[string]*cmn.SQFieldSource) (string, string) {
	normalizedOriginal := cleanIdentifier(field.OriginalField)

	keyCandidates := []string{}
	if field.FieldName != "" {
		keyCandidates = append(keyCandidates, field.FieldName)
	}
	if field.TableName != "" {
		if field.FieldName != "" {
			keyCandidates = append(keyCandidates, field.TableName+"."+field.FieldName)
		}
		if normalizedOriginal != "" {
			keyCandidates = append(keyCandidates, field.TableName+"."+normalizedOriginal)
		}
	}
	if field.OriginalField != "" {
		keyCandidates = append(keyCandidates, field.OriginalField)
	}
	if normalizedOriginal != "" {
		keyCandidates = append(keyCandidates, normalizedOriginal)
	}

	for _, key := range keyCandidates {
		if source, ok := sources[key]; ok && source != nil {
			return extractSourceInfo(source)
		}
	}

	// Try direct lookup via helper
	var source *cmn.SQFieldSource
	if field.TableName != "" {
		source = stmt.FindFieldReference(field.TableName, normalizedOriginal)
		if source == nil && field.FieldName != "" {
			source = stmt.FindFieldReference(field.TableName, cleanIdentifier(field.FieldName))
		}
	} else if normalizedOriginal != "" {
		source = stmt.FindFieldReference("", normalizedOriginal)
	}

	if source != nil {
		return extractSourceInfo(source)
	}

	// Fallback: use table/column hints from SELECT field itself
	if field.TableName != "" {
		return field.TableName, normalizedOriginal
	}

	return "", normalizedOriginal
}

func extractSourceInfo(source *cmn.SQFieldSource) (string, string) {
	if source == nil {
		return "", ""
	}
	var tableName string
	if source.TableSource != nil {
		if source.TableSource.RealName != "" {
			tableName = source.TableSource.RealName
		} else if source.TableSource.Name != "" {
			tableName = source.TableSource.Name
		}
	}
	if tableName == "" && source.SubqueryRef != "" {
		tableName = source.SubqueryRef
	}

	columnName := cleanIdentifier(source.Name)
	if columnName == "" {
		columnName = cleanIdentifier(source.Alias)
	}

	return tableName, columnName
}

func cleanIdentifier(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Trim(trimmed, "`\"[]")
	if dot := strings.LastIndex(trimmed, "."); dot >= 0 {
		trimmed = trimmed[dot+1:]
		trimmed = strings.Trim(trimmed, "`\"[]")
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func normalizeFallbackName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "\n", "_", "\t", "_", "\r", "_")
	name = replacer.Replace(name)
	name = strings.Trim(name, "`\"[]")
	if name == "" {
		return ""
	}
	return name
}
func uniquifyName(base string, counter map[string]int) string {
	count := counter[base]
	if count == 0 {
		counter[base] = 1
		return base
	}
	count++
	counter[base] = count
	return fmt.Sprintf("%s_%d", base, count)
}

type warningCollector struct {
	set map[string]struct{}
}

func newWarningCollector() *warningCollector {
	return &warningCollector{set: make(map[string]struct{})}
}

func (wc *warningCollector) Add(message string) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return
	}
	wc.set[msg] = struct{}{}
}

func (wc *warningCollector) AddAll(messages []string) {
	for _, msg := range messages {
		wc.Add(msg)
	}
}

func (wc *warningCollector) List() []string {
	if len(wc.set) == 0 {
		return nil
	}
	result := make([]string, 0, len(wc.set))
	for msg := range wc.set {
		result = append(result, msg)
	}
	sort.Strings(result)
	return result
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
