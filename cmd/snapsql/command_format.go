package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shibukawa/snapsql/formatter"
)

var (
	ErrFileNotFormatted = errors.New("file is not formatted")
	ErrFormattingErrors = errors.New("some files had formatting errors")
)

// FormatCmd represents the format command
type FormatCmd struct {
	Input  string `arg:"" optional:"" help:"Input file or directory (default: stdin)"`
	Output string `short:"o" help:"Output file (default: stdout, or overwrite input file)"`
	Write  bool   `short:"w" help:"Write result to input file instead of stdout"`
	Check  bool   `short:"c" help:"Check if files are formatted (exit 1 if not)"`
	Diff   bool   `short:"d" help:"Show diff instead of rewriting files"`
}

// Run executes the format command
func (cmd *FormatCmd) Run(ctx *Context) error {
	sqlFormatter := formatter.NewSQLFormatter()

	// Handle different input sources
	if cmd.Input == "" {
		// Read from stdin
		return cmd.formatFromReader(sqlFormatter, os.Stdin, os.Stdout, "<stdin>")
	}

	// Check if input is a file or directory
	info, err := os.Stat(cmd.Input)
	if err != nil {
		return fmt.Errorf("failed to stat input: %w", err)
	}

	if info.IsDir() {
		return cmd.formatDirectory(sqlFormatter, cmd.Input)
	}

	return cmd.formatFile(sqlFormatter, cmd.Input)
}

// formatFromReader formats SQL from a reader and writes to a writer
func (cmd *FormatCmd) formatFromReader(sqlFormatter *formatter.SQLFormatter, reader io.Reader, writer io.Writer, filename string) error {
	// Read all input
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var formatted string

	// Check if this is a Markdown file
	if cmd.isMarkdownFile(filename) {
		markdownFormatter := formatter.NewMarkdownFormatter()

		// Use SnapSQL Markdown formatter for .snap.md files
		if strings.HasSuffix(strings.ToLower(filepath.Base(filename)), ".snap.md") {
			formatted, err = markdownFormatter.FormatSnapSQLMarkdown(string(input))
		} else {
			formatted, err = markdownFormatter.Format(string(input))
		}

		if err != nil {
			return fmt.Errorf("failed to format Markdown in %s: %w", filename, err)
		}
	} else {
		// Format as regular SQL
		formatted, err = sqlFormatter.Format(string(input))
		if err != nil {
			return fmt.Errorf("failed to format SQL in %s: %w", filename, err)
		}
	}

	// Handle check mode
	if cmd.Check {
		if strings.TrimSpace(string(input)) != strings.TrimSpace(formatted) {
			fmt.Fprintf(os.Stderr, "%s is not formatted\n", filename)
			return ErrFileNotFormatted
		}

		return nil
	}

	// Handle diff mode
	if cmd.Diff {
		return cmd.showDiff(string(input), formatted, filename)
	}

	// Write formatted output
	_, err = writer.Write([]byte(formatted))

	return err
}

// formatFile formats a single file
func (cmd *FormatCmd) formatFile(sqlFormatter *formatter.SQLFormatter, filename string) error {
	// Check if it's a SnapSQL file
	if !cmd.isSnapSQLFile(filename) {
		if !cmd.Check {
			fmt.Fprintf(os.Stderr, "Skipping non-SnapSQL file: %s\n", filename)
		}

		return nil
	}

	// Read the file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	// Determine output destination
	var (
		writer     io.Writer
		outputFile *os.File
	)

	if cmd.Write || cmd.Output == filename {
		// Write to temporary file first
		tempFile, err := os.CreateTemp(filepath.Dir(filename), ".snapsql-format-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}

		defer func() {
			tempFile.Close()

			if err == nil {
				// Replace original file with formatted version
				os.Rename(tempFile.Name(), filename)
			} else {
				// Clean up temp file on error
				os.Remove(tempFile.Name())
			}
		}()

		writer = tempFile
		outputFile = tempFile
	} else if cmd.Output != "" {
		// Write to specified output file
		outputFile, err = os.Create(cmd.Output)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", cmd.Output, err)
		}
		defer outputFile.Close()

		writer = outputFile
	} else {
		// Write to stdout
		writer = os.Stdout
	}

	return cmd.formatFromReader(sqlFormatter, file, writer, filename)
}

// formatDirectory formats all SnapSQL files in a directory recursively
func (cmd *FormatCmd) formatDirectory(sqlFormatter *formatter.SQLFormatter, dirPath string) error {
	var hasErrors bool

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process SnapSQL files
		if !cmd.isSnapSQLFile(path) {
			return nil
		}

		// Format the file
		err = cmd.formatFile(sqlFormatter, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", path, err)

			hasErrors = true
			// Continue processing other files
			return nil
		}

		if !cmd.Check && !cmd.Diff {
			fmt.Printf("Formatted: %s\n", path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if hasErrors {
		return ErrFormattingErrors
	}

	return nil
}

// isSnapSQLFile checks if a file is a SnapSQL file
func (cmd *FormatCmd) isSnapSQLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))

	// Check for .snap.sql or .snap.md extensions
	if strings.HasSuffix(base, ".snap.sql") || strings.HasSuffix(base, ".snap.md") {
		return true
	}

	// Also accept plain .sql and .md files
	return ext == ".sql" || ext == ".md"
}

// isMarkdownFile checks if a file is a Markdown file
func (cmd *FormatCmd) isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))

	// Check for .snap.md or .md extensions
	return strings.HasSuffix(base, ".snap.md") || ext == ".md"
}

// showDiff shows the difference between original and formatted content
func (cmd *FormatCmd) showDiff(original, formatted, filename string) error {
	if strings.TrimSpace(original) == strings.TrimSpace(formatted) {
		// No changes needed
		return nil
	}

	fmt.Printf("--- %s (original)\n", filename)
	fmt.Printf("+++ %s (formatted)\n", filename)

	// Simple line-by-line diff
	originalLines := strings.Split(original, "\n")
	formattedLines := strings.Split(formatted, "\n")

	maxLines := len(originalLines)
	if len(formattedLines) > maxLines {
		maxLines = len(formattedLines)
	}

	for i := range maxLines {
		var origLine, formLine string

		if i < len(originalLines) {
			origLine = originalLines[i]
		}

		if i < len(formattedLines) {
			formLine = formattedLines[i]
		}

		if origLine != formLine {
			if origLine != "" {
				fmt.Printf("-%s\n", origLine)
			}

			if formLine != "" {
				fmt.Printf("+%s\n", formLine)
			}
		}
	}

	return nil
}

// Help returns help text for the format command
func (cmd *FormatCmd) Help() string {
	return `Format SnapSQL template files and Markdown files with SQL code blocks.

The format command formats SnapSQL template files (.snap.sql, .snap.md) and regular
SQL/Markdown files according to a consistent style similar to 'go fmt'. It uses 
4-space indentation, trailing comma style, and proper keyword casing.

For Markdown files, it formats SQL code blocks within ` + "```sql" + ` blocks while
preserving the rest of the Markdown content.

Examples:
  # Format a single file and print to stdout
  snapsql format query.snap.sql
  snapsql format README.md

  # Format a file in place
  snapsql format -w query.snap.sql
  snapsql format -w documentation.md

  # Format all files in a directory
  snapsql format -w ./queries/

  # Check if files are properly formatted
  snapsql format -c ./queries/

  # Show diff of what would be changed
  snapsql format -d query.snap.sql

  # Format from stdin
  cat query.sql | snapsql format

Style rules:
- Keywords are uppercase (SELECT, FROM, WHERE, etc.)
- 4-space indentation
- Trailing comma style for multi-line lists
- AND/OR operators at the end of lines
- SnapSQL directives (/*# if */, /*# for */) create new indentation levels
- Inline SnapSQL expressions (/*= expr */) are preserved as-is
- Markdown: SQL code blocks are formatted while preserving document structure`
}
