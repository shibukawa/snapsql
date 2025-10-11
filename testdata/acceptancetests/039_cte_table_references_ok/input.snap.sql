-- @snapsql
-- name: get_active_users
-- response_affinity: many
WITH active_users AS (
  SELECT u.id, u.name, u.email
  FROM users u
  JOIN user_status us ON u.id = us.user_id
  WHERE us.status = 'active'
)
SELECT au.id, au.name, au.email, o.total
FROM active_users au
LEFT JOIN orders o ON au.id = o.user_id
