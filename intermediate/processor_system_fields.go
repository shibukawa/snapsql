package intermediate

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser/parserstep5"
)

// SystemFieldProcessor handles system field validation and processing
type SystemFieldProcessor struct{}

func (s *SystemFieldProcessor) Name() string {
	return "SystemFieldProcessor"
}

func (s *SystemFieldProcessor) Process(ctx *ProcessingContext) error {
	// Extract system fields information from config
	if ctx.Config != nil {
		ctx.SystemFields = extractSystemFieldsInfo(ctx.Config, ctx.Statement)

		// Perform system field validation and get implicit parameters
		systemFieldErr := &parserstep5.GenerateError{}
		ctx.ImplicitParams = CheckSystemFields(ctx.Statement, ctx.Config, ctx.Parameters, systemFieldErr)

		// Check if there were any system field validation errors
		if systemFieldErr.HasErrors() {
			return fmt.Errorf("system field validation failed: %w", systemFieldErr)
		}
	}

	return nil
}
