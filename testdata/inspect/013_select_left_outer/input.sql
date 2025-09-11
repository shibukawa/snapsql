SELECT u.id FROM users u LEFT OUTER JOIN orders o ON o.user_id = u.id;
