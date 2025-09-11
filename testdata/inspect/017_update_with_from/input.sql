UPDATE users u SET name = 'x' WHERE EXISTS (SELECT 1 FROM audits a WHERE a.user_id = u.id);
