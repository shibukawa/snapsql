package schemaimport

import "errors"

var (
	// ErrNotImplemented indicates the requested behaviour has not been implemented yet.
	ErrNotImplemented = errors.New("schemaimport: not implemented")
	// ErrImporterNil indicates the importer receiver was nil.
	ErrImporterNil = errors.New("schemaimport: importer is nil")
	// ErrImporterConfigNil indicates the importer configuration is missing.
	ErrImporterConfigNil = errors.New("schemaimport: importer configuration is nil")
	// ErrSchemaJSONPathMissing indicates the schema JSON path was not provided.
	ErrSchemaJSONPathMissing = errors.New("schemaimport: schema JSON path is not configured")
	// ErrSchemaNotLoaded indicates Convert was called before LoadSchemaJSON.
	ErrSchemaNotLoaded = errors.New("schemaimport: schema not loaded; call LoadSchemaJSON first")
	// ErrSchemaPayloadNil indicates the decoded schema payload is nil.
	ErrSchemaPayloadNil = errors.New("schemaimport: schema payload is nil")
	// ErrDriverMetadataMissing indicates driver metadata was absent.
	ErrDriverMetadataMissing = errors.New("schemaimport: driver metadata is missing")
	// ErrDriverNameEmpty indicates the driver name was empty.
	ErrDriverNameEmpty = errors.New("schemaimport: driver name is empty")
	// ErrSchemaTablesEmpty indicates the schema contained no tables.
	ErrSchemaTablesEmpty = errors.New("schemaimport: no tables present in schema")
	// ErrTblsConfigNotFound indicates no tbls configuration file was discovered.
	ErrTblsConfigNotFound = errors.New("schemaimport: no tbls config found")
)
