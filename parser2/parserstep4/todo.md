# このステップで行うテスト

## clauseレベル

- [ ] アスタリスク（*）による全カラム指定
- [ ] 数式でasがない
- [ ] DISTINCT
- [ ] カラムリスト省略のINSERT
- [ ] ORDER BY/ GROUP BY/ DISTINCT での式や位置指定
- [ ] NATURAL JOIN, USING句
- [ ] サブクエリのエイリアス省略
- [ ] 非推奨・非標準SQL構文

---

## 備考
- parserstep4では上記禁止要素を検出してParseErrorにまとめて返す仕組みを実装する。
- 検出困難なもの（例: DBMS依存の微妙な構文）は後続フェーズで対応する場合あり。

## Distinct

例1: 単一カラム

```sql
SELECT DISTINCT name FROM users;
```

例2: 複数カラム

```sql
SELECT DISTINCT name, age FROM users;
```

例3: 集約関数とDISTINCT

```sql
SELECT COUNT(DISTINCT user_id) FROM orders;
```

例4: DISTINCT ON (PostgreSQL専用)

```sql
SELECT DISTINCT ON (user_id) user_id, created_at FROM orders ORDER BY user_id, created_at DESC;
```
