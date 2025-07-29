/*#
function_name: update_user_with_returning
parameters:
  user_id: int
  new_name: string
*/
UPDATE users 
SET name = /*= new_name */'Updated Name', updated_at = NOW()
WHERE id = /*= user_id */1
RETURNING id, name, email, updated_at;
