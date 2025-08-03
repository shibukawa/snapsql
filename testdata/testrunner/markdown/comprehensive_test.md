---
name: "comprehensive_sqlite_test"
dialect: "sqlite"
---

# Comprehensive SQLite Test Suite

## Description

Comprehensive test suite covering all SQL operations (SELECT/INSERT/UPDATE/DELETE) with various validation strategies.

## Parameters
```yaml
test_name: "Comprehensive Test"
```

## SQL
```sql
-- This will be overridden by individual test cases
SELECT 1 as placeholder;
```

## Test Cases

### Test: SELECT Query Basic

**Parameters:**
```yaml
min_age: 18
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
  age: 30
  status: "active"
  department_id: 2
- id: 3
  name: "Bob Wilson"
  email: "bob@example.com"
  age: 16
  status: "inactive"
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

**Expected Results:**
```yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
  age: 30
  status: "active"
  department_id: 2
```

### Test: INSERT without RETURNING

**Parameters:**
```yaml
new_name: "Alice Johnson"
new_email: "alice@example.com"
new_age: 28
new_dept_id: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Existing User"
  email: "existing@example.com"
  age: 30
  status: "active"
  department_id: 2
```

**Expected Results:**
```yaml
- rows_affected: 1
- users[count]: 2
- users[exists]:
  - name: "Alice Johnson"
    email: "alice@example.com"
    exists: true
```

### Test: INSERT with RETURNING

**Parameters:**
```yaml
new_name: "Charlie Brown"
new_email: "charlie@example.com"
new_age: 35
new_dept_id: 2
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Existing User"
  email: "existing@example.com"
  age: 30
  status: "active"
  department_id: 1
```

**Expected Results:**
```yaml
- id: 2
  name: "Charlie Brown"
  email: "charlie@example.com"
  age: 35
  status: "inactive"
  department_id: 2
```

### Test: INSERT with Verify Query

**Parameters:**
```yaml
new_name: "David Lee"
new_email: "david@example.com"
new_age: 32
new_dept_id: 1
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

### Test: UPDATE without RETURNING

**Parameters:**
```yaml
min_age: 18
new_status: "active"
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

**Expected Results:**
```yaml
- rows_affected: 2
- users[count]: 3
- users[exists]:
  - age: 25
    status: "active"
    exists: true
  - age: 30
    status: "active"
    exists: true
  - age: 16
    status: "inactive"
    exists: true
```

### Test: UPDATE with RETURNING

**Parameters:**
```yaml
target_email: "adult1@example.com"
new_name: "Updated Name"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Original Name"
  email: "adult1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Other User"
  email: "other@example.com"
  age: 30
  status: "active"
  department_id: 2
```

**Expected Results:**
```yaml
- id: 1
  name: "Updated Name"
  email: "adult1@example.com"
  age: 25
  status: "active"
  department_id: 1
```

### Test: UPDATE with Verify Query

**Parameters:**
```yaml
old_dept_id: 1
new_dept_id: 2
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "User 1"
  email: "user1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "User 2"
  email: "user2@example.com"
  age: 28
  status: "active"
  department_id: 1
- id: 3
  name: "User 3"
  email: "user3@example.com"
  age: 30
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
-- Check department transfer results
SELECT 
  COUNT(*) as total_users,
  SUM(CASE WHEN department_id = 1 THEN 1 ELSE 0 END) as eng_users,
  SUM(CASE WHEN department_id = 2 THEN 1 ELSE 0 END) as design_users
FROM users;

-- Check transferred users details
SELECT u.name, u.email, d.name as new_department
FROM users u
JOIN departments d ON u.department_id = d.id
WHERE u.id IN (1, 2)
ORDER BY u.id;
```

**Expected Results:**
```yaml
- total_users: 3
  eng_users: 0
  design_users: 3
- name: "User 1"
  email: "user1@example.com"
  new_department: "Design"
- name: "User 2"
  email: "user2@example.com"
  new_department: "Design"
```

### Test: DELETE without RETURNING

**Parameters:**
```yaml
max_age: 25
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Young User 1"
  email: "young1@example.com"
  age: 22
  status: "active"
  department_id: 1
- id: 2
  name: "Young User 2"
  email: "young2@example.com"
  age: 25
  status: "active"
  department_id: 2
- id: 3
  name: "Old User"
  email: "old@example.com"
  age: 35
  status: "active"
  department_id: 1
```

**Expected Results:**
```yaml
- rows_affected: 2
- users[count]: 1
- users[exists]:
  - age: 22
    exists: false
  - age: 25
    exists: false
  - age: 35
    exists: true
```

### Test: DELETE with RETURNING

**Parameters:**
```yaml
target_status: "inactive"
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Active User 1"
  email: "active1@example.com"
  age: 25
  status: "active"
  department_id: 1
- id: 2
  name: "Inactive User"
  email: "inactive@example.com"
  age: 28
  status: "inactive"
  department_id: 2
- id: 3
  name: "Active User 2"
  email: "active2@example.com"
  age: 30
  status: "active"
  department_id: 1
```

**Expected Results:**
```yaml
- id: 2
  name: "Inactive User"
  email: "inactive@example.com"
  age: 28
  status: "inactive"
  department_id: 2
```

### Test: DELETE with Verify Query

**Parameters:**
```yaml
dept_to_delete: 2
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
