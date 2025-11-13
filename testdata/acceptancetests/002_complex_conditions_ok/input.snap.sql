/*#
name: getFilteredData
function_name: getFilteredData
parameters:
  min_age: int
  max_age: int
  departments: string[]
  active: bool
  has_min_age: bool
  has_max_age: bool
  has_departments: bool
*/
SELECT id, name, age, department 
FROM users
WHERE 1=1
/*# if has_min_age */
AND age >= /*= min_age */18
/*# end */
/*# if has_max_age */
AND age <= /*= max_age */65
/*# end */
/*# if has_departments */
AND department IN (/*= departments */('HR', 'Engineering'))
/*# end */
/*# if active */
AND status = 'active'
/*# end */
