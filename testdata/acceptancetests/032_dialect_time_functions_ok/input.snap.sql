/*#
function_name: get_current_time
*/
SELECT 
    id,
    name,
    NOW() as current_time_now,
    CURRENT_TIMESTAMP as current_time_standard
FROM users
