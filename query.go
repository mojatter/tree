package tree

import (
	"fmt"
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
	if a := (*pn).Array(); a != nil {
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
	if a := (*pn).Array(); a != nil {
		if err := a.Set(key, v); err != nil {
			return err
		}

		*pn = a
		return nil
	}
	return fmt.Errorf("cannot index array with %q", key)
}

func (q MapQuery) Append(pn *Node, v Node) error {
	n := *pn
	key := string(q)
	if en, ok := n.(EditorNode); ok {
		if n.Has(key) {
			if ca := n.Get(key).Array(); ca != nil {
				return en.Set(key, append(ca, v))
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
	if a := (*pn).Array(); a != nil {
		if err := a.Delete(key); err != nil {
			return fmt.Errorf("cannot delete %q", key)
		}

		*pn = a
		return nil
	}
	return fmt.Errorf("cannot delete %q", key)
}

func (q MapQuery) String() string {
	return "." + string(q)
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

func (q ArrayQuery) Exec(n Node) ([]Node, error) {
	if a := n.Array(); a != nil {
		index := q.resolveIndex(len(a))
		if index >= 0 && index < len(a) {
			return []Node{a[index]}, nil
		}
		return nil, nil
	}
	return nil, fmt.Errorf("cannot index array with %d", int(q))
}

func (q ArrayQuery) Set(pn *Node, v Node) error {
	n := *pn
	index := int(q)
	if a := n.Array(); a != nil {
		index = q.resolveIndex(len(a))
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
	n := *pn
	index := int(q)
	if a := n.Array(); a != nil {
		index = q.resolveIndex(len(a))
	}
	if en, ok := n.(EditorNode); ok {
		if n.Has(index) {
			if ca := n.Get(index).Array(); ca != nil {
				return en.Set(index, append(ca, v))
			}
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
		if index < 0 {
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
		return en.Set(index, Array{v})
	}
	if a := n.Array(); a != nil {
		if n.Has(index) {
			if ca := a[index].Array(); ca != nil {
				a[index] = append(ca, v)
				*pn = a
				return nil
			}
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
		if index < 0 {
			return fmt.Errorf("cannot append to array with %d", int(q))
		}
		na := make(Array, index+1)
		copy(na, a)
		na[index] = Array{v}
		*pn = na
		return nil
	}
	return fmt.Errorf("cannot append to array with %d", int(q))
}

func (q ArrayQuery) Delete(pn *Node) error {
	n := *pn
	index := int(q)
	if a := n.Array(); a != nil {
		index = q.resolveIndex(len(a))
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

func (q ArrayRangeQuery) Exec(n Node) ([]Node, error) {
	if a := n.Array(); a != nil {
		from := 0
		if q.From != nil {
			from = *q.From
		}
		to := len(a)
		if q.To != nil {
			to = *q.To
		}
		return a[from:to], nil
	}
	return nil, fmt.Errorf("cannot index array with range %s", q)
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
	_ = Walk(root, func(n Node, keys []any) error {
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
	return Walk(*pn, func(n Node, keys []any) error {
		if n.Has(key) {
			if en, ok := n.(EditorNode); ok {
				_ = en.Set(key, v)
			}
		}
		return nil
	})
}

func (q WalkQuery) Append(pn *Node, v Node) error {
	key := string(q)
	return Walk(*pn, func(n Node, keys []any) error {
		if n.Has(key) {
			if ca := n.Get(key).Array(); ca != nil {
				if en, ok := n.(EditorNode); ok {
					_ = en.Set(key, append(ca, v))
				}
			}
		}
		return nil
	})
}

func (q WalkQuery) Delete(pn *Node) error {
	key := string(q)
	return Walk(*pn, func(n Node, keys []any) error {
		if n.Has(key) {
			if en, ok := n.(EditorNode); ok {
				_ = en.Delete(key)
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

