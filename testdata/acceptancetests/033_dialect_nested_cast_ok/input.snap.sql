/*#
function_name: get_nested_dialect_cast
parameters:
  user_id: int
*/
SELECT 
    id,
    name,
    -- ネストした方言コード: CAST内にNOW()関数
    CAST(NOW() AS TEXT) as current_time_text,
    -- PostgreSQL構文内にTRUEリテラル
    (TRUE)::INTEGER as bool_as_int
FROM users 
WHERE id = /*= user_id */123
