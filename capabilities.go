package snapsql

// Capabilities defines which SQL features are supported by each dialect
var Capabilities = map[Dialect]map[Feature]bool{
	DialectPostgres: {
		FeatureConcat:         true,
		FeatureConcatOperator: true,
		FeatureConcatFunction: true,
		FeatureJson:           true,
		FeatureArray:          true,
	},
	DialectMySQL: {
		FeatureConcat:         true,
		FeatureConcatOperator: false,
		FeatureConcatFunction: true,
		FeatureJson:           true,
		FeatureArray:          false,
	},
	DialectSQLite: {
		FeatureConcat:         true,
		FeatureConcatOperator: true,
		FeatureConcatFunction: false,
		FeatureJson:           false,
		FeatureArray:          false,
	},
}
