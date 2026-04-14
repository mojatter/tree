package tree

import (
	"fmt"
	"regexp"
)

type arrayHolder struct{ a *Array }

func (h *arrayHolder) IsNil() bool                               { return h.a.IsNil() }
func (h *arrayHolder) Type() Type                                { return h.a.Type() }
func (h *arrayHolder) Array() Array                              { return *h.a }
func (h *arrayHolder) Map() Map                                  { return h.a.Map() }
func (h *arrayHolder) Value() Value                              { return h.a.Value() }
func (h *arrayHolder) Has(keys ...any) bool                      { return h.a.Has(keys...) }
func (h *arrayHolder) Get(keys ...any) Node                      { return h.a.Get(keys...) }
func (h *arrayHolder) Each(cb func(key any, v Node) error) error { return h.a.Each(cb) }
func (h *arrayHolder) Find(expr string) ([]Node, error)          { return h.a.Find(expr) }
func (h *arrayHolder) Delete(key any) error                      { return h.a.Delete(key) }
func (h *arrayHolder) Append(v Node) error                       { return h.a.Append(*holdArray(&v)) }
func (h *arrayHolder) Set(key any, v Node) error                 { return h.a.Set(key, *holdArray(&v)) }

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

// Edit parses the edit expression and applies it to the node tree.
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
		v, err = UnmarshalYAML([]byte(right))
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
