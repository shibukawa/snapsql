---
name: "test_delete_verify"
dialect: "sqlite"
---

# DELETE with Verify Query Test

## Description

Test DELETE statement with comprehensive Verify Query validation.

## Parameters
```yaml
dept_to_delete: 2
```

## SQL
```sql
DELETE FROM users WHERE department_id = 2;
```

## Test Cases

### Test: DELETE with Complex Verify Query

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Eng User 1"
  email: "eng1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Design User 1"
  email: "design1@example.com"
  age: 28
  status: "active"
  department_id: 2
- id: 3
  name: "Design User 2"
  email: "design2@example.com"
  age: 30
  status: "active"
  department_id: 2
- id: 4
  name: "Eng User 2"
  email: "eng2@example.com"
  age: 32
  status: "active"
  department_id: 1
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
-- Check deletion results
SELECT 
  COUNT(*) as remaining_users,
  COUNT(CASE WHEN department_id = 1 THEN 1 END) as eng_users,
  COUNT(CASE WHEN department_id = 2 THEN 1 END) as design_users
FROM users;

-- Check remaining users by department
SELECT d.name as department, u.name as user_name
FROM departments d
LEFT JOIN users u ON d.id = u.department_id
WHERE u.id IS NOT NULL
ORDER BY d.name, u.name;

-- Verify deleted users don't exist
SELECT COUNT(*) as deleted_design_users
FROM users
WHERE department_id = 2;
```

**Expected Results:**
```yaml
- remaining_users: 2
  eng_users: 2
  design_users: 0
- department: "Engineering"
  user_name: "Eng User 1"
- department: "Engineering"
  user_name: "Eng User 2"
- deleted_design_users: 0
```
