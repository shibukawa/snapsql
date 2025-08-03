---
name: "test_postgres_basic"
dialect: "postgres"
---

# PostgreSQL Basic Test

## Description

Test basic PostgreSQL operations with RETURNING clause and Verify Query.

## Parameters
```yaml
```

## SQL
```sql
INSERT INTO users (name, email, age, status, department_id) 
VALUES ('PostgreSQL User', 'postgres@example.com', 28, 'active', 1)
RETURNING id, name, email, age, status, department_id;
```

## Test Cases

### Test: PostgreSQL INSERT with RETURNING

**Parameters:**
```yaml
dummy: "value"
```

**Fixtures: users[clear-insert]**
```yaml
- name: "Existing User"
  email: "existing@example.com"
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

**Expected Results:**
```yaml
- name: "PostgreSQL User"
  email: "postgres@example.com"
  age: 28
  status: "active"
  department_id: 1
```
