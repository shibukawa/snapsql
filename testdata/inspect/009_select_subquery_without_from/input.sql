SELECT s.id
FROM (SELECT 1 AS id) s
JOIN users u ON u.id = s.id;

