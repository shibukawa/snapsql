package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	snapsql "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/schemaimport"
)

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

func loadRuntimeTables(ctx *Context) map[string]*snapsql.TableInfo {
	baseDir := resolveConfigBaseDir(ctx.Config)

	opts := schemaimport.Options{WorkingDir: baseDir}
	if ctx.Config != "" {
		opts.TblsConfigPath = ctx.Config
	}

	opts.Verbose = ctx.Verbose
	if ctx.Verbose {
		opts.Logger = func(format string, args ...any) {
			color.Cyan("tbls runtime: "+format, args...)
		}
	}

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
