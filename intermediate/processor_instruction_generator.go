package intermediate

// InstructionGenerator generates execution instructions from tokens
type InstructionGenerator struct{}

func (i *InstructionGenerator) Name() string {
	return "InstructionGenerator"
}

func (i *InstructionGenerator) Process(ctx *ProcessingContext) error {
	// Use existing GenerateInstructions function for all advanced features
	// The TokenTransformer should have already added system field tokens
	instructions := GenerateInstructions(ctx.Tokens, ctx.Expressions)

	// Detect SQL patterns that need dialect-specific handling
	dialectConversions := detectDialectPatterns(ctx.Tokens)

	// Insert dialect-specific instructions where needed
	instructions = insertDialectInstructions(instructions, dialectConversions)

	// Set env_index in loop instructions based on environments
	if len(ctx.Environments) > 0 {
		envs := convertEnvironmentsToEnvs(ctx.Environments)
		setEnvIndexInInstructions(envs, instructions)
	}

	ctx.Instructions = instructions
	return nil
}
