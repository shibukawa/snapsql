/*#
function_name: get_comprehensive_dialect_test_sqlite
parameters:
  user_id: int
*/
SELECT 
    id,
    name,
    CAST(age AS INTEGER) as age_cast_standard,
    CAST(price AS DECIMAL(10,2)) as price_cast_postgresql,
    CAST(salary + bonus AS NUMERIC(12,2)) as total_cast_complex,
    first_name || ' ' || last_name as full_name_mysql,
    first_name || ' ' || last_name as full_name_postgresql,
    CURRENT_TIMESTAMP as time_mysql,
    CURRENT_TIMESTAMP as time_standard,
    TRUE as bool_true,
    FALSE as bool_false,
    RANDOM() as random_mysql,
    RANDOM() as random_postgresql,
    CAST(CURRENT_TIMESTAMP AS TEXT) as nested_cast_time,
    'ID: ' || CAST(id AS TEXT) as nested_concat_cast
FROM users 
WHERE id = /*= user_id */123
  AND active = TRUE
  AND created_at > CURRENT_TIMESTAMP
