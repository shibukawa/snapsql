---
function_name: insert_all_sub_departments
---

# Insert All Sub-Departments

## Description

Insert all sub-departments with nested for loops.

## Parameters

```yaml
departments:
- department_name: string
  department_code: string
  sub_departments:
  - id: string
    name: string
```

## SQL

```sql
INSERT INTO sub_departments (id, name, department_code, department_name)
VALUES
/*# for dept : departments */
    /*# for sub : dept.sub_departments */
    (/*= dept.department_code + "-" + sub.id */'1-101', /*= sub.name */'Engineering Team A', /*= dept.department_code */'1', /*= dept.department_name */'Engineering')
    /*# end */
/*# end */;
```
