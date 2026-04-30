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
	l := &lexer{expr: expr}
	p := &parser{expr: expr, tokens: l.lex()}
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
	tkMinus
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

// lexer holds the lex state: the source string and current position.
type lexer struct {
	expr string
	pos  int
}

// scanQuoted parses the quoted string literal at l.pos and unconditionally
// advances l.pos: past the closing quote on a terminated literal, or just
// past the opening quote on an unterminated one. ok=true means the result
// is worth emitting (terminated and non-empty); empty literals and
// unterminated ones both return ok=false.
func (l *lexer) scanQuoted() (tok token, ok bool) {
	var sb strings.Builder
	quote := l.expr[l.pos]
	end := l.pos + 1
	for end < len(l.expr) && l.expr[end] != quote {
		if l.expr[end] == '\\' && end+1 < len(l.expr) {
			sb.WriteByte(l.expr[end+1])
			end += 2
			continue
		}
		sb.WriteByte(l.expr[end])
		end++
	}
	if end < len(l.expr) {
		l.pos = end + 1
		if sb.Len() == 0 {
			return token{}, false
		}
		return token{kind: tkString, text: sb.String()}, true
	}
	l.pos++
	return token{}, false
}

// scanMethod parses a method-call token (`name(args)`) starting at l.pos
// (where l.expr[l.pos] is a lowercase letter). On success returns the
// tkMethod token with name and raw args text, advancing l.pos past the
// closing `)`. Returns ok=false (and leaves l.pos unchanged) when there
// is no `(` after the lowercase run, or when the args parenthesis is
// unterminated; the caller then falls through to plain ident scanning.
func (l *lexer) scanMethod() (tok token, ok bool) {
	nameEnd := l.pos + 1
	for nameEnd < len(l.expr) && isLower(l.expr[nameEnd]) {
		nameEnd++
	}
	if nameEnd >= len(l.expr) || l.expr[nameEnd] != '(' {
		return token{}, false
	}
	closeIdx := nameEnd + 1
	for closeIdx < len(l.expr) && l.expr[closeIdx] != ')' {
		if l.expr[closeIdx] == '"' || l.expr[closeIdx] == '\'' {
			closeIdx = skipQuoted(l.expr, closeIdx)
			continue
		}
		closeIdx++
	}
	if closeIdx >= len(l.expr) {
		return token{}, false
	}
	tok = token{
		kind: tkMethod,
		text: l.expr[l.pos:nameEnd],
		args: l.expr[nameEnd+1 : closeIdx],
	}
	l.pos = closeIdx + 1
	return tok, true
}

// scanIdent parses a tkIdent token starting at l.pos (where l.expr[l.pos]
// is an isWord byte). A digit-led run absorbs an optional `.[0-9]+`
// fractional part for float literals, plus an optional `[eE][+-]?[0-9]+`
// exponent. prevPathDot suppresses the fractional continuation so path
// steps like `.1.5` stay chained.
func (l *lexer) scanIdent(prevPathDot bool) token {
	start := l.pos
	end := l.pos + 1
	for end < len(l.expr) && isWord(l.expr[end]) {
		end++
	}
	digitLed := isNumber(l.expr[start])
	if digitLed && !prevPathDot &&
		end+1 < len(l.expr) && l.expr[end] == '.' && isNumber(l.expr[end+1]) {
		end++
		for end < len(l.expr) && isWord(l.expr[end]) {
			end++
		}
	}
	// Exponent sign continuation: a digit-led run that ended at `e`/`E`
	// followed by `+`/`-` followed by a digit absorbs the sign and digits.
	// Unsigned exponent (`1e3`) is already covered by the isWord body loop.
	if digitLed && end+1 < len(l.expr) &&
		(l.expr[end] == '+' || l.expr[end] == '-') &&
		end > start && (l.expr[end-1] == 'e' || l.expr[end-1] == 'E') &&
		isNumber(l.expr[end+1]) {
		end++
		for end < len(l.expr) && isWord(l.expr[end]) {
			end++
		}
	}
	l.pos = end
	return token{kind: tkIdent, text: l.expr[start:end]}
}

// lex returns the source as a flat slice of tokens terminated by tkEOF.
// Bytes that don't form a recognized token (whitespace, stray punctuation
// like '+' or '\') are silently skipped, preserving legacy behavior.
func (l *lexer) lex() []token {
	var out []token
	for l.pos < len(l.expr) {
		c := l.expr[l.pos]
		if c == '"' || c == '\'' {
			if tok, ok := l.scanQuoted(); ok {
				out = append(out, tok)
			}
			continue
		}
		if op := matchOp(l.expr, l.pos); op != "" {
			out = append(out, token{kind: cmdKind(op), text: op})
			l.pos += len(op)
			continue
		}
		if isLower(c) {
			if tok, ok := l.scanMethod(); ok {
				out = append(out, tok)
				continue
			}
		}
		if isWord(c) {
			out = append(out, l.scanIdent(prevIsPathDot(out)))
			continue
		}
		l.pos++
	}
	out = append(out, token{kind: tkEOF})
	return out
}

// skipQuoted returns the position past a quoted string literal at expr[p]
// (where expr[p] is the opening quote). Recognizes the same backslash
// escape rules as scanQuoted so embedded `\"` / `\'` don't false-terminate.
// Returns len(expr) for an unterminated literal.
func skipQuoted(expr string, p int) int {
	quote := expr[p]
	p++
	for p < len(expr) && expr[p] != quote {
		if expr[p] == '\\' && p+1 < len(expr) {
			p += 2
			continue
		}
		p++
	}
	if p < len(expr) {
		return p + 1
	}
	return p
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
	case '.', '[', ']', '(', ')', '|', '<', '>', ':', '-':
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
	case "-":
		return tkMinus
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

func isNumber(c byte) bool {
	return c >= '0' && c <= '9'
}

func isWord(c byte) bool {
	return isLower(c) ||
		(c >= 'A' && c <= 'Z') ||
		isNumber(c) ||
		c == '_'
}

// prevIsPathDot reports whether the most recently emitted token is a path-step
// `.` or `..`. Used to suppress float-literal extension after a path separator
// so that `.1.5` stays as chained `MapQuery("1"), MapQuery("5")`.
func prevIsPathDot(out []token) bool {
	if len(out) == 0 {
		return false
	}
	switch out[len(out)-1].kind {
	case tkDot, tkDotDot:
		return true
	}
	return false
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

// skip advances p.pos by n without returning the consumed tokens.
// Use it when the tokens have already been inspected via peek/lookahead
// and only their consumption is needed.
func (p *parser) skip(n int) {
	p.pos += n
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
			p.skip(1)
			return MapQuery(next.text), nil
		}
		return NopQuery{}, nil
	case tkDotDot:
		if next := p.peek(); next.kind == tkIdent || next.kind == tkString {
			p.skip(1)
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
		p.skip(1)
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
		p.skip(1)
		return SelectQuery{}, nil
	}

	if p.peek().kind == tkString &&
		p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].kind == tkRBrack {
		text := p.peek().text
		p.skip(2)
		return MapQuery(text), nil
	}

	if p.peek().kind == tkIdent &&
		p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].kind == tkRBrack {
		text := p.peek().text
		i, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array index: %q", p.expr)
		}
		p.skip(2)
		return ArrayQuery(i), nil
	}

	if p.peek().kind == tkMinus &&
		p.pos+2 < len(p.tokens) &&
		p.tokens[p.pos+1].kind == tkIdent &&
		p.tokens[p.pos+2].kind == tkRBrack {
		text := p.tokens[p.pos+1].text
		i, err := strconv.Atoi("-" + text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array index: %q", p.expr)
		}
		p.skip(3)
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
	p.skip(1)
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
	var from *int
	if p.peek().kind != tkColon {
		if p.peek().kind != tkIdent {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		i, err := strconv.Atoi(p.peek().text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		from = &i
		p.skip(1)
	}
	if p.peek().kind != tkColon {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.skip(1)
	var to *int
	if p.peek().kind != tkRBrack {
		if p.peek().kind != tkIdent {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		i, err := strconv.Atoi(p.peek().text)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
		}
		to = &i
		p.skip(1)
	}
	if p.peek().kind != tkRBrack {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.skip(1)
	return ArrayRangeQuery{From: from, To: to}, nil
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
		p.skip(1)
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
		p.skip(1)
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
		p.skip(1)
		inner, err := p.parseOrExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("syntax error: no right brackets: %q", p.expr)
		}
		p.skip(1)
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
	p.skip(1)
	right, err := p.parseSelectorOperand()
	if err != nil {
		return nil, err
	}
	return Comparator{Left: left, Op: op, Right: right}, nil
}

// parseSelectorOperand parses the LHS or RHS of a comparator: a sequence of
// path steps that stops at any selector-level structural token. A leading
// `tkMinus + tkIdent(numeric)` is consumed as a single negative number
// literal; non-numeric `-X` falls through and errors via parseSequence.
func (p *parser) parseSelectorOperand() (Query, error) {
	if p.peek().kind == tkMinus &&
		p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].kind == tkIdent {
		v := wordValue("-" + p.tokens[p.pos+1].text)
		if _, ok := v.(NumberValue); ok {
			p.skip(2)
			return ValueQuery{v}, nil
		}
	}
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
