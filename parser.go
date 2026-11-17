package tree

import (
	"encoding/csv"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseQuery parses the provided expr to a Query.
// See https://github.com/mojatter/tree/blob/main/docs/query.md for the semantics.
func ParseQuery(expr string) (Query, error) {
	l := &lexer{expr: expr}
	tokens, err := l.lex()
	if err != nil {
		return nil, err
	}
	p := &parser{expr: expr, tokens: tokens}
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

// scanQuoted parses the quoted string at l.pos and advances past its closing quote; ok is false when it is unterminated.
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
		return token{kind: tkString, text: sb.String()}, true
	}
	return token{}, false
}

// scanMethod parses a method-call token (`name(args)`) starting at l.pos
// (where l.expr[l.pos] is a lowercase letter). On success returns the
// tkMethod token with name and raw args text, advancing l.pos past the
// closing `)`. A '"' in the args is skipped by CSV rules, as the args are
// read as CSV; an unterminated '"' keeps the call with the rest of the
// input as args so the CSV reader reports it. Returns ok=false (and
// leaves l.pos unchanged) when there is no `(` after the lowercase run,
// or when the `)` is missing; the caller then scans a plain ident.
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
		switch l.expr[closeIdx] {
		case '"':
			closeIdx = skipCSVQuoted(l.expr, closeIdx)
			if closeIdx >= len(l.expr) {
				// Let the CSV reader report it rather than reading the name as an ident.
				tok = token{kind: tkMethod, text: l.expr[l.pos:nameEnd], args: l.expr[nameEnd+1:]}
				l.pos = len(l.expr)
				return tok, true
			}
			continue
		case '\'':
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

// scanWord parses the keyword, method call or identifier starting at l.pos (a word rune).
// and/or are keywords unless they directly follow a '.', so .orders and .or stay keys.
func (l *lexer) scanWord(prevPathDot bool) token {
	end := wordEnd(l.expr, l.pos)
	if text := l.expr[l.pos:end]; (text == "and" || text == "or") && (l.pos == 0 || l.expr[l.pos-1] != '.') {
		l.pos = end
		return token{kind: cmdKind(text), text: text}
	}
	if isLower(l.expr[l.pos]) {
		if tok, ok := l.scanMethod(); ok {
			return tok
		}
	}
	return l.scanIdent(prevPathDot)
}

// scanIdent parses the tkIdent at l.pos; unless prevPathDot, a digit-led one also takes a fraction (1.5) and a signed exponent (1e+3).
func (l *lexer) scanIdent(prevPathDot bool) token {
	start := l.pos
	end := wordEnd(l.expr, start)
	digitLed := isNumber(l.expr[start])
	if digitLed && !prevPathDot &&
		end+1 < len(l.expr) && l.expr[end] == '.' && isNumber(l.expr[end+1]) {
		end = wordEnd(l.expr, end+1)
	}
	// Unsigned exponent (`1e3`) is already part of the word.
	if digitLed && !prevPathDot && end+1 < len(l.expr) &&
		(l.expr[end] == '+' || l.expr[end] == '-') &&
		(l.expr[end-1] == 'e' || l.expr[end-1] == 'E') &&
		isNumber(l.expr[end+1]) {
		end = wordEnd(l.expr, end+1)
	}
	l.pos = end
	return token{kind: tkIdent, text: l.expr[start:end]}
}

// blanks are the whitespace characters the lexer skips.
const blanks = " \t\n\r"

// lex returns the source as a flat slice of tokens terminated by tkEOF.
// Whitespace is one of blanks; any other character that starts no token is a syntax error.
func (l *lexer) lex() ([]token, error) {
	var out []token
	for l.pos < len(l.expr) {
		c := l.expr[l.pos]
		switch c {
		case ' ', '\t', '\n', '\r':
			l.pos++
			continue
		case '"', '\'':
			tok, ok := l.scanQuoted()
			if !ok {
				return nil, fmt.Errorf("syntax error: unterminated string at offset %d: %q", l.pos, l.expr)
			}
			out = append(out, tok)
			continue
		}
		if op := matchOp(l.expr, l.pos); op != "" {
			out = append(out, token{kind: cmdKind(op), text: op})
			l.pos += len(op)
			continue
		}
		r, size := utf8.DecodeRuneInString(l.expr[l.pos:])
		if !isWordRune(r) {
			return nil, fmt.Errorf("syntax error: unexpected character %q at offset %d: %q", l.expr[l.pos:l.pos+size], l.pos, l.expr)
		}
		out = append(out, l.scanWord(prevIsPathDot(out)))
	}
	out = append(out, token{kind: tkEOF})
	return out, nil
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

// skipCSVQuoted returns the position past the CSV-quoted field at expr[p] (a doubled '"' is literal, '\' is plain), or len(expr) when unterminated.
func skipCSVQuoted(expr string, p int) int {
	for p++; p < len(expr); p++ {
		if expr[p] != '"' {
			continue
		}
		if p+1 < len(expr) && expr[p+1] == '"' {
			p++
			continue
		}
		return p + 1
	}
	return p
}

// matchOp returns the operator or punctuation token starting at expr[i], or "" if none matches.
// Multi-byte operators are probed before single bytes so ".." and "<=" win.
func matchOp(expr string, i int) string {
	rest := expr[i:]
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

// isWordRune reports whether r can appear in an unquoted key: a Unicode letter, digit or mark, or '_'.
func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}

// startsWord reports whether s starts with a word rune.
func startsWord(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return isWordRune(r)
}

// endsWord reports whether s ends with a word rune.
func endsWord(s string) bool {
	r, _ := utf8.DecodeLastRuneInString(s)
	return isWordRune(r)
}

// wordStart returns the index of the run of word runes in s ending at j.
func wordStart(s string, j int) int {
	for j > 0 {
		r, size := utf8.DecodeLastRuneInString(s[:j])
		if !isWordRune(r) {
			break
		}
		j -= size
	}
	return j
}

// wordEnd returns the index past the run of word runes in s starting at i.
func wordEnd(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isWordRune(r) {
			break
		}
		i += size
	}
	return i
}

// prevIsPathDot reports whether the last emitted token is `.` or `..`, after which a digit-led key takes no fraction or signed exponent.
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
// 0 steps are a syntax error, 1 step is returned bare, multiple steps are
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
		if p.peek().kind == tkEOF && !slices.Contains(stops, tkEOF) {
			return nil, fmt.Errorf("syntax error: no right brackets: %q", p.expr)
		}
		return nil, fmt.Errorf("syntax error: empty expression: %q", p.expr)
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
	from, err := p.parseRangeBound(tkColon)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkColon {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.skip(1)
	to, err := p.parseRangeBound(tkRBrack)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkRBrack {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.skip(1)
	return ArrayRangeQuery{From: from, To: to}, nil
}

// parseRangeBound parses one optional bound of `[from?:to?]`. It returns nil
// when the bound is omitted (peek is stop). Accepts a leading `-` for
// negative bounds (Python-style end-relative indexing).
func (p *parser) parseRangeBound(stop tokenKind) (*int, error) {
	if p.peek().kind == stop {
		return nil, nil
	}
	sign := 1
	if p.peek().kind == tkMinus {
		sign = -1
		p.skip(1)
	}
	if p.peek().kind != tkIdent {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	i, err := strconv.Atoi(p.peek().text)
	if err != nil {
		return nil, fmt.Errorf("syntax error: invalid array range: %q", p.expr)
	}
	p.skip(1)
	v := sign * i
	return &v, nil
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
	switch s {
	case "null":
		return Nil
	case "true":
		return BoolValue(true)
	case "false":
		return BoolValue(false)
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil &&
		!math.IsInf(n, 0) && !math.IsNaN(n) {
		return NumberValue(n)
	}
	return StringValue(s)
}
