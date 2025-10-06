package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	snapsql "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/schemaimport"
)

var ErrTblsDatabaseUnavailable = errors.New("tbls database configuration unavailable")

func resolveConfigBaseDir(configPath string) string {
	if configPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "."
		}

		return cwd
	}

	if !filepath.IsAbs(configPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "."
		}

		return filepath.Dir(filepath.Join(cwd, configPath))
	}

	return filepath.Dir(configPath)
}

func isTblsConfigPath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	name := strings.ToLower(filepath.Base(path))

	return strings.HasSuffix(name, ".tbls.yml") || strings.HasSuffix(name, ".tbls.yaml") || name == "tbls.yml" || name == "tbls.yaml"
}

func buildTblsOptions(ctx *Context) schemaimport.Options {
	baseDir := resolveConfigBaseDir(ctx.Config)

	opts := schemaimport.Options{WorkingDir: baseDir, Verbose: ctx.Verbose}

	if isTblsConfigPath(ctx.Config) {
		opts.TblsConfigPath = ctx.Config
	}

	if ctx.Verbose {
		opts.Logger = func(format string, args ...any) {
			color.Cyan("tbls runtime: "+format, args...)
		}
	}

	return opts
}

func resolveDatabaseFromTbls(ctx *Context) (*snapsql.Database, error) {
	opts := buildTblsOptions(ctx)

	cfg, err := schemaimport.ResolveConfig(context.Background(), opts)
	if err != nil {
		if errors.Is(err, schemaimport.ErrTblsConfigNotFound) {
			return nil, ErrTblsDatabaseUnavailable
		}

		return nil, fmt.Errorf("resolve tbls config: %w", err)
	}

	dsn := strings.TrimSpace(cfg.DSN())
	if dsn == "" {
		return nil, ErrTblsDatabaseUnavailable
	}

	db := &snapsql.Database{
		Driver:     normalizeSQLDriverName(determineDriver(dsn)),
		Connection: dsn,
	}

	return db, nil
}

func loadRuntimeTables(ctx *Context) map[string]*snapsql.TableInfo {
	opts := buildTblsOptions(ctx)

	runtime, err := schemaimport.LoadRuntime(context.Background(), opts)
	if err != nil {
		if ctx.Verbose {
			color.Yellow("tbls runtime unavailable: %v", err)
		}

		return nil
	}

	tables := runtime.TablesByName()
	if len(tables) == 0 {
		return nil
	}

	if ctx.Verbose {
		color.Cyan("Loaded %d tables via tbls schema JSON", len(tables))
	}

	return tables
}
