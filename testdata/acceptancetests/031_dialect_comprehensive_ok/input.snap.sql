/*#
function_name: get_comprehensive_dialect_test
parameters:
  user_id: int
*/
SELECT 
    id,
    name,
    -- 型キャスト（両方向）
    CAST(age AS INTEGER) as age_cast_standard,
    price::DECIMAL(10,2) as price_cast_postgresql,
    (salary + bonus)::NUMERIC(12,2) as total_cast_complex,
    
    -- 文字列連結
    CONCAT(first_name, ' ', last_name) as full_name_mysql,
    first_name || ' ' || last_name as full_name_postgresql,
    
    -- 時刻関数（両方向）
    NOW() as time_mysql,
    CURRENT_TIMESTAMP as time_standard,
    
    -- ブール値（両方向）
    TRUE as bool_true,
    FALSE as bool_false,
    
    -- 乱数関数（両方向）
    RAND() as random_mysql,
    RANDOM() as random_postgresql,
    
    -- ネストした方言コード
    CAST(NOW() AS TEXT) as nested_cast_time,
    CONCAT('ID: ', CAST(id AS TEXT)) as nested_concat_cast
    
FROM users 
WHERE id = /*= user_id */123
  AND active = TRUE
  AND created_at > NOW()
