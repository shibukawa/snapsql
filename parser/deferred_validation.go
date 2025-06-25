package parser

import (
	"fmt"
)

// ValidateDeferredAST performs deferred validation of the entire AST (function style)
func ValidateDeferredAST(stmt *SelectStatement) error {
	return validateDeferredNode(stmt)
}

// validateDeferredNode recursively validates nodes
func validateDeferredNode(node AstNode) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *DeferredVariableSubstitution:
		return validateDeferredVariable(n)
	case *SelectStatement:
		return validateDeferredSelectStatement(n)
	case *SelectClause:
		return validateDeferredSelectClause(n)
	case *TemplateForBlock:
		return validateDeferredTemplateForBlock(n)
	case *TemplateIfBlock:
		return validateDeferredTemplateIfBlock(n)
	case *TemplateElseIfBlock:
		return validateDeferredTemplateElseIfBlock(n)
	case *TemplateElseBlock:
		return validateDeferredTemplateElseBlock(n)
	default:
		// 他のノードタイプは何もしない
		return nil
	}
}

// validateDeferredVariable validates deferred variable substitution
func validateDeferredVariable(node *DeferredVariableSubstitution) error {
	// fmt.Printf("DEBUG: Deferred variable validation '%s', schema: %+v\n", node.Expression, node.Namespace.Schema.Parameters)

	// Validate CEL expression using saved namespace
	if err := node.Namespace.ValidateParameterExpression(node.Expression); err != nil {
		return fmt.Errorf("invalid deferred variable expression '%s': %w", node.Expression, err)
	}

	// fmt.Printf("DEBUG: Deferred variable validation successful '%s'\n", node.Expression)
	return nil
}

// validateDeferredSelectStatement validates SELECT statements
func validateDeferredSelectStatement(stmt *SelectStatement) error {
	if stmt.SelectClause != nil {
		if err := validateDeferredNode(stmt.SelectClause); err != nil {
			return err
		}
	}
	if stmt.FromClause != nil {
		if err := validateDeferredNode(stmt.FromClause); err != nil {
			return err
		}
	}
	if stmt.WhereClause != nil {
		if err := validateDeferredNode(stmt.WhereClause); err != nil {
			return err
		}
	}
	if stmt.OrderByClause != nil {
		if err := validateDeferredNode(stmt.OrderByClause); err != nil {
			return err
		}
	}
	if stmt.LimitClause != nil {
		if err := validateDeferredNode(stmt.LimitClause); err != nil {
			return err
		}
	}
	if stmt.OffsetClause != nil {
		if err := validateDeferredNode(stmt.OffsetClause); err != nil {
			return err
		}
	}
	return nil
}

// validateDeferredSelectClause validates SELECT clauses
func validateDeferredSelectClause(clause *SelectClause) error {
	for _, field := range clause.Fields {
		if err := validateDeferredNode(field); err != nil {
			return err
		}
	}
	return nil
}

// validateDeferredTemplateForBlock validates for statement blocks
func validateDeferredTemplateForBlock(block *TemplateForBlock) error {
	for _, content := range block.Content {
		if err := validateDeferredNode(content); err != nil {
			return err
		}
	}
	return nil
}

// validateDeferredTemplateIfBlock validates if statement blocks
func validateDeferredTemplateIfBlock(block *TemplateIfBlock) error {
	for _, content := range block.Content {
		if err := validateDeferredNode(content); err != nil {
			return err
		}
	}

	for _, elseif := range block.ElseIfBlocks {
		if err := validateDeferredNode(elseif); err != nil {
			return err
		}
	}

	if block.ElseBlock != nil {
		if err := validateDeferredNode(block.ElseBlock); err != nil {
			return err
		}
	}

	return nil
}

// validateDeferredTemplateElseIfBlock validates elseif statement blocks
func validateDeferredTemplateElseIfBlock(block *TemplateElseIfBlock) error {
	for _, content := range block.Content {
		if err := validateDeferredNode(content); err != nil {
			return err
		}
	}
	return nil
}

// validateDeferredTemplateElseBlock validates else statement blocks
func validateDeferredTemplateElseBlock(block *TemplateElseBlock) error {
	for _, content := range block.Content {
		if err := validateDeferredNode(content); err != nil {
			return err
		}
	}
	return nil
}
