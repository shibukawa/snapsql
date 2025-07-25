/*#
function_name: get_users_with_conditions
parameters:
  min_age: int
  max_age: int
  include_email: bool
*/
SELECT
    id,
    name,
    /*# if include_email */
    email,
    /*# else */
    'N/A' as email,
    /*# end */
    age
FROM
    users
WHERE
    age >= /*= min_age */18
    AND age <= /*= max_age */65
