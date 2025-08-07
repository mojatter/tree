package tree

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Query is an interface that defines the methods to query a node.
type Query interface {
	Exec(n Node) ([]Node, error)
	String() string
}

// EditorQuery is an interface that defines the methods to edit a node.
type EditorQuery interface {
	Query
	Set(pn *Node, v Node) error
	Append(pn *Node, v Node) error
	Delete(pn *Node) error
}

// NopQuery is a query that implements no-op Exec method.
type NopQuery struct{}

var _ EditorQuery = (*NopQuery)(nil)

// Exec returns the provided node.
func (q NopQuery) Exec(n Node) ([]Node, error) {
	return []Node{n}, nil
}

func (q NopQuery) String() string {
	return "."
}

func (q NopQuery) Set(pn *Node, v Node) error {
	*pn = v
	return nil
}

func (q NopQuery) Append(pn *Node, v Node) error {
	if en, ok := (*pn).(EditorNode); ok {
		if err := en.Append(v); err == nil {
			return nil
		}
	}
	return fmt.Errorf("cannot append to %s", ".")
}

func (q NopQuery) Delete(pn *Node) error {
	return fmt.Errorf("cannot delete %s", ".")
}

// ValueQuery is a query that returns the constant value.
type ValueQuery struct {
	Node
}

// Exec returns the constant value.
func (q ValueQuery) Exec(n Node) ([]Node, error) {
	if q.IsNil() {
		return nil, nil
	}
	return []Node{q.Node}, nil
}

func (q ValueQuery) String() string {
	s, _ := MarshalJSON(q.Node)
	return string(s)
}

// MapQuery is a key of the Map that implements methods of the Query.
type MapQuery string

var _ EditorQuery = (MapQuery)("")

func (q MapQuery) Exec(n Node) ([]Node, error) {
	key := string(q)
	if n.Type().IsValue() {
		return nil, fmt.Errorf("cannot index array with %q", key)
	}
	if n.Has(key) {
		return []Node{n.Get(key)}, nil
	}
	return nil, nil
}

func (q MapQuery) Set(pn *Node, v Node) error {
	key := string(q)
	if en, ok := (*pn).(EditorNode); ok {
		return en.Set(key, v)
	}
	return fmt.Errorf("cannot index array with %q", key)
}

func (q MapQuery) Append(pn *Node, v Node) error {
	n := *pn
	key := string(q)
	if en, ok := (*pn).(EditorNode); ok {
		if n.Has(key) {
			x := n.Get(key)
			if x != nil {
				if ex, ok := x.(EditorNode); ok {
					if err := ex.Append(v); err == nil {
						return nil
					}
				}
			}
			return fmt.Errorf("cannot append to %q", key)
		}
		return en.Set(key, Array{v})
	}
	return fmt.Errorf("cannot append to %q", key)
}

func (q MapQuery) Delete(pn *Node) error {
	key := string(q)
	if en, ok := (*pn).(EditorNode); ok {
		if err := en.Delete(key); err == nil {
			return nil
		}
	}
	return fmt.Errorf("cannot delete %q", key)
}

func (q MapQuery) String() string {
	return "." + string(q)
}

// ArrayQuery is an index of the Array that implements methods of the Query.
type ArrayQuery int

var _ EditorQuery = (ArrayQuery)(0)

func (q ArrayQuery) Exec(n Node) ([]Node, error) {
	if a := n.Array(); a != nil {
		index := int(q)
		if n.Has(index) {
			return []Node{a[index]}, nil
		}
		return nil, nil
	}
	return nil, fmt.Errorf("cannot index array with %d", q)
}

func (q ArrayQuery) Set(pn *Node, v Node) error {
	index := int(q)
	if en, ok := (*pn).(EditorNode); ok {
		return en.Set(index, v)
	}
	return fmt.Errorf("cannot index array with %d", index)
}

func (q ArrayQuery) Append(pn *Node, v Node) error {
	index := int(q)
	n := *pn
	if en, ok := (*pn).(EditorNode); ok {
		if n.Has(index) {
			x := n.Get(index)
			if x != nil {
				if ex, ok := x.(EditorNode); ok {
					if err := ex.Append(v); err == nil {
						return nil
					}
				}
			}
			return fmt.Errorf("cannot append to array with %d", index)
		}
		return en.Set(index, Array{v})
	}
	return fmt.Errorf("cannot append to array with %d", index)
}

func (q ArrayQuery) Delete(pn *Node) error {
	index := int(q)
	if en, ok := (*pn).(EditorNode); ok {
		if err := en.Delete(index); err == nil {
			return nil
		}
	}
	return fmt.Errorf("cannot delete array with %d", index)
}

func (q ArrayQuery) String() string {
	return fmt.Sprintf("[%d]", q)
}

// ArrayRangeQuery represents a range of the Array that implements methods of the Query.
type ArrayRangeQuery []int

func (q ArrayRangeQuery) Exec(n Node) ([]Node, error) {
	if len(q) != 2 {
		return nil, fmt.Errorf("invalid array range %s", q)
	}
	if a := n.Array(); a != nil {
		from, to := q[0], q[1]
		if from == -1 {
			return a[:to], nil
		} else if q[1] == -1 {
			return a[from:], nil
		}
		return a[from:to], nil
	}
	return nil, fmt.Errorf("cannot index array with range %d:%d", q[0], q[1])
}

func (q ArrayRangeQuery) String() string {
	ss := make([]string, len(q))
	for i, r := range q {
		if r != -1 {
			ss[i] = strconv.Itoa(r)
		}
	}
	return "[" + strings.Join(ss, ":") + "]"
}

// SlurpQuery is a special query that works in FilterQuery.
type SlurpQuery struct{}

// Exec returns the provided node into a single node array.
// FilterQuery calls q.Exec(Array(results)), which has the effect of to slurp
// all the results into a single node array.
func (q SlurpQuery) Exec(n Node) ([]Node, error) {
	return []Node{n}, nil
}

func (q SlurpQuery) String() string {
	return " | "
}

type CountQuery struct{}

func (q CountQuery) Exec(n Node) ([]Node, error) {
	switch n.Type() {
	case TypeArray:
		return ToNodeValues(len(n.Array())), nil
	case TypeMap:
		return ToNodeValues(len(n.Map())), nil
	}
	return ToNodeValues(0), nil
}

func (q CountQuery) String() string {
	return "count()"
}

type KeysQuery struct{}

func (q KeysQuery) Exec(n Node) ([]Node, error) {
	switch n.Type() {
	case TypeArray:
		a := n.Array()
		keys := make(Array, len(a))
		for i := 0; i < len(a); i++ {
			keys[i] = NumberValue(i)
		}
		return []Node{keys}, nil
	case TypeMap:
		strKeys := n.Map().Keys()
		keys := make(Array, len(strKeys))
		for i := range strKeys {
			keys[i] = StringValue(strKeys[i])
		}
		return []Node{keys}, nil
	}
	return nil, nil
}

func (q KeysQuery) String() string {
	return "keys()"
}

type ValuesQuery struct{}

func (q ValuesQuery) Exec(n Node) ([]Node, error) {
	switch n.Type() {
	case TypeArray:
		return []Node{n.Array()}, nil
	case TypeMap:
		m := n.Map()
		keys := m.Keys()
		values := make(Array, len(keys))
		for i, key := range keys {
			values[i] = m[key]
		}
		return []Node{values}, nil
	}
	return nil, nil
}

func (q ValuesQuery) String() string {
	return "values()"
}

// FilterQuery consists of multiple queries that filter the nodes in order.
type FilterQuery []Query

func (qs FilterQuery) Exec(n Node) ([]Node, error) {
	rs := []Node{n}
	for _, q := range qs {
		switch q.(type) {
		case SlurpQuery:
			nrs, err := q.Exec(Array(rs))
			if err != nil {
				return nil, err
			}
			rs = nrs
			continue
		}
		var nrs []Node
		for _, r := range rs {
			if r == nil {
				continue
			}
			nr, err := q.Exec(r)
			if err != nil {
				return nil, err
			}
			nrs = append(nrs, nr...)
		}
		rs = nrs
	}
	return rs, nil
}

func (qs FilterQuery) execForEdit(n Node) ([]Node, error) {
	rs := []Node{n}
	for i, q := range qs[:len(qs)-1] {
		switch q.(type) {
		case SlurpQuery:
			nrs, err := q.Exec(Array(rs))
			if err != nil {
				return nil, err
			}
			rs = nrs
			continue
		}
		var nrs []Node
		for _, r := range rs {
			if r == nil {
				continue
			}
			nr, err := q.Exec(r)
			if err != nil {
				return nil, err
			}
			if len(nr) == 0 {
				var empty Node
				switch qs[i+1].(type) {
				case MapQuery:
					empty = Map{}
				case ArrayQuery:
					empty = Array{}
				}
				if empty != nil {
					if eq, ok := q.(EditorQuery); ok {
						if err = eq.Set(&r, empty); err != nil {
							return nil, err
						}
						if nr, err = eq.Exec(r); err != nil {
							return nil, err
						}
					}
				}
			}
			nrs = append(nrs, nr...)
		}
		rs = nrs
	}
	return rs, nil
}

func (qs FilterQuery) String() string {
	ss := make([]string, len(qs))
	for i, q := range qs {
		ss[i] = q.String()
	}
	return strings.Join(ss, "")
}

// WalkQuery is a key of each nodes that implements methods of the Query.
type WalkQuery string

var _ EditorQuery = (WalkQuery)("")

// Exec walks the specified root node and collects matching nodes using itself as a key.
func (q WalkQuery) Exec(root Node) ([]Node, error) {
	key := string(q)
	var r []Node
	// NOTE: Walk returns no error.
	Walk(root, func(n Node, keys []interface{}) error {
		if n == nil {
			return nil
		}
		if n.Has(key) {
			r = append(r, n.Get(key))
		}
		return nil
	})
	return r, nil
}

func (q WalkQuery) Set(pn *Node, v Node) error {
	key := string(q)
	return Walk(*pn, func(n Node, keys []interface{}) error {
		if n.Has(key) {
			if en, ok := n.(EditorNode); ok {
				en.Set(key, v)
			}
		}
		return nil
	})
}

func (q WalkQuery) Append(pn *Node, v Node) error {
	key := string(q)
	return Walk(*pn, func(n Node, keys []interface{}) error {
		if n.Has(key) {
			if nv := n.Get(key); nv != nil {
				if env, ok := nv.(EditorNode); ok {
					env.Append(v)
				}
			}
		}
		return nil
	})
}

func (q WalkQuery) Delete(pn *Node) error {
	key := string(q)
	return Walk(*pn, func(n Node, keys []interface{}) error {
		if n.Has(key) {
			if en, ok := n.(EditorNode); ok {
				en.Delete(key)
			}
		}
		return nil
	})
}

func (q WalkQuery) String() string {
	return ".." + string(q)
}

// Selector checks if a node is eligible for selection.
type Selector interface {
	Matches(n Node) (bool, error)
	String() string
}

// And represents selectors that combines each selector with and.
type And []Selector

// Matches returns true if all selectors returns true.
func (a And) Matches(n Node) (bool, error) {
	for _, s := range a {
		ok, err := s.Matches(n)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

func (a And) String() string {
	ss := make([]string, len(a))
	for i, s := range a {
		ss[i] = s.String()
	}
	return "(" + strings.Join(ss, " and ") + ")"
}

// Or represents selectors that combines each selector with or.
type Or []Selector

// Matches returns true if anyone returns true.
func (o Or) Matches(n Node) (bool, error) {
	for _, s := range o {
		ok, err := s.Matches(n)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (o Or) String() string {
	ss := make([]string, len(o))
	for i, s := range o {
		ss[i] = s.String()
	}
	return "(" + strings.Join(ss, " or ") + ")"
}

// Evaluator represents a evaluatable selector.
type Evaluator struct {
	Query Query
}

var _ Selector = (*Evaluator)(nil)

func (e Evaluator) Matches(n Node) (bool, error) {
	rs, err := e.Query.Exec(n)
	if err != nil {
		return false, err
	}
	switch len(rs) {
	case 0:
		return false, nil
	case 1:
		r := rs[0]
		switch r.Type() {
		case TypeBoolValue:
			return r.Value().Bool(), nil
		case TypeNumberValue:
			return r.Value().Float64() > 0, nil
		case TypeStringValue:
			return r.Value().String() != "", nil
		case TypeArray:
			return len(r.Array()) > 0, nil
		case TypeMap:
			return len(r.Map()) > 0, nil
		}
		return false, nil
	}
	return true, nil
}

func (e Evaluator) String() string {
	return e.Query.String()
}

// Comparator represents a comparable selector.
type Comparator struct {
	Left  Query
	Op    Operator
	Right Query
}

var _ Selector = (*Comparator)(nil)

// Matches evaluates left and right using the operator. (eg. .id == 0)
func (c Comparator) Matches(n Node) (bool, error) {
	l, err := c.Left.Exec(n)
	if err != nil {
		return false, err
	}
	r, err := c.Right.Exec(n)
	if err != nil {
		return false, err
	}
	var l0, r0 Node
	switch len(l) {
	case 0:
		l0 = nil
	case 1:
		l0 = l[0]
	default:
		return false, fmt.Errorf("%q returns no single value %+v", c.Left, l)
	}
	switch len(r) {
	case 0:
		r0 = nil
	case 1:
		r0 = r[0]
	default:
		return false, fmt.Errorf("%q returns no single value %+v", c.Right, r)
	}
	if l0 == nil || r0 == nil {
		return (l0 == nil && r0 == nil), nil
	}
	return l0.Value().Compare(c.Op, r0.Value()), nil
}

func (c Comparator) String() string {
	return fmt.Sprintf("%s %s %s", c.Left, c.Op, c.Right)
}

// SelectQuery returns nodes that matched by selectors.
type SelectQuery struct {
	Selector
}

func (q SelectQuery) Exec(n Node) ([]Node, error) {
	if a := n.Array(); a != nil {
		if q.Selector == nil {
			return a, nil
		}
		var rs []Node
		for _, nn := range a {
			ok, err := q.Selector.Matches(nn)
			if err != nil {
				return nil, err
			}
			if ok {
				rs = append(rs, nn)
			}
		}
		return rs, nil
	}
	if m := n.Map(); m != nil {
		if q.Selector == nil {
			return m.Values(), nil
		}
		var rs []Node
		for _, nn := range m.Values() {
			ok, err := q.Selector.Matches(nn)
			if err != nil {
				return nil, err
			}
			if ok {
				rs = append(rs, nn)
			}
		}
		return rs, nil
	}
	return nil, nil
}

func (q SelectQuery) String() string {
	if q.Selector == nil {
		return "[]"
	}
	return "[" + q.Selector.String() + "]"
}

var (
	_ Selector = (And)(nil)
	_ Selector = (Or)(nil)
	_ Selector = (*Comparator)(nil)
	_ Selector = (*SelectQuery)(nil)
)

// ParseQuery parses the provided expr to a Query.
// See https://github.com/jarxorg/tree#Query
func ParseQuery(expr string) (Query, error) {
	token, err := tokenizeQuery(expr)
	if err != nil {
		return nil, err
	}
	return tokenToQuery(token, expr)
}

type token struct {
	cmd      string
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

var tokenRegexp = regexp.MustCompile(`"([^"]*)"|(and|or|==|<=|>=|!=|~=|\.\.|[\.\[\]\(\)\|<>:]|[a-z]+\(\))|(\w+)`)

// tokenizeQuery parses a query expression string into a token tree.
// Uses regular expressions to identify different token types.
func tokenizeQuery(expr string) (*token, error) {
	current := &token{}
	ms := tokenRegexp.FindAllStringSubmatch(expr, -1)
	for _, m := range ms {
		quoted := m[1]
		cmd := m[2]
		word := m[3]
		// NOTE: detect node name
		if quoted != "" || word != "" {
			value := quoted
			if value == "" {
				value = word
			}
			var lastChild *token
			if len(current.children) > 0 {
				lastChild = current.children[len(current.children)-1]
			}
			if lastChild != nil && (lastChild.cmd == "." || lastChild.cmd == "..") {
				lastChild.value = value
				lastChild.quoted = quoted != ""
				continue
			}
			t := &token{value: value, quoted: quoted != ""}
			current.children = append(current.children, t)
			continue
		}
		// NOTE: detect keywords
		t := &token{cmd: cmd, parent: current}
		switch cmd {
		case "]", ")":
			if (cmd == "]" && current.cmd != "[") || (cmd == ")" && current.cmd != "(") {
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
	case "count()":
		return CountQuery{}, nil
	case "keys()":
		return KeysQuery{}, nil
	case "values()":
		return ValuesQuery{}, nil
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

// Find finds a node from n using the Query.
func Find(n Node, expr string) ([]Node, error) {
	if n.IsNil() {
		return nil, nil
	}
	q, err := ParseQuery(expr)
	if err != nil {
		return nil, err
	}
	return q.Exec(n)
}

type arrayHolder struct{ a *Array }

func (h *arrayHolder) IsNil() bool                                       { return h.a.IsNil() }
func (h *arrayHolder) Type() Type                                        { return h.a.Type() }
func (h *arrayHolder) Array() Array                                      { return *h.a }
func (h *arrayHolder) Map() Map                                          { return h.a.Map() }
func (h *arrayHolder) Value() Value                                      { return h.a.Value() }
func (h *arrayHolder) Has(keys ...interface{}) bool                      { return h.a.Has(keys...) }
func (h *arrayHolder) Get(keys ...interface{}) Node                      { return h.a.Get(keys...) }
func (h *arrayHolder) Each(cb func(key interface{}, v Node) error) error { return h.a.Each(cb) }
func (h *arrayHolder) Find(expr string) ([]Node, error)                  { return h.a.Find(expr) }
func (h *arrayHolder) Delete(key interface{}) error                      { return h.a.Delete(key) }
func (h *arrayHolder) Append(v Node) error                               { return h.a.Append(*holdArray(&v)) }
func (h *arrayHolder) Set(key interface{}, v Node) error                 { return h.a.Set(key, *holdArray(&v)) }

var _ EditorNode = (*arrayHolder)(nil)

// holdArray wraps Array nodes with arrayHolder for edit operations.
// Recursively processes nested structures.
func holdArray(pn *Node) *Node {
	n := *pn
	if a := n.Array(); a != nil {
		ah := &arrayHolder{&a}
		*pn = ah
		for i, nn := range a {
			if nn != nil {
				holdArray(&nn)
				a[i] = nn
			}
		}
	} else if m := n.Map(); m != nil {
		for key, nn := range m {
			if nn != nil {
				holdArray(&nn)
				m[key] = nn
			}
		}
	}
	return pn
}

// unholdArray unwraps arrayHolder nodes back to regular Array nodes.
// Recursively processes nested structures.
func unholdArray(pn *Node) {
	n := *pn
	if a := n.Array(); a != nil {
		if ah, ok := n.(*arrayHolder); ok {
			a = *ah.a
			*pn = a
		}
		for i, nn := range a {
			if nn != nil {
				unholdArray(&nn)
				a[i] = nn
			}
		}
	} else if m := n.Map(); m != nil {
		for key, nn := range m {
			if nn != nil {
				unholdArray(&nn)
				m[key] = nn
			}
		}
	}
}

var editRegexp = regexp.MustCompile(`^([^\+]+) ?((=|\+=) ?(.+)|(\^\?))$`)

func Edit(pn *Node, expr string) error {
	ms := editRegexp.FindStringSubmatch(expr)
	if len(ms) != 6 {
		return fmt.Errorf("syntax error: invalid edit expression %q, %v", expr, ms)
	}
	left, op, right := ms[1], ms[3], ms[4]
	if op == "" {
		op = ms[5]
	}

	var v Node
	if right != "" {
		var err error
		v, err = UnmarshalJSON([]byte(right))
		if err != nil {
			return err
		}
	}
	q, err := ParseQuery(left)
	if err != nil {
		return err
	}

	holdArray(pn)
	defer unholdArray(pn)

	return editQuery(pn, q, op, v)
}

// editQuery applies an edit operation to a node using the specified query.
// Supports FilterQuery and EditorQuery types.
func editQuery(pn *Node, q Query, op string, v Node) error {
	switch tq := q.(type) {
	case FilterQuery:
		return execForEdit(pn, tq, op, v)
	case EditorQuery:
		return execEdit(pn, tq, op, v)
	}
	return fmt.Errorf("syntax error: unsupported edit query: %s", q)
}

// execForEdit executes a FilterQuery for edit operations.
// Handles multi-step queries for complex edit paths.
func execForEdit(pn *Node, fq FilterQuery, op string, v Node) error {
	l := len(fq)
	if l == 0 {
		return nil
	}

	nn := []Node{*pn}
	if l > 1 {
		var err error
		nn, err = fq.execForEdit(*pn)
		if err != nil {
			return err
		}
	}

	q := fq[l-1]
	for _, n := range nn {
		if err := editQuery(&n, q, op, v); err != nil {
			return err
		}
	}
	return nil
}

// execEdit executes an EditorQuery with the specified operation.
// Supports set (=), append (+=), and delete (^?) operations.
func execEdit(pn *Node, eq EditorQuery, op string, v Node) error {
	switch op {
	case "=":
		return eq.Set(pn, v)
	case "+=":
		return eq.Append(pn, v)
	case "^?":
		return eq.Delete(pn)
	}
	return fmt.Errorf("syntax error: unsupported edit operation %q", op)
}
