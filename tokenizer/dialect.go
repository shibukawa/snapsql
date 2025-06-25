package tokenizer

import "strings"

// Define keyword maps as package global variables
var (
	postgresqlKeywordsMap = map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true,
		"AND": true, "OR": true, "NOT": true, "IN": true, "EXISTS": true, "BETWEEN": true,
		"LIKE": true, "IS": true, "NULL": true, "ORDER": true, "BY": true, "GROUP": true,
		"HAVING": true, "UNION": true, "ALL": true, "DISTINCT": true, "AS": true, "WITH": true,
		"OVER": true, "PARTITION": true, "ROWS": true, "RANGE": true, "UNBOUNDED": true,
		"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true,
		"RETURNING": true, "ARRAY_AGG": true,
	}

	postgresqlReservedWordsMap = map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true,
	}

	mysqlKeywordsMap = map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true,
		"AND": true, "OR": true, "NOT": true, "IN": true, "EXISTS": true, "BETWEEN": true,
		"LIKE": true, "IS": true, "NULL": true, "ORDER": true, "BY": true, "GROUP": true,
		"HAVING": true, "UNION": true, "ALL": true, "DISTINCT": true, "AS": true, "WITH": true,
		"OVER": true, "PARTITION": true, "ROWS": true, "RANGE": true, "UNBOUNDED": true,
		"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true,
		"LIMIT": true, "OFFSET": true,
	}

	mysqlReservedWordsMap = map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true,
	}

	sqliteKeywordsMap = map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true,
		"AND": true, "OR": true, "NOT": true, "IN": true, "EXISTS": true, "BETWEEN": true,
		"LIKE": true, "IS": true, "NULL": true, "ORDER": true, "BY": true, "GROUP": true,
		"HAVING": true, "UNION": true, "ALL": true, "DISTINCT": true, "AS": true, "WITH": true,
		"OVER": true, "PARTITION": true, "ROWS": true, "RANGE": true, "UNBOUNDED": true,
		"PRECEDING": true, "FOLLOWING": true, "CURRENT": true, "ROW": true,
		"LIMIT": true,
	}

	sqliteReservedWordsMap = map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
		"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true,
	}
)

// SqlDialect is the interface for database dialects
type SqlDialect interface {
	Name() string
	IsKeyword(word string) bool
	IsReservedWord(word string) bool
	QuoteIdentifier(identifier string) string
	QuoteLiteral(literal string) string
}

// PostgreSQLDialect is the implementation for PostgreSQL dialect
type PostgreSQLDialect struct{}

// NewPostgreSQLDialect creates a PostgreSQL dialect
func NewPostgreSQLDialect() *PostgreSQLDialect {
	return &PostgreSQLDialect{}
}

func (d *PostgreSQLDialect) Name() string {
	return "PostgreSQL"
}

func (d *PostgreSQLDialect) IsKeyword(word string) bool {
	return postgresqlKeywordsMap[strings.ToUpper(word)]
}

func (d *PostgreSQLDialect) IsReservedWord(word string) bool {
	return postgresqlReservedWordsMap[strings.ToUpper(word)]
}

func (d *PostgreSQLDialect) QuoteIdentifier(identifier string) string {
	return `"` + identifier + `"`
}

func (d *PostgreSQLDialect) QuoteLiteral(literal string) string {
	return `'` + strings.ReplaceAll(literal, `'`, `''`) + `'`
}

// MySQLDialect is the implementation for MySQL dialect
type MySQLDialect struct{}

// NewMySQLDialect creates a MySQL dialect
func NewMySQLDialect() *MySQLDialect {
	return &MySQLDialect{}
}

func (d *MySQLDialect) Name() string {
	return "MySQL"
}

func (d *MySQLDialect) IsKeyword(word string) bool {
	return mysqlKeywordsMap[strings.ToUpper(word)]
}

func (d *MySQLDialect) IsReservedWord(word string) bool {
	return mysqlReservedWordsMap[strings.ToUpper(word)]
}

func (d *MySQLDialect) QuoteIdentifier(identifier string) string {
	return "`" + identifier + "`"
}

func (d *MySQLDialect) QuoteLiteral(literal string) string {
	return `'` + strings.ReplaceAll(literal, `'`, `''`) + `'`
}

// SQLiteDialect is the implementation for SQLite dialect
type SQLiteDialect struct{}

// NewSQLiteDialect creates a SQLite dialect
func NewSQLiteDialect() *SQLiteDialect {
	return &SQLiteDialect{}
}

func (d *SQLiteDialect) Name() string {
	return "SQLite"
}

func (d *SQLiteDialect) IsKeyword(word string) bool {
	return sqliteKeywordsMap[strings.ToUpper(word)]
}

func (d *SQLiteDialect) IsReservedWord(word string) bool {
	return sqliteReservedWordsMap[strings.ToUpper(word)]
}

func (d *SQLiteDialect) QuoteIdentifier(identifier string) string {
	return `"` + identifier + `"`
}

func (d *SQLiteDialect) QuoteLiteral(literal string) string {
	return `'` + strings.ReplaceAll(literal, `'`, `''`) + `'`
}

// DetectDialect automatically detects the dialect from SQL content
func DetectDialect(sql string) SqlDialect {
	upperSQL := strings.ToUpper(sql)

	// PostgreSQL specific
	if strings.Contains(upperSQL, "RETURNING") ||
		strings.Contains(upperSQL, "ARRAY_AGG") ||
		strings.Contains(upperSQL, "::") { // cast operator
		return NewPostgreSQLDialect()
	}

	// MySQL specific
	if (strings.Contains(upperSQL, "LIMIT") && strings.Contains(upperSQL, "OFFSET")) ||
		strings.Contains(upperSQL, "`") { // backtick
		return NewMySQLDialect()
	}

	// Default is SQLite
	return NewSQLiteDialect()
}
