package intermediate

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateInstructions generates instructions from tokens
func GenerateInstructions(tokens []tokenizer.Token) []Instruction {
	instructions := []Instruction{}
	
	// Buffer for static content
	var staticBuffer strings.Builder
	
	// Track if we need to add a space before the next token
	needSpace := false
	
	// Track the position of the first significant token for the current instruction
	var currentInstructionPos string
	
	// Track if we're inside a dummy literal block
	inDummyBlock := false
	
	// Helper function to add a static instruction if the buffer is not empty
	flushStaticBuffer := func() {
		if staticBuffer.Len() > 0 {
			// Normalize whitespace in the buffer content
			content := normalizeWhitespace(staticBuffer.String())
			if content != "" {
				instructions = append(instructions, Instruction{
					Op:    OpEmitStatic,
					Pos:   currentInstructionPos,
					Value: content,
				})
			}
			staticBuffer.Reset()
			needSpace = false
			currentInstructionPos = "" // Reset position for next instruction
		}
	}
	
	// Helper function to get position string
	getPos := func(token tokenizer.Token) string {
		return fmt.Sprintf("%d:%d", token.Position.Line, token.Position.Column)
	}
	
	// Process tokens
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		
		// Skip header comments
		if token.Type == tokenizer.BLOCK_COMMENT && isHeaderComment(token.Value) {
			continue
		}
		
		// Check for DUMMY_START and DUMMY_END tokens
		if token.Type == tokenizer.DUMMY_START {
			inDummyBlock = true
			continue
		}
		if token.Type == tokenizer.DUMMY_END {
			inDummyBlock = false
			continue
		}
		
		// Skip tokens inside dummy blocks
		if inDummyBlock {
			continue
		}
		
		// Update current instruction position if this is a significant token and we don't have one yet
		if currentInstructionPos == "" && 
		   token.Type != tokenizer.WHITESPACE && 
		   token.Type != tokenizer.LINE_COMMENT && 
		   token.Type != tokenizer.BLOCK_COMMENT {
			currentInstructionPos = getPos(token)
		}
		
		// Check for LIMIT and OFFSET keywords
		if token.Type == tokenizer.LIMIT {
			// Flush any pending content
			flushStaticBuffer()
			
			// Add EMIT_SYSTEM_LIMIT instruction
			instructions = append(instructions, Instruction{
				Op:           OpEmitSystemLimit,
				Pos:          getPos(token),
				DefaultValue: "10", // Default limit value
			})
			
			// Skip the LIMIT keyword and the following number
			// Find the next number token and skip it
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == tokenizer.NUMBER {
					i = j // Skip to this token (the loop will increment i)
					break
				}
			}
			
			continue
		} else if token.Type == tokenizer.OFFSET {
			// Flush any pending content
			flushStaticBuffer()
			
			// Add EMIT_SYSTEM_OFFSET instruction
			instructions = append(instructions, Instruction{
				Op:           OpEmitSystemOffset,
				Pos:          getPos(token),
				DefaultValue: "0", // Default offset value
			})
			
			// Skip the OFFSET keyword and the following number
			// Find the next number token and skip it
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == tokenizer.NUMBER {
					i = j // Skip to this token (the loop will increment i)
					break
				}
			}
			
			continue
		}
		
		// Process token based on type
		switch token.Type {
		case tokenizer.WHITESPACE:
			// Handle whitespace specially
			if staticBuffer.Len() > 0 {
				// If we have content in the buffer, mark that we need a space
				needSpace = true
			}
			// Don't add whitespace directly to the buffer
			
		case tokenizer.BLOCK_COMMENT:
			// Check if this is a directive
			if token.Directive != nil {
				// Flush static buffer before processing directive
				flushStaticBuffer()
				
				// Get position for this instruction
				pos := getPos(token)
				
				// Process directive based on type
				switch token.Directive.Type {
				case "if":
					// Add IF instruction
					instructions = append(instructions, Instruction{
						Op:        OpIf,
						Pos:       pos,
						Condition: token.Directive.Condition,
					})
					
				case "elseif":
					// Add ELSE_IF instruction
					instructions = append(instructions, Instruction{
						Op:        OpElseIf,
						Pos:       pos,
						Condition: token.Directive.Condition,
					})
					
				case "else":
					// Add ELSE instruction
					instructions = append(instructions, Instruction{
						Op:  OpElse,
						Pos: pos,
					})
					
				case "end":
					// Add END instruction
					instructions = append(instructions, Instruction{
						Op:  OpEnd,
						Pos: pos,
					})
					
				case "for":
					// Add FOR instruction
					parts := strings.Split(token.Directive.Condition, ":")
					if len(parts) == 2 {
						variable := strings.TrimSpace(parts[0])
						collection := strings.TrimSpace(parts[1])
						
						instructions = append(instructions, Instruction{
							Op:         OpFor,
							Pos:        pos,
							Variable:   variable,
							Collection: collection,
						})
					}
					
				case "variable":
					// Extract variable name from /*= variable_name */
					varName := extractVariableName(token.Value)
					
					// Add EMIT_PARAM instruction
					instructions = append(instructions, Instruction{
						Op:    OpEmitParam,
						Pos:   pos,
						Param: varName,
					})
				}
				
				// Reset position for next instruction
				currentInstructionPos = ""
			} else if !isHeaderComment(token.Value) {
				// Regular comment - add a space if needed
				if needSpace {
					staticBuffer.WriteString(" ")
					needSpace = false
				}
				// Append comment to buffer
				staticBuffer.WriteString(token.Value)
			}
			
		case tokenizer.LINE_COMMENT:
			// Add a space if needed
			if needSpace {
				staticBuffer.WriteString(" ")
				needSpace = false
			}
			// Append comment to buffer
			staticBuffer.WriteString(token.Value)
			
		case tokenizer.DUMMY_LITERAL:
			// Skip dummy literals - they should not appear in the output
			continue
			
		default:
			// Add a space if needed
			if needSpace {
				staticBuffer.WriteString(" ")
				needSpace = false
			}
			// Append token value to buffer
			staticBuffer.WriteString(token.Value)
		}
		
		// If this is the last token, flush the buffer
		if i == len(tokens)-1 {
			flushStaticBuffer()
		}
	}
	
	return instructions
}

// normalizeWhitespace normalizes whitespace in a string:
// - Removes leading whitespace
// - Collapses multiple spaces into a single space
// - Preserves newlines but removes extra spaces around them
func normalizeWhitespace(s string) string {
	var result strings.Builder
	
	// Remove leading whitespace
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	
	// Process the rest of the string
	prevIsSpace := false
	prevIsNewline := false
	
	for _, r := range s {
		isSpace := unicode.IsSpace(r) && r != '\n'
		isNewline := r == '\n'
		
		if isNewline {
			// Always keep newlines
			result.WriteRune(r)
			prevIsSpace = false
			prevIsNewline = true
		} else if isSpace {
			// Only add a space if the previous character wasn't a space or newline
			if !prevIsSpace && !prevIsNewline {
				result.WriteRune(' ')
			}
			prevIsSpace = true
			prevIsNewline = false
		} else {
			// Non-whitespace character
			result.WriteRune(r)
			prevIsSpace = false
			prevIsNewline = false
		}
	}
	
	return result.String()
}

// isHeaderComment checks if a comment is a header comment (/*# ... */)
func isHeaderComment(comment string) bool {
	return strings.HasPrefix(comment, "/*#") && strings.HasSuffix(comment, "*/")
}

// extractVariableName extracts the variable name from a variable token
// Format: /*= variable_name */placeholder
func extractVariableName(value string) string {
	// Remove /*= and */
	value = strings.TrimPrefix(value, "/*=")
	
	// Split by */
	parts := strings.Split(value, "*/")
	if len(parts) > 0 {
		// Return trimmed variable name
		return strings.TrimSpace(parts[0])
	}
	
	return ""
}

// extractPlaceholder extracts the placeholder from a variable token
// Format: /*= variable_name */placeholder
func extractPlaceholder(value string) string {
	// Split by */
	parts := strings.Split(value, "*/")
	if len(parts) > 1 {
		// Return trimmed placeholder
		return strings.TrimSpace(parts[1])
	}
	
	return ""
}
