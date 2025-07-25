/*#
name: getFilteredData
function_name: getFilteredData
parameters:
  min_age: int
  max_age: int
  departments: string[]
  active: bool
*/
SELECT id, name, age, department 
FROM users
WHERE 1=1
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */
/*# if max_age > 0 */
AND age <= /*= max_age */65
/*# end */
/*# if departments.size() > 0 */
AND department IN (/*= departments */('HR', 'Engineering'))
/*# end */
/*# if active */
AND status = 'active'
/*# end */
