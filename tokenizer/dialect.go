package tokenizer

// KeywordInfo holds information about a SQL keyword.
type KeywordInfo struct {
	// Keyword is always true (for quick lookup)
	Keyword bool
	// StrictReserved is true if the keyword is strict reserved in any major DB (PostgreSQL/MySQL/SQLite)
	StrictReserved bool
}

// KeywordSet is a map of all reserved/used SQL keywords (all upper-case) with strict reserved info.
// SnapSQL SQL keyword list for all supported DBMS (strictest union, reserved only)
// This file provides a unified keyword set for parser-level reserved word checks.
// No dialect logic, just a single set.
var KeywordSet = map[string]KeywordInfo{
	// Row locking and concurrency control
	"SHARE": {true, true}, "NO": {true, true}, "NOWAIT": {true, true}, "SKIP": {true, true}, "LOCKED": {true, true},
	// --- Common SQL reserved words (strictest union of PostgreSQL, MySQL, SQLite) ---
	"ALTER": {true, true}, "ASC": {true, true}, "BETWEEN": {true, true},
	"CASE": {true, true}, "CHECK": {true, true}, "COLUMN": {true, false}, "CONSTRAINT": {true, true}, "CREATE": {true, true}, "CROSS": {true, true},
	"CURRENT_DATE": {true, true}, "CURRENT_TIME": {true, true}, "CURRENT_TIMESTAMP": {true, true}, "DATABASE": {true, true},
	"DEFAULT": {true, true}, "DESC": {true, true}, "DROP": {true, true}, "ELSE": {true, true},
	"END": {true, true}, "EXCEPT": {true, true}, "EXISTS": {true, true}, "FOREIGN": {true, true},
	"FULL": {true, true}, "IF": {true, true}, "IN": {true, true}, "INDEX": {true, true},
	"INNER": {true, true}, "INTERSECT": {true, true}, "IS": {true, true}, "JOIN": {true, true},
	"KEY": {true, true}, "LEFT": {true, true}, "LIKE": {true, true}, "MATCH": {true, true}, "NATURAL": {true, true},
	"OUTER": {true, true}, "PRIMARY": {true, true},
	"REFERENCES": {true, true}, "RIGHT": {true, true}, "TABLE": {true, true}, "THEN": {true, true},
	"TO": {true, true}, "UNION": {true, true}, "UNIQUE": {true, true}, "USING": {true, true},
	"VIEW": {true, true}, "WHEN": {true, true},

	"NULLS": {true, false}, "FIRST": {true, false}, "LAST": {true, false},

	// --- PostgreSQL/extended (strict reserved) ---
	"SIMILAR": {true, true}, "OVER": {true, true}, "PARTITION": {true, true}, "RANGE": {true, true}, "ROWS": {true, true},
	"UNBOUNDED": {true, true}, "PRECEDING": {true, true}, "FOLLOWING": {true, true}, "CURRENT": {true, true}, "ROW": {true, true},
	"RETURNING": {true, true}, "WINDOW": {true, true}, "LATERAL": {true, true}, "ONLY": {true, true},
	"VARIADIC": {true, true}, "VERBOSE": {true, true}, "SETOF": {true, true}, "USER": {true, true},

	// --- MySQL/SQLite/extended (strict reserved) ---
	"REGEXP": {true, true}, "XOR": {true, true}, "REPLACE": {true, true}, "SHOW": {true, true}, "TRIGGER": {true, true},
	"UNLOCK": {true, true}, "ZEROFILL": {true, true}, "MOD": {true, true}, "DIV": {true, true}, "LOCK": {true, true}, "UNSIGNED": {true, true}, "SIGNED": {true, true},
	"STRAIGHT_JOIN": {true, true}, "SQL_BIG_RESULT": {true, true}, "SQL_CALC_FOUND_ROWS": {true, true}, "SQL_SMALL_RESULT": {true, true},
	"HIGH_PRIORITY": {true, true}, "LOW_PRIORITY": {true, true}, "DELAYED": {true, true}, "IGNORE": {true, true},

	// --- SQL standard/common function and type keywords (PostgreSQL/MySQL/SQLite strict reserved or widely used) ---
	"COALESCE": {true, false}, "GREATEST": {true, false}, "LEAST": {true, false}, "NULLIF": {true, false}, "OVERLAY": {true, false}, "POSITION": {true, false}, "SUBSTRING": {true, false}, "TRIM": {true, false},
	"BIT": {true, false}, "BOOLEAN": {true, false}, "BINARY": {true, false}, "BOTH": {true, false}, "CHAR": {true, false}, "CHARACTER": {true, false}, "NATIONAL": {true, false}, "NCHAR": {true, false}, "NVARCHAR": {true, false}, "VARYING": {true, false},

	// XML-related (PostgreSQL)
	"XMLATTRIBUTES": {true, false}, "XMLCONCAT": {true, false}, "XMLELEMENT": {true, false}, "XMLEXISTS": {true, false}, "XMLFOREST": {true, false}, "XMLPARSE": {true, false}, "XMLPI": {true, false}, "XMLROOT": {true, false}, "XMLSERIALIZE": {true, false},

	// Other common reserved/used
	"AUTHORIZATION": {true, false}, "CHECKPOINT": {true, false}, "CLUSTER": {true, false}, "COMMENT": {true, false}, "CONCURRENTLY": {true, false}, "CYCLE": {true, false}, "DEALLOCATE": {true, false}, "DISCARD": {true, false}, "DO": {true, false}, "FREEZE": {true, false}, "LISTEN": {true, false}, "LOAD": {true, false}, "MOVE": {true, false}, "NOTIFY": {true, false}, "OUT": {true, false}, "PREPARE": {true, false}, "REASSIGN": {true, false}, "REFRESH": {true, false}, "REINDEX": {true, false}, "RELEASE": {true, false}, "RESET": {true, false}, "RESTART": {true, false}, "REVOKE": {true, false}, "SECURITY": {true, false}, "SEQUENCE": {true, false}, "UNLISTEN": {true, false}, "UNLOGGED": {true, false}, "VACUUM": {true, false},

	// Types
	"BIGINT": {true, false}, "INT": {true, false}, "INTEGER": {true, false}, "SMALLINT": {true, false}, "DEC": {true, false}, "DECIMAL": {true, false}, "NUMERIC": {true, false}, "REAL": {true, false}, "FLOAT": {true, false}, "DOUBLE": {true, false}, "PRECISION": {true, false}, "SERIAL": {true, false}, "SMALLSERIAL": {true, false}, "BIGSERIAL": {true, false}, "MONEY": {true, false}, "DATE": {true, false}, "TIME": {true, false}, "TIMESTAMP": {true, false}, "INTERVAL": {true, false}, "TEXT": {true, false}, "UUID": {true, false}, "JSON": {true, false}, "JSONB": {true, false}, "BYTEA": {true, false}, "ARRAY": {true, false}, "ENUM": {true, false},

	// SQLite/compat
	"AUTOINCREMENT": {true, false}, "GLOB": {true, false}, "PRAGMA": {true, false}, "RECURSIVE": {true, false}, "TEMP": {true, false}, "TEMPORARY": {true, false}, "WITHOUT": {true, false},
}
