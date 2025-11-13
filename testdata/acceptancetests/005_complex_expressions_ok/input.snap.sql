/*#
name: getComplexData
function_name: getComplexData
parameters:
  user_id: int
  username: string
  display_name: bool
  start_date: string
  end_date: string
  display_value: string
  has_date_range: bool
  has_order_clause: bool
  page_size_value: int
  page_offset_value: int
*/
SELECT 
  id, 
  name,
  /*= display_value */ display_name
FROM users
WHERE 
  /*# if has_date_range */
  created_at BETWEEN /*= start_date */'2023-01-01' AND /*= end_date */'2023-12-31'
  /*# end */
  /*# if has_order_clause */
ORDER BY username
  /*# end */
LIMIT /*= page_size_value */10
OFFSET /*= page_offset_value */0
