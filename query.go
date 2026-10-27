package tree

import (
	"fmt"
	"slices"
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
	if isNullLike(*pn) {
		*pn = Array{v} // null reads as missing, as for a key
		return nil
	}
	if a := orNil(*pn).Array(); a != nil {
		*pn = append(a, v)
		return nil
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

// Exec returns the constant value, with a Go nil as null.
func (q ValueQuery) Exec(n Node) ([]Node, error) {
	return []Node{orNil(q.Node)}, nil
}

func (q ValueQuery) String() string {
	if q.Node != nil && q.Node.Type().IsStringValue() {
		return quoteString(q.Node.Value().String())
	}
	s, _ := MarshalJSON(q.Node)
	return string(s)
}

// MapQuery is a key of the Map that implements methods of the Query.
type MapQuery string

var _ EditorQuery = (MapQuery)("")

// Exec returns the value at the key, or null when n is null, a scalar or lacks the key.
// An array reads a numeric key as an index, as Array.Get does.
func (q MapQuery) Exec(n Node) ([]Node, error) {
	return []Node{orNil(n.Get(string(q)))}, nil
}

func (q MapQuery) Set(pn *Node, v Node) error {
	n := orNil(*pn)
	key := string(q)
	if en, ok := n.(EditorNode); ok {
		return en.Set(key, v)
	}
	if a := n.Array(); a != nil {
		if err := a.Set(key, v); err != nil {
			return err
		}

		*pn = a
		return nil
	}
	return fmt.Errorf("cannot index array with %q", key)
}

func (q MapQuery) Append(pn *Node, v Node) error {
	n := orNil(*pn)
	key := string(q)
	if en, ok := n.(EditorNode); ok {
		if n.Has(key) && !isNullLike(n.Get(key)) {
			if ca := n.Get(key).Array(); ca != nil {
				return en.Set(key, append(ca, v))
			}
			return fmt.Errorf("cannot append to %q", key)
		}
		return en.Set(key, Array{v})
	}
	if n.Array() != nil {
		if i, err := strconv.Atoi(key); err == nil && i >= 0 {
			return ArrayQuery(i).Append(pn, v) // a numeric key appends at that index, as Set does
		}
	}
	return fmt.Errorf("cannot append to %q", key)
}

func (q MapQuery) Delete(pn *Node) error {
	n := orNil(*pn)
	key := string(q)
	if en, ok := n.(EditorNode); ok {
		if err := en.Delete(key); err == nil {
			return nil
		}
	}
	if a := n.Array(); a != nil {
		if err := a.Delete(key); err != nil {
			return fmt.Errorf("cannot delete %q", key)
		}

		*pn = a
		return nil
	}
	return fmt.Errorf("cannot delete %q", key)
}

func (q MapQuery) String() string {
	return formatKey(".", string(q))
}

func formatKey(prefix, key string) string {
	if isIdent(key) {
		return prefix + key
	}
	return prefix + quoteString(key)
}

var quoteReplacer = strings.NewReplacer(`"`, `\"`, `\`, `\\`)

// quoteString double-quotes s as the lexer reads it back: only '"' and '\' are escaped.
func quoteString(s string) string {
	return `"` + quoteReplacer.Replace(s) + `"`
}

func isIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isWord(s[i]) {
			return false
		}
	}
	return true
}

// ArrayQuery is an index of the Array that implements methods of the Query.
type ArrayQuery int

var _ EditorQuery = (ArrayQuery)(0)

// resolveIndex returns the absolute index for q against an array of length n.
// A non-negative index passes through unchanged. A negative index is resolved
// jq-style: -1 maps to the last element. The result may still be out of range
// (negative or >= n); callers handle that.
func (q ArrayQuery) resolveIndex(n int) int {
	i := int(q)
	if i < 0 {
		return n + i
	}
	return i
}

// Exec returns the element at the index, or null when it is out of range or n is not an array (a map reads the key "0").
func (q ArrayQuery) Exec(n Node) ([]Node, error) {
	if a := n.Array(); a != nil {
		index := q.resolveIndex(len(a))
		if index >= 0 && index < len(a) {
			return []Node{orNil(a[index])}, nil
		}
		return []Node{Nil}, nil
	}
	return []Node{orNil(n.Get(int(q)))}, nil
}

func (q ArrayQuery) Set(pn *Node, v Node) error {
	n := orNil(*pn)
	index := int(q)
	if a := n.Array(); a != nil {
		if index = q.resolveIndex(len(a)); index < 0 {
			return fmt.Errorf("cannot index array with %d", int(q))
		}
	}
	if en, ok := n.(EditorNode); ok {
		return en.Set(index, v)
	}
	if a := n.Array(); a != nil {
		if err := a.Set(index, v); err != nil {
			return err
		}

		*pn = a
		return nil
	}
	return fmt.Errorf("cannot index array with %d", int(q))
}

func (q ArrayQuery) Append(pn *Node, v Node) error {
	n := orNil(*pn)
	index := int(q)
	if a := n.Array(); a != nil {
		if index = q.resolveIndex(len(a)); index < 0 {
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
	}
	if en, ok := n.(EditorNode); ok {
		if n.Has(index) && !isNullLike(n.Get(index)) {
			if ca := n.Get(index).Array(); ca != nil {
				return en.Set(index, append(ca, v))
			}
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
		return en.Set(index, Array{v})
	}
	if a := n.Array(); a != nil {
		if n.Has(index) && !isNullLike(a[index]) {
			if ca := a[index].Array(); ca != nil {
				a[index] = append(ca, v)
				*pn = a
				return nil
			}
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
		if index >= len(a) {
			var err error
			if a, err = growArray(a, index); err != nil {
				return err
			}
		}
		a[index] = Array{v}
		*pn = a
		return nil
	}
	return fmt.Errorf("cannot append to array with %d", int(q))
}

func (q ArrayQuery) Delete(pn *Node) error {
	n := orNil(*pn)
	index := int(q)
	if a := n.Array(); a != nil {
		if index = q.resolveIndex(len(a)); index < 0 {
			return fmt.Errorf("cannot delete array with %d", int(q))
		}
	}
	if en, ok := n.(EditorNode); ok {
		if err := en.Delete(index); err == nil {
			return nil
		}
	}
	if a := n.Array(); a != nil {
		if err := a.Delete(index); err != nil {
			return fmt.Errorf("cannot delete array with %d", int(q))
		}

		*pn = a
		return nil
	}
	return fmt.Errorf("cannot delete array with %d", int(q))
}

func (q ArrayQuery) String() string {
	return fmt.Sprintf("[%d]", q)
}

// ArrayRangeQuery represents a range of the Array that implements methods of
// the Query. From/To use *int so that omitted bounds (`[1:]`, `[:5]`) are
// represented as nil rather than a magic sentinel.
type ArrayRangeQuery struct {
	From *int
	To   *int
}

// Exec returns the elements in range, or nothing when n is not an array.
func (q ArrayRangeQuery) Exec(n Node) ([]Node, error) {
	if a := n.Array(); a != nil {
		from := resolveRangeBound(q.From, len(a), 0)
		to := resolveRangeBound(q.To, len(a), len(a))
		if from >= to {
			return nil, nil
		}
		return orNilAll(a[from:to]), nil
	}
	return nil, nil
}

// resolveRangeBound resolves a *int bound for ArrayRangeQuery.Exec against
// an array of length n. Nil falls back to defaultV. Negative bounds are
// interpreted Python-style (end-relative: `-1` is the last element). The
// result is clamped to [0, n].
func resolveRangeBound(bound *int, n, defaultV int) int {
	if bound == nil {
		return defaultV
	}
	v := *bound
	if v < 0 {
		v += n
	}
	if v < 0 {
		return 0
	}
	if v > n {
		return n
	}
	return v
}

func (q ArrayRangeQuery) String() string {
	f := ""
	if q.From != nil {
		f = strconv.Itoa(*q.From)
	}
	t := ""
	if q.To != nil {
		t = strconv.Itoa(*q.To)
	}
	return "[" + f + ":" + t + "]"
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
			nr, err := q.Exec(orNil(r))
			if err != nil {
				return nil, err
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
	for i := range len(ss) - 1 {
		if isNopQuery(qs[i]) && strings.HasPrefix(ss[i+1], ".") {
			ss[i] = "" // a bare . before .b would print ..b, which parses as a walk
		}
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
	_ = Walk(root, func(n Node, keys []any) error {
		if n == nil {
			return nil
		}
		if n.Has(key) {
			r = append(r, orNil(n.Get(key)))
		}
		return nil
	})
	return r, nil
}

// walkEdit calls fn on each editable node holding the key, descendants first, skipping values this edit has stored.
func (q WalkQuery) walkEdit(n Node, as assigned, fn func(en EditorNode) error) error {
	if isGoNil(n) || as.has(n) {
		return nil
	}
	if err := n.Each(func(key any, c Node) error {
		if key == nil {
			return nil
		}
		return q.walkEdit(c, as, fn)
	}); err != nil {
		return err
	}
	if en, ok := n.(EditorNode); ok && n.Has(string(q)) {
		return fn(en)
	}
	return nil
}

// edit applies op at every match, descendants first, recording stored copies in as and stopping at the first error.
func (q WalkQuery) edit(pn *Node, op string, v Node, as assigned) error {
	key := string(q)
	return q.walkEdit(*pn, as, func(en EditorNode) error {
		switch op {
		case "=":
			c := cloneEditValue(v)
			as.add(c)
			return en.Set(key, c)
		case "+=":
			child := en.Get(key)
			if isNullLike(child) {
				c := cloneEditValue(v)
				as.add(c)
				return en.Set(key, Array{c})
			}
			ca := child.Array()
			if ca == nil {
				return nil
			}
			c := cloneEditValue(v)
			as.add(c)
			return en.Set(key, append(ca, c))
		case "^?":
			return en.Delete(key)
		}
		return fmt.Errorf("syntax error: unsupported edit operation %q", op)
	})
}

// Set stores a copy of v (for built-in containers) at every match, descendants first, stopping at the first error.
func (q WalkQuery) Set(pn *Node, v Node) error {
	as := assigned{}
	as.add(v) // never edit inside v itself when it is part of the document
	return q.edit(pn, "=", v, as)
}

// Append appends a copy of v (for built-in containers) to every matched array, descendants first, stopping at the first error.
func (q WalkQuery) Append(pn *Node, v Node) error {
	as := assigned{}
	as.add(v) // never edit inside v itself when it is part of the document
	return q.edit(pn, "+=", v, as)
}

// Delete removes the key at every match, descendants first, stopping at the first error.
func (q WalkQuery) Delete(pn *Node) error {
	return q.edit(pn, "^?", nil, nil)
}

func (q WalkQuery) String() string {
	return formatKey("..", string(q))
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
	return slices.ContainsFunc(rs, truthy), nil
}

// truthy reports whether n counts as true in a selector.
func truthy(n Node) bool {
	r := orNil(n)
	switch r.Type() {
	case TypeBoolValue:
		return r.Value().Bool()
	case TypeNumberValue:
		return r.Value().Float64() > 0
	case TypeStringValue:
		return r.Value().String() != ""
	case TypeArray:
		return len(r.Array()) > 0
	case TypeMap:
		return len(r.Map()) > 0
	}
	return false
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
// A side that yields nothing matches nothing, as in jq; a null side compares as null.
// A container side compares by Equal for == and != and never matches an ordered operator.
func (c Comparator) Matches(n Node) (bool, error) {
	l, err := c.Left.Exec(n)
	if err != nil {
		return false, err
	}
	r, err := c.Right.Exec(n)
	if err != nil {
		return false, err
	}
	if len(l) > 1 {
		return false, fmt.Errorf("%s returns more than one value %s", c.Left, formatResults(l))
	}
	if len(r) > 1 {
		return false, fmt.Errorf("%s returns more than one value %s", c.Right, formatResults(r))
	}
	if len(l) == 0 || len(r) == 0 {
		return false, nil
	}
	ln, rn := orNil(l[0]), orNil(r[0])
	if ln.Type().IsValue() && rn.Type().IsValue() {
		return ln.Value().Compare(c.Op, rn.Value()), nil
	}
	switch c.Op {
	case EQ:
		return Equal(ln, rn), nil
	case NE:
		return !Equal(ln, rn), nil
	}
	return false, nil
}

// formatResults formats query results for an error message, with null as null.
func formatResults(ns []Node) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		if isNull(n) {
			s[i] = "null"
		} else {
			s[i] = fmt.Sprintf("%+v", n)
		}
	}
	return "[" + strings.Join(s, " ") + "]"
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
			return orNilAll(a), nil
		}
		var rs []Node
		for _, nn := range a {
			nn = orNil(nn)
			ok, err := q.Matches(nn)
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
			ok, err := q.Matches(nn)
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

// Find runs expr on n, reading a Go nil n as null.
// A key or index step with no value yields null, not nothing.
func Find(n Node, expr string) ([]Node, error) {
	q, err := ParseQuery(expr)
	if err != nil {
		return nil, err
	}
	return q.Exec(orNil(n))
}
