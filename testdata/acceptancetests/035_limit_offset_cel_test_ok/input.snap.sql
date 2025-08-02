/*#
function_name: get_users_with_cel_limit_offset
parameters:
  min_age: int
  page_size: int
  page: int
*/
SELECT
    id,
    name,
    age
FROM
    users
WHERE
    age >= /*= min_age */18
LIMIT /*= page_size != 0 ? page_size : 10 */10
OFFSET /*= page > 0 ? (page - 1) * page_size : 0 */0
