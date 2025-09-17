---
name: "test_insert_verify"
dialect: "sqlite"
---

# INSERT with Verify Query Test

## Description

Test INSERT statement with comprehensive Verify Query validation.

## Parameters
```yaml
new_name: "David Lee"
new_email: "david@example.com"
new_age: 32
new_dept_id: 1
```

## SQL
```sql
INSERT INTO users (name, email, age, status, department_id) 
VALUES ('David Lee', 'david@example.com', 32, 'inactive', 1);
```

## Test Cases

### Test: INSERT with Complex Verify Query

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Existing User 1"
  email: "existing1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Existing User 2"
  email: "existing2@example.com"
  age: 28
  status: "active"
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
-- Check total user count
SELECT COUNT(*) as total_users FROM users;

-- Check new user details with department
SELECT u.name, u.email, u.age, d.name as department_name
FROM users u
JOIN departments d ON u.department_id = d.id
WHERE u.email = 'david@example.com';

-- Check department statistics
SELECT d.name as department, COUNT(u.id) as user_count
FROM departments d
LEFT JOIN users u ON d.id = u.department_id
GROUP BY d.id, d.name
ORDER BY d.name;
```

**Expected Results:**
```yaml
- total_users: 3
- name: "David Lee"
  email: "david@example.com"
  age: 32
  department_name: "Engineering"
- department: "Design"
  user_count: 1
- department: "Engineering"
  user_count: 2
```
