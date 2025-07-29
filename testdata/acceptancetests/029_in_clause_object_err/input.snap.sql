/*#
function_name: getUsersByFilter
parameters:
  filter: ./UserFilter
*/
SELECT id, name FROM users WHERE department_id IN (/*= filter */1, 2, 3)
