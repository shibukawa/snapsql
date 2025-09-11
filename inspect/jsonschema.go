package inspect

// InspectOptions controls inspect behavior.
type InspectOptions struct {
	InspectMode bool // always true for inspect; hook for future flags
	Strict      bool // if true, abort on partial/unsupported constructs
	Pretty      bool // pretty-print JSON (used by CLI layer)
}

// TableRef describes a referenced table in the query.
type TableRef struct {
	Name     string `json:"name"`
	Alias    string `json:"alias,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Source   string `json:"source"`    // main|join|cte|subquery
	JoinType string `json:"join_type"` // none|inner|left|right|full|cross|natural|natural_left|natural_right|natural_full|unknown
}

// InspectResult is the JSON-serializable output model.
type InspectResult struct {
	Statement string     `json:"statement"`
	Tables    []TableRef `json:"tables"`
	Notes     []string   `json:"notes,omitempty"`
}
