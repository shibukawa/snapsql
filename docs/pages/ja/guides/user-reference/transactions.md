# トランザクション

生成されたクライアントはトランザクション制御を呼び出し側に任せる設計です。`executor` 引数には `*sql.Tx` を渡すことでトランザクション内でクエリを実行できます。

例:

```go
tx, err := db.BeginTx(ctx, nil)
defer tx.Rollback()
res, err := UpdateCardTitle(ctx, tx, 123, "New")
if err != nil { return err }
return tx.Commit()
```

設計理由:

- アプリケーションがトランザクション境界を明確に制御できるようにするため
- リードレプリカや複雑な接続ルーティングが必要な場合に柔軟性を保つため
