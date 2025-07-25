/*#
function_name: find_user
parameters:
  user_id: int
*/
SELECT
    id,
    name,
    age
FROM
    users
WHERE
    id = /*= user_id */1