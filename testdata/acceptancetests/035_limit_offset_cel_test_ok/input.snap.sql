/*#
function_name: get_users_with_cel_limit_offset
parameters:
  min_age: int
  page_limit: int
  page_offset: int
*/
SELECT
    id,
    name,
    age
FROM
    users
WHERE
    age >= /*= min_age */18
LIMIT /*= page_limit */10
OFFSET /*= page_offset */0
