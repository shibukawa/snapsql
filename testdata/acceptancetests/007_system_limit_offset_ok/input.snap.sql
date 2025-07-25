/*#
function_name: get_users_with_limit_offset
parameters:
  min_age: int
  max_age: int
*/
SELECT
    id,
    name,
    age
FROM
    users
WHERE
    age >= /*= min_age */18
    AND age <= /*= max_age */65
LIMIT 10
OFFSET 20
