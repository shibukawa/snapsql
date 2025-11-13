package explang

import (
	"errors"
	"fmt"
	"unicode"
)

// ErrInvalidExpression indicates that an expression string could not be parsed.
var ErrInvalidExpression = errors.New("explang: invalid expression")

// ParseSteps parses an explang expression into a flattened list of access steps.
// startLine/startColumn allow callers to provide the 1-based location of the
// first rune of expr within a larger SQL/template, so Position metadata remains accurate.
func ParseSteps(expr string, startLine, startColumn int) ([]Step, error) {
	p := newParser(expr, startLine, startColumn)
	if err := p.parse(); err != nil {
		return nil, err
	}

	return p.steps, nil
}

type parser struct {
	src        []rune
	pos        int
	steps      []Step
	baseLine   int
	baseColumn int
}

func newParser(expr string, startLine, startColumn int) *parser {
	if startLine < 1 {
		startLine = 1
	}

	if startColumn < 1 {
		startColumn = 1
	}

	return &parser{src: []rune(expr), baseLine: startLine, baseColumn: startColumn}
}

func (p *parser) parse() error {
	if err := p.parseRootIdentifier(); err != nil {
		return err
	}

	for {
		p.skipWhitespace()

		if p.eof() {
			return nil
		}

		safe := false

		safeStart := -1
		if p.peek() == '?' {
			safeStart = p.pos
			p.pos++
			safe = true

			p.skipWhitespace()
		}

		switch p.peek() {
		case '.':
			if err := p.parseMemberStep(safe, safeStart); err != nil {
				return err
			}
		case '[':
			if err := p.parseIndexStep(safe, safeStart); err != nil {
				return err
			}
		default:
			if safe {
				return fmt.Errorf("%w: safe access operator must be followed by '.' or '[' at position %d", ErrInvalidExpression, p.pos+1)
			}

			if p.eof() {
				return nil
			}

			return fmt.Errorf("%w: unexpected character '%c' at position %d", ErrInvalidExpression, p.peek(), p.pos+1)
		}
	}
}

func (p *parser) parseRootIdentifier() error {
	p.skipWhitespace()

	ident, start, end, ok := p.readIdentifier()
	if !ok {
		if p.eof() {
			return fmt.Errorf("%w: expected identifier at position %d", ErrInvalidExpression, p.pos+1)
		}

		return fmt.Errorf("%w: unexpected character '%c' at position %d", ErrInvalidExpression, p.peek(), p.pos+1)
	}

	p.steps = append(p.steps, Step{Kind: StepIdentifier, Identifier: ident, Pos: p.makePosition(start, end)})

	return nil
}

func (p *parser) parseMemberStep(safe bool, safeStart int) error {
	dotStart := p.pos
	p.pos++
	p.skipWhitespace()

	ident, _, end, ok := p.readIdentifier()
	if !ok {
		return fmt.Errorf("%w: expected identifier after '.' at position %d", ErrInvalidExpression, p.pos+1)
	}

	spanStart := dotStart
	if safe && safeStart >= 0 {
		spanStart = safeStart
	}

	p.steps = append(p.steps, Step{Kind: StepMember, Property: ident, Safe: safe, Pos: p.makePosition(spanStart, end)})

	return nil
}

func (p *parser) parseIndexStep(safe bool, safeStart int) error {
	bracketStart := p.pos
	p.pos++
	p.skipWhitespace()

	idx, _, _, ok := p.readNumber()
	if !ok {
		return fmt.Errorf("%w: expected integer index after '[' at position %d", ErrInvalidExpression, p.pos+1)
	}

	p.skipWhitespace()

	if !p.match(']') {
		return fmt.Errorf("%w: expected ']' to close index at position %d", ErrInvalidExpression, p.pos+1)
	}

	spanStart := bracketStart
	if safe && safeStart >= 0 {
		spanStart = safeStart
	}

	p.steps = append(p.steps, Step{Kind: StepIndex, Index: idx, Safe: safe, Pos: p.makePosition(spanStart, p.pos)})

	return nil
}

func (p *parser) skipWhitespace() {
	for unicode.IsSpace(p.peek()) {
		p.pos++
	}
}

func (p *parser) match(r rune) bool {
	if p.peek() != r {
		return false
	}

	p.pos++

	return true
}

func (p *parser) readIdentifier() (string, int, int, bool) {
	if !isIdentStart(p.peek()) {
		return "", 0, 0, false
	}

	start := p.pos

	p.pos++
	for isIdentPart(p.peek()) {
		p.pos++
	}

	return string(p.src[start:p.pos]), start, p.pos, true
}

func (p *parser) readNumber() (int, int, int, bool) {
	if !unicode.IsDigit(p.peek()) {
		return 0, 0, 0, false
	}

	start := p.pos
	for unicode.IsDigit(p.peek()) {
		p.pos++
	}

	var val int
	for _, r := range p.src[start:p.pos] {
		val = val*10 + int(r-'0')
	}

	return val, start, p.pos, true
}

func (p *parser) peek() rune {
	if p.pos >= len(p.src) {
		return 0
	}

	return p.src[p.pos]
}

func (p *parser) eof() bool {
	return p.pos >= len(p.src)
}

func (p *parser) makePosition(start, end int) Position {
	pos := p.positionAt(start)
	pos.Length = end - start

	return pos
}

func (p *parser) positionAt(offset int) Position {
	if offset < 0 {
		offset = 0
	}

	if offset > len(p.src) {
		offset = len(p.src)
	}

	line := p.baseLine

	col := p.baseColumn
	for i := range offset {
		r := p.src[i]
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}

	return Position{Offset: offset, Line: line, Column: col}
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || unicode.IsDigit(r)
}
