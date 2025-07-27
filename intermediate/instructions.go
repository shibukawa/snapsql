package intermediate

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql/tokenizer"
)

// LimitOffsetClauseInfo holds information about LIMIT and OFFSET clause processing
type LimitOffsetClauseInfo struct {
	// LIMIT clause info
	HasLimit             bool // Whether LIMIT clause exists
	HasLimitCondition    bool // Whether LIMIT clause is wrapped in condition
	LimitTokenIndex      int  // Index of LIMIT token
	LimitValueTokenIndex int  // Index of value token after LIMIT
	LimitConditionStart  int  // Start index of LIMIT condition tokens
	LimitConditionEnd    int  // End index of LIMIT condition tokens

	// OFFSET clause info
	HasOffset             bool // Whether OFFSET clause exists
	HasOffsetCondition    bool // Whether OFFSET clause is wrapped in condition
	OffsetTokenIndex      int  // Index of OFFSET token
	OffsetValueTokenIndex int  // Index of value token after OFFSET
	OffsetConditionStart  int  // Start index of OFFSET condition tokens
	OffsetConditionEnd    int  // End index of OFFSET condition tokens
}

// isSelectStatement checks if the tokens represent a SELECT statement
func isSelectStatement(tokens []tokenizer.Token) bool {
	for _, token := range tokens {
		if token.Type == tokenizer.SELECT {
			return true
		}
		// Stop at the first significant keyword
		if token.Type == tokenizer.INSERT ||
			token.Type == tokenizer.UPDATE ||
			token.Type == tokenizer.DELETE {
			return false
		}
	}
	return false
}

// detectLimitOffsetClause analyzes tokens to detect LIMIT and OFFSET clause patterns
func detectLimitOffsetClause(tokens []tokenizer.Token) *LimitOffsetClauseInfo {
	info := &LimitOffsetClauseInfo{}

	for i, token := range tokens {
		// Look for LIMIT keyword
		if token.Type == tokenizer.LIMIT {
			info.HasLimit = true
			info.LimitTokenIndex = i

			// Look for the value token after LIMIT
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == tokenizer.NUMBER ||
					(tokens[j].Directive != nil && tokens[j].Directive.Type == "variable") {
					info.LimitValueTokenIndex = j
					break
				}
			}

			// Check if LIMIT clause is wrapped in a condition
			info.HasLimitCondition = checkForCondition(tokens, i, info.LimitValueTokenIndex, &info.LimitConditionStart, &info.LimitConditionEnd)
		}

		// Look for OFFSET keyword
		if token.Type == tokenizer.OFFSET {
			info.HasOffset = true
			info.OffsetTokenIndex = i

			// Look for the value token after OFFSET
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == tokenizer.NUMBER ||
					(tokens[j].Directive != nil && tokens[j].Directive.Type == "variable") {
					info.OffsetValueTokenIndex = j
					break
				}
			}

			// Check if OFFSET clause is wrapped in a condition
			info.HasOffsetCondition = checkForCondition(tokens, i, info.OffsetValueTokenIndex, &info.OffsetConditionStart, &info.OffsetConditionEnd)
		}
	}

	return info
}

// checkForCondition checks if a clause is wrapped in a condition
func checkForCondition(tokens []tokenizer.Token, keywordIndex, valueIndex int, conditionStart, conditionEnd *int) bool {
	// Look backwards for IF directive
	for j := keywordIndex - 1; j >= 0; j-- {
		if tokens[j].Directive != nil && tokens[j].Directive.Type == "if" {
			// Check if there's a matching END after the value
			endIndex := valueIndex
			if endIndex == 0 {
				endIndex = keywordIndex + 1
			}
			for k := endIndex + 1; k < len(tokens); k++ {
				if tokens[k].Directive != nil && tokens[k].Directive.Type == "end" {
					*conditionStart = j
					*conditionEnd = k
					return true
				}
			}
		}
	}
	return false
}

// GenerateInstructions generates instructions from tokens with expression index references
func GenerateInstructions(tokens []tokenizer.Token, expressions []string) []Instruction {
	instructions := []Instruction{}

	// Detect LIMIT and OFFSET clause patterns
	limitOffsetInfo := detectLimitOffsetClause(tokens)

	// Detect conditional boundaries
	boundaries := findConditionalBoundaries(tokens)

	// Buffer for static content
	var staticBuffer strings.Builder

	// Track if we need to add a space before the next token
	needSpace := false

	// Track the position of the first significant token for the current instruction
	var currentInstructionPos string

	// Track if we're inside a dummy literal block
	inDummyBlock := false

	// Directive stack to track nested structures
	var directiveStack []string

	// Helper function to find expression index
	findExpressionIndex := func(expr string) int {
		for i, e := range expressions {
			if e == expr {
				return i
			}
		}
		return -1 // Should not happen if expressions are properly extracted
	}

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

		// Check for boundary instructions
		if boundaryType, exists := boundaries[i]; exists {
			if boundaryType == "BOUNDARY" {
				// Flush any pending content before placing boundary
				flushStaticBuffer()

				// Add BOUNDARY instruction
				instructions = append(instructions, Instruction{
					Op:  OpBoundary,
					Pos: getPos(token),
				})
			}
		}

		// Check for LIMIT clause processing
		if limitOffsetInfo.HasLimit && i == limitOffsetInfo.LimitTokenIndex {
			// Flush any pending content
			flushStaticBuffer()

			// Handle different LIMIT clause patterns
			if limitOffsetInfo.HasLimitCondition {
				// Case 3: LIMIT clause with condition
				// IF_SYSTEM_LIMIT, EMIT_STATIC(LIMIT ), ELSE, 今の出力, END
				instructions = append(instructions, Instruction{
					Op:  OpIfSystemLimit,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:    OpEmitStatic,
					Value: "LIMIT ",
					Pos:   getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpEmitSystemLimit,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpElse,
					Pos: getPos(token),
				})

				// Skip to the condition start and let normal processing handle the condition
				i = limitOffsetInfo.LimitConditionStart - 1 // -1 because loop will increment
				continue

			} else {
				// Case 1: LIMIT already exists without condition
				// EMIT_STATIC(LIMIT ), IF_SYSTEM_LIMIT, EMIT_SYSTEM_LIMIT, ELSE, EMIT_STATIC(value), END
				instructions = append(instructions, Instruction{
					Op:    OpEmitStatic,
					Value: "LIMIT ",
					Pos:   getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpIfSystemLimit,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpEmitSystemLimit,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpElse,
					Pos: getPos(token),
				})

				// Find and emit the original value
				if limitOffsetInfo.LimitValueTokenIndex > 0 && limitOffsetInfo.LimitValueTokenIndex < len(tokens) {
					valueToken := tokens[limitOffsetInfo.LimitValueTokenIndex]
					if valueToken.Directive != nil && valueToken.Directive.Type == "variable" {
						// Handle variable substitution
						exprIndex := findExpressionIndex(valueToken.Directive.Condition)
						instructions = append(instructions, Instruction{
							Op:        OpEmitEval,
							ExprIndex: &exprIndex,
							Pos:       getPos(valueToken),
						})
					} else {
						// Handle literal value
						instructions = append(instructions, Instruction{
							Op:    OpEmitStatic,
							Value: strings.TrimSpace(valueToken.Value),
							Pos:   getPos(valueToken),
						})
					}
				}

				instructions = append(instructions, Instruction{
					Op:  OpEnd,
					Pos: getPos(token),
				})

				// Skip to after the value token
				if limitOffsetInfo.LimitValueTokenIndex > 0 {
					i = limitOffsetInfo.LimitValueTokenIndex
				}
			}
			continue
		}

		// Check for OFFSET clause processing
		if limitOffsetInfo.HasOffset && i == limitOffsetInfo.OffsetTokenIndex {
			// Flush any pending content
			flushStaticBuffer()

			// Handle different OFFSET clause patterns
			if limitOffsetInfo.HasOffsetCondition {
				// Case 3: OFFSET clause with condition
				// IF_SYSTEM_OFFSET, EMIT_STATIC(OFFSET ), ELSE, 今の出力, END
				instructions = append(instructions, Instruction{
					Op:  OpIfSystemOffset,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:    OpEmitStatic,
					Value: "OFFSET ",
					Pos:   getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpEmitSystemOffset,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpElse,
					Pos: getPos(token),
				})

				// Skip to the condition start and let normal processing handle the condition
				i = limitOffsetInfo.OffsetConditionStart - 1 // -1 because loop will increment
				continue

			} else {
				// Case 1: OFFSET already exists without condition
				// EMIT_STATIC(OFFSET ), IF_SYSTEM_OFFSET, EMIT_SYSTEM_OFFSET, ELSE, EMIT_STATIC(value), END
				instructions = append(instructions, Instruction{
					Op:    OpEmitStatic,
					Value: "OFFSET ",
					Pos:   getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpIfSystemOffset,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpEmitSystemOffset,
					Pos: getPos(token),
				})

				instructions = append(instructions, Instruction{
					Op:  OpElse,
					Pos: getPos(token),
				})

				// Find and emit the original value
				if limitOffsetInfo.OffsetValueTokenIndex > 0 && limitOffsetInfo.OffsetValueTokenIndex < len(tokens) {
					valueToken := tokens[limitOffsetInfo.OffsetValueTokenIndex]
					if valueToken.Directive != nil && valueToken.Directive.Type == "variable" {
						// Handle variable substitution
						exprIndex := findExpressionIndex(valueToken.Directive.Condition)
						instructions = append(instructions, Instruction{
							Op:        OpEmitEval,
							ExprIndex: &exprIndex,
							Pos:       getPos(valueToken),
						})
					} else {
						// Handle literal value
						instructions = append(instructions, Instruction{
							Op:    OpEmitStatic,
							Value: strings.TrimSpace(valueToken.Value),
							Pos:   getPos(valueToken),
						})
					}
				}

				instructions = append(instructions, Instruction{
					Op:  OpEnd,
					Pos: getPos(token),
				})

				// Skip to after the value token
				if limitOffsetInfo.OffsetValueTokenIndex > 0 {
					i = limitOffsetInfo.OffsetValueTokenIndex
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
					// Find expression index for condition
					exprIndex := findExpressionIndex(token.Directive.Condition)
					if exprIndex == -1 {
						// Fallback to condition for backward compatibility
						instructions = append(instructions, Instruction{
							Op:        OpIf,
							Pos:       pos,
							Condition: token.Directive.Condition,
						})
					} else {
						// Use expression index
						instructions = append(instructions, Instruction{
							Op:        OpIf,
							Pos:       pos,
							ExprIndex: &exprIndex,
						})
					}

					// Push to stack
					directiveStack = append(directiveStack, "if")

				case "elseif":
					// Find expression index for condition
					exprIndex := findExpressionIndex(token.Directive.Condition)
					if exprIndex == -1 {
						// Fallback to condition for backward compatibility
						instructions = append(instructions, Instruction{
							Op:        OpElseIf,
							Pos:       pos,
							Condition: token.Directive.Condition,
						})
					} else {
						// Use expression index
						instructions = append(instructions, Instruction{
							Op:        OpElseIf,
							Pos:       pos,
							ExprIndex: &exprIndex,
						})
					}

					// Update stack top to elseif
					if len(directiveStack) > 0 {
						directiveStack[len(directiveStack)-1] = "elseif"
					}

				case "else":
					// Add ELSE instruction
					instructions = append(instructions, Instruction{
						Op:  OpElse,
						Pos: pos,
					})

					// Update stack top to else
					if len(directiveStack) > 0 {
						directiveStack[len(directiveStack)-1] = "else"
					}

				case "end":
					// Check if this is ending a for loop by looking at the directive stack
					if len(directiveStack) > 0 && directiveStack[len(directiveStack)-1] == "for" {
						// This is ending a for loop - add LOOP_END instruction
						instructions = append(instructions, Instruction{
							Op:  OpLoopEnd,
							Pos: pos,
						})
						// Pop from stack
						directiveStack = directiveStack[:len(directiveStack)-1]
					} else {
						// This is ending an if block - add END instruction
						instructions = append(instructions, Instruction{
							Op:  OpEnd,
							Pos: pos,
						})
						// Pop from stack if it's an if-related directive
						if len(directiveStack) > 0 && (directiveStack[len(directiveStack)-1] == "if" || directiveStack[len(directiveStack)-1] == "elseif" || directiveStack[len(directiveStack)-1] == "else") {
							directiveStack = directiveStack[:len(directiveStack)-1]
						}
					}

				case "for":
					// Add LOOP_START instruction
					parts := strings.Split(token.Directive.Condition, ":")
					if len(parts) == 2 {
						variable := strings.TrimSpace(parts[0])
						collection := strings.TrimSpace(parts[1])

						// Find expression index for collection
						collectionExprIndex := findExpressionIndex(collection)

						if collectionExprIndex == -1 {
							// Fallback to collection string for backward compatibility
							instructions = append(instructions, Instruction{
								Op:         OpLoopStart,
								Pos:        pos,
								Variable:   variable,
								Collection: collection,
							})
						} else {
							// Use expression index for collection
							instructions = append(instructions, Instruction{
								Op:                  OpLoopStart,
								Pos:                 pos,
								Variable:            variable,
								CollectionExprIndex: &collectionExprIndex,
							})
						}

						// Push to stack
						directiveStack = append(directiveStack, "for")
					}

				case "variable":
					// Extract variable name from /*= variable_name */
					varName := extractVariableName(token.Value)

					// Find expression index
					exprIndex := findExpressionIndex(varName)
					if exprIndex == -1 {
						// Fallback to param for backward compatibility
						instructions = append(instructions, Instruction{
							Op:    OpEmitEval,
							Pos:   pos,
							Param: varName,
						})
					} else {
						// Use expression index
						instructions = append(instructions, Instruction{
							Op:        OpEmitEval,
							Pos:       pos,
							ExprIndex: &exprIndex,
						})
					}
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
			// Check if this token should use EMIT_UNLESS_BOUNDARY
			if boundaryType, exists := boundaries[i]; exists && boundaryType == "EMIT_UNLESS_BOUNDARY" {
				// Flush any pending content before EMIT_UNLESS_BOUNDARY
				flushStaticBuffer()

				// Add EMIT_UNLESS_BOUNDARY instruction
				instructions = append(instructions, Instruction{
					Op:    OpEmitUnlessBoundary,
					Value: token.Value,
					Pos:   getPos(token),
				})
			} else {
				// Normal static content processing
				// Add a space if needed
				if needSpace {
					staticBuffer.WriteString(" ")
					needSpace = false
				}
				// Append token value to buffer
				staticBuffer.WriteString(token.Value)
			}
		}

		// If this is the last token, flush the buffer
		if i == len(tokens)-1 {
			flushStaticBuffer()
		}
	}

	// Handle case 2: No LIMIT clause exists (only for SELECT statements)
	if !limitOffsetInfo.HasLimit && isSelectStatement(tokens) {
		// IF_SYSTEM_LIMIT, EMIT_STATIC(LIMIT ), EMIT_SYSTEM_LIMIT, END
		instructions = append(instructions, Instruction{
			Op:  OpIfSystemLimit,
			Pos: "0:0", // No specific position since LIMIT doesn't exist
		})

		instructions = append(instructions, Instruction{
			Op:    OpEmitStatic,
			Value: " LIMIT ",
			Pos:   "0:0",
		})

		instructions = append(instructions, Instruction{
			Op:  OpEmitSystemLimit,
			Pos: "0:0",
		})

		instructions = append(instructions, Instruction{
			Op:  OpEnd,
			Pos: "0:0",
		})
	}

	// Handle case 2: No OFFSET clause exists (only for SELECT statements)
	if !limitOffsetInfo.HasOffset && isSelectStatement(tokens) {
		// IF_SYSTEM_OFFSET, EMIT_STATIC(OFFSET ), EMIT_SYSTEM_OFFSET, END
		instructions = append(instructions, Instruction{
			Op:  OpIfSystemOffset,
			Pos: "0:0", // No specific position since OFFSET doesn't exist
		})

		instructions = append(instructions, Instruction{
			Op:    OpEmitStatic,
			Value: " OFFSET ",
			Pos:   "0:0",
		})

		instructions = append(instructions, Instruction{
			Op:  OpEmitSystemOffset,
			Pos: "0:0",
		})

		instructions = append(instructions, Instruction{
			Op:  OpEnd,
			Pos: "0:0",
		})
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

// isHeaderComment checks if a comment is a header comment (function definition)
func isHeaderComment(comment string) bool {
	if !strings.HasPrefix(comment, "/*#") || !strings.HasSuffix(comment, "*/") {
		return false
	}

	// Extract content between /*# and */
	content := strings.TrimSpace(comment[3 : len(comment)-2])

	// Check if it's a directive (if, elseif, else, for, end)
	if strings.HasPrefix(content, "if ") || content == "if" ||
		strings.HasPrefix(content, "elseif ") || content == "elseif" ||
		content == "else" ||
		strings.HasPrefix(content, "for ") || content == "for" ||
		content == "end" {
		return false // This is a directive, not a header comment
	}

	// If it contains function_name or parameters, it's a header comment
	return strings.Contains(content, "function_name:") || strings.Contains(content, "parameters:")
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
