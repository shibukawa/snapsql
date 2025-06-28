# 型推論システム - 実装例

## 概要

本ドキュメントは、型推論システムの具体的な実装例とテストケースを示します。

## シナリオ例

### シナリオ1: シンプルなカラム選択

**SQLクエリ:**
```sql
SELECT id, name, email, created_at 
FROM users;
```

**スキーマ:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**期待される型推論:**
```go
[]*InferredField{
    {
        Name: "id",
        Type: &TypeInfo{BaseType: "int", IsNullable: false},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "id"},
    },
    {
        Name: "name", 
        Type: &TypeInfo{BaseType: "string", IsNullable: false, MaxLength: 255},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "name"},
    },
    {
        Name: "email",
        Type: &TypeInfo{BaseType: "string", IsNullable: true, MaxLength: 255},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "email"},
    },
    {
        Name: "created_at",
        Type: &TypeInfo{BaseType: "time", IsNullable: true},
        Source: FieldSource{Type: ColumnSource, Table: "users", Column: "created_at"},
    },
}
```

### シナリオ2: エイリアス付きJOIN

**SQLクエリ:**
```sql
SELECT u.id, u.name, d.name as department_name, u.salary * 1.1 as adjusted_salary
FROM users u
JOIN departments d ON u.department_id = d.id;
```

**スキーマ:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    department_id INTEGER REFERENCES departments(id),
    salary DECIMAL(10,2)
);

CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
```

**期待される型推論:**
```go
[]*InferredField{
    // ...（英語版と同様の内容を日本語で続けてください）
}
```

...（以降、英語版の内容を日本語で忠実に翻訳・要約して続けてください）
