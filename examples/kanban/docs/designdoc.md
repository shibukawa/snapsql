# Kanban Sample Application Design

## Overview

The Kanban sample demonstrates how SnapSQL-generated queries back a Vue-based board management UI. This document captures the user-facing screens, the interactions available on each screen, and the HTTP APIs plus SQL statements that power those interactions. It also highlights API gaps where queries exist but are not yet wired into the handlers.

## Screen Inventory

| Screen | Description | Primary Data Sources |
| --- | --- | --- |
| Board Dashboard | Landing view listing all boards and providing creation entry points. | `GET /api/boards`, `POST /api/boards` |
| Board Detail | Board-level workspace showing columns (lists) and cards with inline editing actions. | `GET /api/boards/{id}`, `GET /api/boards/{id}/tree`, list & card mutation endpoints |
| Card Detail Drawer | Drawer shown from the board detail view for inspecting a single card and its comments. | `GET /api/cards/{id}/comments`, `POST /api/cards/{id}/comments` |

## Operations by Screen

### Board Detail

| User Action | API (method & path) | Notes |
| --- | --- | --- |
| View board list | `GET /api/boards` | Uses generated `query.BoardList` sequence. |
| Load board tree (lists + cards) | `GET /api/boards/{id}/tree` | Returns a denormalised hierarchy produced by `query.BoardTree`; listsはテンプレート由来で読み取り専用。Done 判定はテンプレート内の `stage_order` 最大値を参照する。 |
| Create card | `POST /api/cards` | Uses `query.CardCreate` and re-queries the inserted row for response payloads. |
| Update card title/description | `POST|PATCH /api/cards/{id}` | Inline SQL updates cards; generated `card_update.snap.md` exists for future migration. |
| Move card to another list | `POST|PATCH /api/cards/{id}/move` | Inline SQL updates cards; generated `card_move.snap.md` exists for future migration. |
| Reorder card within list | `POST|PATCH /api/cards/{id}/reorder` | Inline SQL updates cards; generated `card_reorder.snap.md` exists for future migration. |
| Create a board | `POST /api/boards` | ワークフロー: (1) 既存アクティブボードを `query.BoardArchive` でアーカイブ、(2) `query.BoardCreate` で新規作成、(3) `query.ListCreate` でテンプレート複製、(4) テンプレートの `stage_order` 最大値を Done ステージとみなし、それ未満のリストにあるカードを `UPDATE cards` で新ボードへ移行、(5) `query.BoardGet` で応答を組み立てる。 |

### Card Detail Drawer

| User Action | API (method & path) | Notes |
| --- | --- | --- |
| View comments on a card | `GET /api/cards/{id}/comments` | Uses generated `query.CardCommentList`. |
| Add comment to a card | `POST /api/cards/{id}/comments` | Generated `card_comment_create.snap.md` exists; handler inserts manually and performs lookup for response. |

> リストのアーカイブ / リネーム / リオーダー / 作成は API から提供しない。テンプレートを更新したい場合は新しいボードを作成し、テンプレート再展開とカード移行で対応する。Done 判定もテンプレートに依存し、`list_templates` の最大 `stage_order` が Done ステージとなる。

## Outstanding Gaps

- **Board creationフローのテスト拡充**: カード移行とテンプレート複製を含むシナリオテストが未整備。`list_templates` の最大 `stage_order` を Done とみなした移行ロジックを検証する。
- **Inline SQLの置き換え**: カード更新・移動・コメント作成など一部ハンドラが依然として手書きSQLを保持。既存SnapSQLクエリへの移行で統一性を高める。
- **Comment fetch consistency after creation**: ハンドラが `fetchCardCommentByID` で再読込している。`card_comment_create` の戻り値を直接レスポンスに使えるよう改善余地あり。
- **テンプレート変更検討**: `stage_order < 4` を Done 判定に利用しているため、将来的に list_templates へ Done フラグを追加して柔軟性を確保する案を検討する。

This document should be kept in sync whenever new screens, API endpoints, or queries are added to the Kanban sample.
