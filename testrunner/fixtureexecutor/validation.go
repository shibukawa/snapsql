package fixtureexecutor

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/shibukawa/snapsql"
)

// ValidationStrategy represents the validation strategy for DML queries
type ValidationStrategy string

const (
	DirectResult  ValidationStrategy = "direct"    // RETURNING clause present
	NumericResult ValidationStrategy = "numeric"   // Numeric validation (rows_affected, last_insert_id)
	TableState    ValidationStrategy = "table"     // Table state validation
	Existence     ValidationStrategy = "exists"    // Existence check
	Count         ValidationStrategy = "count"     // Row count validation
	Conditional   ValidationStrategy = "where"     // Conditional validation
	Aggregate     ValidationStrategy = "aggregate" // Aggregate validation
	Cascade       ValidationStrategy = "cascade"   // Related table validation
	Diff          ValidationStrategy = "diff"      // Change diff validation
)

// ValidationSpec represents a validation specification
type ValidationSpec struct {
	Strategy   ValidationStrategy
	TableName  string
	Parameters map[string]any
	Expected   any
}

// parseValidationSpecs parses expected results into validation specifications
func parseValidationSpecs(expectedResults []map[string]any) ([]ValidationSpec, error) {
	var specs []ValidationSpec

	for _, result := range expectedResults {
		for key, value := range result {
			spec, err := parseValidationSpec(key, value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse validation spec for key '%s': %w", key, err)
			}

			specs = append(specs, spec)
		}
	}

	return specs, nil
}

// parseValidationSpec parses a single validation specification
func parseValidationSpec(key string, value any) (ValidationSpec, error) {
	// Check for strategy syntax: table_name[strategy]
	if strings.Contains(key, "[") && strings.HasSuffix(key, "]") {
		// Extract table name and strategy
		parts := strings.SplitN(key, "[", 2)
		if len(parts) != 2 {
			return ValidationSpec{}, fmt.Errorf("%w: %s", snapsql.ErrInvalidValidationSpecFormat, key)
		}

		tableName := parts[0]
		strategyPart := strings.TrimSuffix(parts[1], "]")
		strategy := ValidationStrategy(strategyPart)

		return ValidationSpec{
			Strategy:  strategy,
			TableName: tableName,
			Expected:  value,
		}, nil
	}

	// Handle numeric validation keys
	switch key {
	case "rows_affected", "last_insert_id":
		return ValidationSpec{
			Strategy: NumericResult,
			Expected: map[string]any{key: value},
		}, nil
	default:
		// Default to table state validation
		return ValidationSpec{
			Strategy:  TableState,
			TableName: key,
			Expected:  value,
		}, nil
	}
}

// validateResult validates the query result against the expected specifications
func (e *Executor) validateResult(tx *sql.Tx, result *ValidationResult, specs []ValidationSpec) error {
	for _, spec := range specs {
		err := e.validateSingleSpec(tx, result, spec)
		if err != nil {
			return fmt.Errorf("validation failed for %s[%s]: %w", spec.TableName, spec.Strategy, err)
		}
	}

	return nil
}

// validateSingleSpec validates a single specification
func (e *Executor) validateSingleSpec(tx *sql.Tx, result *ValidationResult, spec ValidationSpec) error {
	switch spec.Strategy {
	case DirectResult:
		return e.validateDirectResult(result, spec)
	case NumericResult:
		return e.validateNumericResult(result, spec)
	case TableState:
		return e.validateTableState(tx, spec)
	case Existence:
		return e.validateExistence(tx, spec)
	case Count:
		return e.validateCount(tx, spec)
	default:
		return fmt.Errorf("%w: %s", snapsql.ErrUnsupportedValidationStrategy, spec.Strategy)
	}
}

// validateDirectResult validates direct query results (RETURNING clause)
func (e *Executor) validateDirectResult(result *ValidationResult, spec ValidationSpec) error {
	expected, ok := spec.Expected.([]map[string]any)
	if !ok {
		return snapsql.ErrExpectedDataMustBeArray
	}

	if len(result.Data) != len(expected) {
		return fmt.Errorf("%w: expected %d rows, got %d rows", snapsql.ErrResultRowCountMismatch, len(expected), len(result.Data))
	}

	for i, expectedRow := range expected {
		actualRow := result.Data[i]

		err := compareRows(expectedRow, actualRow)
		if err != nil {
			return fmt.Errorf("row %d mismatch: %w", i, err)
		}
	}

	return nil
}

// validateNumericResult validates numeric results (rows_affected, last_insert_id)
func (e *Executor) validateNumericResult(result *ValidationResult, spec ValidationSpec) error {
	expected, ok := spec.Expected.(map[string]any)
	if !ok {
		return snapsql.ErrExpectedNumericMustBeMap
	}

	for key, expectedValue := range expected {
		switch key {
		case "rows_affected":
			expectedRows, err := convertToInt64(expectedValue)
			if err != nil {
				return fmt.Errorf("invalid rows_affected value: %w", err)
			}

			if result.RowsAffected != expectedRows {
				return fmt.Errorf("%w: expected rows_affected %d, got %d", snapsql.ErrResultRowCountMismatch, expectedRows, result.RowsAffected)
			}
		case "last_insert_id":
			// TODO: Implement last_insert_id validation
			// This requires getting the last insert ID from the database
			return snapsql.ErrLastInsertIdNotImplemented
		default:
			return fmt.Errorf("%w: %s", snapsql.ErrUnsupportedNumericValidationKey, key)
		}
	}

	return nil
}

// validateTableState validates table state after DML operation
func (e *Executor) validateTableState(tx *sql.Tx, spec ValidationSpec) error {
	// Handle both array and single object formats
	var expected []map[string]any

	switch v := spec.Expected.(type) {
	case []map[string]any:
		expected = v
	case []any:
		// Convert []any to []map[string]any
		for _, item := range v {
			if row, ok := item.(map[string]any); ok {
				expected = append(expected, row)
			} else {
				return fmt.Errorf("%w, got %T", snapsql.ErrTableStateValidationItemMustBeObject, item)
			}
		}
	case map[string]any:
		expected = []map[string]any{v}
	default:
		return fmt.Errorf("%w, got %T", snapsql.ErrTableStateValidationMustBeArray, spec.Expected)
	}

	// Query the table to get current state
	query := "SELECT * FROM " + spec.TableName
	ctx := context.Background()

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query table %s: %w", spec.TableName, err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %w", err)
	}

	// Read actual data
	var actualData []map[string]any

	for rows.Next() {
		values := make([]interface{}, len(columns))

		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]any)

		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		actualData = append(actualData, row)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during row iteration: %w", err)
	}

	// Compare expected and actual data
	if len(expected) != len(actualData) {
		return fmt.Errorf("%w: expected %d rows in table %s, got %d rows", snapsql.ErrTableRowCountMismatch, len(expected), spec.TableName, len(actualData))
	}

	// TODO: Implement more sophisticated comparison (order-independent, partial matching)
	for i, expectedRow := range expected {
		if i >= len(actualData) {
			return fmt.Errorf("%w %d in table %s", snapsql.ErrMissingRowInTable, i, spec.TableName)
		}

		err := compareRows(expectedRow, actualData[i])
		if err != nil {
			return fmt.Errorf("table %s row %d mismatch: %w", spec.TableName, i, err)
		}
	}

	return nil
}

// validateExistence validates existence of records
func (e *Executor) validateExistence(tx *sql.Tx, spec ValidationSpec) error {
	// Handle both array and single object formats
	var expectedRows []map[string]any

	switch v := spec.Expected.(type) {
	case []map[string]any:
		expectedRows = v
	case []any:
		// Convert []any to []map[string]any
		for _, item := range v {
			if row, ok := item.(map[string]any); ok {
				expectedRows = append(expectedRows, row)
			} else {
				return fmt.Errorf("%w, got %T", snapsql.ErrExistenceValidationItemMustBeObject, item)
			}
		}
	case map[string]any:
		expectedRows = []map[string]any{v}
	default:
		return fmt.Errorf("%w, got %T", snapsql.ErrExistenceValidationMustBeArray, spec.Expected)
	}

	for _, expectedRow := range expectedRows {
		exists, ok := expectedRow["exists"].(bool)
		if !ok {
			return snapsql.ErrExistenceValidationRequiresExists
		}

		// Build WHERE clause from other fields
		var (
			conditions []string
			args       []interface{}
		)

		for key, value := range expectedRow {
			if key != "exists" {
				conditions = append(conditions, key+" = ?")
				args = append(args, value)
			}
		}

		if len(conditions) == 0 {
			return snapsql.ErrExistenceValidationRequiresCondition
		}

		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", spec.TableName, strings.Join(conditions, " AND "))

		var count int64

		ctx := context.Background()

		err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existence: %w", err)
		}

		actualExists := count > 0
		if actualExists != exists {
			return fmt.Errorf("%w: expected exists=%t, got exists=%t for conditions %v", snapsql.ErrExistenceValidationMismatch, exists, actualExists, expectedRow)
		}
	}

	return nil
}

// validateCount validates row count
func (e *Executor) validateCount(tx *sql.Tx, spec ValidationSpec) error {
	expectedCount, err := convertToInt64(spec.Expected)
	if err != nil {
		return fmt.Errorf("invalid count value: %w", err)
	}

	query := "SELECT COUNT(*) FROM " + spec.TableName

	var actualCount int64

	ctx := context.Background()

	err = tx.QueryRowContext(ctx, query).Scan(&actualCount)
	if err != nil {
		return fmt.Errorf("failed to count rows in table %s: %w", spec.TableName, err)
	}

	if actualCount != expectedCount {
		return fmt.Errorf("%w: expected count %d, got count %d", snapsql.ErrCountMismatch, expectedCount, actualCount)
	}

	return nil
}

// compareRows compares two row maps with type tolerance
func compareRows(expected, actual map[string]any) error {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			return fmt.Errorf("%w '%s'", snapsql.ErrMissingField, key)
		}

		// Try to normalize numeric types for comparison
		if !compareValues(expectedValue, actualValue) {
			return fmt.Errorf("%w '%s': expected %v (%T), got %v (%T)", snapsql.ErrFieldValueMismatch, key, expectedValue, expectedValue, actualValue, actualValue)
		}
	}

	return nil
}

// compareValues compares two values with type tolerance
func compareValues(expected, actual any) bool {
	// Direct equality check first
	if reflect.DeepEqual(expected, actual) {
		return true
	}

	// Try numeric conversion
	expectedInt, expectedErr := convertToInt64(expected)

	actualInt, actualErr := convertToInt64(actual)
	if expectedErr == nil && actualErr == nil {
		return expectedInt == actualInt
	}

	// Try string conversion
	expectedStr := fmt.Sprintf("%v", expected)
	actualStr := fmt.Sprintf("%v", actual)

	return expectedStr == actualStr
}

// convertToInt64 converts various numeric types to int64
func convertToInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint64:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("%w: %T", snapsql.ErrCannotConvertToInt64, value)
	}
}
