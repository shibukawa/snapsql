package schemaimport

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tblsconfig "github.com/k1LoW/tbls/config"
)

// ResolveConfig hydrates a Config with information sourced from tbls configuration files and defaults.
func ResolveConfig(ctx context.Context, opts Options) (Config, error) {
	_ = ctx

	cfg := NewConfig(opts)

	root := cfg.WorkingDir
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Config{}, fmt.Errorf("schemaimport: resolve working directory: %w", err)
	}

	cfg.WorkingDir = absRoot

	configPath, err := resolveTblsConfigPath(absRoot, cfg.TblsConfigPath)
	if err != nil {
		return Config{}, err
	}

	cfg.logf("Resolved tbls config path: %s", configPath)

	tblsCfg, err := tblsconfig.New()
	if err != nil {
		return Config{}, fmt.Errorf("schemaimport: initialise tbls config: %w", err)
	}

	if err := tblsCfg.Load(configPath); err != nil {
		return Config{}, fmt.Errorf("schemaimport: load tbls config: %w", err)
	}

	cfg.TblsConfigPath = configPath
	cfg.TblsConfig = tblsCfg

	schemaJSONPath, docPath := resolvedSchemaPaths(cfg.SchemaJSONPath, tblsCfg, configPath, absRoot)
	cfg.SchemaJSONPath = schemaJSONPath
	cfg.DocPath = docPath
	cfg.OutputDir = resolvedOutputDir(cfg.OutputDir, absRoot)
	cfg.logf("Resolved schema JSON: %s", schemaJSONPath)
	cfg.logf("Resolved docPath: %s", docPath)

	if cfg.OutputDir != "" {
		cfg.logf("Output directory set to: %s", cfg.OutputDir)
	}

	if len(cfg.Include) > 0 || len(cfg.Exclude) > 0 {
		cfg.logf("Filters include=%v exclude=%v", cfg.Include, cfg.Exclude)
	}

	cfg.Include = append([]string(nil), cfg.Include...)
	cfg.Exclude = append([]string(nil), cfg.Exclude...)

	return cfg, nil
}

func resolveTblsConfigPath(root, explicitPath string) (string, error) {
	if explicitPath != "" {
		if !filepath.IsAbs(explicitPath) {
			explicitPath = filepath.Join(root, explicitPath)
		}

		explicitPath = filepath.Clean(explicitPath)
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("schemaimport: tbls config %q: %w", explicitPath, err)
		}

		return explicitPath, nil
	}

	for _, candidate := range tblsconfig.DefaultConfigFilePaths {
		full := filepath.Join(root, candidate)

		info, err := os.Stat(full)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return "", fmt.Errorf("schemaimport: stat %q: %w", full, err)
		}

		if info.IsDir() {
			continue
		}

		return filepath.Clean(full), nil
	}

	return "", fmt.Errorf("%w in %q", ErrTblsConfigNotFound, root)
}

func resolvedSchemaPaths(explicitPath string, tblsCfg *tblsconfig.Config, configPath, root string) (string, string) {
	if explicitPath != "" {
		resolved := explicitPath
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(root, resolved)
		}

		resolved = filepath.Clean(resolved)

		return resolved, filepath.Dir(resolved)
	}

	docPath := tblsCfg.DocPath
	if strings.TrimSpace(docPath) == "" {
		docPath = tblsconfig.DefaultDocPath
	}

	base := filepath.Dir(configPath)
	if base == "" {
		base = root
	}

	if !filepath.IsAbs(docPath) {
		docPath = filepath.Join(base, docPath)
	}

	docPath = filepath.Clean(docPath)

	return filepath.Join(docPath, tblsconfig.SchemaFileName), docPath
}

func resolvedOutputDir(outputDir, root string) string {
	if outputDir == "" {
		return ""
	}

	if filepath.IsAbs(outputDir) {
		return filepath.Clean(outputDir)
	}

	return filepath.Clean(filepath.Join(root, outputDir))
}
