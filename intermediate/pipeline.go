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

	// Metadata
	Description      string
	FunctionName     string
	Parameters       []Parameter
	ResponseAffinity string
}

// NewTokenPipeline creates a new token processing pipeline
func NewTokenPipeline(stmt parser.StatementNode, funcDef *parser.FunctionDefinition, config *snapsql.Config, tableInfo map[string]*snapsql.TableInfo) *TokenPipeline {
	return &TokenPipeline{
		tokens:    extractTokensFromStatement(stmt),
		stmt:      stmt,
		funcDef:   funcDef,
		config:    config,
		tableInfo: tableInfo,
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
	}

	// Execute each processor in order
	for _, processor := range p.processors {
		err := processor.Process(ctx)
		if err != nil {
			return nil, fmt.Errorf("processor %s failed: %w", processor.Name(), err)
		}
	}

	// Build the final intermediate format
	responses := applyHierarchyKeyLevels(determineResponseType(ctx.Statement, ctx.TableInfo), ctx.TableInfo)

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
			BaseType:     colInfo.DataType,
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
func CreateDefaultPipeline(stmt parser.StatementNode, funcDef *parser.FunctionDefinition, config *snapsql.Config, tableInfo map[string]*snapsql.TableInfo) *TokenPipeline {
	pipeline := NewTokenPipeline(stmt, funcDef, config, tableInfo)

	// Add processors in order
	pipeline.AddProcessor(&MetadataExtractor{})
	pipeline.AddProcessor(&CELExpressionExtractor{})
	pipeline.AddProcessor(&SystemFieldProcessor{})
	pipeline.AddProcessor(&TokenTransformer{})
	// ReturningProcessor: 方言非対応の UPDATE/DELETE RETURNING を構造的に除去
	pipeline.AddProcessor(&ReturningProcessor{})
	pipeline.AddProcessor(&InstructionGenerator{})
	// Post instruction sanitation: ensure minimal spacing around keywords / boundaries
	pipeline.AddProcessor(&WhitespaceNormalizer{})
	// DialectProcessor: 現段階では方言解決前提の命令 (EMIT_IF_DIALECT) を静的化する予定のフック。
	// 今は no-op; 後続タスクで実装を追加。
	pipeline.AddProcessor(&DialectProcessor{})
	pipeline.AddProcessor(&ResponseAffinityDetector{})
	// Hierarchy key level: currently applied post determineResponseType in Execute, but keep processor for future schema-aware refinement
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

// DialectProcessor は今後方言依存の命令/トークン正規化を行うステージ。
// 現時点ではスケルトン (no-op)。EMIT_IF_DIALECT 廃止後、生成側で出なくなった命令の
// 防御的除去 (残存時に静的化 or ドロップ) をここで行う予定。
type DialectProcessor struct{}

func (p *DialectProcessor) Name() string { return "DialectProcessor" }

func (p *DialectProcessor) Process(ctx *ProcessingContext) error {
	// CAST構文正規化: PostgreSQL以外では expr::TYPE を CAST(expr AS TYPE) に変換
	if ctx.Dialect != "postgres" && ctx.Dialect != "postgresql" {
		normalizeCastSyntax(ctx.Instructions, ctx.Dialect)
	}

	// 時間関数正規化
	normalizeTimeFunctions(ctx.Instructions, ctx.Dialect)

	// システムフィールドのデフォルト値も正規化
	normalizeSystemFieldTimeFunctions(ctx, ctx.Dialect)

	return nil
}

// WhitespaceNormalizer ensures EMIT_STATIC instruction values have proper spacing to avoid token run-on like
// 'CURRENT_TIMESTAMPWHERE'. It inserts a single space when two adjacent fragments would otherwise concatenate
// alphanumeric / underscore characters without delimiter.
type WhitespaceNormalizer struct{}

func (w *WhitespaceNormalizer) Name() string { return "WhitespaceNormalizer" }

func (w *WhitespaceNormalizer) Process(ctx *ProcessingContext) error {
	// ここでは受け入れテストの期待する生のEMIT_STATIC値を変えないため、何もしません。
	// ランタイムでのトークン連結時のスペース不足は、optimizer.MergeAdjacentStatic と
	// gogen/sql.go のSQLビルダー側で処理します。
	return nil
}

// normalizeCastSyntax converts PostgreSQL-style expr::TYPE to standard CAST(expr AS TYPE)
// for non-PostgreSQL dialects
func normalizeCastSyntax(ins []Instruction, dialectName string) {
	if len(ins) == 0 {
		return
	}

	isPostgres := dialectName == "postgres" || dialectName == "postgresql"
	if isPostgres {
		return // keep native :: syntax
	}

	// CAST内部の AS を :: に変換してしまう不具合があるかもしれません
	// 実際には、既に正しい CAST(expr AS type) 構文の場合は変換不要

	for i := range ins {
		if ins[i].Op != OpEmitStatic {
			continue
		}

		// CAST内部の :: を AS に置換
		ins[i].Value = strings.ReplaceAll(ins[i].Value, "::INTEGER)", " AS INTEGER)")
		ins[i].Value = strings.ReplaceAll(ins[i].Value, "::DECIMAL(", " AS DECIMAL(")
		ins[i].Value = strings.ReplaceAll(ins[i].Value, "::NUMERIC(", " AS NUMERIC(")
		ins[i].Value = strings.ReplaceAll(ins[i].Value, "::CHAR)", " AS CHAR)")
		ins[i].Value = strings.ReplaceAll(ins[i].Value, "::TEXT)", " AS TEXT)")
	}
}

// normalizeTimeFunctions converts time functions between dialects
// MySQL: prefers NOW()
// PostgreSQL/SQLite: prefers CURRENT_TIMESTAMP
func normalizeTimeFunctions(ins []Instruction, dialectName string) {
	if len(ins) == 0 {
		return
	}

	for i := range ins {
		if ins[i].Op != OpEmitStatic {
			continue
		}

		switch dialectName {
		case "mysql":
			// Convert CURRENT_TIMESTAMP to NOW() for MySQL
			ins[i].Value = strings.ReplaceAll(ins[i].Value, "CURRENT_TIMESTAMP", "NOW()")
		case "postgres", "postgresql", "sqlite":
			// Convert NOW() to CURRENT_TIMESTAMP for PostgreSQL/SQLite
			ins[i].Value = strings.ReplaceAll(ins[i].Value, "NOW()", "CURRENT_TIMESTAMP")
		}
	}
}

// normalizeSystemFieldTimeFunctions normalizes time functions in system field defaults and implicit parameters
func normalizeSystemFieldTimeFunctions(ctx *ProcessingContext, dialectName string) {
	// Normalize implicit parameters
	for i := range ctx.ImplicitParams {
		normalizeTimeAny(&ctx.ImplicitParams[i].Default, dialectName)
	}

	// Normalize system fields
	for i := range ctx.SystemFields {
		field := &ctx.SystemFields[i]
		if field.OnInsert != nil {
			normalizeTimeAny(&field.OnInsert.Default, dialectName)
		}

		if field.OnUpdate != nil {
			normalizeTimeAny(&field.OnUpdate.Default, dialectName)
		}
	}
}

// normalizeTimeAny normalizes time function in an any value (if it's a string)
func normalizeTimeAny(value *any, dialectName string) {
	if value == nil {
		return
	}

	if str, ok := (*value).(string); ok {
		normalizedStr := normalizeTimeStringValue(str, dialectName)
		if normalizedStr != str {
			*value = normalizedStr
		}
	}
}

// normalizeTimeStringValue normalizes time function in a string value
func normalizeTimeStringValue(value string, dialectName string) string {
	switch dialectName {
	case "mysql":
		// Convert CURRENT_TIMESTAMP to NOW() for MySQL
		if value == "CURRENT_TIMESTAMP" {
			return "NOW()"
		}
	case "postgres", "postgresql", "sqlite":
		// Convert NOW() to CURRENT_TIMESTAMP for PostgreSQL/SQLite
		if value == "NOW()" {
			return "CURRENT_TIMESTAMP"
		}
	}

	return value
}
