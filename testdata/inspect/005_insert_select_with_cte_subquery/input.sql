WITH latest AS (
  SELECT user_id FROM orders
)
INSERT INTO snapshots (user_id)
SELECT l.user_id
FROM latest l
JOIN (SELECT 1 AS dummy) s ON 1=1;

