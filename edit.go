package tree

import (
	"fmt"
	"regexp"
	"strconv"
)

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
// For multi-step queries it uses resolveAndEdit for recursive descent
// with write-back, so that mutations to Array nodes (value types)
// propagate back to their parents without wrapping the entire tree.
func execForEdit(pn *Node, fq FilterQuery, op string, v Node) error {
	l := len(fq)
	if l == 0 {
		return nil
	}
	if l == 1 {
		return editQuery(pn, fq[0], op, v)
	}

	// Split at SlurpQuery (pipe) if present among intermediate steps.
	for i, q := range fq[:l-1] {
		if _, ok := q.(SlurpQuery); ok {
			left := FilterQuery(fq[:i])
			right := FilterQuery(fq[i+1:])

			var results []Node
			if len(left) > 0 {
				var err error
				results, err = left.Exec(*pn)
				if err != nil {
					return err
				}
			} else {
				results = []Node{*pn}
			}

			var slurped Node = Array(results)
			return execForEdit(&slurped, right, op, v)
		}
	}

	return resolveAndEdit(pn, fq, 0, op, v)
}

// resolveAndEdit walks the FilterQuery path step by step, writing back
// modified Array nodes at each level so changes propagate to the root.
func resolveAndEdit(pn *Node, fq FilterQuery, step int, op string, v Node) error {
	if step >= len(fq)-1 {
		return editQuery(pn, fq[len(fq)-1], op, v)
	}

	q := fq[step]
	n := *pn

	switch tq := q.(type) {
	case MapQuery:
		return resolveMapStep(pn, n, string(tq), fq, step, op, v)

	case ArrayQuery:
		index := int(tq)
		if a := n.Array(); a != nil {
			index = tq.resolveIndex(len(a))
		}
		return resolveArrayStep(pn, n, index, fq, step, op, v)

	case WalkQuery:
		key := string(tq)
		return Walk(n, func(wn Node, keys []any) error {
			if wn.Has(key) {
				if en, ok := wn.(EditorNode); ok {
					child := wn.Get(key)

					if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
						return err
					}

					return en.Set(key, child)
				}
			}
			return nil
		})

	case SelectQuery:
		return resolveSelectStep(pn, n, tq, fq, step, op, v)

	default:
		// Fallback: use Exec (no write-back, works for reference types).
		results, err := q.Exec(n)
		if err != nil {
			return err
		}
		for _, r := range results {
			child := r

			if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
				return err
			}
		}
		return nil
	}
}

// resolveMapStep handles a MapQuery intermediate step with write-back.
func resolveMapStep(pn *Node, n Node, key string, fq FilterQuery, step int, op string, v Node) error {
	if m := n.Map(); m != nil {
		child, exists := m[key]
		if !exists || child == nil {
			child = emptyIntermediate(fq[step+1])
			if child == nil {
				return nil
			}
			m[key] = child
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
			return err
		}

		m[key] = child
		return nil
	}

	if a := n.Array(); a != nil {
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 {
			return nil
		}
		if i >= len(a) {
			na := make(Array, i+1)
			copy(na, a)
			a = na
		}
		child := a[i]
		if child == nil {
			child = emptyIntermediate(fq[step+1])
			if child == nil {
				return nil
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
			return err
		}

		a[i] = child
		*pn = a
		return nil
	}

	return nil
}

// resolveArrayStep handles an ArrayQuery intermediate step with write-back.
func resolveArrayStep(pn *Node, n Node, index int, fq FilterQuery, step int, op string, v Node) error {
	if en, ok := n.(EditorNode); ok {
		child := n.Get(index)
		if child.IsNil() {
			child = emptyIntermediate(fq[step+1])
			if child == nil {
				return nil
			}
			if err := en.Set(index, child); err != nil {
				return err
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
			return err
		}

		return en.Set(index, child)
	}

	if a := n.Array(); a != nil {
		if index >= len(a) {
			na := make(Array, index+1)
			copy(na, a)
			a = na
		}
		child := a[index]
		if child == nil {
			child = emptyIntermediate(fq[step+1])
			if child == nil {
				return nil
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
			return err
		}

		a[index] = child
		*pn = a
		return nil
	}

	return nil
}

// resolveSelectStep handles a SelectQuery intermediate step with write-back.
func resolveSelectStep(pn *Node, n Node, sq SelectQuery, fq FilterQuery, step int, op string, v Node) error {
	if a := n.Array(); a != nil {
		for i, elem := range a {
			if sq.Selector != nil {
				ok, err := sq.Matches(elem)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
			}
			child := elem

			if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
				return err
			}

			a[i] = child
		}
		*pn = a
		return nil
	}

	if m := n.Map(); m != nil {
		for _, k := range m.Keys() {
			elem := m[k]
			if sq.Selector != nil {
				ok, err := sq.Matches(elem)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
			}
			child := elem

			if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
				return err
			}

			m[k] = child
		}
		return nil
	}

	return nil
}

// emptyIntermediate returns an empty node appropriate for the next
// query step, or nil if no intermediate creation is needed.
func emptyIntermediate(nextQuery Query) Node {
	switch nextQuery.(type) {
	case MapQuery:
		return Map{}
	case ArrayQuery:
		return Array{}
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
