/*#
function_name: get_comprehensive_dialect_test_mysql
parameters:
  user_id: int
*/
SELECT 
    id,
    name,
    CAST(age AS INTEGER) as age_cast_standard,
    CAST(price AS DECIMAL(10,2)) as price_cast_postgresql,
    CAST(salary + bonus AS NUMERIC(12,2)) as total_cast_complex,
    CONCAT(first_name, ' ', last_name) as full_name_mysql,
    CONCAT(first_name, ' ', last_name) as full_name_postgresql,
    NOW() as time_mysql,
    NOW() as time_standard,
    1 as bool_true,
    0 as bool_false,
    RAND() as random_mysql,
    RAND() as random_postgresql,
    CAST(NOW() AS CHAR) as nested_cast_time,
    CONCAT('ID: ', CAST(id AS CHAR)) as nested_concat_cast
FROM users 
WHERE id = /*= user_id */123
  AND active = 1
  AND created_at > NOW()
