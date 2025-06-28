# 20250628-json-type-policy.md

## Handling of JSON Types in SnapSQL Type Inference Engine

### Policy & Constraints

- Currently, JSON types (e.g., PostgreSQL's json/jsonb, MySQL's json, SQLite's json) are treated as `any` type in the type inference engine.
- This is because providing strict type representations or structures for JSON types for each DB dialect would make it difficult to unify and normalize pull-type inference.
- For example, functions like `JSONB_BUILD_OBJECT` also return `any`.
- In the future, if stricter type representations (object, array, scalar, null, etc.) or schema inference are required, the design and pull specification will be extended.

### Impact

- In type inference tests, the expected value for return values or column types of JSON type is `any`.
- DB dialect-specific type names (json, jsonb, json1, etc.) are internally normalized to `any`.
- Any restrictions or caveats due to the return type being `any` should be considered on the application side.

---

This policy may be revised in the future according to extensions of the pull-type inference specification or use cases.
