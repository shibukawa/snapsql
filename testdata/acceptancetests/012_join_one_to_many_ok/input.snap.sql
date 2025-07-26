/*#
function_name: get_user_with_jobs
parameters:
  user_id: int
*/
SELECT 
    u.id, 
    u.name, 
    u.email,
    j.id AS jobs__id,
    j.title AS jobs__title,
    j.company AS jobs__company
FROM users u
LEFT JOIN jobs j ON u.id = j.user_id
WHERE u.id = /*= user_id */1
