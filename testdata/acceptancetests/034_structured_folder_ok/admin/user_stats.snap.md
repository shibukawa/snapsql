# User Statistics Query

## Overview
This query provides comprehensive user statistics for administrative purposes.

## SQL Template

```sql
/*#
function_name: get_user_statistics
description: Get comprehensive user statistics
parameters:
  start_date: time
  end_date: time
response_affinity: one
*/
SELECT 
    COUNT(*) as total_users,
    COUNT(CASE WHEN created_at >= /*= start_date */'2024-01-01' THEN 1 END) as new_users,
    AVG(EXTRACT(EPOCH FROM (NOW() - created_at))/86400) as avg_days_since_registration
FROM users 
WHERE created_at BETWEEN /*= start_date */'2024-01-01' AND /*= end_date */'2024-12-31';
```

## Expected Output Structure
- `total_users`: Total number of users
- `new_users`: Users created in the specified period
- `avg_days_since_registration`: Average days since user registration
