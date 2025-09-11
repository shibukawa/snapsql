SELECT u.id FROM users u FULL OUTER JOIN orders o ON o.user_id = u.id;
