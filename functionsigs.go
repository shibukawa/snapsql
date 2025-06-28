package snapsql

// FunctionSignature defines the return type and nullability for a SQL function
// ReturnTypeByArg: trueなら最初の引数の型を返す
// NullableByArg: trueなら引数のnullableを伝搬
// CastType: trueならCAST(... AS type)の型を返す
type FunctionSignature struct {
	ReturnType      string
	ReturnTypeByArg bool
	Nullable        bool
	NullableByArg   bool
	CastType        bool
}

// FunctionSignatures maps Dialect to function name to signature
var FunctionSignatures = map[Dialect]map[string]FunctionSignature{
	DialectPostgres: {
		"LENGTH":    {ReturnType: "int", NullableByArg: true},
		"COALESCE":  {ReturnTypeByArg: true, NullableByArg: true},
		"IFNULL":    {ReturnTypeByArg: true, NullableByArg: true},
		"CAST":      {CastType: true, NullableByArg: true},
		"UPPER":     {ReturnType: "string", NullableByArg: true},
		"NOW":       {ReturnType: "time", Nullable: false},
		"DATE_ADD":  {ReturnType: "time", NullableByArg: true},
		"SUBSTRING": {ReturnType: "string", NullableByArg: true},
		"TRIM":      {ReturnType: "string", NullableByArg: true},
		// ウインドウ関数
		"ROW_NUMBER":         {ReturnType: "int", Nullable: false},
		"RANK":               {ReturnType: "int", Nullable: false},
		"DENSE_RANK":         {ReturnType: "int", Nullable: false},
		"SUM":                {ReturnTypeByArg: true, NullableByArg: true},
		"AVG":                {ReturnTypeByArg: true, NullableByArg: true},
		"COUNT":              {ReturnType: "int", Nullable: false},
		"MIN":                {ReturnTypeByArg: true, NullableByArg: true},
		"MAX":                {ReturnTypeByArg: true, NullableByArg: true},
		"FIRST_VALUE":        {ReturnTypeByArg: true, NullableByArg: true},
		"LAST_VALUE":         {ReturnTypeByArg: true, NullableByArg: true},
		"LEAD":               {ReturnTypeByArg: true, NullableByArg: true},
		"LAG":                {ReturnTypeByArg: true, NullableByArg: true},
		"ARRAY":              {ReturnType: "array", NullableByArg: true},
		"UNNEST":             {ReturnType: "any", NullableByArg: true},
		"JSONB_BUILD_OBJECT": {ReturnType: "any", NullableByArg: true},
	},
	DialectMySQL: {
		"LENGTH":    {ReturnType: "int", NullableByArg: true},
		"COALESCE":  {ReturnTypeByArg: true, NullableByArg: true},
		"IFNULL":    {ReturnTypeByArg: true, NullableByArg: true},
		"CAST":      {CastType: true, NullableByArg: true},
		"UPPER":     {ReturnType: "string", NullableByArg: true},
		"NOW":       {ReturnType: "time", Nullable: false},
		"DATE_ADD":  {ReturnType: "time", NullableByArg: true},
		"SUBSTRING": {ReturnType: "string", NullableByArg: true},
		"TRIM":      {ReturnType: "string", NullableByArg: true},
		// ウインドウ関数
		"ROW_NUMBER":  {ReturnType: "int", Nullable: false},
		"RANK":        {ReturnType: "int", Nullable: false},
		"DENSE_RANK":  {ReturnType: "int", Nullable: false},
		"SUM":         {ReturnTypeByArg: true, NullableByArg: true},
		"AVG":         {ReturnTypeByArg: true, NullableByArg: true},
		"COUNT":       {ReturnType: "int", Nullable: false},
		"MIN":         {ReturnTypeByArg: true, NullableByArg: true},
		"MAX":         {ReturnTypeByArg: true, NullableByArg: true},
		"FIRST_VALUE": {ReturnTypeByArg: true, NullableByArg: true},
		"LAST_VALUE":  {ReturnTypeByArg: true, NullableByArg: true},
		"LEAD":        {ReturnTypeByArg: true, NullableByArg: true},
		"LAG":         {ReturnTypeByArg: true, NullableByArg: true},
	},
	DialectSQLite: {
		"LENGTH":    {ReturnType: "int", NullableByArg: true},
		"COALESCE":  {ReturnTypeByArg: true, NullableByArg: true},
		"IFNULL":    {ReturnTypeByArg: true, NullableByArg: true},
		"CAST":      {CastType: true, NullableByArg: true},
		"UPPER":     {ReturnType: "string", NullableByArg: true},
		"NOW":       {ReturnType: "time", Nullable: false},
		"DATE_ADD":  {ReturnType: "time", NullableByArg: true},
		"SUBSTRING": {ReturnType: "string", NullableByArg: true},
		"TRIM":      {ReturnType: "string", NullableByArg: true},
		// ウインドウ関数
		"ROW_NUMBER":  {ReturnType: "int", Nullable: false},
		"RANK":        {ReturnType: "int", Nullable: false},
		"DENSE_RANK":  {ReturnType: "int", Nullable: false},
		"SUM":         {ReturnTypeByArg: true, NullableByArg: true},
		"AVG":         {ReturnTypeByArg: true, NullableByArg: true},
		"COUNT":       {ReturnType: "int", Nullable: false},
		"MIN":         {ReturnTypeByArg: true, NullableByArg: true},
		"MAX":         {ReturnTypeByArg: true, NullableByArg: true},
		"FIRST_VALUE": {ReturnTypeByArg: true, NullableByArg: true},
		"LAST_VALUE":  {ReturnTypeByArg: true, NullableByArg: true},
		"LEAD":        {ReturnTypeByArg: true, NullableByArg: true},
		"LAG":         {ReturnTypeByArg: true, NullableByArg: true},
	},
}
