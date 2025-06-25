-- CTE（Common Table Expression）を含む複雑なクエリ
WITH RECURSIVE employee_hierarchy AS (
    -- アンカー部分
    SELECT employee_id, name, manager_id, 0 as level
    FROM employees
    WHERE manager_id IS NULL
    
    UNION ALL
    
    -- 再帰部分
    SELECT e.employee_id, e.name, e.manager_id, eh.level + 1
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.employee_id
    WHERE eh.level < 10
),
department_stats AS (
    SELECT 
        department_id,
        COUNT(*) as employee_count,
        AVG(salary) as avg_salary,
        MAX(salary) as max_salary,
        MIN(salary) as min_salary
    FROM employees
    WHERE active = true
    GROUP BY department_id
    HAVING COUNT(*) > 5
),
top_performers AS (
    SELECT 
        employee_id,
        name,
        department_id,
        salary,
        performance_rating,
        ROW_NUMBER() OVER (PARTITION BY department_id ORDER BY performance_rating DESC, salary DESC) as dept_rank
    FROM employees
    WHERE performance_rating >= 4.0
)
SELECT 
    eh.name,
    eh.level,
    ds.employee_count,
    ds.avg_salary,
    tp.dept_rank,
    CASE 
        WHEN tp.dept_rank <= 3 THEN 'Top Performer'
        WHEN eh.level = 0 THEN 'Executive'
        WHEN eh.level <= 2 THEN 'Senior Management'
        ELSE 'Regular Employee'
    END as employee_category,
    (SELECT COUNT(*) FROM projects pr WHERE pr.lead_id = eh.employee_id) as projects_led
FROM employee_hierarchy eh
LEFT JOIN department_stats ds ON eh.department_id = ds.department_id
LEFT JOIN top_performers tp ON eh.employee_id = tp.employee_id
WHERE eh.level <= 5
  AND EXISTS (
      SELECT 1 FROM employee_reviews er 
      WHERE er.employee_id = eh.employee_id 
        AND er.review_date >= '2024-01-01'
  )
ORDER BY eh.level, eh.name;
