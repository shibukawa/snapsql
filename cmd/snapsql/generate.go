package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateIntermediateFormat generates intermediate format from a single file
func generateIntermediateFormat(inputFile, outputDir string, pretty bool) error {
	// Check if the input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Read the input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", inputFile, err)
	}

	// TODO: Implement proper intermediate format generation
	// For now, this is a placeholder that indicates the feature is not yet implemented
	_ = content
	_ = outputDir
	_ = pretty

	fmt.Printf("Intermediate format generation for %s is not yet implemented\n", inputFile)
	fmt.Println("This feature will be available in a future version.")

	return nil
}

// generateIntermediateFormats generates intermediate formats for all files in a directory
func generateIntermediateFormats(inputDir, outputDir string, pretty bool) error {
	// Check if the input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", inputDir)
	}

	// Walk through the input directory
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process only .snap.sql and .snap.md files
		if strings.HasSuffix(path, ".snap.sql") || strings.HasSuffix(path, ".snap.md") {
			return generateIntermediateFormat(path, outputDir, pretty)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to process input directory: %v", err)
	}

	return nil
}
