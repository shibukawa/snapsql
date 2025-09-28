# `tbls`-Backed Schema Import Command

## Overview
We will replace the legacy `snapsql pull` command—which connects directly to live databases—with a new importer that consumes schema metadata already exported by [`tbls`](https://github.com/k1LoW/tbls). SnapSQL will never shell out to `tbls`; instead it reads the JSON artefacts produced by `tbls doc` and converts them into SnapSQL's internal schema structures that downstream commands can consume directly. All SnapSQL subcommands (generate, test, etc.) will read `.tbls.yml` on demand to discover DSNs and schema file locations so configuration lives in a single place.

## Goals
- Provide runtime primitives that transform `tbls` JSON into SnapSQL's internal schema structures without invoking external binaries.
- Preserve behavioural expectations established by the legacy `snapsql pull` command (output layout, defaults, filters).
- Centralise DSN/schema discovery so other SnapSQL commands reuse `.tbls.yml` instead of maintaining bespoke configuration.

## Non-Goals
- Running `tbls doc` on behalf of the user or managing `tbls` installations.
- Persisting alternate schema artefacts (YAML, SQL dumps, etc.).
- Supporting databases that `tbls` does not already emit.

## User Experience
1. Users manage a `.tbls.yml` (or `tbls.yml`) that defines their database DSN, include/exclude filters, and documentation output path. They execute `tbls doc --format json` as part of their build/test workflow, producing `schema.json` inside the configured `docPath`.
2. Users run `Runtime entry points`. The command locates `.tbls.yml`, determines the JSON file path (defaults to `<docPath>/schema.json`, overridable via CLI), and loads that file.
3. SnapSQL converts the JSON into in-memory structures that downstream commands consume directly. Subsequent operations (`snapsql generate`, `snapsql test`) read `.tbls.yml` again to obtain DSNs or schema paths at runtime—no additional SnapSQL-specific configuration files are introduced.

## Runtime API Surface
```
Runtime entry points

APIs:
  schemaimport.ResolveConfig(ctx, Options) -> Config
  schemaimport.LoadRuntime(ctx, Options) -> Runtime
  Runtime.TablesByName() -> map[string]*snapsql.TableInfo
  Runtime.Schemas -> []snapsql.DatabaseSchema
  Options{WorkingDir, TblsConfigPath, SchemaJSONPath, Include/Exclude, Flags}
```

Compatibility notes:
- Defaults mirror the former `snapsql pull` command so existing scripts require minimal changes beyond running `tbls doc` beforehand.
- CLI warnings highlight when SnapSQL filters exclude objects that `tbls` already omitted to aid debugging.

## High-Level Flow
```text
resolve config (.tbls.yml, snapsql.yaml)
   ↓
select schema JSON (CLI flag or <docPath>/schema.json)
   ↓
parse JSON into schema.Schema (tbls module)
   ↓
convert to snapsql.DatabaseSchema[] (apply SnapSQL filters, compute types)
   ↓
expose SnapSQL schema structures for downstream commands
```

## Inputs
- **`tbls` configuration**: `.tbls.yml` / `tbls.yml`. Provides DSN, filters, and `docPath`. SnapSQL reads this file each time it needs DSN or schema metadata—no additional persisted config.
- **Schema JSON**: Typically `<docPath>/schema.json`, but the CLI can override with `--schema-json` for deterministic fixtures or alternative workflows.
- **SnapSQL configuration (`snapsql.yaml`)**: Continues to define system columns, generator settings, and default include/exclude patterns applied after JSON load.

## Mapping `tbls` JSON → SnapSQL Types
- `schema.Schema` → collection of `snapsql.DatabaseSchema` objects (one per `table.Schema`);
  `schema.Driver.Name` informs `DatabaseInfo.Type`, and `schema.Driver.DatabaseVersion` becomes `DatabaseInfo.Version`.
- `schema.Table` entries map to `snapsql.TableInfo`:
  - `Schema`, `Name`, `Comment`, `Type` (table/view) copied directly.
  - `Columns` preserve order from JSON; each column sets `Nullable`, `Default`, `Comment`, length/precision metadata when available.
  - Existing SnapSQL type mappers (currently in `pull`) migrate into `schemaimport/typemapper` and continue to infer SnapSQL types from `schema.Column.Type` + driver.
  - Primary and foreign keys derive from `table.Constraints` (type lookups) and `table.Relations` for cross-table metadata.
- `schema.Index` entries become `snapsql.IndexInfo`, inferring uniqueness via `Index.Unique` when present or by parsing `Def` as a fallback.
- Views map to `snapsql.ViewInfo` carrying `Def` and `Comment`.
- Any unrecognised fields are logged under verbose mode for diagnostics but otherwise ignored.

## Filters & Overrides
- SnapSQL applies its own include/exclude patterns after loading JSON to ensure legacy behaviour. We reuse the wildcard matcher previously housed in `pull`.
- CLI `--include`/`--exclude` flags still override configuration and operate on the loaded tables/views.
- `--include-views=false` removes entries where `table.Type == "VIEW"`.
- `--include-indexes=false` clears the `Indexes` slice while leaving constraint metadata intact (so downstream generation remains deterministic).

## Expected Inputs and Outputs
- **Config resolution**
  - *Inputs*: Paths provided via CLI (`--tbls-config`, `--schema-json`), working directory, SnapSQL runtime configuration structures, and environment variables (future use).
  - *Outputs*: A `schemaimport.Config` instance containing resolved config path, doc path, schema JSON path, output directory, include/exclude slices, and flags for views/indexes/schema-aware/dry-run, plus the raw `tbls/config.Config` for downstream consumers.
- **Importer construction**
  - *Inputs*: A resolved `schemaimport.Config`, JSON decoder, filesystem handles.
  - *Outputs*: An `schemaimport.Importer` ready to load JSON (`LoadSchemaJSON`) and convert it to `[]snapsql.DatabaseSchema` via `Convert` (returning data structures for downstream consumers). Each method will eventually accept a `context.Context`.
## Implementation Plan
1. **Package Layout**
   - Create `schemaimport` package with:
     - `Config` struct capturing resolved `.tbls.yml`, SnapSQL config, CLI overrides.
     - `Importer` orchestration with methods `LoadSchemaJSON`, `Convert()`.
     - Relocated utilities: type mapper, wildcard matcher, JSON helpers.
   - Update `cmd/snapsql` to route runtime responsibilities through this package, removing the legacy `pull` command entirely.

2. **Config Resolution**
   - Implement search for `.tbls.yml` / `tbls.yml` (and `.yaml` variants) relative to the invocation directory, matching tbls precedence rules.
   - Load the file via `github.com/k1LoW/tbls/config` to reuse canonical parsing, gaining access to `DocPath`, `DSN`, includes/excludes.
   - Merge SnapSQL config and CLI overrides, producing effective include/exclude lists and output path. Configuration is read every invocation—no caching layer or re-serialisation.

3. **JSON Source Selection**
   - Determine the schema JSON path: CLI `--schema-json` > `.tbls.yml` `DocPath` + `schema.SchemaFileName` > SnapSQL defaults.
   - Validate file existence; if missing, emit actionable guidance (e.g. “Run `tbls doc --format json` before importing”).

4. **Parsing & Validation**
   - Unmarshal JSON into `schema.Schema` (from `github.com/k1LoW/tbls/schema`).
   - Ensure required driver metadata (`schema.Driver.Name`) and table structures exist; surface descriptive errors otherwise.

5. **Conversion**
   - Port the existing pull mappers into `schemaimport` (e.g., type mappers) and adjust them to operate on `schema.Schema` inputs.
   - Preserve column order and compute `ColumnOrder` indices.
   - Attach `DatabaseInfo` metadata and propagate comments/labels where available.

6. **Integration**
   - Provide APIs that return `[]snapsql.DatabaseSchema` so commands like `generate` and `test` can work with live JSON results without writing YAML.

7. **CLI Surface**
   - Integrate runtime helpers with existing commands (generate/test) so they can consume tbls JSON directly.
   - Provide feature flags (e.g., --verbose) to surface runtime details without new subcommands.

8. **Shared Runtime Helpers**
   - Expose `schemaimport.ReadConfig()` (or similar) for other commands to obtain DSNs, schema JSON paths, and include/exclude lists by re-reading `.tbls.yml`.
   - Update `generate`, `test`, and related commands to call this helper rather than duplicating config logic in future phases.

9. **Deprecation Hooks**
   - Identify remaining `pull` dependencies and plan replacement using runtime helpers (see roadmap).

## Testing Strategy
- Unit tests for config resolution and JSON path derivation covering: multiple config filenames, overridden doc paths, environment variables, and missing files.
- Fixture-based conversion tests that load known `tbls` JSON exports (e.g. from `examples/kanban/dbdoc/schema.json`) and assert the converted in-memory structures match expectations.
- Failure-mode tests for absent drivers, empty tables array, and unsupported database types.
- Regression tests for type mapping across PostgreSQL/MySQL/SQLite inputs after relocating mapper logic.

## Rollout Plan
- Roll out alongside the existing workflow until JSON runtime becomes the sole path (pull already retired).
- Update documentation/examples to instruct users to run `tbls doc` and rely on runtime helpers instead of schema YAML.
- Collect feedback to ensure runtime helpers cover previous pull use cases.

## Risks & Mitigations
- **Stale JSON artefacts**: Mitigate via `--dry-run` summarising timestamps and reminding users to regenerate when stale; document workflow expectations.
- **Config divergence**: If SnapSQL filters conflict with `.tbls.yml` includes/excludes, emit combined summaries so users can reconcile differences.
- **Schema version drift**: Pin a minimum supported `tbls` version and validate `schema.Schema.Version` fields when available.
- **Dependency footprint**: Monitor the impact of importing `tbls/config` and `tbls/schema`; if heavy, consider extracting only needed struct definitions while documenting trade-offs.

## Open Questions
- Do we need an option to accept multiple schema JSON files (one per schema) or is a single consolidated file sufficient?
- Should we validate JSON freshness (timestamps) before using cached results?
- How should we surface differences between `.tbls.yml` filters and SnapSQL filters (warnings vs. hard errors)?
