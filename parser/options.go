package parser

// Options controls parser behaviors that can be relaxed or enabled.
type Options struct {
	// InspectMode relaxes validations intended for code generation/runtime execution.
	// When true, parser steps that require strict directive/variable checks may skip them
	// in favor of extracting structural information.
	InspectMode bool
}

// DefaultOptions provides the default parser options (all strict validations enabled).
var DefaultOptions = Options{}
