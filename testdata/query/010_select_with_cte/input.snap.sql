WITH u AS (
  SELECT id, name FROM users
)
SELECT id, name FROM u WHERE id = 1;
