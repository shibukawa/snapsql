package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/joho/godotenv"
)

// loadEnvFiles loads .env files if they exist
func loadEnvFiles() error {
	// Try to load .env file from current directory
	if fileExists(".env") {
		if err := godotenv.Load(".env"); err != nil {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
	}
	return nil
}

// expandEnvVars expands environment variables in the format ${VAR} or $VAR
func expandEnvVars(s string) string {
	// Pattern for ${VAR} format
	re1 := regexp.MustCompile(`\$\{([^}]+)\}`)
	s = re1.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		return os.Getenv(varName)
	})

	// Pattern for $VAR format (word boundaries)
	re2 := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	s = re2.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[1:] // Remove $
		return os.Getenv(varName)
	})

	return s
}

// expandConfigEnvVars recursively expands environment variables in config
func expandConfigEnvVars(config *Config) {
	// Expand database connections
	for name, db := range config.Databases {
		db.Connection = expandEnvVars(db.Connection)
		db.Driver = expandEnvVars(db.Driver)
		db.Schema = expandEnvVars(db.Schema)
		db.Database = expandEnvVars(db.Database)
		config.Databases[name] = db
	}

	// Expand constant files
	for i, file := range config.ConstantFiles {
		config.ConstantFiles[i] = expandEnvVars(file)
	}

	// Expand generation paths
	config.Generation.InputDir = expandEnvVars(config.Generation.InputDir)

	// Expand generator output paths
	for name, generator := range config.Generation.Generators {
		generator.Output = expandEnvVars(generator.Output)
		config.Generation.Generators[name] = generator
	}
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// writeFile writes content to a file, creating directories if necessary
func writeFile(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	return os.WriteFile(path, []byte(content), 0644)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// isDirectory checks if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
