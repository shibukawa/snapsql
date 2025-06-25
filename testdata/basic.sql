-- 基本的なSELECT文
SELECT id, name, email FROM users WHERE active = true;

-- JOINを含むクエリ
SELECT 
    u.id,
    u.name,
    p.title,
    p.created_at
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
WHERE u.active = true
ORDER BY p.created_at DESC;

-- 複雑なWHERE句
SELECT * FROM products 
WHERE (category = 'electronics' AND price > 100) 
   OR (category = 'books' AND rating >= 4.0)
   OR (featured = true AND stock > 0);
