package tree

import (
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
	return parseTokens(expr, tokens)
}

// parseTokens parses the tokens lexed from expr.
func parseTokens(expr string, tokens []token) (Query, error) {
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
// use in error messages); args holds a tkMethod's decoded arguments.
type token struct {
	kind tokenKind
	text string
	args []string
}

// lexer holds the lex state: the source string and current position.
type lexer struct {
	expr string
	pos  int
	edit bool   // stop at the first edit operator, as lex describes
	op   string // the edit operator lex stopped at, at pos
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

// scanMethod parses the method call `name(args)` at l.pos (a lowercase letter); ok is false when no `(` follows the name.
func (l *lexer) scanMethod() (tok token, ok bool, err error) {
	nameEnd := l.pos + 1
	for nameEnd < len(l.expr) && isLower(l.expr[nameEnd]) {
		nameEnd++
	}
	if nameEnd >= len(l.expr) || l.expr[nameEnd] != '(' {
		return token{}, false, nil
	}
	var args []string
	var closers []byte // the closers expected for the open parentheses and brackets, innermost last
	argStart := nameEnd + 1
	for i := argStart; i < len(l.expr); i++ {
		c := l.expr[i]
		switch {
		case c == '"' || c == '\'':
			q := lexer{expr: l.expr, pos: i}
			if _, ok := q.scanQuoted(); !ok {
				return token{}, false, l.unterminated(i)
			}
			i = q.pos - 1
		case c == '(':
			closers = append(closers, ')')
		case c == '[':
			closers = append(closers, ']')
		case len(closers) > 0 && (c == ')' || c == ']'):
			if closers[len(closers)-1] != c {
				return token{}, false, l.unexpected(i)
			}
			closers = closers[:len(closers)-1]
		case c == ']':
			return token{}, false, l.unexpected(i)
		case len(closers) == 0 && (c == ',' || c == ')'):
			start, end := trimBlanks(l.expr, argStart, i)
			if start == end && (c == ',' || args != nil) {
				return token{}, false, fmt.Errorf("syntax error: empty argument at offset %d: %q", start, l.expr)
			}
			if start != end {
				arg, err := l.methodArg(start, end)
				if err != nil {
					return token{}, false, err
				}
				args = append(args, arg)
			}
			if c == ')' {
				tok = token{kind: tkMethod, text: l.expr[l.pos:nameEnd], args: args}
				l.pos = i + 1
				return tok, true, nil
			}
			argStart = i + 1
		}
	}
	return token{}, false, errNoRightBrackets(l.expr)
}

// methodArg returns the argument l.expr[start:end], trimmed of blanks: a lone string literal's contents, else its text.
func (l *lexer) methodArg(start, end int) (string, error) {
	if c := l.expr[start]; c != '"' && c != '\'' {
		return l.expr[start:end], nil
	}
	q := lexer{expr: l.expr, pos: start}
	tok, _ := q.scanQuoted() // terminated: scanMethod skipped it
	if rest, _ := trimBlanks(l.expr, q.pos, end); rest < end {
		return "", l.unexpected(rest)
	}
	return tok.text, nil
}

// trimBlanks returns the bounds of s[start:end] without the blanks around it.
func trimBlanks(s string, start, end int) (int, int) {
	t := strings.TrimLeft(s[start:end], blanks)
	start = end - len(t)
	return start, start + len(strings.TrimRight(t, blanks))
}

// unterminated returns the syntax error for the string that opens at offset i.
func (l *lexer) unterminated(i int) error {
	return fmt.Errorf("syntax error: unterminated string at offset %d: %q", i, l.expr)
}

// errNoRightBrackets returns the syntax error for a bracket that expr never closes.
func errNoRightBrackets(expr string) error {
	return fmt.Errorf("syntax error: no right brackets: %q", expr)
}

// unexpected returns the syntax error for the character at offset i.
func (l *lexer) unexpected(i int) error {
	_, size := utf8.DecodeRuneInString(l.expr[i:])
	return fmt.Errorf("syntax error: unexpected character %q at offset %d: %q", l.expr[i:i+size], i, l.expr)
}

// scanWord parses the keyword, method call or identifier starting at l.pos (a word rune).
// and/or are keywords unless they directly follow a '.', so .orders and .or stay keys.
func (l *lexer) scanWord(prevPathDot bool) (token, error) {
	end := wordEnd(l.expr, l.pos)
	if text := l.expr[l.pos:end]; (text == "and" || text == "or") && (l.pos == 0 || l.expr[l.pos-1] != '.') {
		l.pos = end
		return token{kind: cmdKind(text), text: text}, nil
	}
	if isLower(l.expr[l.pos]) {
		if tok, ok, err := l.scanMethod(); err != nil || ok {
			return tok, err
		}
	}
	return l.scanIdent(prevPathDot), nil
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
// In edit mode it stops at the first edit operator outside parentheses and brackets, or anywhere after a closer without an opener, leaving it in l.op with l.pos at it.
func (l *lexer) lex() ([]token, error) {
	var out []token
	depth, stray := 0, false // stray: a closer had no opener
	for l.pos < len(l.expr) {
		start := l.pos
		switch l.expr[start] {
		case ' ', '\t', '\n', '\r':
			l.pos++
			continue
		case '"', '\'':
			tok, ok := l.scanQuoted()
			if !ok {
				return nil, l.unterminated(start)
			}
			if err := l.checkKeyAfterBlank(out, start); err != nil {
				return nil, err
			}
			out = append(out, tok)
			continue
		}
		if op := matchOp(l.expr, start); op != "" {
			switch op {
			case "(", "[":
				depth++
			case ")", "]":
				if depth--; depth < 0 {
					stray = true
				}
			}
			out = append(out, token{kind: cmdKind(op), text: op})
			l.pos += len(op)
			continue
		}
		if l.edit && (depth == 0 || stray) {
			if l.op = matchEditOp(l.expr[start:]); l.op != "" {
				break
			}
		}
		if r, _ := utf8.DecodeRuneInString(l.expr[start:]); !isWordRune(r) {
			return nil, l.unexpected(start)
		}
		tok, err := l.scanWord(prevIsPathDot(out))
		if err != nil {
			return nil, err
		}
		if tok.kind == tkIdent {
			if err := l.checkKeyAfterBlank(out, start); err != nil {
				return nil, err
			}
		}
		out = append(out, tok)
	}
	out = append(out, token{kind: tkEOF})
	return out, nil
}

// checkKeyAfterBlank rejects a key at start that a blank separates from the . or .. before it.
func (l *lexer) checkKeyAfterBlank(out []token, start int) error {
	if !prevIsPathDot(out) {
		return nil
	}
	i := len(strings.TrimRight(l.expr[:start], blanks))
	if i == start {
		return nil
	}
	return fmt.Errorf("syntax error: unexpected blank at offset %d: %q", i, l.expr)
}

// editOps lists the edit operators.
var editOps = []string{"+=", "^?", "="}

// matchEditOp returns the edit operator s starts with, or "".
func matchEditOp(s string) string {
	for _, op := range editOps {
		if strings.HasPrefix(s, op) {
			return op
		}
	}
	return ""
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

// parseSequence parses steps until tkEOF or one of stops: one step is returned bare, more as a FilterQuery.
// No steps, an empty left side of a pipe and a literal next to a step are syntax errors.
func (p *parser) parseSequence(stops ...tokenKind) (Query, error) {
	var steps []Query
	for {
		k := p.peek().kind
		if k == tkEOF || slices.Contains(stops, k) {
			break
		}
		if k == tkPipe && (len(steps) == 0 || isSlurpQuery(steps[len(steps)-1])) {
			return nil, fmt.Errorf("syntax error: empty expression: %q", p.expr)
		}
		step, err := p.parseStep()
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	if len(steps) > 1 {
		for _, s := range steps {
			if v, ok := s.(ValueQuery); ok {
				return nil, p.errLiteral(v)
			}
		}
	}
	if len(steps) == 0 {
		if p.peek().kind == tkEOF && !slices.Contains(stops, tkEOF) {
			return nil, errNoRightBrackets(p.expr)
		}
		return nil, fmt.Errorf("syntax error: empty expression: %q", p.expr)
	}
	if len(steps) == 1 {
		return steps[0], nil
	}
	return FilterQuery(steps), nil
}

// errLiteral returns the syntax error for a literal that is neither a whole query nor a comparison side.
func (p *parser) errLiteral(v ValueQuery) error {
	return fmt.Errorf("syntax error: unexpected literal %s: %q", v, p.expr)
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
			return nil, errNoRightBrackets(p.expr)
		}
		p.skip(1)
		return inner, nil
	case tkPipe:
		return SlurpQuery{}, nil
	case tkMethod:
		return NewMethodQuery(t.text, t.args...)
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
		return nil, errNoRightBrackets(p.expr)
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
			return nil, errNoRightBrackets(p.expr)
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
		if v, ok := left.(ValueQuery); ok {
			return nil, p.errLiteral(v)
		}
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
			if k := p.peek().kind; k != tkEOF && !slices.Contains(selectorStops, k) {
				return nil, p.errLiteral(ValueQuery{v})
			}
			return ValueQuery{v}, nil
		}
	}
	return p.parseSequence(selectorStops...)
}

// selectorStops are the tokens that end a comparison side.
var selectorStops = []tokenKind{
	tkAnd, tkOr,
	tkEQ, tkNE, tkLT, tkLE, tkGT, tkGE, tkRE,
	tkRParen, tkRBrack,
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
