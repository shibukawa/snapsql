---
name: "complex_verify_test"
dialect: "sqlite"
---

# Complex Verify Query Test

## Description

Test complex Verify Query with JOINs and aggregations.

## Parameters
```yaml
```

## SQL
```sql
UPDATE users SET status = 'active' WHERE age >= 18;
```

## Test Cases

### Test: Update with Complex Verification

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Minor User"
  email: "minor@example.com"
  age: 16
  status: "inactive"
  department_id: 1
- id: 2
  name: "Adult User 1"
  email: "adult1@example.com"
  age: 25
  status: "inactive"
  department_id: 1
- id: 3
  name: "Adult User 2"
  email: "adult2@example.com"
  age: 30
  status: "inactive"
  department_id: 2
```

**Fixtures: departments[clear-insert]**
```yaml
- id: 1
  name: "Engineering"
  description: "Software development"
- id: 2
  name: "Design"
  description: "UI/UX design"
```

**Verify Query:**
```sql
-- Check activation statistics
SELECT 
  COUNT(*) as total_users,
  SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_users,
  SUM(CASE WHEN status = 'inactive' THEN 1 ELSE 0 END) as inactive_users
FROM users;

-- Check active users by department
SELECT d.name as department, COUNT(u.id) as active_count
FROM departments d
LEFT JOIN users u ON d.id = u.department_id AND u.status = 'active'
GROUP BY d.id, d.name
ORDER BY d.name;

-- Check that only adults were activated
SELECT COUNT(*) as activated_minors
FROM users 
WHERE status = 'active' AND age < 18;
```

**Expected Results:**
```yaml
- total_users: 3
  active_users: 2
  inactive_users: 1
- department: "Design"
  active_count: 1
- department: "Engineering"
  active_count: 1
- activated_minors: 0
```
