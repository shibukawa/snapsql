/*#
function_name: get_user_by_id
parameters:
  user_id: int
*/
SELECT id, name, email 
FROM users 
WHERE id = /*= user_id */123
