package parsercommon

// SubqueryAnalysisResult contains subquery analysis information from parserstep7
// This provides a public interface to parserstep7 functionality without exposing internal types
type SubqueryAnalysisResult struct {
	HasSubqueries    bool                   // Whether the statement contains subqueries
	SubqueryTables   []string               // List of available subquery table names
	FieldSources     map[string]interface{} // Field source information (opaque interface)
	TableReferences  map[string]interface{} // Table reference information (opaque interface)
	DependencyInfo   interface{}            // Dependency graph information (opaque interface)
	ProcessingOrder  []string               // Recommended processing order for subqueries
	ValidationErrors []ValidationError      // Validation errors from subquery analysis
	HasErrors        bool                   // Whether analysis had errors
}

// ValidationError represents a validation error from subquery analysis
type ValidationError struct {
	ErrorType string // Type of validation error
	Message   string // Error message
	NodeID    string // Related node ID (if applicable)
	Position  string // Position information (if applicable)
}

// GetSubqueryTables returns available subquery table names
func (sai *SubqueryAnalysisResult) GetSubqueryTables() []string {
	if sai == nil {
		return []string{}
	}
	return sai.SubqueryTables
}

// HasSubqueryAnalysis returns true if subquery analysis information is available
func (sai *SubqueryAnalysisResult) HasSubqueryAnalysis() bool {
	return sai != nil && sai.HasSubqueries
}

// GetProcessingOrder returns the recommended processing order for subqueries
func (sai *SubqueryAnalysisResult) GetProcessingOrder() []string {
	if sai == nil {
		return []string{}
	}
	return sai.ProcessingOrder
}

// GetValidationErrors returns validation errors from subquery analysis
func (sai *SubqueryAnalysisResult) GetValidationErrors() []ValidationError {
	if sai == nil {
		return []ValidationError{}
	}
	return sai.ValidationErrors
}

// HasAnalysisErrors returns true if subquery analysis had errors
func (sai *SubqueryAnalysisResult) HasAnalysisErrors() bool {
	return sai != nil && sai.HasErrors
}
