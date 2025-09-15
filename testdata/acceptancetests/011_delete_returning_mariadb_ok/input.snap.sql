/*#
function_name: delete_user_with_returning_mariadb
parameters:
  user_id: int
*/
DELETE FROM users 
WHERE id = /*= user_id */1
RETURNING id, name, email;
