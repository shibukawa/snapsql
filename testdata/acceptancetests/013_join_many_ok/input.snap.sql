/*#
function_name: get_users_with_jobs
parameters:
  department: string
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
WHERE u.department = /*= department */'engineering'
