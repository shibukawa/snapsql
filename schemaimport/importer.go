package schemaimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	tblsschema "github.com/k1LoW/tbls/schema"

	snapsql "github.com/shibukawa/snapsql"
)

const (
	snapTypeString   = "string"
	snapTypeInt      = "int"
	snapTypeFloat    = "float"
	snapTypeBool     = "bool"
	snapTypeDate     = "date"
	snapTypeTime     = "time"
	snapTypeDateTime = "datetime"
	snapTypeJSON     = "json"
	snapTypeArray    = "array"
	snapTypeBinary   = "binary"
)

// Importer orchestrates loading schema JSON, converting it, and writing YAML outputs.
type Importer struct {
	cfg          *Config
	schema       *tblsschema.Schema
	schemaLoaded bool
}

// NewImporter constructs an Importer from a Config.
func NewImporter(cfg Config) *Importer {
	copyCfg := cfg
	return &Importer{cfg: &copyCfg}
}

// Config returns the resolved configuration backing the importer.
func (i *Importer) Config() *Config {
	if i == nil {
		return nil
	}

	return i.cfg
}

// LoadSchemaJSON loads the tbls JSON artefact into memory ready for conversion.
func (i *Importer) LoadSchemaJSON(ctx context.Context) error {
	if i == nil {
		return ErrImporterNil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if i.cfg == nil {
		return ErrImporterConfigNil
	}

	path := i.cfg.SchemaJSONPath
	if strings.TrimSpace(path) == "" {
		return ErrSchemaJSONPathMissing
	}

	if !filepath.IsAbs(path) {
		base := i.cfg.WorkingDir
		if base == "" {
			base = "."
		}

		path = filepath.Join(base, path)
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("schemaimport: open schema JSON %q: %w", path, err)
	}
	defer file.Close()

	schema, err := decodeSchemaJSON(file)
	if err != nil {
		return fmt.Errorf("schemaimport: decode schema JSON %q: %w", path, err)
	}

	if err := validateSchema(schema); err != nil {
		return fmt.Errorf("schemaimport: invalid schema JSON %q: %w", path, err)
	}

	i.logf("Loaded schema JSON (%s) tables=%d", schema.Driver.Name, len(schema.Tables))

	if err := ctx.Err(); err != nil {
		return err
	}

	i.schema = schema
	i.schemaLoaded = true

	return nil
}

// Convert transforms the loaded tbls schema into SnapSQL's internal structures.
func (i *Importer) Convert(ctx context.Context) ([]snapsql.DatabaseSchema, error) {
	if i == nil {
		return nil, ErrImporterNil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !i.schemaLoaded || i.schema == nil {
		return nil, ErrSchemaNotLoaded
	}

	schemas := make(map[string]*snapsql.DatabaseSchema)

	driverName := normalizeDriverName(i.schema.Driver.Name)
	dbInfo := snapsql.DatabaseInfo{
		Type:    driverName,
		Version: i.schema.Driver.DatabaseVersion,
		Name:    inferDatabaseName(i.cfg, i.schema),
	}

	i.logf("Converting schema for driver=%s tables=%d", driverName, len(i.schema.Tables))

	for _, tbl := range i.schema.Tables {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if tbl == nil {
			continue
		}

		schemaName, tableName := splitSchemaAndName(tbl.Name, i.schema.Driver)

		switch strings.ToUpper(tbl.Type) {
		case "VIEW":
			schema := ensureDatabaseSchema(schemas, schemaName, dbInfo)
			view := convertView(tbl, schemaName, tableName)
			schema.Views = append(schema.Views, view)
		default:
			schema := ensureDatabaseSchema(schemas, schemaName, dbInfo)
			table := convertTable(tbl, schemaName, tableName, driverName)
			schema.Tables = append(schema.Tables, table)
		}
	}

	results := make([]snapsql.DatabaseSchema, 0, len(schemas))
	for _, schema := range schemas {
		schema.DatabaseInfo = dbInfo
		if schema.Name != "" {
			schema.DatabaseInfo.Name = schema.Name
		}

		results = append(results, *schema)
	}

	i.logf("Converted schema JSON -> %d database schema(s)", len(results))

	return results, nil
}

// hasLoadedSchema reports whether a schema JSON payload has been loaded.
func (i *Importer) hasLoadedSchema() bool {
	if i == nil {
		return false
	}

	return i.schemaLoaded
}

func decodeSchemaJSON(r io.Reader) (*tblsschema.Schema, error) {
	dec := json.NewDecoder(r)

	var schema tblsschema.Schema
	if err := dec.Decode(&schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

func validateSchema(s *tblsschema.Schema) error {
	if s == nil {
		return ErrSchemaPayloadNil
	}

	if s.Driver == nil {
		return ErrDriverMetadataMissing
	}

	if strings.TrimSpace(s.Driver.Name) == "" {
		return ErrDriverNameEmpty
	}

	if len(s.Tables) == 0 {
		return ErrSchemaTablesEmpty
	}

	return nil
}

func (i *Importer) logf(format string, args ...any) {
	if i == nil || i.cfg == nil {
		return
	}

	i.cfg.logf(format, args...)
}
func ensureDatabaseSchema(target map[string]*snapsql.DatabaseSchema, schemaName string, info snapsql.DatabaseInfo) *snapsql.DatabaseSchema {
	key := schemaName
	if key == "" {
		key = info.Name
	}

	if key == "" {
		key = "default"
	}

	if existing, ok := target[key]; ok {
		if existing.Name == "" {
			existing.Name = key
		}

		return existing
	}

	schema := &snapsql.DatabaseSchema{
		Name:         schemaName,
		Tables:       []*snapsql.TableInfo{},
		Views:        []*snapsql.ViewInfo{},
		DatabaseInfo: info,
	}

	if schema.Name == "" {
		schema.Name = key
	}

	target[key] = schema

	return schema
}

func convertTable(tbl *tblsschema.Table, schemaName, tableName, driver string) *snapsql.TableInfo {
	columns := make(map[string]*snapsql.ColumnInfo)
	order := make([]string, 0, len(tbl.Columns))

	for _, col := range tbl.Columns {
		if col == nil {
			continue
		}

		columns[col.Name] = &snapsql.ColumnInfo{
			Name:         col.Name,
			DataType:     normalizeColumnType(col, driver),
			Nullable:     col.Nullable,
			DefaultValue: nullStringValue(col.Default),
			Comment:      col.Comment,
			IsPrimaryKey: col.PK,
		}

		order = append(order, col.Name)
	}

	constraints := convertConstraints(tbl)
	markPrimaryKeysFromConstraints(columns, constraints)

	indexes := convertIndexes(tbl)

	return &snapsql.TableInfo{
		Name:        tableName,
		Schema:      schemaName,
		Columns:     columns,
		ColumnOrder: order,
		Constraints: constraints,
		Indexes:     indexes,
		Comment:     tbl.Comment,
	}
}

func convertView(tbl *tblsschema.Table, schemaName, tableName string) *snapsql.ViewInfo {
	definition := tbl.Def

	return &snapsql.ViewInfo{
		Name:       tableName,
		Schema:     schemaName,
		Definition: definition,
		Comment:    tbl.Comment,
	}
}

func convertConstraints(tbl *tblsschema.Table) []snapsql.ConstraintInfo {
	constraints := make([]snapsql.ConstraintInfo, 0, len(tbl.Constraints))

	for _, c := range tbl.Constraints {
		if c == nil {
			continue
		}

		info := snapsql.ConstraintInfo{
			Name:              c.Name,
			Type:              strings.ToUpper(c.Type),
			Columns:           append([]string(nil), c.Columns...),
			ReferencedColumns: append([]string(nil), c.ReferencedColumns...),
			Definition:        c.Def,
		}

		if c.ReferencedTable != nil {
			info.ReferencedTable = *c.ReferencedTable
		}

		constraints = append(constraints, info)
	}

	return constraints
}

func convertIndexes(tbl *tblsschema.Table) []snapsql.IndexInfo {
	indexes := make([]snapsql.IndexInfo, 0, len(tbl.Indexes))

	for _, idx := range tbl.Indexes {
		if idx == nil {
			continue
		}

		indexes = append(indexes, snapsql.IndexInfo{
			Name:     idx.Name,
			Columns:  append([]string(nil), idx.Columns...),
			IsUnique: isUniqueIndex(idx),
			Type:     parseIndexType(idx),
		})
	}

	return indexes
}

func markPrimaryKeysFromConstraints(columns map[string]*snapsql.ColumnInfo, constraints []snapsql.ConstraintInfo) {
	if len(columns) == 0 || len(constraints) == 0 {
		return
	}

	lowerIndexed := make(map[string]*snapsql.ColumnInfo, len(columns))
	for name, col := range columns {
		lowerIndexed[strings.ToLower(name)] = col
	}

	for _, constraint := range constraints {
		if constraint.Type != "PRIMARY KEY" {
			continue
		}

		for _, name := range constraint.Columns {
			if column, ok := columns[name]; ok {
				column.IsPrimaryKey = true
				continue
			}

			if column, ok := lowerIndexed[strings.ToLower(name)]; ok {
				column.IsPrimaryKey = true
			}
		}
	}
}

func normalizeDriverName(driver string) string {
	switch strings.ToLower(driver) {
	case "postgresql":
		return "postgres"
	case "postgres", "pgx":
		return "postgres"
	case "mysql":
		return "mysql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return strings.ToLower(driver)
	}
}

func normalizeColumnType(col *tblsschema.Column, driver string) string {
	if col == nil {
		return ""
	}

	raw := strings.TrimSpace(col.Type)

	normalized := strings.ToLower(raw)
	if idx := strings.Index(normalized, "("); idx >= 0 {
		normalized = normalized[:idx]
	}

	normalized = strings.TrimSpace(normalized)

	switch strings.ToLower(driver) {
	case "postgres", "postgresql":
		if snap := mapPostgresType(normalized); snap != "" {
			return snap
		}
	case "mysql":
		if snap := mapMySQLType(normalized); snap != "" {
			return snap
		}
	case "sqlite", "sqlite3":
		if snap := mapSQLiteType(normalized); snap != "" {
			return snap
		}
	}

	return inferGenericType(normalized)
}

func mapPostgresType(t string) string {
	switch t {
	case "integer", "int", "int4", "bigint", "int8", "smallint", "int2", "serial", "bigserial", "smallserial":
		return snapTypeInt
	case "text", "varchar", "character", "char", "bpchar":
		return snapTypeString
	case "numeric", "decimal", "real", "float4", "double precision", "float8", "float":
		return snapTypeFloat
	case "boolean", "bool":
		return snapTypeBool
	case "date":
		return snapTypeDate
	case "time", "time with time zone", "time without time zone", "timetz":
		return snapTypeTime
	case "timestamp", "timestamp with time zone", "timestamp without time zone", "timestamptz":
		return snapTypeDateTime
	case "json", "jsonb":
		return snapTypeJSON
	case "bytea":
		return snapTypeBinary
	case "uuid", "inet", "cidr", "macaddr", "interval", "bit", "varbit":
		return snapTypeString
	}

	if strings.HasSuffix(t, "[]") {
		return snapTypeArray
	}

	return ""
}

func mapMySQLType(t string) string {
	switch t {
	case "int", "integer", "bigint", "smallint", "tinyint", "mediumint":
		return snapTypeInt
	case "varchar", "char", "text", "tinytext", "mediumtext", "longtext":
		return snapTypeString
	case "decimal", "numeric", "float", "double", "real":
		return snapTypeFloat
	case "boolean", "bool":
		return snapTypeBool
	case "date":
		return snapTypeDate
	case "time":
		return snapTypeTime
	case "datetime", "timestamp":
		return snapTypeDateTime
	case "json":
		return snapTypeJSON
	case "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary":
		return snapTypeBinary
	}

	return ""
}

func mapSQLiteType(t string) string {
	t = strings.ToLower(t)
	switch {
	case strings.Contains(t, "int"):
		return snapTypeInt
	case strings.Contains(t, "char") || strings.Contains(t, "text") || strings.Contains(t, "clob"):
		return snapTypeString
	case strings.Contains(t, "real") || strings.Contains(t, "floa") || strings.Contains(t, "doubt"):
		return snapTypeFloat
	case strings.Contains(t, "bool"):
		return snapTypeBool
	case strings.Contains(t, "datetime"):
		return snapTypeDateTime
	case strings.Contains(t, "date"):
		return snapTypeDate
	case strings.Contains(t, "time"):
		return snapTypeTime
	case strings.Contains(t, "json"):
		return snapTypeJSON
	}

	return ""
}

func inferGenericType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch {
	case t == "":
		return snapTypeString
	case strings.HasSuffix(t, "[]"):
		return snapTypeArray
	case strings.Contains(t, "int"):
		return snapTypeInt
	case strings.Contains(t, "char") || strings.Contains(t, "text") || strings.Contains(t, "clob"):
		return snapTypeString
	case strings.Contains(t, "bool"):
		return snapTypeBool
	case strings.Contains(t, "json"):
		return snapTypeJSON
	case strings.Contains(t, "datetime"):
		return snapTypeDateTime
	case strings.Contains(t, "date"):
		return snapTypeDate
	case strings.Contains(t, "time"):
		return snapTypeTime
	case strings.Contains(t, "real") || strings.Contains(t, "floa") || strings.Contains(t, "doubt"):
		return snapTypeFloat
	case strings.Contains(t, "blob") || strings.Contains(t, "binary"):
		return snapTypeBinary
	}

	return snapTypeString
}

func splitSchemaAndName(fullName string, driver *tblsschema.Driver) (string, string) {
	schemaName := ""
	tableName := fullName

	if idx := strings.Index(fullName, "."); idx >= 0 {
		schemaName = fullName[:idx]
		tableName = fullName[idx+1:]
	} else if driver != nil && driver.Meta != nil && driver.Meta.CurrentSchema != "" {
		schemaName = driver.Meta.CurrentSchema
	}

	return schemaName, tableName
}

func nullStringValue(v sql.NullString) string {
	if v.Valid {
		return v.String
	}

	return ""
}

func isUniqueIndex(idx *tblsschema.Index) bool {
	if idx == nil {
		return false
	}

	def := strings.ToUpper(idx.Def)

	return strings.Contains(def, "UNIQUE")
}

func parseIndexType(idx *tblsschema.Index) string {
	if idx == nil {
		return ""
	}

	def := strings.ToUpper(idx.Def)
	switch {
	case strings.Contains(def, "PRIMARY"):
		return "PRIMARY"
	case strings.Contains(def, "UNIQUE"):
		return "UNIQUE"
	default:
		return ""
	}
}

func inferDatabaseName(cfg *Config, schema *tblsschema.Schema) string {
	if cfg != nil && cfg.TblsConfig != nil {
		if dsn := strings.TrimSpace(cfg.TblsConfig.DSN.URL); dsn != "" {
			if name := extractDatabaseNameFromDSN(dsn); name != "" {
				return name
			}
		}

		if cfg.TblsConfig.Name != "" {
			return cfg.TblsConfig.Name
		}
	}

	if schema != nil && schema.Name != "" {
		return schema.Name
	}

	return ""
}

func extractDatabaseNameFromDSN(dsn string) string {
	if dsn == "" {
		return ""
	}

	if strings.HasPrefix(dsn, "sqlite://") {
		trimmed := strings.TrimPrefix(dsn, "sqlite://")
		trimmed = strings.TrimSuffix(trimmed, "/")

		base := filepath.Base(trimmed)
		if base != "." && base != "" {
			if ext := filepath.Ext(base); ext != "" {
				base = strings.TrimSuffix(base, ext)
			}

			return base
		}

		return "sqlite"
	}

	if u, err := url.Parse(dsn); err == nil {
		if name := strings.TrimPrefix(u.Path, "/"); name != "" {
			return name
		}
	}

	return ""
}
