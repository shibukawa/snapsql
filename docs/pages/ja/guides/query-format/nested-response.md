# レスポンスのネスト

JOIN 結果を階層化して返すためのルールを説明します。

ルール:

- フィールド名に `AS parent__child__...` のように `__` 区切りを使うと、ネストされたスライス/オブジェクトにマッピングされます。
- 各階層は主キーとなるフィールドを含める必要があります。
- NULL の子要素は空スライスとして扱われます。

サンプル:

```sql
SELECT
  b.id,
  b.name,
  l.id AS lists__id,
  l.name AS lists__name,
  c.id AS lists__cards__id,
  c.title AS lists__cards__title
FROM boards b
LEFT JOIN lists l ON l.board_id = b.id
LEFT JOIN cards c ON c.list_id = l.id
```

詳しい挙動は `examples/kanban` の生成コードを参照してください。
