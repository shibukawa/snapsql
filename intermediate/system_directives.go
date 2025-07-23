package intermediate

// NewSystemDirectives creates a new system directives map with default values
func NewSystemDirectives() map[string]interface{} {
	return map[string]interface{}{
		"version":      "1.0",
		"engine":       "snapsql",
		"allow_unsafe": false, // Default to safe mode
	}
}
