package markdownparser

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mattn/go-sqlite3"
)

// ExpectedError represents detailed error expectations for test cases
// Supports both simple string format and detailed YAML format
type ExpectedError struct {
	Type           ErrorType `yaml:"type"`
	Constraint     string    `yaml:"constraint,omitempty"`
	Column         string    `yaml:"column,omitempty"`
	MessagePattern string    `yaml:"message_pattern,omitempty"`
	SQLState       string    `yaml:"sqlstate,omitempty"`
}

// ClassifyDatabaseError classifies a database error into one of the predefined ErrorType constants
// Returns an empty string if the error cannot be classified
func ClassifyDatabaseError(err error) ErrorType {
	if err == nil {
		return ""
	}

	// PostgreSQL errors (via pgx)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return classifyPostgresError(pgErr)
	}

	// MySQL errors
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return classifyMySQLError(myErr)
	}

	// SQLite errors
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return classifySQLiteError(sqliteErr)
	}

	// Check for common error messages as fallback
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no rows") {
		return ErrorTypeNotFound
	}

	return ""
}

// classifyPostgresError classifies PostgreSQL errors based on SQLSTATE codes
// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
func classifyPostgresError(err *pgconn.PgError) ErrorType {
	switch err.Code {
	// Class 23: Integrity Constraint Violation
	case "23505": // unique_violation
		return ErrorTypeUniqueViolation
	case "23503": // foreign_key_violation
		return ErrorTypeForeignKeyViolation
	case "23502": // not_null_violation
		return ErrorTypeNotNullViolation
	case "23514": // check_violation
		return ErrorTypeCheckViolation

	// Class 22: Data Exception
	case "22001": // string_data_right_truncation
		return ErrorTypeDataTooLong
	case "22003": // numeric_value_out_of_range
		return ErrorTypeNumericOverflow
	case "22P02": // invalid_text_representation
		return ErrorTypeInvalidTextRepresentation

	default:
		return ""
	}
}

// classifyMySQLError classifies MySQL errors based on error numbers
// See: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
func classifyMySQLError(err *mysql.MySQLError) ErrorType {
	switch err.Number {
	// Integrity constraint violations
	case 1062: // ER_DUP_ENTRY
		return ErrorTypeUniqueViolation
	case 1451, 1452: // ER_ROW_IS_REFERENCED, ER_NO_REFERENCED_ROW
		return ErrorTypeForeignKeyViolation
	case 1048, 1364: // ER_BAD_NULL_ERROR, ER_NO_DEFAULT_FOR_FIELD
		return ErrorTypeNotNullViolation
	case 3819: // ER_CHECK_CONSTRAINT_VIOLATED
		return ErrorTypeCheckViolation

	// Data exceptions
	case 1406: // ER_DATA_TOO_LONG
		return ErrorTypeDataTooLong
	case 1264, 1690: // ER_WARN_DATA_OUT_OF_RANGE, ER_DATA_OUT_OF_RANGE
		return ErrorTypeNumericOverflow
	case 1265, 1366: // ER_WARN_DATA_TRUNCATED, ER_TRUNCATED_WRONG_VALUE
		return ErrorTypeInvalidTextRepresentation

	default:
		return ""
	}
}

// classifySQLiteError classifies SQLite errors based on extended error codes
// See: https://www.sqlite.org/rescode.html
func classifySQLiteError(err sqlite3.Error) ErrorType {
	switch err.ExtendedCode {
	// Constraint violations
	case sqlite3.ErrConstraintUnique, sqlite3.ErrConstraintPrimaryKey:
		return ErrorTypeUniqueViolation
	case sqlite3.ErrConstraintForeignKey:
		return ErrorTypeForeignKeyViolation
	case sqlite3.ErrConstraintNotNull:
		return ErrorTypeNotNullViolation
	case sqlite3.ErrConstraintCheck:
		return ErrorTypeCheckViolation
	}

	// Check base error codes for other cases
	switch err.Code {
	case sqlite3.ErrMismatch:
		return ErrorTypeInvalidTextRepresentation
	case sqlite3.ErrTooBig:
		return ErrorTypeDataTooLong
	}

	return ""
}

// MatchesExpectedError checks if the actual error matches the expected error type
// Returns (true, "") if they match, or (false, explanation) if they don't
func MatchesExpectedError(actualErr error, expectedType string) (bool, string) {
	if actualErr == nil {
		return false, "expected error but got no error"
	}

	actualType := ClassifyDatabaseError(actualErr)
	if actualType == "" {
		return false, "unable to classify database error: " + actualErr.Error()
	}

	normalizedExpected := normalizeErrorType(expectedType)
	if string(actualType) != normalizedExpected {
		return false, "error type mismatch: expected " + normalizedExpected + ", got " + string(actualType)
	}

	return true, ""
}
