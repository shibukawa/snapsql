# Subquery with Table References

This test verifies that subqueries in FROM clause are properly handled:
- Subquery tokens are captured and re-parsed
- SELECT fields from subquery are extracted
- Table references inside subquery are tracked
- Type inference works with derived tables from subqueries

```sql
SELECT 
    sq.user_id,
    sq.user_name,
    sq.order_count,
    o.total
FROM (
    SELECT 
        u.id AS user_id,
        u.name AS user_name,
        COUNT(o.id) AS order_count
    FROM users u
    JOIN orders o ON o.user_id = u.id
    GROUP BY u.id, u.name
) sq
JOIN orders o ON o.user_id = sq.user_id
WHERE sq.order_count > 1
```
