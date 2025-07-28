/*#
function_name: getUsersByDepartments
parameters:
  department_ids: int[]
*/
SELECT id, name FROM users WHERE department_id IN (/*= department_ids */1, 2, 3)
