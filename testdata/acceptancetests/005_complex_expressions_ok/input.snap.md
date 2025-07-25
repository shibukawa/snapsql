---
function_name: getComplexData
---

# Get Complex Data

## Description

Get complex data with various conditional expressions.

## Parameters

```yaml
user_id: int
username: string
display_name: bool
start_date: string
end_date: string
sort_field: string
sort_direction: string
page_size: int
page: int
```

## SQL

```sql
SELECT 
  id, 
  name,
  /*= display_name ? username : "Anonymous" */
FROM users
WHERE 
  /*# if start_date != "" && end_date != "" */
  created_at BETWEEN /*= start_date */'2023-01-01' AND /*= end_date */'2023-12-31'
  /*# end */
  /*# if sort_field != "" */
ORDER BY /*= sort_field + " " + (sort_direction != "" ? sort_direction : "ASC") */name
  /*# end */
LIMIT /*= page_size != 0 ? page_size : 10 */10
OFFSET /*= page > 0 ? (page - 1) * page_size : 0 */0
```
