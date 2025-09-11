SELECT u.id FROM users u RIGHT OUTER JOIN orders o ON o.user_id = u.id;
