/*#
function_name: find_user_by_id
description: Find a user by their ID from hierarchical structure
parameters:
  user_id: int
response_affinity: one
*/
SELECT 
    id,
    name,
    email,
    created_at
FROM users 
WHERE id = /*= user_id */1;
