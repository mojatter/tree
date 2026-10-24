package tree

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// editOps lists the edit operators.
var editOps = []string{"+=", "^?", "="}

// Edit parses the edit expression and applies it to the node tree.
// The RHS after "=" or "+=" is parsed as YAML; on error, matches edited before the failing one stay edited.
// A step the edit cannot write through, such as a method other than first() and last(), is a syntax error.
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
	fq, err := editPath(q)
	if err != nil {
		return err
	}
	if isNullLike(*pn) { // a null root is replaced like a null reached by key
		// Probe with "=" so that ^? stops only a query = would create containers for; others leave the root as under a key.
		if c := emptyIntermediate("=", dropNopSteps(fq)); c != nil {
			if op == "^?" {
				return nil // deleting from a null root leaves it as it is
			}
			*pn = c
		}
	}

	return execForEdit(pn, fq, op, v)
}

// indexQuery is a method query that an edit writes through as the array index it returns.
type indexQuery interface {
	Query
	editIndex() ArrayQuery
}

// editStep returns the step an edit applies for q on n: the index of an indexQuery, rejecting a map, which [0] would edit by key "0", and q itself otherwise.
func editStep(n Node, q Query) (Query, error) {
	iq, ok := q.(indexQuery)
	if !ok {
		return q, nil
	}
	if !isNullLike(n) && n.Type().IsMap() {
		return nil, fmt.Errorf("cannot edit %s of a map", q)
	}
	return iq.editIndex(), nil
}

// editPath flattens groups without a pipe and rejects a step the edit cannot write through, before any data is read.
// Steps before the last pipe are only read, so only the steps after it are checked.
func editPath(q Query) (FilterQuery, error) {
	fq := flattenGroups(q)
	seg := dropNopSteps(fq[lastPipe(fq)+1:])
	for i, s := range seg {
		if !canEditStep(s, i == len(seg)-1) {
			return nil, errUnsupportedEditQuery(s)
		}
	}
	return fq, nil
}

// flattenGroups returns q as a FilterQuery with its parenthesised groups without a pipe spliced in.
func flattenGroups(q Query) FilterQuery {
	fq, ok := q.(FilterQuery)
	if !ok {
		return FilterQuery{q}
	}
	flat := make(FilterQuery, 0, len(fq))
	for _, s := range fq {
		if g, ok := s.(FilterQuery); ok && lastPipe(g) < 0 {
			flat = append(flat, flattenGroups(g)...)
			continue
		}
		flat = append(flat, s)
	}
	return flat
}

// lastPipe returns the index of the last pipe in fq, or -1.
func lastPipe(fq FilterQuery) int {
	for i, q := range slices.Backward(fq) {
		if _, ok := q.(SlurpQuery); ok {
			return i
		}
	}
	return -1
}

// canEditStep reports whether an edit can write through q, as the last step when last is true.
func canEditStep(q Query, last bool) bool {
	switch q.(type) {
	case MapQuery, ArrayQuery, WalkQuery, indexQuery:
		return true
	case SelectQuery, ArrayRangeQuery:
		return !last
	}
	_, ok := q.(EditorQuery)
	return last && ok
}

// errUnsupportedEditQuery reports a step an edit cannot write through, with a group in parentheses.
func errUnsupportedEditQuery(q Query) error {
	if vq, ok := q.(ValueQuery); ok && isNull(vq.Node) {
		return errors.New("syntax error: unsupported edit query: ()") // an empty group parses as a null literal
	}
	if _, ok := q.(FilterQuery); ok {
		return fmt.Errorf("syntax error: unsupported edit query: (%s)", q)
	}
	return fmt.Errorf("syntax error: unsupported edit query: %s", q)
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
		default:
			if isWord(expr[i]) {
				i = skipWord(expr, i)
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

// skipParen returns the position past the first unquoted ')' after the grouping '(' at expr[p], or p+1 when unterminated, so '=' inside parentheses never splits the edit.
func skipParen(expr string, p int) int {
	for i := p + 1; i < len(expr); {
		switch expr[i] {
		case '"', '\'':
			i = skipQuoted(expr, i)
			continue
		case ')':
			return i + 1
		}
		if isWord(expr[i]) {
			i = skipWord(expr, i)
			continue
		}
		i++
	}
	return p + 1
}

// skipWord returns the position past the keyword, method call or identifier the lexer reads at expr[p] (an isWord byte).
func skipWord(expr string, p int) int {
	afterDot := p > 0 && expr[p-1] == '.'
	if kw := matchOp(expr, p, afterDot); kw != "" {
		return p + len(kw)
	}
	l := lexer{expr: expr, pos: p}
	if isLower(expr[p]) {
		if _, ok := l.scanMethod(); ok {
			return l.pos
		}
	}
	l.scanIdent(afterDot)
	return l.pos
}

// editQuery applies an edit operation to a node using the specified query.
// Supports FilterQuery, EditorQuery and indexQuery types.
func editQuery(pn *Node, q Query, op string, v Node) error {
	q, err := editStep(orNil(*pn), q)
	if err != nil {
		return err
	}
	if aq, ok := q.(ArrayQuery); ok && aq < 0 && isNullLike(*pn) {
		return nil // a null root has no element to count from the end and stays as it is, as under a key
	}
	switch tq := q.(type) {
	case FilterQuery:
		return execForEdit(pn, tq, op, v)
	case EditorQuery:
		return execEdit(pn, tq, op, v)
	}
	return errUnsupportedEditQuery(q)
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
	// Split at SlurpQuery (pipe) if present among intermediate steps; pipes first, so a lone . next to one is kept.
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

	if fq = dropNopSteps(fq); len(fq) == 1 {
		return editQuery(pn, fq[0], op, v)
	}

	as := newAssigned(fq)
	as.add(v) // never edit inside v itself when it is part of the document

	return resolveAndEdit(pn, fq, 0, op, v, as)
}

// dropNopSteps removes bare . and .. steps, which pass the node through, keeping one when nothing else is left.
func dropNopSteps(fq FilterQuery) FilterQuery {
	kept := make(FilterQuery, 0, len(fq))
	for i, q := range fq {
		if !isNopQuery(q) {
			kept = append(kept, q)
			continue
		}
		if i == len(fq)-1 && len(kept) > 0 {
			if !canEditStep(kept[len(kept)-1], true) {
				kept = append(kept, q) // a trailing . after a step without write-back, such as [] or a slice, is the editable last step
			}
		}
	}
	if len(kept) == 0 {
		return FilterQuery{NopQuery{}}
	}
	return kept
}

// isNopQuery reports whether q is a bare . or .., by value or pointer.
func isNopQuery(q Query) bool {
	switch q.(type) {
	case NopQuery, *NopQuery:
		return true
	}
	return false
}

// resolveAndEdit walks the FilterQuery path step by step, writing back
// modified Array nodes at each level so changes propagate to the root.
func resolveAndEdit(pn *Node, fq FilterQuery, step int, op string, v Node, as assigned) error {
	if as.has(*pn) {
		return nil // never edit inside a value this edit has stored
	}
	if step >= len(fq)-1 {
		return editLast(pn, fq[len(fq)-1], op, v, as)
	}

	n := orNil(*pn)
	q, err := editStep(n, fq[step])
	if err != nil {
		return err
	}

	switch tq := q.(type) {
	case MapQuery:
		return resolveMapStep(pn, n, string(tq), fq, step, op, v, as)

	case ArrayQuery:
		index := int(tq)
		if a := n.Array(); a != nil {
			if index = tq.resolveIndex(len(a)); index < 0 {
				return fmt.Errorf("cannot index array with %d", int(tq))
			}
		}
		return resolveArrayStep(pn, n, index, fq, step, op, v, as)

	case WalkQuery:
		key := string(tq)
		return tq.walkEdit(n, as, func(en EditorNode) error {
			child := en.Get(key)
			if isNullLike(child) {
				if child = emptyIntermediate(op, fq[step+1:]); child == nil {
					return nil // a null match stays as it is
				}
			}

			if err := resolveAndEdit(&child, fq, step+1, op, cloneEditValue(v), as); err != nil {
				return err
			}

			return en.Set(key, child)
		})

	case SelectQuery:
		return resolveSelectStep(pn, n, tq, fq, step, op, v, as)

	case ArrayRangeQuery:
		return resolveRangeStep(pn, n, tq, fq, step, op, v, as)
	}
	return errUnsupportedEditQuery(q)
}

// resolveMapStep handles a MapQuery intermediate step with write-back.
func resolveMapStep(pn *Node, n Node, key string, fq FilterQuery, step int, op string, v Node, as assigned) error {
	if en, ok := n.(EditorNode); ok && !n.IsNil() {
		child := en.Get(key)
		if isNullLike(child) {
			child = emptyIntermediate(op, fq[step+1:])
			if child == nil {
				return nil
			}
			if err := en.Set(key, child); err != nil {
				return err
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
			return err
		}

		return en.Set(key, child)
	}

	if m := n.Map(); m != nil {
		child, exists := m[key]
		if !exists || isNullLike(child) {
			child = emptyIntermediate(op, fq[step+1:])
			if child == nil {
				return nil
			}
			m[key] = child
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
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
		if isNullLike(child) {
			child = emptyIntermediate(op, fq[step+1:])
			if child == nil {
				return nil
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
			return err
		}

		a[i] = child
		*pn = a
		return nil
	}

	if n.IsNil() {
		return nil // a null root the edit creates nothing for stays as it is
	}
	return fmt.Errorf("cannot index array with %q", key)
}

// resolveArrayStep handles an ArrayQuery intermediate step with write-back.
func resolveArrayStep(pn *Node, n Node, index int, fq FilterQuery, step int, op string, v Node, as assigned) error {
	if en, ok := n.(EditorNode); ok && !isNullLike(n) {
		child := n.Get(index)
		if isNull(child) || child.IsNil() { // a typed-nil child is replaced here too, as before
			child = emptyIntermediate(op, fq[step+1:])
			if child == nil {
				return nil
			}
			if err := en.Set(index, child); err != nil {
				return err
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
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
		if isNullLike(child) {
			child = emptyIntermediate(op, fq[step+1:])
			if child == nil {
				return nil
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
			return err
		}

		a[index] = child
		*pn = a
		return nil
	}

	if n.IsNil() {
		return nil // a null root the edit creates nothing for stays as it is
	}
	return fmt.Errorf("cannot index array with %d", index)
}

// resolveSelectStep handles a SelectQuery intermediate step with write-back.
func resolveSelectStep(pn *Node, n Node, sq SelectQuery, fq FilterQuery, step int, op string, v Node, as assigned) error {
	if a := n.Array(); a != nil {
		var match func(Node) (bool, error)
		if sq.Selector != nil {
			match = sq.Matches
		}
		return editElements(pn, n, a, 0, len(a), match, fq, step, op, v, as)
	}

	if m := n.Map(); m != nil {
		for _, k := range m.Keys() {
			elem := orNil(m[k])
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
			if isNullLike(child) {
				if child = emptyIntermediate(op, fq[step+1:]); child == nil {
					continue // a null element stays as it is
				}
			}

			if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
				return err
			}

			m[k] = child
		}
		return nil
	}

	return nil
}

// resolveRangeStep handles an ArrayRangeQuery intermediate step with write-back, like [i] for each index in range.
func resolveRangeStep(pn *Node, n Node, rq ArrayRangeQuery, fq FilterQuery, step int, op string, v Node, as assigned) error {
	a := n.Array()
	if a == nil {
		if isNullLike(n) {
			return nil // a null root the edit creates nothing for stays as it is
		}
		return fmt.Errorf("cannot index array with range %s", rq)
	}
	from := resolveRangeBound(rq.From, len(a), 0)
	to := resolveRangeBound(rq.To, len(a), len(a))
	return editElements(pn, n, a, from, to, nil, fq, step, op, v, as)
}

// editElements edits a[from:to], skipping elements match rejects, and writes them back through n when it is an EditorNode.
func editElements(pn *Node, n Node, a Array, from, to int, match func(Node) (bool, error), fq FilterQuery, step int, op string, v Node, as assigned) error {
	en, isEditor := n.(EditorNode)
	for i := from; i < to; i++ {
		child := orNil(a[i])
		if match != nil {
			ok, err := match(child)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		if isNullLike(child) {
			if child = emptyIntermediate(op, fq[step+1:]); child == nil {
				continue // a null element stays as it is
			}
		}

		if err := resolveAndEdit(&child, fq, step+1, op, v, as); err != nil {
			return err
		}

		if isEditor {
			if err := en.Set(i, child); err != nil {
				return err
			}
			continue
		}
		a[i] = child
	}
	if !isEditor {
		*pn = a
	}
	return nil
}

// isNullLike reports whether n is null or a nil Map or Array, which an edit replaces like null.
func isNullLike(n Node) bool {
	switch tn := n.(type) {
	case Map:
		return tn == nil
	case Array:
		return tn == nil
	}
	return isNull(n)
}

// emptyIntermediate returns an empty node for the next step of rest, or nil when a delete or a later step could not create its container.
func emptyIntermediate(op string, rest FilterQuery) Node {
	if op == "^?" || len(rest) == 0 {
		return nil
	}
	for _, q := range rest {
		switch tq := q.(type) {
		case MapQuery:
		case ArrayQuery:
			if tq < 0 {
				return nil // an empty array has no element to count from the end
			}
		case indexQuery:
			if tq.editIndex() < 0 {
				return nil
			}
		default:
			return nil
		}
	}
	switch rest[0].(type) {
	case MapQuery:
		return Map{}
	case ArrayQuery, indexQuery:
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

// assigned holds the containers this edit has stored, keyed by identity, so a later walk never edits inside them.
type assigned map[nodeKey]Node

// nodeKey identifies a container by its data pointer and, for a slice, its length.
type nodeKey struct {
	ptr uintptr
	len int
}

// newAssigned returns an empty set when the query walks, the only case in which an edit can reach a stored value.
func newAssigned(fq FilterQuery) assigned {
	for _, q := range fq {
		if _, ok := q.(WalkQuery); ok {
			return assigned{}
		}
	}
	return nil
}

// add records v when it has an identity, keeping it alive so its address is not reused during the edit.
func (as assigned) add(v Node) {
	if as == nil {
		return
	}
	if id := nodeID(v); id.ptr != 0 {
		as[id] = v
	}
}

// has reports whether n was stored by this edit.
func (as assigned) has(n Node) bool {
	if len(as) == 0 {
		return false
	}
	_, ok := as[nodeID(n)]
	return ok
}

// nodeID identifies a map, pointer or non-empty slice (by data pointer and length), custom nodes included; other values have none.
func nodeID(n Node) nodeKey {
	v := reflect.ValueOf(n)
	switch v.Kind() {
	case reflect.Map, reflect.Pointer:
		return nodeKey{ptr: v.Pointer()}
	case reflect.Slice:
		if v.Len() > 0 {
			return nodeKey{ptr: v.Pointer(), len: v.Len()}
		}
	}
	return nodeKey{}
}

// editLast applies the final step and records the stored value.
func editLast(pn *Node, q Query, op string, v Node, as assigned) error {
	if wq, ok := q.(WalkQuery); ok {
		return wq.edit(pn, op, v, as)
	}
	if err := editQuery(pn, q, op, v); err != nil {
		return err
	}
	as.add(v)
	return nil
}
