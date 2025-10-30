package snapsql

// MockTestCase represents a mock scenario described in a Markdown test case.
type MockTestCase struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	Responses      []MockResponse `json:"responses"`
	ResultOrdered  bool           `json:"result_ordered,omitempty"`
	SlowQueryMilli int64          `json:"slow_query_threshold_ms,omitempty"`
	VerifyQuery    string         `json:"verify_query,omitempty"`
}

// MockResponse models a single response emitted by a mock test case.
type MockResponse struct {
	Expected []map[string]any       `json:"expected,omitempty"`
	Tables   []MockTableExpectation `json:"tables,omitempty"`
	Error    *MockError             `json:"error,omitempty"`
	Result   *MockSQLResult         `json:"result,omitempty"`
}

// MockTableExpectation captures expectations for a specific table.
type MockTableExpectation struct {
	Table        string           `json:"table"`
	Strategy     string           `json:"strategy,omitempty"`
	Rows         []map[string]any `json:"rows,omitempty"`
	ExternalFile string           `json:"external_file,omitempty"`
}

// MockSQLResult encodes a synthetic sql.Result outcome.
type MockSQLResult struct {
	RowsAffected *int64 `json:"rows_affected,omitempty"`
	LastInsertID *int64 `json:"last_insert_id,omitempty"`
}

// MockError describes an error returned by a mock response.
type MockError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
