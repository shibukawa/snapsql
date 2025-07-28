package intermediate

// CELExpressionExtractor extracts CEL expressions and environment variables
type CELExpressionExtractor struct{}

func (c *CELExpressionExtractor) Name() string {
	return "CELExpressionExtractor"
}

func (c *CELExpressionExtractor) Process(ctx *ProcessingContext) error {
	// Extract CEL expressions and environment variables from the statement
	expressions, envs := ExtractFromStatement(ctx.Statement)

	ctx.Expressions = expressions

	// Convert [][]EnvVar to []string for now (simplified)
	var envStrings []string
	for _, envGroup := range envs {
		for _, env := range envGroup {
			envStrings = append(envStrings, env.Name)
		}
	}
	ctx.Environments = envStrings

	return nil
}
