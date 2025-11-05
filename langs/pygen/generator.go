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
	// Validate dialect is specified
	if g.Dialect == "" {
		return errors.New("dialect must be specified")
	}

	// Validate supported dialects
	switch g.Dialect {
	case snapsql.DialectPostgres, snapsql.DialectMySQL, snapsql.DialectSQLite:
		// Valid dialects
	default:
		return fmt.Errorf("unsupported dialect: %s (supported: postgres, mysql, sqlite)", g.Dialect)
	}

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
		Dialect:      string(g.Dialect),

		// Initialize empty slices
		ResponseStructs: []responseStructData{},
		CELEnvironments: []celEnvironmentData{},
		CELPrograms:     []celProgramData{},
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

	// Check for CEL usage
	data.HasCEL = len(g.Format.CELEnvironments) > 0 || len(g.Format.CELExpressions) > 0

	// Process CEL environments
	if data.HasCEL {
		celEnvs, err := processCELEnvironments(g.Format)
		if err != nil {
			return nil, fmt.Errorf("failed to process CEL environments: %w", err)
		}

		data.CELEnvironments = celEnvs

		// Process CEL programs
		celProgs, err := generateCELPrograms(g.Format, celEnvs)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CEL programs: %w", err)
		}

		data.CELPrograms = celProgs
	}

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
	responseStruct, err := processResponseStruct(g.Format)
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
		data.ResponseStructs = []responseStructData{*responseStruct}

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
	queryExecution, err := generateQueryExecution(g.Format, responseStruct, string(g.Dialect))
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
	mockData := processMockData(g.MockPath, g.Format.FunctionName, data.ResponseAffinity, data.ReturnType, string(g.Dialect), queryType)
	data.MockData = mockData
	data.HasMock = mockData.HasMock

	// Process WHERE clause metadata for mutations
	data.MutationKind = getMutationKind(g.Format.StatementType)
	if data.MutationKind != "" && g.Format.WhereClauseMeta != nil {
		data.WhereMeta = convertWhereMeta(g.Format.WhereClauseMeta)
	}

	return data, nil
}

// convertToCELType converts a SnapSQL type to a CEL type name
func convertToCELType(snapType string) string {
	// Handle arrays
	if before, ok := strings.CutSuffix(snapType, "[]"); ok {
		baseType := before
		baseCELType := convertToCELType(baseType)

		return "ListType(celpy." + baseCELType + ")"
	}

	// Normalize temporal aliases
	normalized := normalizeTemporalAlias(snapType)

	switch normalized {
	case "int", "int32", "int64":
		return "IntType"
	case "string":
		return "StringType"
	case "bool":
		return "BoolType"
	case "float", "float32", "float64", "double":
		return "DoubleType"
	case "decimal":
		return "DoubleType"
	case "timestamp", "datetime", "date", "time":
		return "TimestampType"
	case "bytes":
		return "BytesType"
	case "any":
		return "AnyType"
	default:
		return "DynType"
	}
}

// processCELEnvironments processes CEL environments and returns template data for Python
func processCELEnvironments(format *intermediate.IntermediateFormat) ([]celEnvironmentData, error) {
	envs := make([]celEnvironmentData, len(format.CELEnvironments))

	for i, env := range format.CELEnvironments {
		envData := celEnvironmentData{
			Index:     env.Index,
			Container: env.Container,
			Variables: make([]celVariableData, 0),
		}

		// Set default container name if empty
		if envData.Container == "" {
			if env.Index == 0 {
				envData.Container = "root"
			} else {
				envData.Container = fmt.Sprintf("env_%d", env.Index)
			}
		}

		// Process variables from parameters for environment 0
		if i == 0 {
			for _, param := range format.Parameters {
				envData.Variables = append(envData.Variables, celVariableData{
					Name:    param.Name,
					CELType: convertToCELType(param.Type),
				})
			}
		}

		// Process additional variables
		for _, v := range env.AdditionalVariables {
			envData.Variables = append(envData.Variables, celVariableData{
				Name:    v.Name,
				CELType: convertToCELType(v.Type),
			})
		}

		envs[i] = envData
	}

	return envs, nil
}

// generateCELPrograms generates CEL program initialization code for Python
func generateCELPrograms(format *intermediate.IntermediateFormat, envs []celEnvironmentData) ([]celProgramData, error) {
	programs := make([]celProgramData, len(format.CELExpressions))

	for i, expr := range format.CELExpressions {
		program := celProgramData{
			Index:          i,
			Expression:     expr.Expression,
			EnvironmentIdx: expr.EnvironmentIndex,
		}
		programs[i] = program
	}

	return programs, nil
}
