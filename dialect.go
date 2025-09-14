package snapsql

// Dialect represents supported database dialects
// This type is shared across all packages
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectMySQL    Dialect = "mysql"
	DialectSQLite   Dialect = "sqlite"
	DialectMariaDB  Dialect = "mariadb"
)

// Feature represents DB-specific feature flags
type Feature int

const (
	FeatureConcat         Feature = iota + 1
	FeatureConcatOperator         // ||
	FeatureConcatFunction         // CONCAT()
	FeatureJson                   // JSON系関数
	FeatureArray                  // ARRAY系関数
	// Add more features as needed
)
