-- ウインドウ関数を含むクエリ
SELECT 
    employee_id,
    name,
    department,
    salary,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) as dept_rank,
    RANK() OVER (ORDER BY salary DESC) as overall_rank,
    SUM(salary) OVER (PARTITION BY department ORDER BY salary ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) as running_total,
    LAG(salary, 1) OVER (PARTITION BY department ORDER BY salary) as prev_salary,
    LEAD(salary, 1) OVER (PARTITION BY department ORDER BY salary) as next_salary,
    PERCENT_RANK() OVER (PARTITION BY department ORDER BY salary) as percentile
FROM employees
WHERE active = true
ORDER BY department, salary DESC;
