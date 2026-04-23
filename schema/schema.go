package schema

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/mojatter/tree"
)

// Rule validates a single tree.Node reached via query q.
//
// q is a display path used for error messages; when Validate (top-level
// function) drives the call, q is the query string, with array-element
// rules (queries ending in []) having their suffix expanded to the
// concrete index (e.g. .arr[0], .arr[1]).
type Rule interface {
	Validate(n tree.Node, q string) error
}

// QueryRules maps a tree query to the rule applied at each match.
type QueryRules map[string]Rule

// ErrQuery indicates that a rule's query string is malformed (tree.Find
// rejected it). It represents a bug in the rules definition, not a
// data violation, so Validate and ValidateWithPrefix fail fast and
// return only this error instead of joining it with data-level
// violation errors. Callers that want to distinguish can errors.As
// into *ErrQuery.
type ErrQuery struct {
	Query string // joined display path (prefix + rule query) that failed to parse
	Err   error  // underlying tree.Find error
}

// Error implements error.
func (e *ErrQuery) Error() string {
	return fmt.Sprintf("query %q: %s", e.Query, e.Err)
}

// Unwrap returns the underlying tree.Find error.
func (e *ErrQuery) Unwrap() error {
	return e.Err
}

// Validate executes every (query, rule) entry in rules against n.
// Errors from all rules are collected and joined.
//
// For each query q, tree.Find is called. If q yields no matches, the
// rule is invoked once with tree.Nil (so Require{} can fire). When q
// ends with "[]", matches are reported with a concrete path:
// .arr[0]/.arr[1]... for array parents and .obj["a"]/.obj["b"]...
// for map parents (keys sorted). If the base of "[]" resolves to more
// than one parent, numeric indices are used as the fallback.
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
		r := rules[q]
		vs, err := tree.Find(n, q)
		if err != nil {
			return nil, &ErrQuery{Query: joinPath(prefix, q), Err: err}
		}
		if len(vs) == 0 {
			if err := r.Validate(tree.Nil, joinPath(prefix, q)); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if base, ok := strings.CutSuffix(q, "[]"); ok {
			sub, fe := validateExpanded(n, base, vs, r, prefix)
			if fe != nil {
				return nil, fe
			}
			errs = append(errs, sub...)
			continue
		}
		display := joinPath(prefix, q)
		for _, v := range vs {
			if err := r.Validate(v, display); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs, nil
}

// validateExpanded runs r against each matched value of a "[]"-suffixed
// query. When the base query resolves to a single map parent, error
// paths include the concrete key (e.g. .obj["a"]); otherwise numeric
// indices are used (e.g. .arr[0]). prefix is prepended to the base
// so nested contexts (from Every) render full paths.
//
// A parse failure on the base query is propagated as *ErrQuery; in
// practice this should not fire when the caller already parsed the
// full "[]"-suffixed query successfully, but it is handled for safety.
func validateExpanded(root tree.Node, base string, vs []tree.Node, r Rule, prefix string) ([]error, *ErrQuery) {
	display := joinPath(prefix, base)
	parents, err := tree.Find(root, base)
	if err != nil {
		return nil, &ErrQuery{Query: joinPath(prefix, base), Err: err}
	}
	if len(parents) == 1 && parents[0].Type().IsMap() {
		m := parents[0].Map()
		var errs []error
		for _, k := range m.Keys() {
			qi := fmt.Sprintf("%s[%q]", display, k)
			if err := r.Validate(m[k], qi); err != nil {
				errs = append(errs, err)
			}
		}
		return errs, nil
	}
	var errs []error
	for i, v := range vs {
		qi := fmt.Sprintf("%s[%d]", display, i)
		if err := r.Validate(v, qi); err != nil {
			errs = append(errs, err)
		}
	}
	return errs, nil
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
