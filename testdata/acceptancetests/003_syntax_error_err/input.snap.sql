/*#
function_name: invalid_query
parameters:
  user_id: int
*/
SELECT id, name, email 
FROM users 
WHERE id = /*= user_id 123  -- Missing closing comment
