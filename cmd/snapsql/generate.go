package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

// GenerateCommand represents the generate command
type GenerateCommand struct {
	Input  string `help:"Input SQL file or directory" short:"i" default:"."`
	Output string `help:"Output directory" short:"o" default:"./generated"`
	Pretty bool   `help:"Pretty print JSON output" short:"p" default:"true"`
}

// Run executes the generate command
func (cmd *GenerateCommand) Run() error {
	// Create output directory if it doesn't exist
	err := os.MkdirAll(cmd.Output, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Check if input is a file or directory
	info, err := os.Stat(cmd.Input)
	if err != nil {
		return fmt.Errorf("failed to stat input: %v", err)
	}

	if info.IsDir() {
		// Process all SQL files in the directory
		return processDirectory(cmd.Input, cmd.Output, cmd.Pretty)
	}

	// Process a single file
	return processFile(cmd.Input, cmd.Output, cmd.Pretty)
}

// processDirectory processes all SQL files in a directory
func processDirectory(inputDir, outputDir string, pretty bool) error {
	// Walk the directory
	return filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the file is a SQL file
		if !strings.HasSuffix(path, ".snap.sql") && !strings.HasSuffix(path, ".snap.md") {
			return nil
		}

		// Process the file
		return processFile(path, outputDir, pretty)
	})
}

// processFile processes a single SQL file
func processFile(inputFile, outputDir string, pretty bool) error {
	// Read the file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", inputFile, err)
	}

	// Generate the intermediate format
	result, err := intermediate.GenerateIntermediateFormat(string(content), inputFile)
	if err != nil {
		return fmt.Errorf("failed to generate intermediate format for %s: %v", inputFile, err)
	}

	// Create the output file path
	relPath, err := filepath.Rel(filepath.Dir(outputDir), inputFile)
	if err != nil {
		relPath = filepath.Base(inputFile)
	}
	outputFile := filepath.Join(outputDir, strings.TrimSuffix(relPath, filepath.Ext(relPath))+".json")

	// Create the output directory if it doesn't exist
	err = os.MkdirAll(filepath.Dir(outputFile), 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory for %s: %v", outputFile, err)
	}

	// Marshal the result to JSON
	var jsonData []byte
	if pretty {
		jsonData, err = json.MarshalIndent(result, "", "  ")
	} else {
		jsonData, err = json.Marshal(result)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal result to JSON: %v", err)
	}

	// Write the output file
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file %s: %v", outputFile, err)
	}

	fmt.Printf("Generated %s\n", outputFile)
	return nil
}
