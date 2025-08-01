package markdownparser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark/ast"
)

// ASTSection represents a markdown section with AST nodes
type ASTSection struct {
	Heading     ast.Node   // The heading node
	HeadingText string     // Extracted heading text
	StartLine   int        // Line number where section starts
	Content     []ast.Node // All nodes between this heading and the next
}

// extractSectionsFromAST extracts sections and title from AST
func extractSectionsFromAST(doc ast.Node, content []byte) (string, map[string]ASTSection) {
	sections := make(map[string]ASTSection)
	var title string
	var currentSection *ASTSection

	// Walk through the AST
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			headingText := extractTextFromHeadingNode(node, content)

			// H1 is title
			if node.Level == 1 {
				title = headingText
				return ast.WalkContinue, nil
			}

			// Create new section for H2
			if node.Level == 2 {
				sectionName := strings.ToLower(headingText)
				currentSection = &ASTSection{
					Heading:     node,
					HeadingText: headingText,
					StartLine:   node.Lines().At(0).Start,
					Content:     make([]ast.Node, 0),
				}
				sections[sectionName] = *currentSection
				return ast.WalkContinue, nil
			}

			// For H3 and below, add to current section if exists
			if currentSection != nil && node.Level > 2 {
				currentSection.Content = append(currentSection.Content, node)
				sections[strings.ToLower(currentSection.HeadingText)] = *currentSection
			}

		case *ast.FencedCodeBlock, *ast.Paragraph, *ast.List:
			// Add content to current section
			if currentSection != nil {
				currentSection.Content = append(currentSection.Content, node)
				sections[strings.ToLower(currentSection.HeadingText)] = *currentSection
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return "", nil
	}

	return title, sections
}

// extractTextFromHeadingNode extracts text content from a heading node
func extractTextFromHeadingNode(n ast.Node, content []byte) string {
	var text strings.Builder
	err := ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindText {
			text.Write(n.Text(content))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text.String())
}

// validateRequiredSections checks if all required sections are present
func validateRequiredSections(sections map[string]ASTSection) error {
	// Description or Overview is required
	descriptionFound := false
	for sectionName := range sections {
		if sectionName == "description" || sectionName == "overview" {
			descriptionFound = true
			break
		}
	}
	if !descriptionFound {
		return fmt.Errorf("%w: description or overview", ErrMissingRequiredSection)
	}

	// SQL is required
	if _, exists := sections["sql"]; !exists {
		return fmt.Errorf("%w: sql", ErrMissingRequiredSection)
	}

	return nil
}

// extractSQLFromASTNodes extracts SQL content from AST nodes
func extractSQLFromASTNodes(nodes []ast.Node, content []byte) (string, int) {
	for _, node := range nodes {
		if codeBlock, ok := node.(*ast.FencedCodeBlock); ok {
			info := string(codeBlock.Info.Text(content))
			if strings.ToLower(strings.TrimSpace(info)) == "sql" {
				var sql strings.Builder
				lines := codeBlock.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					sql.Write(line.Value(content))
					if i < lines.Len()-1 {
						sql.WriteString("\n")
					}
				}
				return sql.String(), codeBlock.Lines().At(0).Start
			}
		}
	}
	return "", 0
}

// extractParameterTextFromASTNodes extracts raw parameter text and type from AST nodes
func extractParameterTextFromASTNodes(nodes []ast.Node, content []byte) (string, string, error) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FencedCodeBlock:
			info := string(n.Info.Text(content))
			infoLower := strings.ToLower(strings.TrimSpace(info))
			
			// Extract content
			var textContent strings.Builder
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				textContent.Write(line.Value(content))
				if i < lines.Len()-1 {
					textContent.WriteString("\n")
				}
			}
			
			if infoLower == "yaml" || infoLower == "yml" {
				return textContent.String(), "yaml", nil
			} else if infoLower == "json" {
				return textContent.String(), "json", nil
			}
			
		case *ast.List:
			// Extract parameter definitions from list items
			var listContent strings.Builder
			ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering && n.Kind() == ast.KindText {
					listContent.Write(n.Text(content))
				}
				return ast.WalkContinue, nil
			})
			return listContent.String(), "list", nil
		}
	}
	return "", "", fmt.Errorf("no parameter code block or list found")
}
func parseParameterSection(nodes []ast.Node, content []byte) (map[string]any, error) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FencedCodeBlock:
			info := string(n.Info.Text(content))
			infoLower := strings.ToLower(strings.TrimSpace(info))
			
			if infoLower == "yaml" || infoLower == "yml" {
				// Extract YAML content
				var yamlContent strings.Builder
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					yamlContent.Write(line.Value(content))
					if i < lines.Len()-1 {
						yamlContent.WriteString("\n")
					}
				}
				
				// Parse YAML
				var params map[string]any
				if err := yaml.Unmarshal([]byte(yamlContent.String()), &params); err != nil {
					return nil, fmt.Errorf("failed to parse YAML parameters: %w", err)
				}
				return params, nil
				
			} else if infoLower == "json" {
				// Extract JSON content
				var jsonContent strings.Builder
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					jsonContent.Write(line.Value(content))
					if i < lines.Len()-1 {
						jsonContent.WriteString("\n")
					}
				}
				
				// Parse JSON
				var params map[string]any
				if err := json.Unmarshal([]byte(jsonContent.String()), &params); err != nil {
					return nil, fmt.Errorf("failed to parse JSON parameters: %w", err)
				}
				return params, nil
			}
		}
	}
	
	// No parameter block found
	return nil, nil
}

// extractParameterBlock extracts parameter definitions from AST nodes
func extractParameterBlock(nodes []ast.Node, content []byte) string {
	var parameterContent strings.Builder

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.FencedCodeBlock:
			info := string(n.Info.Text(content))
			if strings.ToLower(strings.TrimSpace(info)) == "yaml" ||
				strings.ToLower(strings.TrimSpace(info)) == "json" {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					parameterContent.Write(line.Value(content))
					if i < lines.Len()-1 {
						parameterContent.WriteString("\n")
					}
				}
				return parameterContent.String()
			}
		case *ast.List:
			// Extract parameter definitions from list items
			ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering && n.Kind() == ast.KindText {
					text := string(n.Text(content))
					if strings.Contains(text, ":") {
						parameterContent.WriteString(text)
						parameterContent.WriteString("\n")
					}
				}
				return ast.WalkContinue, nil
			})
		}
	}

	return parameterContent.String()
}

// extractTextFromASTNodes extracts plain text content from AST nodes
func extractTextFromASTNodes(nodes []ast.Node, content []byte) (string, error) {
	var textContent strings.Builder
	
	for _, node := range nodes {
		ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				switch n.Kind() {
				case ast.KindText:
					textContent.Write(n.Text(content))
				case ast.KindParagraph:
					// Add space between paragraphs
					if textContent.Len() > 0 {
						textContent.WriteString(" ")
					}
				}
			}
			return ast.WalkContinue, nil
		})
	}
	
	return strings.TrimSpace(textContent.String()), nil
}
