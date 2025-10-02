# Kanban Sample

This directory hosts the end-to-end Kanban example for SnapSQL. It contains the query templates, generated Go code, HTTP handlers, and the reference frontend.

## SQL Test

```bash
% snapsql test --schema=./sql/schema.sql

=== Fixture Test Summary ===
Tests: 17 total, 17 passed, 0 failed
Duration: 0.019s

All fixture tests passed! ✅
```

## Code generation

To generate code, run [tbls](https://github.com/k1LoW/tbls) and generate schema file(`dbdoc/schema.json`) for snapsql. It requires initialized database to generate.

```bash
% sqlite3 kanban.db < sql/schema.sql
% tbls doc
```

Run SnapSQL commands from this directory so that `snapsql.yaml` and all relative paths resolve correctly:

```bash
cd examples/kanban
snapsql generate
```

The command updates:

- `internal/query/` – typed Go wrappers generated from the SQL templates

## Database fixtures

SQLite databases for local testing are stored in this directory (`kanban.db` and `snapsql.db`). Use the `--init` flag on the demo server to recreate them when needed:

```bash
go run ./cmd/kanban --init
```

## Debug Mode

Launch the backend server and frontend dev server independently.

```bash
% go run ./cmd/kanban
2025/10/03 07:56:00 Connected to database: ~/examples/kanban/cmd/kanban/kanban.db
2025/10/03 07:56:00 info: serving API-only mode (no frontend assets)
2025/10/03 07:56:00 Backend server starting on :8080
```

```bash
% cd frontend
% npm install
% npm run dev

> frontend@0.0.0 dev
> vite


  VITE v7.1.7  ready in 163 ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: use --host to expose
  ➜  press h + enter to show help
```

Then open frontend's dev server via browsers.

## Production build

The Vue frontend lives under `frontend/`. Build assets before running the demo server so the embedded files are present. Then build backend. Go module bundles frontend's assets in it.

```bash
% cd frontend
% npm install
% npm run build
% cd ..
% go build ./cmd/kanban
% ./kanban
```
