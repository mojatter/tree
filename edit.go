package tree

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// editOps lists the edit operators.
var editOps = []string{"+=", "^?", "="}

// Edit parses the edit expression and applies it to the node tree.
// The RHS after "=" or "+=" is parsed as YAML.
func Edit(pn *Node, expr string) error {
	left, op, right, ok := splitEdit(expr)
	if !ok {
		return fmt.Errorf("syntax error: invalid edit expression %q", expr)
	}

	var v Node
	if op != "^?" {
		var err error
		v, err = parseEditValue(right)
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

// parseEditValue parses right as one YAML document and rejects anything after it.
func parseEditValue(right string) (Node, error) {
	dec := yaml.NewDecoder(strings.NewReader(right))
	v, err := DecodeYAML(dec)
	if errors.Is(err, io.EOF) {
		return ToNode(nil), nil // comment-only value
	}
	if err != nil {
		return nil, err
	}
	var rest any
	if err := dec.Decode(&rest); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("syntax error: multiple YAML documents in edit value %q", right)
	}
	return v, nil
}

// splitEdit splits expr at its first edit operator outside quotes and method arguments.
func splitEdit(expr string) (left, op, right string, ok bool) {
	for i := 0; i < len(expr); {
		switch expr[i] {
		case '"', '\'':
			i = skipQuoted(expr, i)
			continue
		case '(':
			i = skipParen(expr, i)
			continue
		case '+':
			if i+1 >= len(expr) || expr[i+1] != '=' {
				return "", "", "", false // bare '+' on the LHS is a mistyped append, not a key
			}
		case '=', '<', '>', '!', '~':
			if i+1 < len(expr) && expr[i+1] == '=' {
				i += 2 // comparison operator, not an assignment
				continue
			}
		}
		for _, o := range editOps {
			if strings.HasPrefix(expr[i:], o) {
				// Keep the RHS verbatim apart from leading blanks so YAML block scalars survive.
				left, right = strings.TrimSpace(expr[:i]), strings.TrimLeft(expr[i+len(o):], " \t")
				if strings.TrimSpace(right) == "" {
					right = ""
				}
				if left == "" || (o == "^?") != (right == "") {
					return "", "", "", false
				}
				return left, o, right, true
			}
		}
		i++
	}
	return "", "", "", false
}

// skipParen returns the position past the first unquoted ')' after expr[p], or p+1 when unterminated, so '=' inside parentheses never splits the edit.
func skipParen(expr string, p int) int {
	for i := p + 1; i < len(expr); {
		switch expr[i] {
		case '"', '\'':
			i = skipQuoted(expr, i)
			continue
		case ')':
			return i + 1
		}
		i++
	}
	return p + 1
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
	if sameNode(*pn, v) {
		return nil // never edit inside the value assigned by an earlier match
	}
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
			if index = tq.resolveIndex(len(a)); index < 0 {
				return fmt.Errorf("cannot index array with %d", int(tq))
			}
		}
		return resolveArrayStep(pn, n, index, fq, step, op, v)

	case WalkQuery:
		key := string(tq)
		for _, t := range tq.walkTargets(n, v) {
			en, ok := t.live(n)
			if !ok || !en.Has(key) {
				continue
			}
			child := en.Get(key)

			if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
				return err
			}

			if err := en.Set(key, child); err != nil {
				return err
			}
		}
		return nil

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
	if en, ok := n.(EditorNode); ok && !n.IsNil() {
		child := en.Get(key)
		if child.Type().IsNilValue() {
			child = emptyIntermediate(op, fq[step+1])
			if child == nil {
				return nil
			}
			if err := en.Set(key, child); err != nil {
				return err
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v); err != nil {
			return err
		}

		return en.Set(key, child)
	}

	if m := n.Map(); m != nil {
		child, exists := m[key]
		if !exists || child == nil {
			child = emptyIntermediate(op, fq[step+1])
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
			return fmt.Errorf("cannot index array with %v", key)
		}
		if i >= len(a) {
			if a, err = growArray(a, i); err != nil {
				return err
			}
		}
		child := a[i]
		if child == nil {
			child = emptyIntermediate(op, fq[step+1])
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

	if n.IsNil() {
		return nil // null not reached by a map key is skipped (#147)
	}
	return fmt.Errorf("cannot index array with %q", key)
}

// resolveArrayStep handles an ArrayQuery intermediate step with write-back.
func resolveArrayStep(pn *Node, n Node, index int, fq FilterQuery, step int, op string, v Node) error {
	if en, ok := n.(EditorNode); ok {
		child := n.Get(index)
		if child.IsNil() {
			child = emptyIntermediate(op, fq[step+1])
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
			var err error
			if a, err = growArray(a, index); err != nil {
				return err
			}
		}
		child := a[index]
		if child == nil {
			child = emptyIntermediate(op, fq[step+1])
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

	if n.IsNil() {
		return nil // null not reached by a map key is skipped (#147)
	}
	return fmt.Errorf("cannot index array with %d", index)
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
// query step, or nil if none should be created (always for a delete).
func emptyIntermediate(op string, nextQuery Query) Node {
	if op == "^?" {
		return nil
	}
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
