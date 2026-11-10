package schema

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/mojatter/tree"
)

// Rule validates a single tree.Node reached via query q.
//
// q is a display path used for error messages; when Validate (top-level
// function) drives the call, q is the query string, with each [] after
// the last pipe, outside selectors and groups with a pipe, expanded to
// the concrete index or key (e.g. .arr[0].k).
type Rule interface {
	Validate(n tree.Node, q string) error
}

// QueryRules maps a tree query to the rule applied at each match.
type QueryRules map[string]Rule

// ErrQuery indicates that a rule's query string is malformed (parsing
// or running it failed). It represents a bug in the rules definition, not a
// data violation, so Validate and ValidateWithPrefix fail fast and
// return only this error instead of joining it with data-level
// violation errors. Callers that want to distinguish can errors.As
// into *ErrQuery.
type ErrQuery struct {
	Query string // joined display path (prefix + rule query) that failed to parse or run
	Err   error  // underlying tree error
}

// Error implements error.
func (e *ErrQuery) Error() string {
	return fmt.Sprintf("query %q: %s", e.Query, e.Err)
}

// Unwrap returns the underlying tree error.
func (e *ErrQuery) Unwrap() error {
	return e.Err
}

// Validate executes every (query, rule) entry in rules against n.
// Errors from all rules are collected and joined.
//
// For each query q, the rule runs on every match. If q yields no matches, the
// rule is invoked once with tree.Nil (so Require{} can fire). Each "[]"
// after the last pipe, outside selectors and groups with a pipe, expands,
// so matches are reported with a concrete path: .arr[0].k for array parents
// and .obj["a"].k for map parents (keys sorted). An empty array or map
// before a "[]" fires nothing; docs/schema.md has the full rules.
func Validate(n tree.Node, rules QueryRules) error {
	return ValidateWithPrefix(n, rules, "")
}

// ValidateWithPrefix is Validate with a path prefix prepended to every
// error location. It is intended for custom Rule implementations that
// re-apply a QueryRules set to a subtree (think Every, but driven by
// the enclosing node's content): the Rule can delegate to
// ValidateWithPrefix(child, subRules, q) so nested error paths come
// out as q + subquery (e.g. .handlers["c"].type) instead of bare
// subqueries that the caller would otherwise have to splice in by
// unwrapping and re-wrapping errors.
//
// prefix is concatenated as-is; pass "" to get the same behavior as
// Validate.
func ValidateWithPrefix(n tree.Node, rules QueryRules, prefix string) error {
	errs, fe := validateNode(n, rules, prefix)
	if fe != nil {
		return fe
	}
	return errors.Join(errs...)
}

// validateNode is the shared driver behind Validate and Every. prefix
// is prepended to every display path and query key so nested element
// validation (via Every) can attribute errors to .parent[i].child.
//
// When a rule query fails to parse, it returns immediately with a
// non-nil *ErrQuery and nil validation errors: a malformed query is a
// bug in the rules, so reporting it alongside data-level violations
// would just produce noise.
func validateNode(n tree.Node, rules QueryRules, prefix string) ([]error, *ErrQuery) {
	qs := slices.Sorted(maps.Keys(rules))

	var errs []error
	for _, q := range qs {
		pq, err := tree.ParseQuery(q)
		if err != nil {
			return nil, &ErrQuery{Query: joinPath(prefix, q), Err: err}
		}
		fq := flattenGroups(pq)
		text := q
		if firstEach(fq) >= 0 {
			text = fq.String() // one spelling for the expanded and unexpanded paths of a rule
		}
		sub, err := validateSteps(n, fq, prefix, text, rules[q])
		if err != nil {
			return nil, &ErrQuery{Query: joinPath(prefix, q), Err: err}
		}
		errs = append(errs, sub...)
	}
	return errs, nil
}

// validateSteps runs r on what fq yields from n, at the path prefix + text.
// The first [] after the last pipe expands: the rest of fq runs on each element with its index or key in the path.
// A step that yields nothing, other than an empty array or map before a [], passes Nil once; a nil Array or Map is null.
func validateSteps(n tree.Node, fq tree.FilterQuery, prefix, text string, r Rule) ([]error, error) {
	i := firstEach(fq)
	if i < 0 {
		vs, err := fq.Exec(n)
		if err != nil {
			return nil, err
		}
		if len(vs) == 0 {
			vs = []tree.Node{tree.Nil}
		}
		var errs []error
		display := joinPath(prefix, text)
		for _, v := range vs {
			if err := r.Validate(v, display); err != nil {
				errs = append(errs, err)
			}
		}
		return errs, nil
	}
	base, rest := fq[:i], fq[i+1:]
	parents, err := append(tree.FilterQuery{tree.NopQuery{}}, base...).Exec(n)
	if err != nil {
		return nil, err
	}
	basePath, restText := joinPath(prefix, base.String()), rest.String()
	var errs []error
	expanded := false
	for _, p := range parents {
		switch {
		case p.IsNil():
			continue // a nil Array or Map is null, as Require reads it
		case p.Type().IsArray():
			a := p.Array()
			for j := range a {
				sub, err := validateSteps(a.Get(j), rest, fmt.Sprintf("%s[%d]", basePath, j), restText, r)
				if err != nil {
					return nil, err
				}
				errs = append(errs, sub...)
			}
		case p.Type().IsMap():
			m := p.Map()
			for _, k := range m.Keys() {
				sub, err := validateSteps(m.Get(k), rest, fmt.Sprintf("%s[%q]", basePath, k), restText, r)
				if err != nil {
					return nil, err
				}
				errs = append(errs, sub...)
			}
		default:
			continue
		}
		expanded = true
	}
	if !expanded {
		if err := r.Validate(tree.Nil, joinPath(prefix, text)); err != nil {
			errs = append(errs, err)
		}
	}
	return errs, nil
}

// flattenGroups inlines groups without a pipe, as the edit path does, so a [] inside one expands.
func flattenGroups(q tree.Query) tree.FilterQuery {
	fq, ok := q.(tree.FilterQuery)
	if !ok {
		return tree.FilterQuery{q}
	}
	flat := make(tree.FilterQuery, 0, len(fq))
	for _, s := range fq {
		if g, ok := s.(tree.FilterQuery); ok && !slices.ContainsFunc(g, isPipe) {
			flat = append(flat, flattenGroups(g)...)
			continue
		}
		flat = append(flat, s)
	}
	return flat
}

func isPipe(q tree.Query) bool {
	_, ok := q.(tree.SlurpQuery)
	return ok
}

// firstEach returns the index of the first [] after the last pipe in fq, or -1.
func firstEach(fq tree.FilterQuery) int {
	start := 0
	for i, q := range fq {
		if isPipe(q) {
			start = i + 1
		}
	}
	for i := start; i < len(fq); i++ {
		if s, ok := fq[i].(tree.SelectQuery); ok && s.Selector == nil {
			return i
		}
	}
	return -1
}

// joinPath concatenates a prefix path and a rule query into the
// display path used in error messages. A rule query of "." refers to
// the current node, so it drops to the bare prefix (or "." when the
// prefix is empty for a top-level Validate call).
func joinPath(prefix, q string) string {
	if q == "." {
		if prefix == "" {
			return "."
		}
		return prefix
	}
	return prefix + q
}
