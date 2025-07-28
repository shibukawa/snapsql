package intermediate

// ResponseAffinityDetector detects response affinity based on statement type
type ResponseAffinityDetector struct{}

func (r *ResponseAffinityDetector) Name() string {
	return "ResponseAffinityDetector"
}

func (r *ResponseAffinityDetector) Process(ctx *ProcessingContext) error {
	// Use existing DetermineResponseAffinity function
	affinity := DetermineResponseAffinity(ctx.Statement, ctx.TableInfo)
	ctx.ResponseAffinity = string(affinity)
	return nil
}
