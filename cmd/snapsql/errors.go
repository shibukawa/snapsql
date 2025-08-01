package main

import "errors"

// Sentinel errors for command operations
var (
	ErrGeneratorNotConfigured = errors.New("generator not configured")
	ErrPluginNotFound         = errors.New("plugin not found")
	ErrInputFileNotExist      = errors.New("input file does not exist")
	ErrNoDatabasesConfigured  = errors.New("no databases configured")
	ErrEnvironmentNotFound    = errors.New("environment not found")
	ErrMissingDBOrEnv         = errors.New("missing database or environment")
	ErrEmptyConnectionString  = errors.New("empty connection string")
	ErrEmptyDatabaseType      = errors.New("empty database type")
)
