/*#
function_name: get_monthly_sales
description: Get monthly sales report
parameters:
  year: int
  month: int
response_affinity: many
*/
SELECT 
    DATE_TRUNC('day', created_at) as sale_date,
    COUNT(*) as order_count,
    SUM(total) as total_sales
FROM orders 
WHERE EXTRACT(YEAR FROM created_at) = /*= year */2024
  AND EXTRACT(MONTH FROM created_at) = /*= month */1
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY sale_date;
