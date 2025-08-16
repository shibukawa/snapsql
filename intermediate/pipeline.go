package intermediate

import (
	"fmt"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
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
	}

	// Execute each processor in order
	for _, processor := range p.processors {
		err := processor.Process(ctx)
		if err != nil {
			return nil, fmt.Errorf("processor %s failed: %w", processor.Name(), err)
		}
	}

	// Build the final intermediate format
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
		Responses:          determineResponseType(ctx.Statement, ctx.TableInfo), // Add type inference result
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

// CreateDefaultPipeline creates a pipeline with default processors
func CreateDefaultPipeline(stmt parser.StatementNode, funcDef *parser.FunctionDefinition, config *snapsql.Config, tableInfo map[string]*snapsql.TableInfo) *TokenPipeline {
	pipeline := NewTokenPipeline(stmt, funcDef, config, tableInfo)

	// Add processors in order
	pipeline.AddProcessor(&MetadataExtractor{})
	pipeline.AddProcessor(&CELExpressionExtractor{})
	pipeline.AddProcessor(&SystemFieldProcessor{})
	pipeline.AddProcessor(&TokenTransformer{})
	pipeline.AddProcessor(&InstructionGenerator{})
	pipeline.AddProcessor(&ResponseAffinityDetector{})

	return pipeline
}
