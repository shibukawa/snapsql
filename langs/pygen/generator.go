package pygen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

// Generator generates Python code from intermediate format
type Generator struct {
	PackageName string
	OutputPath  string
	Format      *intermediate.IntermediateFormat
	MockPath    string
	Dialect     snapsql.Dialect // Target database dialect (postgres, mysql, sqlite)
}

// Option is a function that configures Generator
type Option func(*Generator)

// WithPackageName sets the package name for generated code
func WithPackageName(name string) Option {
	return func(g *Generator) {
		g.PackageName = name
	}
}

// WithDialect sets the target database dialect
func WithDialect(dialect snapsql.Dialect) Option {
	return func(g *Generator) {
		g.Dialect = dialect
	}
}

// WithOutputPath sets the output path for generated code
func WithOutputPath(path string) Option {
	return func(g *Generator) {
		g.OutputPath = path
	}
}

// WithMockPath sets the mock data path
func WithMockPath(path string) Option {
	return func(g *Generator) {
		g.MockPath = path
	}
}

// New creates a new Generator
func New(format *intermediate.IntermediateFormat, opts ...Option) *Generator {
	g := &Generator{
		PackageName: "generated", // Default package name
		Format:      format,
		Dialect:     "", // Must be specified via WithDialect
	}
	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Generate generates Python code and writes it to the writer
func (g *Generator) Generate(w io.Writer) error {
	// Prepare template data
	data, err := g.prepareTemplateData()
	if err != nil {
		return fmt.Errorf("failed to prepare template data: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("python").Funcs(getTemplateFuncs()).Parse(pythonTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template to buffer first to catch any errors
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write to output
	_, err = w.Write(buf.Bytes())

	return err
}

// prepareTemplateData prepares the data structure for the Python template
func (g *Generator) prepareTemplateData() (*templateData, error) {
	// Generate timestamp
	timestamp := time.Now().Format(time.RFC3339)

	// Initialize template data
	data := &templateData{
		Timestamp:    timestamp,
		FunctionName: g.Format.FunctionName,
		Description:  g.Format.Description,
		Dialect:      g.Dialect,

		// Initialize empty slices
		ResponseStructs: []responseStructData{},
		Parameters:      []parameterData{},
		ImplicitParams:  []implicitParamData{},
		Validations:     []validationData{},

		// Default SQL builder (will be populated later)
		SQLBuilder: sqlBuilderData{
			IsStatic:  true,
			StaticSQL: "SELECT 1",
			Args:      []string{},
		},

		// Default query execution (will be populated later)
		QueryExecution: queryExecutionData{
			Code: "# TODO: Implement query execution",
		},

		// Default return type
		ReturnType:            "Any",
		ReturnTypeDescription: "Query result",
	}

	data.ExplangExpressions = buildExplangExpressionData(g.Format)

	// Process parameters
	params, err := g.processParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to process parameters: %w", err)
	}

	data.Parameters = params

	// Process implicit parameters (system columns)
	implicitParams, err := g.processImplicitParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to process implicit parameters: %w", err)
	}

	data.ImplicitParams = implicitParams
	data.HasImplicitParams = len(implicitParams) > 0

	// Process validations
	data.Validations = g.processValidations(params)
	data.HasValidation = len(data.Validations) > 0

	// Process response structures
	responseStructs, responseStruct, err := processResponseStruct(g.Format)
	if err != nil {
		// No response fields is acceptable for some statement types
		if !errors.Is(err, ErrNoResponseFields) {
			return nil, fmt.Errorf("failed to process response structure: %w", err)
		}
		// For statements with no response (INSERT/UPDATE/DELETE), set appropriate return type
		data.ReturnType = "int"
		data.ReturnTypeDescription = "Number of affected rows"
	} else {
		// Add response struct to template data
		data.ResponseStructs = responseStructs

		// Set return type based on response affinity
		// Note: For "many" affinity, the template will wrap this in AsyncGenerator
		data.ReturnType = responseStruct.ClassName
		data.ReturnTypeDescription = "Result of " + g.Format.FunctionName
	}

	// Process SQL building
	sqlBuilder, err := processSQLBuilder(g.Format, g.Dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to process SQL builder: %w", err)
	}

	data.SQLBuilder = *sqlBuilder

	// Process query execution
	queryExecution, err := generateQueryExecution(g.Format, responseStruct, g.Dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query execution: %w", err)
	}

	data.QueryExecution = *queryExecution

	// Set response affinity for template
	data.ResponseAffinity = g.Format.ResponseAffinity
	if data.ResponseAffinity == "" {
		data.ResponseAffinity = "none"
	}

	// Determine query type from statement type
	queryType := "select"
	if g.Format.StatementType != "" {
		queryType = strings.ToLower(g.Format.StatementType)
	}

	// Process mock data
	mockData := processMockData(g.MockPath, g.Format.FunctionName, data.ResponseAffinity, data.ReturnType, g.Dialect, queryType)
	data.MockData = mockData
	data.HasMock = mockData.HasMock

	// Process WHERE clause metadata for mutations
	data.MutationKind = getMutationKind(g.Format.StatementType)
	if data.MutationKind != "" && g.Format.WhereClauseMeta != nil {
		data.WhereMeta = convertWhereMeta(g.Format.WhereClauseMeta)
	}

	return data, nil
}
