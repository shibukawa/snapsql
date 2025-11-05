# SQL Templating

SQLクエリのテンプレート化について説明します。

## テンプレートの必要性

SnapSQLのテンプレート機能により：

- 同じパターンのクエリを再利用できる
- 動的なクエリ生成が可能
- コードの保守性が向上する

## テンプレートの基本

テンプレートは `{{variable}}` の形式で変数を埋め込みます：

```sql
SELECT * FROM {{table_name}}
WHERE {{condition_column}} = /*= value */1
ORDER BY {{order_column}} {{order_direction}};
```

## テンプレートファイルの作成

`templates/user_queries.md`:

````markdown
# User Queries Template

## Description

ユーザー関連の汎用クエリテンプレート

## Parameters

```yaml
table_name: string
condition_column: string
order_column: string
order_direction: string
value: any
```

## SQL

```sql
SELECT * FROM {{table_name}}
WHERE {{condition_column}} = /*= value */1
ORDER BY {{order_column}} {{order_direction}};
```

## Test Cases

### Get active users

**Parameters:**
```yaml
table_name: users
condition_column: status
order_column: created_at
order_direction: DESC
value: active
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice"
  status: "active"
```
````

## テンプレートの使用

テンプレートを使用するには、クエリファイルで `template` フィールドを指定：

````markdown
# Get Active Users

## Description

アクティブなユーザーを取得

## Template

```yaml
template: templates/user_queries.md
parameters:
  table_name: users
  condition_column: status
  order_column: created_at
  order_direction: DESC
```

## Parameters

```yaml
value: active
```

## Test Cases

### Get active users

**Parameters:**
```yaml
value: active
```

**Expected Results:**
```yaml
- id: 1
  name: "Alice"
  status: "active"
```
````

## 高度なテンプレート機能

### 条件分岐

<code v-pre>{{#if condition}}...{{/if}}</code> で条件分岐：

```sql
SELECT * FROM users
WHERE 1=1
{{#if status}}
  AND status = /*= status */'active'
{{/if}}
{{#if name}}
  AND name LIKE /*= name */'%test%'
{{/if}}
ORDER BY created_at DESC;
```

### ループ

<code v-pre>{{#each items}}...{{/each}}</code> で繰り返し：

```sql
SELECT * FROM users
WHERE id IN (
  {{#each user_ids}}
    /*= . */1{{#unless @last}},{{/unless}}
  {{/each}}
);
```

## テンプレートのベストプラクティス

### 1. 再利用性の高い設計

- 共通パターンをテンプレート化
- パラメータを柔軟に設定可能に

### 2. 適切な粒度

- テンプレートが大きくなりすぎないよう分割
- 単一責任の原則を適用

### 3. ドキュメント化

- テンプレートの使用方法を明確に記述
- パラメータの意味を説明

## 次のステップ

SQLテンプレート化が完了したら、[テストの実行](./testing) に進みましょう。

## 関連セクション

* [テンプレート構文の詳細](../guides/query-format/template-syntax.md)
* [クエリフォーマット入門](../guides/query-format/index.md)
* [共通型とレスポンス型](../guides/query-format/common-types.md)