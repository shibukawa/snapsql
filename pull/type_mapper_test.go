package pull

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestPostgreSQLTypeMapper(t *testing.T) {
	mapper := NewPostgreSQLTypeMapper()

	t.Run("MapBasicTypes", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"integer", TypeInt},
			{"bigint", TypeInt},
			{"smallint", TypeInt},
			{"serial", TypeInt},
			{"bigserial", TypeInt},
			{"character varying(255)", TypeString},
			{"varchar(100)", TypeString},
			{"text", TypeString},
			{"char(10)", TypeString},
			{"character(5)", TypeString},
			{"numeric", TypeFloat},
			{"decimal", TypeFloat},
			{"real", TypeFloat},
			{"double precision", TypeFloat},
			{"float", TypeFloat},
			{"boolean", TypeBool},
			{"bool", TypeBool},
			{"date", TypeDate},
			{"time", TypeTime},
			{"time with time zone", TypeTime},
			{"timestamp", TypeDateTime},
			{"timestamp with time zone", TypeDateTime},
			{"timestamptz", TypeDateTime},
			{"json", TypeJSON},
			{"jsonb", TypeJSON},
			{"bytea", TypeBinary},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Failed to map PostgreSQL type: %s", tc.dbType)
		}
	})

	t.Run("MapArrayTypes", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"integer[]", TypeArray},
			{"text[]", TypeArray},
			{"varchar(255)[]", TypeArray},
			{"boolean[]", TypeArray},
			{"json[]", TypeArray},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Failed to map PostgreSQL array type: %s", tc.dbType)
		}
	})

	t.Run("MapUnknownTypes", func(t *testing.T) {
		testCases := []string{
			"unknown_type",
			"custom_enum",
			"geometry",
			"point",
		}

		for _, dbType := range testCases {
			result := mapper.MapType(dbType)
			assert.Equal(t, TypeString, result, "Unknown type should fallback to string: %s", dbType)
		}
	})

	t.Run("GetSnapSQLType", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"integer", TypeInt},
			{"text", TypeString},
			{"boolean", TypeBool},
			{"timestamp with time zone", TypeDateTime},
			{"unknown_type", TypeString},
		}

		for _, tc := range testCases {
			result := mapper.GetSnapSQLType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "GetSnapSQLType failed for: %s", tc.dbType)
		}
	})
}

func TestMySQLTypeMapper(t *testing.T) {
	mapper := NewMySQLTypeMapper()

	t.Run("MapBasicTypes", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"int", TypeInt},
			{"integer", TypeInt},
			{"bigint", TypeInt},
			{"smallint", TypeInt},
			{"tinyint", TypeInt},
			{"mediumint", TypeInt},
			{"varchar(255)", TypeString},
			{"char(10)", TypeString},
			{"text", TypeString},
			{"mediumtext", TypeString},
			{"longtext", TypeString},
			{"tinytext", TypeString},
			{"decimal", TypeFloat},
			{"numeric", TypeFloat},
			{"float", TypeFloat},
			{"double", TypeFloat},
			{"real", TypeFloat},
			{"boolean", TypeBool},
			{"bool", TypeBool},
			{"tinyint(1)", TypeBool},
			{"date", TypeDate},
			{"time", TypeTime},
			{"datetime", TypeDateTime},
			{"timestamp", TypeDateTime},
			{"year", TypeInt},
			{"json", TypeJSON},
			{"blob", TypeBinary},
			{"mediumblob", TypeBinary},
			{"longblob", TypeBinary},
			{"tinyblob", TypeBinary},
			{"binary", TypeBinary},
			{"varbinary", TypeBinary},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Failed to map MySQL type: %s", tc.dbType)
		}
	})

	t.Run("MapSpecialBooleanTypes", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"tinyint(1)", TypeBool},
			{"bool", TypeBool},
			{"boolean", TypeBool},
			{"tinyint(2)", TypeInt}, // Not boolean if not tinyint(1)
			{"tinyint", TypeInt},    // Default tinyint is not boolean
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Failed to map MySQL boolean type: %s", tc.dbType)
		}
	})

	t.Run("MapUnknownTypes", func(t *testing.T) {
		testCases := []string{
			"unknown_type",
			"custom_enum",
			"geometry",
			"point",
			"linestring",
		}

		for _, dbType := range testCases {
			result := mapper.MapType(dbType)
			assert.Equal(t, TypeString, result, "Unknown type should fallback to string: %s", dbType)
		}
	})
}

func TestSQLiteTypeMapper(t *testing.T) {
	mapper := NewSQLiteTypeMapper()

	t.Run("MapBasicTypes", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"INTEGER", TypeInt},
			{"integer", TypeInt},
			{"INT", TypeInt},
			{"BIGINT", TypeInt},
			{"SMALLINT", TypeInt},
			{"TEXT", TypeString},
			{"text", TypeString},
			{"VARCHAR", TypeString},
			{"varchar(255)", TypeString},
			{"CHAR", TypeString},
			{"CHARACTER", TypeString},
			{"REAL", TypeFloat},
			{"real", TypeFloat},
			{"DOUBLE", TypeFloat},
			{"FLOAT", TypeFloat},
			{"NUMERIC", TypeFloat},
			{"DECIMAL", TypeFloat},
			{"BOOLEAN", TypeBool},
			{"boolean", TypeBool},
			{"BOOL", TypeBool},
			{"DATE", TypeDate},
			{"date", TypeDate},
			{"TIME", TypeTime},
			{"time", TypeTime},
			{"DATETIME", TypeDateTime},
			{"datetime", TypeDateTime},
			{"TIMESTAMP", TypeDateTime},
			{"timestamp", TypeDateTime},
			{"BLOB", TypeBinary},
			{"blob", TypeBinary},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Failed to map SQLite type: %s", tc.dbType)
		}
	})

	t.Run("MapDynamicTypes", func(t *testing.T) {
		// SQLite allows any type name, should fallback to string
		testCases := []string{
			"custom_type",
			"user_defined",
			"anything_goes",
			"",
		}

		for _, dbType := range testCases {
			result := mapper.MapType(dbType)
			assert.Equal(t, TypeString, result, "Dynamic type should fallback to string: %s", dbType)
		}
	})

	t.Run("MapCaseInsensitive", func(t *testing.T) {
		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"INTEGER", TypeInt},
			{"integer", TypeInt},
			{"Integer", TypeInt},
			{"TEXT", TypeString},
			{"text", TypeString},
			{"Text", TypeString},
			{"REAL", TypeFloat},
			{"real", TypeFloat},
			{"Real", TypeFloat},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Case insensitive mapping failed for: %s", tc.dbType)
		}
	})
}

func TestTypeMapperFactory(t *testing.T) {
	t.Run("CreatePostgreSQLMapper", func(t *testing.T) {
		mapper, err := NewTypeMapper("postgresql")
		assert.NoError(t, err)
		assert.NotZero(t, mapper)

		// Test that it's actually a PostgreSQL mapper
		result := mapper.MapType("integer")
		assert.Equal(t, TypeInt, result)
	})

	t.Run("CreateMySQLMapper", func(t *testing.T) {
		mapper, err := NewTypeMapper("mysql")
		assert.NoError(t, err)
		assert.NotZero(t, mapper)

		// Test that it's actually a MySQL mapper
		result := mapper.MapType("int")
		assert.Equal(t, TypeInt, result)
	})

	t.Run("CreateSQLiteMapper", func(t *testing.T) {
		mapper, err := NewTypeMapper("sqlite")
		assert.NoError(t, err)
		assert.NotZero(t, mapper)

		// Test that it's actually a SQLite mapper
		result := mapper.MapType("INTEGER")
		assert.Equal(t, TypeInt, result)
	})

	t.Run("CreateUnsupportedMapper", func(t *testing.T) {
		mapper, err := NewTypeMapper("unsupported")
		assert.Error(t, err)
		assert.Zero(t, mapper)
		assert.Equal(t, ErrUnsupportedDatabase, err)
	})

	t.Run("CreateEmptyTypeMapper", func(t *testing.T) {
		mapper, err := NewTypeMapper("")
		assert.Error(t, err)
		assert.Zero(t, mapper)
		assert.Equal(t, ErrEmptyDatabaseType, err)
	})
}

func TestTypeConstants(t *testing.T) {
	t.Run("VerifyTypeConstants", func(t *testing.T) {
		assert.Equal(t, "string", TypeString)
		assert.Equal(t, "int", TypeInt)
		assert.Equal(t, "float", TypeFloat)
		assert.Equal(t, "bool", TypeBool)
		assert.Equal(t, "date", TypeDate)
		assert.Equal(t, "time", TypeTime)
		assert.Equal(t, "datetime", TypeDateTime)
		assert.Equal(t, "json", TypeJSON)
		assert.Equal(t, "array", TypeArray)
		assert.Equal(t, "binary", TypeBinary)
	})
}

func TestComplexTypeMappingScenarios(t *testing.T) {
	t.Run("PostgreSQLComplexTypes", func(t *testing.T) {
		mapper := NewPostgreSQLTypeMapper()

		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"character varying(255)", TypeString},
			{"varchar(100)", TypeString},
			{"numeric(10,2)", TypeFloat},
			{"decimal(8,2)", TypeFloat},
			{"timestamp(6) with time zone", TypeDateTime},
			{"time(3) without time zone", TypeTime},
			{"bit varying(64)", TypeString},
			{"interval", TypeString},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Complex PostgreSQL type mapping failed: %s", tc.dbType)
		}
	})

	t.Run("MySQLComplexTypes", func(t *testing.T) {
		mapper := NewMySQLTypeMapper()

		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"varchar(255)", TypeString},
			{"char(10)", TypeString},
			{"decimal(10,2)", TypeFloat},
			{"float(7,4)", TypeFloat},
			{"enum('a','b','c')", TypeString},
			{"set('x','y','z')", TypeString},
			{"tinyint(1) unsigned", TypeBool},
			{"int(11) unsigned", TypeInt},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "Complex MySQL type mapping failed: %s", tc.dbType)
		}
	})

	t.Run("SQLiteFlexibleTypes", func(t *testing.T) {
		mapper := NewSQLiteTypeMapper()

		testCases := []struct {
			dbType       string
			expectedType string
		}{
			{"VARCHAR(255)", TypeString},
			{"DECIMAL(10,2)", TypeFloat},
			{"UNSIGNED BIG INT", TypeInt},
			{"CHARACTER(20)", TypeString},
			{"NATIVE CHARACTER(70)", TypeString},
			{"VARYING CHARACTER(255)", TypeString},
			{"NCHAR(55)", TypeString},
			{"NVARCHAR(100)", TypeString},
			{"CLOB", TypeString},
		}

		for _, tc := range testCases {
			result := mapper.MapType(tc.dbType)
			assert.Equal(t, tc.expectedType, result, "SQLite flexible type mapping failed: %s", tc.dbType)
		}
	})
}
