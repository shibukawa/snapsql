---
name: "fixture_upsert"
dialect: "postgres"
---

# Fixture Upsert Strategy Test

## Description
Validate upsert strategy inserts new rows and updates existing ones based on primary key.

## SQL
```sql
-- Include email to validate upserted column changes
SELECT id, name, email FROM users ORDER BY id;
```

## Test Cases

### Test: Upsert rows

**Parameters:**
```yaml
dummy: 1
```

**Fixtures: users[clear-insert]**
```yaml
- id: 1
  name: "Alice"
  email: "alice@example.com"
- id: 2
  name: "Bob"
  email: "bob@example.com"
```

**Fixtures: users[upsert]**
```yaml
- id: 1
  name: "Alice-updated"
  email: "alice2@example.com"
- id: 3
  name: "Charlie"
  email: "charlie@example.com"
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice-updated"
  email: "alice2@example.com"
- id: 2
  name: "Bob"
  email: "bob@example.com"
- id: 3
  name: "Charlie"
  email: "charlie@example.com"
```
