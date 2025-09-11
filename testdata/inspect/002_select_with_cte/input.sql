WITH recent AS (
  SELECT user_id FROM orders
)
SELECT u.id
FROM users u
LEFT JOIN recent r ON r.user_id = u.id;

