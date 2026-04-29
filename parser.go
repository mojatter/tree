package tree

import (
	"encoding/csv"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ParseQuery parses the provided expr to a Query.
// See https://github.com/mojatter/tree#Query
func ParseQuery(expr string) (Query, error) {
	p := &parser{expr: expr, tokens: lex(expr)}
	q, err := p.parseSequence(tkEOF)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkEOF {
		return nil, fmt.Errorf("syntax error: no left bracket: %q", expr)
	}
	return q, nil
}

// tokenKind is the kind tag attached to each lexer token.
type tokenKind int

const (
	tkEOF tokenKind = iota
	tkDot
	tkDotDot
	tkLBrack
	tkRBrack
	tkLParen
	tkRParen
	tkColon
	tkPipe
	tkEQ
	tkNE
	tkLT
	tkLE
	tkGT
	tkGE
	tkRE
	tkAnd
	tkOr
	tkIdent
	tkString
	tkMethod
)

// operator returns the Operator that this tokenKind represents, or the
// empty Operator if k is not a comparison operator token.
func (k tokenKind) operator() Operator {
	switch k {
	case tkEQ:
		return EQ
	case tkNE:
		return NE
	case tkLT:
		return LT
	case tkLE:
		return LE
	case tkGT:
		return GT
	case tkGE:
		return GE
	case tkRE:
		return RE
	}
	return ""
}

// token is a single lexer token. text holds the original source slice (for
// use in error messages); args is populated only for tkMethod and carries
// the raw argument string between the parentheses.
type token struct {
	kind tokenKind
	text string
	args string
}

// lex returns expr into a flat slice of tokens terminated by tkEOF. Bytes
// that don't form a recognized token (whitespace, stray punctuation like
// '+' or '\') are silently skipped, preserving legacy behavior.
func lex(expr string) []token {
	var out []token
	for i := 0; i < len(expr); {
		c := expr[i]
		if c == '"' {
			end := i + 1
			for end < len(expr) && expr[end] != '"' {
				end++
			}
			if end < len(expr) {
				if end > i+1 {
					out = append(out, token{kind: tkString, text: expr[i+1 : end]})
				}
				i = end + 1
				continue
			}
			i++
			continue
		}
		if op := matchOp(expr, i); op != "" {
			out = append(out, token{kind: cmdKind(op), text: op})
			i += len(op)
			continue
		}
		if isLower(c) {
			nameEnd := i + 1
			for nameEnd < len(expr) && isLower(expr[nameEnd]) {
				nameEnd++
			}
			if nameEnd < len(expr) && expr[nameEnd] == '(' {
				closeIdx := nameEnd + 1
				for closeIdx < len(expr) && expr[closeIdx] != ')' {
					closeIdx++
				}
				if closeIdx < len(expr) {
					out = append(out, token{
						kind: tkMethod,
						text: expr[i:nameEnd],
						args: expr[nameEnd+1 : closeIdx],
					})
					i = closeIdx + 1
					continue
				}
			}
		}
		if isWord(c) {
			end := i + 1
			for end < len(expr) && isWord(expr[end]) {
				end++
			}
			out = append(out, token{kind: tkIdent, text: expr[i:end]})
			i = end
			continue
		}
		i++
	}
	out = append(out, token{kind: tkEOF})
	return out
}

// matchOp returns the operator, keyword, or punctuation token starting
// at expr[i], or "" if none matches. The probe order mirrors the alternation
// in the legacy regex so leftmost-first semantics are preserved.
func matchOp(expr string, i int) string {
	rest := expr[i:]
	for _, kw := range []string{"and", "or"} {
		if strings.HasPrefix(rest, kw) {
			return kw
		}
	}
	for _, op := range []string{"==", "<=", ">=", "!=", "~=", ".."} {
		if strings.HasPrefix(rest, op) {
			return op
		}
	}
	switch expr[i] {
	case '.', '[', ']', '(', ')', '|', '<', '>', ':':
		return string(expr[i])
	}
	return ""
}

func cmdKind(cmd string) tokenKind {
	switch cmd {
	case ".":
		return tkDot
	case "..":
		return tkDotDot
	case "[":
		return tkLBrack
	case "]":
		return tkRBrack
	case "(":
		return tkLParen
	case ")":
		return tkRParen
	case ":":
		return tkColon
	case "|":
		return tkPipe
	case "==":
		return tkEQ
	case "!=":
		return tkNE
	case "<":
		return tkLT
	case "<=":
		return tkLE
	case ">":
		return tkGT
	case ">=":
		return tkGE
	case "~=":
		return tkRE
	case "and":
		return tkAnd
	case "or":
		return tkOr
	}
	return tkEOF
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func isWord(c byte) bool {
	return isLower(c) ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// parser holds the recursive-descent state.
type parser struct {
	expr   string
	tokens []token
	pos    int
}

func (p *parser) peek() token {
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

// parseSequence consumes 0 or more steps until tkEOF or one of stops is seen.
// 0 steps yield ValueQuery{Nil}, 1 step is returned bare, multiple steps are
// wrapped in a FilterQuery.
func (p *parser) parseSequence(stops ...tokenKind) (Query, error) {
	var steps []Query
	for {
		k := p.peek().kind
		if k == tkEOF || slices.Contains(stops, k) {
			break
		}
		step, err := p.parseStep()
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return ValueQuery{Nil}, nil
	}
	if len(steps) == 1 {
		return steps[0], nil
	}
	return FilterQuery(steps), nil
}

// parseStep parses a single path step.
func (p *parser) parseStep() (Query, error) {
	t := p.advance()
	switch t.kind {
	case tkDot:
		if next := p.peek(); next.kind == tkIdent || next.kind == tkString {
			p.advance()
			return MapQuery(next.text), nil
		}
		return NopQuery{}, nil
	case tkDotDot:
		if next := p.peek(); next.kind == tkIdent || next.kind == tkString {
			p.advance()
			return WalkQuery(next.text), nil
		}
		return NopQuery{}, nil
	case tkLBrack:
		return p.parseBracket()
	case tkLParen:
		inner, err := p.parseSequence(tkRParen)
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("syntax error: no right brackets: %q", p.expr)
		}
		p.advance()
		return inner, nil
	case tkPipe:
		return SlurpQuery{}, nil
	case tkMethod:
		var args []string
		if t.args != "" {
			r := csv.NewReader(strings.NewReader(t.args))
			var err error
			args, err = r.Read()
			if err != nil {
				return nil, fmt.Errorf("failed to parse args: %w", err)
			}
		}
		return NewMethodQuery(t.text, args...)
	case tkIdent:
		return ValueQuery{wordValue(t.text)}, nil
	case tkString:
		return ValueQuery{StringValue(t.text)}, nil
	case tkRBrack, tkRParen:
		return nil, fmt.Errorf("syntax error: no left bracket: %q", p.expr)
	default:
		return nil, fmt.Errorf("syntax error: invalid token %s: %q", t.text, p.expr)
	}
}

// parseBracket parses the contents of `[...]`. The leading `[` is already
// consumed; this function consumes through the matching `]`.
func (p *parser) parseBracket() (Query, error) {
	if p.peek().kind == tkRBrack {
		p.advance()
		return SelectQuery{}, nil
	}

	if p.peek().kind == tkIdent &&
		p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].kind == tkRBrack {
		text := p.peek().text
		i, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array index: %q", p.expr)
		}
		p.advance()
		p.advance()
		return ArrayQuery(i), nil
	}

	if p.bracketHasColon() {
		return p.parseArrayRange()
	}

	sel, err := p.parseSelector()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkRBrack {
		return nil, fmt.Errorf("syntax error: no right brackets: %q", p.expr)
	}
	p.advance()
	return SelectQuery{Selector: sel}, nil
}

// bracketHasColon reports whether a `:` appears at the bracket's top level
// (i.e. with balanced sub-brackets/parens) before the matching `]`.
func (p *parser) bracketHasColon() bool {
	depth := 0
	for i := p.pos; i < len(p.tokens); i++ {
		switch p.tokens[i].kind {
		case tkLBrack, tkLParen:
			depth++
		case tkRBrack:
			if depth == 0 {
				return false
			}
			depth--
		case tkRParen:
			depth--
		case tkColon:
			if depth == 0 {
				return true
			}
		case tkEOF:
			return false
		}
	}
	return false
}

// parseArrayRange parses `[from? : to?]`. The leading `[` is already consumed.
func (p *parser) parseArrayRange() (Query, error) {
	from := -1
	if p.peek().kind != tkColon {
		if p.peek().kind != tkIdent {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		i, err := strconv.Atoi(p.peek().text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		from = i
		p.advance()
	}
	if p.peek().kind != tkColon {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.advance()
	to := -1
	if p.peek().kind != tkRBrack {
		if p.peek().kind != tkIdent {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		i, err := strconv.Atoi(p.peek().text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		to = i
		p.advance()
	}
	if p.peek().kind != tkRBrack {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.advance()
	return ArrayRangeQuery{from, to}, nil
}

// parseSelector parses a complete selector expression. The result is always
// wrapped in And or Or at the top level so that callers see the same shape
// the legacy parser produced.
func (p *parser) parseSelector() (Selector, error) {
	sel, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}
	switch sel.(type) {
	case Or, And:
		return sel, nil
	}
	return And{sel}, nil
}

func (p *parser) parseOrExpr() (Selector, error) {
	first, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkOr {
		return first, nil
	}
	sels := []Selector{first}
	for p.peek().kind == tkOr {
		p.advance()
		next, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		sels = append(sels, next)
	}
	return Or(sels), nil
}

func (p *parser) parseAndExpr() (Selector, error) {
	first, err := p.parseCompExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkAnd {
		return first, nil
	}
	sels := []Selector{first}
	for p.peek().kind == tkAnd {
		p.advance()
		next, err := p.parseCompExpr()
		if err != nil {
			return nil, err
		}
		sels = append(sels, next)
	}
	return And(sels), nil
}

func (p *parser) parseCompExpr() (Selector, error) {
	if p.peek().kind == tkLParen {
		p.advance()
		inner, err := p.parseOrExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("syntax error: no right brackets: %q", p.expr)
		}
		p.advance()
		return inner, nil
	}
	left, err := p.parseSelectorOperand()
	if err != nil {
		return nil, err
	}
	op := p.peek().kind.operator()
	if op == "" {
		return Evaluator{Query: left}, nil
	}
	p.advance()
	right, err := p.parseSelectorOperand()
	if err != nil {
		return nil, err
	}
	return Comparator{Left: left, Op: op, Right: right}, nil
}

// parseSelectorOperand parses the LHS or RHS of a comparator: a sequence of
// path steps that stops at any selector-level structural token.
func (p *parser) parseSelectorOperand() (Query, error) {
	return p.parseSequence(
		tkAnd, tkOr,
		tkEQ, tkNE, tkLT, tkLE, tkGT, tkGE, tkRE,
		tkRParen, tkRBrack,
	)
}

// wordValue parses an unquoted bare word into a literal Node.
func wordValue(s string) Node {
	if s == "" {
		return Nil
	}
	if s == "true" {
		return BoolValue(true)
	}
	if s == "false" {
		return BoolValue(false)
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return NumberValue(n)
	}
	return StringValue(s)
}
