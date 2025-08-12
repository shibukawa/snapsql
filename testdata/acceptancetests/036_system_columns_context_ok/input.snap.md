# Create User with System Columns Context

## Description

Create a new user with automatic system column handling via context.

## Parameters

```yaml
name: string
email: string
```

## SQL

```sql
INSERT INTO users (name, email) VALUES (/*= name */'John', /*= email */'john@example.com')
```
