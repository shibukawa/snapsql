package schemaimport

import (
	"fmt"

	tblsconfig "github.com/k1LoW/tbls/config"
)

// Config contains the fully resolved settings for running the schema import pipeline.
type Config struct {
	WorkingDir     string
	TblsConfigPath string
	DocPath        string
	SchemaJSONPath string
	OutputDir      string
	Include        []string
	Exclude        []string
	IncludeViews   bool
	IncludeIndexes bool
	SchemaAware    bool
	DryRun         bool
	Experimental   bool
	Verbose        bool

	logger func(format string, args ...any)

	TblsConfig *tblsconfig.Config
}

// NewConfig creates a Config from Options, applying defaults and copying slices.
func NewConfig(opts Options) Config {
	include := append([]string(nil), opts.Include...)
	exclude := append([]string(nil), opts.Exclude...)

	cfg := Config{
		WorkingDir:     opts.WorkingDir,
		TblsConfigPath: opts.TblsConfigPath,
		SchemaJSONPath: opts.SchemaJSONPath,
		OutputDir:      opts.OutputDir,
		Include:        include,
		Exclude:        exclude,
		IncludeViews:   true,
		IncludeIndexes: true,
		SchemaAware:    true,
		DryRun:         opts.DryRun,
		Experimental:   opts.Experimental,
		Verbose:        opts.Verbose,
		logger:         opts.Logger,
	}

	if opts.IncludeViews != nil {
		cfg.IncludeViews = *opts.IncludeViews
	}

	if opts.IncludeIndexes != nil {
		cfg.IncludeIndexes = *opts.IncludeIndexes
	}

	if opts.SchemaAware != nil {
		cfg.SchemaAware = *opts.SchemaAware
	}

	return cfg
}

// DSN returns the resolved database connection string from the tbls configuration.
func (c Config) DSN() string {
	if c.TblsConfig == nil {
		return ""
	}

	return c.TblsConfig.DSN.URL
}

func (c Config) logf(format string, args ...any) {
	if !c.Verbose || c.logger == nil {
		return
	}

	c.logger(fmt.Sprintf(format, args...))
}
