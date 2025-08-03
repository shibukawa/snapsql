package formatter

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// MarkdownFormatter formats SQL code blocks within Markdown files
type MarkdownFormatter struct {
	sqlFormatter *SQLFormatter
}

// NewMarkdownFormatter creates a new Markdown formatter
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{
		sqlFormatter: NewSQLFormatter(),
	}
}

// Format formats SQL code blocks within a Markdown file
func (f *MarkdownFormatter) Format(markdown string) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	
	var inSQLBlock bool
	var sqlBlockContent strings.Builder
	var blockIndent string
	
	// Regular expressions for SQL code blocks
	sqlBlockStartRe := regexp.MustCompile(`^(\s*)\x60{3}sql\s*$`)
	codeBlockEndRe := regexp.MustCompile(`^(\s*)\x60{3}\s*$`)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if !inSQLBlock {
			// Check if this line starts a SQL code block
			if match := sqlBlockStartRe.FindStringSubmatch(line); match != nil {
				inSQLBlock = true
				blockIndent = match[1] // Capture the indentation
				sqlBlockContent.Reset()
				result.WriteString(line)
				result.WriteString("\n")
				continue
			}
			
			// Regular line, just copy it
			result.WriteString(line)
			result.WriteString("\n")
		} else {
			// We're inside a SQL code block
			if match := codeBlockEndRe.FindStringSubmatch(line); match != nil {
				// End of SQL code block
				inSQLBlock = false
				
				// Format the accumulated SQL content
				sqlContent := sqlBlockContent.String()
				if strings.TrimSpace(sqlContent) != "" {
					formattedSQL, err := f.sqlFormatter.Format(sqlContent)
					if err != nil {
						// If formatting fails, use original content
						formattedSQL = sqlContent
					}
					
					// Add the formatted SQL with proper indentation
					sqlLines := strings.Split(strings.TrimRight(formattedSQL, "\n"), "\n")
					for _, sqlLine := range sqlLines {
						if strings.TrimSpace(sqlLine) != "" {
							result.WriteString(blockIndent)
							result.WriteString(sqlLine)
						}
						result.WriteString("\n")
					}
				}
				
				// Add the closing code block marker
				result.WriteString(line)
				result.WriteString("\n")
			} else {
				// Accumulate SQL content (remove the block indentation)
				sqlLine := line
				if strings.HasPrefix(line, blockIndent) {
					sqlLine = line[len(blockIndent):]
				}
				sqlBlockContent.WriteString(sqlLine)
				sqlBlockContent.WriteString("\n")
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading markdown: %w", err)
	}
	
	return strings.TrimRight(result.String(), "\n"), nil
}

// FormatFromReader formats SQL code blocks from a reader and writes to a writer
func (f *MarkdownFormatter) FormatFromReader(reader io.Reader, writer io.Writer) error {
	// Read all input
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Format the markdown
	formatted, err := f.Format(string(input))
	if err != nil {
		return fmt.Errorf("failed to format markdown: %w", err)
	}

	// Write formatted output
	_, err = writer.Write([]byte(formatted))
	return err
}

// IsMarkdownFile checks if a file is a Markdown file
func IsMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))
	
	// Check for .snap.md or .md extensions
	return strings.HasSuffix(base, ".snap.md") || ext == ".md"
}

// FormatSnapSQLMarkdown formats SnapSQL Markdown files with special handling
func (f *MarkdownFormatter) FormatSnapSQLMarkdown(markdown string) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	
	var inSQLSection bool
	var inCodeBlock bool
	var sqlContent strings.Builder
	var sectionIndent string
	
	// Regular expressions for SnapSQL Markdown sections
	sqlSectionRe := regexp.MustCompile(`^(\s*)##\s+SQL\s*$`)
	sectionHeaderRe := regexp.MustCompile(`^(\s*)##\s+`)
	codeBlockRe := regexp.MustCompile(`^(\s*)\x60{3}(sql)?\s*$`)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if !inSQLSection {
			// Check if this line starts the SQL section
			if match := sqlSectionRe.FindStringSubmatch(line); match != nil {
				inSQLSection = true
				sectionIndent = match[1]
				result.WriteString(line)
				result.WriteString("\n")
				continue
			}
			
			// Regular line, just copy it
			result.WriteString(line)
			result.WriteString("\n")
		} else {
			// We're inside the SQL section
			if sectionHeaderRe.MatchString(line) {
				// New section started, end SQL section
				inSQLSection = false
				inCodeBlock = false
				
				// Process any accumulated SQL content
				if sqlContent.Len() > 0 {
					f.processSQLContent(&result, sqlContent.String(), sectionIndent)
					sqlContent.Reset()
				}
				
				result.WriteString(line)
				result.WriteString("\n")
				continue
			}
			
			if !inCodeBlock {
				// Check if this line starts a code block
				if match := codeBlockRe.FindStringSubmatch(line); match != nil {
					inCodeBlock = true
					sqlContent.Reset()
					result.WriteString(line)
					result.WriteString("\n")
					continue
				}
				
				// Regular line in SQL section
				result.WriteString(line)
				result.WriteString("\n")
			} else {
				// We're inside a code block
				if codeBlockRe.MatchString(line) {
					// End of code block
					inCodeBlock = false
					
					// Format the accumulated SQL content
					if sqlContent.Len() > 0 {
						f.processSQLContent(&result, sqlContent.String(), sectionIndent)
						sqlContent.Reset()
					}
					
					result.WriteString(line)
					result.WriteString("\n")
				} else {
					// Accumulate SQL content
					sqlContent.WriteString(line)
					sqlContent.WriteString("\n")
				}
			}
		}
	}
	
	// Handle case where file ends while in SQL section
	if inSQLSection && sqlContent.Len() > 0 {
		f.processSQLContent(&result, sqlContent.String(), sectionIndent)
	}
	
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading markdown: %w", err)
	}
	
	return strings.TrimRight(result.String(), "\n"), nil
}

// processSQLContent formats SQL content and adds it to the result
func (f *MarkdownFormatter) processSQLContent(result *strings.Builder, sqlContent, indent string) {
	if strings.TrimSpace(sqlContent) == "" {
		return
	}
	
	formattedSQL, err := f.sqlFormatter.Format(sqlContent)
	if err != nil {
		// If formatting fails, use original content
		formattedSQL = sqlContent
	}
	
	// Add the formatted SQL with proper indentation
	sqlLines := strings.Split(strings.TrimRight(formattedSQL, "\n"), "\n")
	for _, sqlLine := range sqlLines {
		if strings.TrimSpace(sqlLine) != "" {
			result.WriteString(indent)
			result.WriteString(sqlLine)
		}
		result.WriteString("\n")
	}
}
