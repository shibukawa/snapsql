SELECT s.id
FROM (SELECT id FROM users) s
JOIN orders o ON o.user_id = s.id;

