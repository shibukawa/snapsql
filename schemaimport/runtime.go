package schemaimport

import (
	"context"
	"strings"

	snapsql "github.com/shibukawa/snapsql"
)

// Runtime holds resolved tbls configuration alongside converted SnapSQL schemas.
type Runtime struct {
	Config  Config
	Schemas []snapsql.DatabaseSchema
}

// LoadRuntime resolves tbls configuration from opts, loads schema JSON, and converts it to SnapSQL structures.
func LoadRuntime(ctx context.Context, opts Options) (*Runtime, error) {
	cfg, err := ResolveConfig(ctx, opts)
	if err != nil {
		return nil, err
	}

	importer := NewImporter(cfg)
	if err := importer.LoadSchemaJSON(ctx); err != nil {
		return nil, err
	}

	schemas, err := importer.Convert(ctx)
	if err != nil {
		return nil, err
	}

	if cfg.Verbose {
		tables := 0
		views := 0

		for _, db := range schemas {
			tables += len(db.Tables)
			views += len(db.Views)
		}

		cfg.logf("Runtime prepared: schemas=%d tables=%d views=%d", len(schemas), tables, views)
	}

	return &Runtime{Config: cfg, Schemas: schemas}, nil
}

// TablesByName returns a lookup map keyed by table name and schema-qualified name.
func (r *Runtime) TablesByName() map[string]*snapsql.TableInfo {
	tables := make(map[string]*snapsql.TableInfo)
	if r == nil {
		return tables
	}

	for _, db := range r.Schemas {
		for _, tbl := range db.Tables {
			if tbl == nil {
				continue
			}

			key := tbl.Name
			tables[key] = tbl

			if strings.Contains(key, ".") {
				continue
			}

			if tbl.Schema != "" {
				qualified := tbl.Schema + "." + tbl.Name
				tables[qualified] = tbl
			}
		}
	}

	return tables
}
