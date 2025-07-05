package parsercommon

// TableName represents a table name
type TableName struct {
	Name   string
	Schema string // Optional schema name
}

func (n TableName) String() string {
	return "TABLE"
}

// FieldName represents a field/column name
type FieldName struct {
	Name      string
	TableName string // Optional table qualifier
}

func (n FieldName) String() string {
	return "FIELD"
}

// SelectItem represents an item in SELECT clause
type SelectItem struct {
	Expression AstNode // Expression, FieldName, etc.
	Alias      string  // Optional alias
}

func (n SelectItem) String() string {
	return "SELECT_ITEM"
}

// TableReference represents a table reference in FROM clause
type TableReference struct {
	Table AstNode // TableName, SubQuery, JoinClause, etc.
	Alias string  // Optional alias
}

func (n TableReference) String() string {
	return "TABLE_REF"
}
