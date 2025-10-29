package intermediate

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// TokenPipeline represents a token processing pipeline
type TokenPipeline struct {
	tokens     []tokenizer.Token
	stmt       parser.StatementNode
	funcDef    *parser.FunctionDefinition
	config     *snapsql.Config
	tableInfo  map[string]*snapsql.TableInfo
	typeInfo   map[string]any
	processors []TokenProcessor
}

// TokenProcessor defines the interface for token processing stages
type TokenProcessor interface {
	Process(ctx *ProcessingContext) error
	Name() string
}

// ProcessingContext holds the context for token processing
type ProcessingContext struct {
	Tokens      []tokenizer.Token
	Statement   parser.StatementNode
	FunctionDef *parser.FunctionDefinition
	Config      *snapsql.Config
	TableInfo   map[string]*snapsql.TableInfo
	TypeInfoMap map[string]any

	// Selected dialect (normalized lowercase) from config; empty means default postgres
	Dialect string

	// Processing results
	Environments   []string
	ImplicitParams []ImplicitParameter
	SystemFields   []SystemFieldInfo
	Instructions   []Instruction

	// Enhanced CEL information
	CELExpressions  []CELExpression
	CELEnvironments []CELEnvironment

	// Table references extracted from the statement
	TableReferences []TableReferenceInfo

	// Metadata
	Description      string
	FunctionName     string
	Parameters       []Parameter
	ResponseAffinity string
}

// NewTokenPipeline creates a new token processing pipeline
func NewTokenPipeline(stmt parser.StatementNode, funcDef *parser.FunctionDefinition, config *snapsql.Config, tableInfo map[string]*snapsql.TableInfo, typeInfo map[string]any) *TokenPipeline {
	return &TokenPipeline{
		tokens:    extractTokensFromStatement(stmt),
		stmt:      stmt,
		funcDef:   funcDef,
		config:    config,
		tableInfo: tableInfo,
		typeInfo:  typeInfo,
	}
}

// AddProcessor adds a token processor to the pipeline
func (p *TokenPipeline) AddProcessor(processor TokenProcessor) {
	p.processors = append(p.processors, processor)
}

// Execute runs the token processing pipeline
func (p *TokenPipeline) Execute() (*IntermediateFormat, error) {
	ctx := &ProcessingContext{
		Tokens:      p.tokens,
		Statement:   p.stmt,
		FunctionDef: p.funcDef,
		Config:      p.config,
		TableInfo:   p.tableInfo,
		Dialect:     normalizeDialect(p.config),
		TypeInfoMap: p.typeInfo,
	}

	// Execute each processor in order
	for _, processor := range p.processors {
		err := processor.Process(ctx)
		if err != nil {
			return nil, fmt.Errorf("processor %s failed: %w", processor.Name(), err)
		}
	}

	// Build the final intermediate format
	responsesRaw, responseWarnings := determineResponseType(ctx.Statement, ctx.TableInfo)
	responses := applyHierarchyKeyLevels(responsesRaw, ctx.TableInfo)

	if len(responses) == 0 {
		fallbackResponses, err := buildDMLReturningResponses(ctx.Statement, ctx.TableInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to build RETURNING responses: %w", err)
		}

		if len(fallbackResponses) > 0 {
			responses = applyHierarchyKeyLevels(fallbackResponses, ctx.TableInfo)
		}
	}

	result := &IntermediateFormat{
		FormatVersion:      "1",
		Description:        ctx.Description,
		FunctionName:       ctx.FunctionName,
		Parameters:         ctx.Parameters,
		CELExpressions:     ctx.CELExpressions,
		CELEnvironments:    ctx.CELEnvironments,
		Envs:               convertEnvironmentsToEnvs(ctx.Environments), // Convert environments to Envs format
		Instructions:       ctx.Instructions,
		ImplicitParameters: ctx.ImplicitParams,
		SystemFields:       ctx.SystemFields,
		ResponseAffinity:   ctx.ResponseAffinity,
		Responses:          responses,
		TableReferences:    ctx.TableReferences, // Add table references
	}

	if len(responseWarnings) > 0 {
		result.Warnings = append(result.Warnings, responseWarnings...)
	}

	return result, nil
}

// convertEnvironmentsToEnvs converts []string environments to [][]EnvVar format
func convertEnvironmentsToEnvs(environments []string) [][]EnvVar {
	if len(environments) == 0 {
		return nil
	}

	var envs [][]EnvVar

	// envs[0] is always empty (parameters only)
	envs = append(envs, []EnvVar{})

	// Build cumulative environments for nested loops
	for i := range environments {
		// Create environment for this level (includes all previous levels + current)
		var envLevel []EnvVar
		for j := 0; j <= i; j++ {
			envLevel = append(envLevel, EnvVar{
				Name: environments[j],
				Type: "any", // Default type for environment variables
			})
		}

		envs = append(envs, envLevel)
	}

	return envs
}

// buildDMLReturningResponses synthesizes response metadata for DML statements with RETURNING
// clauses when type inference fails to populate responses. It leverages schema information and
// the parser AST to map returned columns back to their originating tables and columns.
func buildDMLReturningResponses(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) ([]Response, error) {
	if stmt == nil || len(tableInfo) == 0 {
		return nil, nil
	}

	switch s := stmt.(type) {
	case *parser.InsertIntoStatement:
		if s.Returning == nil || s.Into == nil {
			return nil, nil
		}

		baseTable, aliasMap := resolveTableReference(s.Into.Table, tableInfo)
		if baseTable == "" {
			return nil, nil
		}

		return buildResponsesFromReturningClause(s.Returning, baseTable, aliasMap, tableInfo), nil
	case *parser.UpdateStatement:
		if s.Returning == nil || s.Update == nil {
			return nil, nil
		}

		baseTable, aliasMap := resolveTableReference(s.Update.Table, tableInfo)
		if baseTable == "" {
			return nil, nil
		}

		return buildResponsesFromReturningClause(s.Returning, baseTable, aliasMap, tableInfo), nil
	case *parser.DeleteFromStatement:
		if s.Returning == nil || s.From == nil {
			return nil, nil
		}

		baseTable, aliasMap := resolveTableReference(s.From.Table, tableInfo)
		if baseTable == "" {
			return nil, nil
		}

		return buildResponsesFromReturningClause(s.Returning, baseTable, aliasMap, tableInfo), nil
	default:
		return nil, nil
	}
}

func resolveTableReference(ref cmn.TableReference, tableInfo map[string]*snapsql.TableInfo) (string, map[string]string) {
	aliasMap := map[string]string{}

	canonical := canonicalTableName(ref, tableInfo)
	if canonical == "" {
		return "", aliasMap
	}

	if ref.Name != "" {
		aliasMap[strings.ToLower(ref.Name)] = canonical
	}

	if ref.TableName != "" {
		aliasMap[strings.ToLower(ref.TableName)] = canonical
	}

	aliasMap[strings.ToLower(canonical)] = canonical

	return canonical, aliasMap
}

func canonicalTableName(ref cmn.TableReference, tableInfo map[string]*snapsql.TableInfo) string {
	candidates := []string{ref.TableName, ref.Name}

	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}

		if tbl := lookupTableInfo(tableInfo, cand); tbl != nil {
			return tbl.Name
		}
	}

	return ""
}

func buildResponsesFromReturningClause(returning *parser.ReturningClause, baseTable string, aliasMap map[string]string, tableInfo map[string]*snapsql.TableInfo) []Response {
	if returning == nil {
		return nil
	}

	responses := make([]Response, 0, len(returning.Fields))

	for _, field := range returning.Fields {
		columnName := normalizeColumnName(field.FieldName)
		if columnName == "" {
			columnName = normalizeColumnName(field.OriginalField)
		}

		if columnName == "" {
			continue
		}

		tableName := baseTable

		if field.TableName != "" {
			if mapped := aliasMap[strings.ToLower(field.TableName)]; mapped != "" {
				tableName = mapped
			} else if tbl := lookupTableInfo(tableInfo, field.TableName); tbl != nil {
				tableName = tbl.Name
			}
		}

		tblInfo := lookupTableInfo(tableInfo, tableName)
		if tblInfo == nil {
			continue
		}

		colInfo := lookupColumnInfo(tblInfo, columnName)
		if colInfo == nil {
			continue
		}

		responses = append(responses, Response{
			Name:         columnName,
			Type:         colInfo.DataType,
			IsNullable:   colInfo.Nullable,
			MaxLength:    colInfo.MaxLength,
			Precision:    colInfo.Precision,
			Scale:        colInfo.Scale,
			SourceTable:  tblInfo.Name,
			SourceColumn: columnName,
		})
	}

	return responses
}

func normalizeColumnName(name string) string {
	if name == "" {
		return ""
	}

	trimmed := strings.TrimSpace(name)
	trimmed = strings.Trim(trimmed, "`\"[]")

	if idx := strings.Index(trimmed, "::"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	if dot := strings.LastIndex(trimmed, "."); dot >= 0 {
		trimmed = trimmed[dot+1:]
	}

	return strings.TrimSpace(trimmed)
}

func lookupTableInfo(tableInfo map[string]*snapsql.TableInfo, name string) *snapsql.TableInfo {
	if len(tableInfo) == 0 || name == "" {
		return nil
	}

	if tbl, ok := tableInfo[name]; ok {
		return tbl
	}

	lower := strings.ToLower(name)
	for key, tbl := range tableInfo {
		if strings.EqualFold(key, name) || strings.ToLower(tbl.Name) == lower {
			return tbl
		}
	}

	return nil
}

func lookupColumnInfo(table *snapsql.TableInfo, column string) *snapsql.ColumnInfo {
	if table == nil || column == "" {
		return nil
	}

	if col, ok := table.Columns[column]; ok {
		return col
	}

	lower := strings.ToLower(column)
	for key, col := range table.Columns {
		if strings.EqualFold(key, column) {
			return col
		}

		if strings.ToLower(key) == lower {
			return col
		}
	}

	return nil
}

// CreateDefaultPipeline creates a pipeline with default processors
func CreateDefaultPipeline(stmt parser.StatementNode, funcDef *parser.FunctionDefinition, config *snapsql.Config, tableInfo map[string]*snapsql.TableInfo, typeInfoMap map[string]any) *TokenPipeline {
	pipeline := NewTokenPipeline(stmt, funcDef, config, tableInfo, typeInfoMap)

	// Add processors in order
	pipeline.AddProcessor(&MetadataExtractor{})
	pipeline.AddProcessor(&TableReferencesProcessor{})
	pipeline.AddProcessor(&InstructionGenerator{})
	pipeline.AddProcessor(&ResponseAffinityDetector{})
	pipeline.AddProcessor(&HierarchyKeyLevelProcessor{})

	return pipeline
}

// normalizeDialect returns a normalized dialect string (postgres, mysql, sqlite, mariadb)
func normalizeDialect(cfg *snapsql.Config) string {
	if cfg == nil || cfg.Dialect == "" {
		return "postgres"
	}

	d := strings.ToLower(cfg.Dialect)
	switch d {
	case "postgres", "postgresql", "pg":
		return "postgres"
	case "mysql":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	case "mariadb":
		return "mariadb"
	default:
		return d
	}
}
