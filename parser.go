package tree

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

// ParseQuery parses the provided expr to a Query.
// See https://github.com/mojatter/tree#Query
func ParseQuery(expr string) (Query, error) {
	t, err := tokenizeQuery(expr)
	if err != nil {
		return nil, err
	}
	return tokenToQuery(t, expr)
}

// scannedToken mirrors the capture-group layout of the legacy regex-based
// scanner so that tokenizeQuery can consume it with the same field access
// pattern the regex matches used to provide.
type scannedToken struct {
	quoted  string
	cmd     string
	method  string
	argsStr string
	word    string
}

// scanTokens walks expr left-to-right and yields tokens equivalent to those
// the legacy tokenRegexp would produce. Bytes that the legacy regex did not
// match (whitespace, stray punctuation like '+' or '\'', etc.) are silently
// skipped, preserving the original behavior.
func scanTokens(expr string) []scannedToken {
	var out []scannedToken
	for i := 0; i < len(expr); {
		c := expr[i]

		if c == '"' {
			end := i + 1
			for end < len(expr) && expr[end] != '"' {
				end++
			}
			if end < len(expr) {
				out = append(out, scannedToken{quoted: expr[i+1 : end]})
				i = end + 1
				continue
			}
			i++
			continue
		}

		if op := matchScannerOp(expr, i); op != "" {
			out = append(out, scannedToken{cmd: op})
			i += len(op)
			continue
		}

		if c >= 'a' && c <= 'z' {
			nameEnd := i + 1
			for nameEnd < len(expr) && expr[nameEnd] >= 'a' && expr[nameEnd] <= 'z' {
				nameEnd++
			}
			if nameEnd < len(expr) && expr[nameEnd] == '(' {
				closeIdx := nameEnd + 1
				for closeIdx < len(expr) && expr[closeIdx] != ')' {
					closeIdx++
				}
				if closeIdx < len(expr) {
					out = append(out, scannedToken{
						cmd:     expr[i : closeIdx+1],
						method:  expr[i:nameEnd],
						argsStr: expr[nameEnd+1 : closeIdx],
					})
					i = closeIdx + 1
					continue
				}
			}
		}

		if isWordByte(c) {
			end := i + 1
			for end < len(expr) && isWordByte(expr[end]) {
				end++
			}
			out = append(out, scannedToken{word: expr[i:end]})
			i = end
			continue
		}

		i++
	}
	return out
}

// matchScannerOp returns the operator, keyword, or punctuation token starting
// at expr[i], or "" if none matches. The probe order mirrors the alternation
// in the legacy regex so leftmost-first semantics are preserved.
func matchScannerOp(expr string, i int) string {
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

func isWordByte(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

type token struct {
	cmd      string
	method   string
	argsStr  string
	quoted   bool
	value    string
	parent   *token
	children []*token
}

// toValue converts a token to a Node value based on its type and content.
// Handles quoted strings, booleans, numbers, and defaults to string values.
func (t *token) toValue() Node {
	if !t.quoted {
		if t.value == "" {
			return Nil
		}
		if t.value == "true" {
			return BoolValue(true)
		}
		if t.value == "false" {
			return BoolValue(false)
		}
		if n, err := strconv.ParseFloat(t.value, 64); err == nil {
			return NumberValue(n)
		}
	}
	return StringValue(t.value)
}

// indexOfCmd finds the index of a child token with the specified command.
// Returns -1 if not found.
func (t *token) indexOfCmd(cmd string) int {
	for i, c := range t.children {
		if c.cmd == cmd {
			return i
		}
	}
	return -1
}

func (t *token) Args() ([]string, error) {
	if t.argsStr == "" {
		return nil, nil
	}
	r := csv.NewReader(strings.NewReader(t.argsStr))
	args, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to parse args: %w", err)
	}
	return args, nil
}

// tokenizeQuery walks expr through the hand-written scanner and assembles
// the resulting tokens into the bracket-aware tree that tokenToQuery
// consumes.
func tokenizeQuery(expr string) (*token, error) {
	current := &token{}
	for _, st := range scanTokens(expr) {
		if st.quoted != "" || st.word != "" {
			value := st.quoted
			quoted := st.quoted != ""
			if value == "" {
				value = st.word
			}
			var lastChild *token
			if len(current.children) > 0 {
				lastChild = current.children[len(current.children)-1]
			}
			if lastChild != nil && (lastChild.cmd == "." || lastChild.cmd == "..") {
				lastChild.value = value
				lastChild.quoted = quoted
				continue
			}
			current.children = append(current.children, &token{value: value, quoted: quoted})
			continue
		}
		t := &token{cmd: st.cmd, method: st.method, argsStr: st.argsStr, parent: current}
		switch st.cmd {
		case "]", ")":
			if (st.cmd == "]" && current.cmd != "[") || (st.cmd == ")" && current.cmd != "(") {
				return nil, fmt.Errorf("syntax error: no left bracket: %q", expr)
			}
			current = current.parent
		case "[", "(":
			current.children = append(current.children, t)
			current = t
		default:
			current.children = append(current.children, t)
		}
	}
	if current.parent != nil {
		return nil, fmt.Errorf("syntax error: no right brackets: %q", expr)
	}
	return current, nil
}

// tokenToQuery converts a token tree into a Query object.
// Recursively processes tokens based on their command type.
func tokenToQuery(t *token, expr string) (Query, error) {
	if t.method != "" {
		args, err := t.Args()
		if err != nil {
			return nil, err
		}
		return NewMethodQuery(t.method, args...)
	}
	child := len(t.children)
	switch t.cmd {
	case "":
		if child == 0 {
			return ValueQuery{t.toValue()}, nil
		}
	case "|":
		return SlurpQuery{}, nil
	case ".":
		if t.value != "" {
			return MapQuery(t.value), nil
		}
		return NopQuery{}, nil
	case "..":
		if t.value != "" {
			return WalkQuery(t.value), nil
		}
		return NopQuery{}, nil
	case "[":
		if child == 0 {
			return SelectQuery{}, nil
		}
		if child == 1 {
			i, err := strconv.Atoi(t.children[0].value)
			if err != nil {
				return nil, fmt.Errorf("syntax error: invalid array index: %q", expr)
			}
			return ArrayQuery(i), nil
		}
		if i := t.indexOfCmd(":"); i != -1 {
			return tokensToArrayRangeQuery(t.children, i, expr)
		}
		selector, err := tokensToSelector(t.children, expr)
		if err != nil {
			return nil, err
		}
		return SelectQuery{selector}, nil
	}
	if child == 0 {
		return nil, fmt.Errorf("syntax error: invalid token %s: %q", t.cmd, expr)
	}
	if child == 1 {
		return tokenToQuery(t.children[0], expr)
	}
	var fq FilterQuery
	for _, c := range t.children {
		q, err := tokenToQuery(c, expr)
		if err != nil {
			return nil, err
		}
		fq = append(fq, q)
	}
	return fq, nil
}

// tokensToArrayRangeQuery creates an ArrayRangeQuery from tokens.
// Handles array slice notation like [from:to].
func tokensToArrayRangeQuery(ts []*token, i int, expr string) (Query, error) {
	from := -1
	to := -1
	if j := i - 1; j >= 0 {
		var err error
		from, err = strconv.Atoi(ts[j].value)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", expr)
		}
	}
	if j := i + 1; j < len(ts) {
		var err error
		to, err = strconv.Atoi(ts[j].value)
		if err != nil {
			return nil, fmt.Errorf("syntax error: invalid array range: %q", expr)
		}
	}
	return ArrayRangeQuery{from, to}, nil
}

// tokensToSelector converts tokens into a Selector for filtering operations.
// Handles logical operators (and/or) and comparison operators.
func tokensToSelector(ts []*token, expr string) (Selector, error) {
	andOr := ""
	var groups [][]*token
	off := 0
	for i, t := range ts {
		switch t.cmd {
		case "and", "or":
			if andOr != "" && andOr != t.cmd {
				return nil, fmt.Errorf("syntax error: mixed and|or: %q", expr)
			}
			andOr = t.cmd
			groups = append(groups, ts[off:i])
			off = i + 1
		case "(":
			groups = append(groups, ts[off:i])
			groups = append(groups, []*token{t})
			off = i + 1
		}
	}
	groups = append(groups, ts[off:])

	var ss []Selector
	for _, group := range groups {
		op := -1
	GROUP:
		for i, t := range group {
			if t.cmd == "(" {
				sss, err := tokensToSelector(t.children, expr)
				if err != nil {
					return nil, err
				}
				ss = append(ss, sss)
				break
			}
			switch Operator(t.cmd) {
			case EQ, GT, GE, LT, LE, NE, RE:
				op = i
				break GROUP
			}
		}
		if op == -1 {
			if len(groups) == 1 && len(group) > 0 {
				q, err := tokenToQuery(&token{children: group}, expr)
				if err != nil {
					return nil, err
				}
				ss = append(ss, Evaluator{Query: q})
			}
			continue
		}
		left, err := tokenToQuery(&token{children: group[0:op]}, expr)
		if err != nil {
			return nil, err
		}
		right, err := tokenToQuery(&token{children: group[op+1:]}, expr)
		if err != nil {
			return nil, err
		}
		ss = append(ss, Comparator{left, Operator(group[op].cmd), right})
	}
	if andOr == "or" {
		return Or(ss), nil
	}
	return And(ss), nil
}
