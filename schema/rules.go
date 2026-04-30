package schema

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"

	"github.com/mojatter/tree"
)

// Required wraps r so the node must exist. It is shorthand for
// And{Require{}, r} and works uniformly for both leaf rules and
// composites. If r is already a Required wrapper, Required(r) is
// returned unchanged so double wrapping does not produce duplicate
// "required" errors.
func Required(r Rule) Rule {
	if a, ok := r.(And); ok && len(a) > 0 {
		if _, ok := a[0].(Require); ok {
			return r
		}
	}
	return And{Require{}, r}
}

// IntPtr returns a pointer to v.
//
// Deprecated: Use [tree.IntPtr].
func IntPtr(v int) *int { return tree.IntPtr(v) }

// Int64Ptr returns a pointer to v.
//
// Deprecated: Use [tree.Int64Ptr].
func Int64Ptr(v int64) *int64 { return tree.Int64Ptr(v) }

// Float64Ptr returns a pointer to v.
//
// Deprecated: Use [tree.Float64Ptr].
func Float64Ptr(v float64) *float64 { return tree.Float64Ptr(v) }

// Require fires an error when the node is tree.Nil (query matched
// nothing). It is the only rule that treats Nil as a violation.
type Require struct{}

// Validate implements Rule.
func (Require) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return fmt.Errorf("%s: required", q)
	}
	return nil
}

// And passes only if every child rule passes. All child errors are
// collected (not short-circuited) so the caller sees every violation
// at once.
type And []Rule

// Validate implements Rule.
func (rs And) Validate(n tree.Node, q string) error {
	var errs []error
	for _, r := range rs {
		if err := r.Validate(n, q); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Or passes if at least one child rule passes. On Nil it returns nil
// (skip) to stay consistent with leaf rules; use Required(Or{...}) to
// force presence.
type Or []Rule

// Validate implements Rule.
func (rs Or) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	var errs []error
	for _, r := range rs {
		err := r.Validate(n, q)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Every applies a set of QueryRules to each element of an array or
// each value of a map. Error paths include the concrete index
// (.foo[0]) or key (.foo["a"]) of the element. On Nil it returns nil
// (skip); wrap with Required to require presence.
//
// Unlike the bare "[]" suffix at the top level, which flattens
// matches and therefore cannot detect a specific element that is
// missing a required sub-field, Every runs its rules starting from
// each element individually, so Required(r) in Rules fires per
// element.
type Every struct {
	Rules QueryRules
}

// Validate implements Rule.
func (r Every) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	var errs []error
	switch {
	case n.Type().IsArray():
		for i, v := range n.Array() {
			prefix := fmt.Sprintf("%s[%d]", q, i)
			sub, fe := validateNode(v, r.Rules, prefix)
			if fe != nil {
				return fe
			}
			errs = append(errs, sub...)
		}
	case n.Type().IsMap():
		m := n.Map()
		for _, k := range m.Keys() {
			prefix := fmt.Sprintf("%s[%q]", q, k)
			sub, fe := validateNode(m[k], r.Rules, prefix)
			if fe != nil {
				return fe
			}
			errs = append(errs, sub...)
		}
	default:
		return fmt.Errorf("%s: expected array or map, got %s", q, n.Type())
	}
	return errors.Join(errs...)
}

// Not passes if the inner rule fails. On Nil it returns nil so a
// missing node is not treated as "matched"; wrap with Required to
// require presence.
type Not struct {
	Rule Rule
}

// Validate implements Rule.
func (r Not) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if err := r.Rule.Validate(n, q); err != nil {
		return nil
	}
	return fmt.Errorf("%s: unexpected match", q)
}

// String matches a string-typed value. All constraint fields are
// optional; unset ones do not constrain. Regex is a pre-compiled
// pattern so Validate has no per-call compile cost; use
// regexp.MustCompile (or regexp.Compile) at construction time.
type String struct {
	Enum   []string
	Regex  *regexp.Regexp
	MinLen *int
	MaxLen *int
}

// Validate implements Rule.
func (r String) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsStringValue() {
		return fmt.Errorf("%s: expected string, got %s", q, n.Type())
	}
	s := n.Value().String()
	if len(r.Enum) > 0 && !slices.Contains(r.Enum, s) {
		return fmt.Errorf("%s: value %q not in enum %v", q, s, r.Enum)
	}
	if r.Regex != nil && !r.Regex.MatchString(s) {
		return fmt.Errorf("%s: value %q does not match %q", q, s, r.Regex)
	}
	if r.MinLen != nil && len(s) < *r.MinLen {
		return fmt.Errorf("%s: length %d less than min %d", q, len(s), *r.MinLen)
	}
	if r.MaxLen != nil && len(s) > *r.MaxLen {
		return fmt.Errorf("%s: length %d greater than max %d", q, len(s), *r.MaxLen)
	}
	return nil
}

// Int matches an integer-valued number (fractional values are
// rejected). tree stores numbers as float64, so "is integer" is
// checked by comparing the value with its truncation.
type Int struct {
	Min *int64
	Max *int64
}

// Validate implements Rule.
func (r Int) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsNumberValue() {
		return fmt.Errorf("%s: expected number, got %s", q, n.Type())
	}
	f := n.Value().Float64()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("%s: expected integer, got %v", q, f)
	}
	// int64 range check before conversion; values outside (MinInt64,
	// MaxInt64] cannot be represented exactly as int64.
	if f < math.MinInt64 || f > math.MaxInt64 {
		return fmt.Errorf("%s: expected integer, got %v", q, f)
	}
	if f != math.Trunc(f) {
		return fmt.Errorf("%s: expected integer, got %v", q, f)
	}
	i := int64(f)
	if r.Min != nil && i < *r.Min {
		return fmt.Errorf("%s: value %d less than min %d", q, i, *r.Min)
	}
	if r.Max != nil && i > *r.Max {
		return fmt.Errorf("%s: value %d greater than max %d", q, i, *r.Max)
	}
	return nil
}

// Float matches any number with optional float bounds.
type Float struct {
	Min *float64
	Max *float64
}

// Validate implements Rule.
func (r Float) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsNumberValue() {
		return fmt.Errorf("%s: expected number, got %s", q, n.Type())
	}
	f := n.Value().Float64()
	// NaN fails any ordered comparison, which would silently slip past
	// Min/Max checks; reject it explicitly so the rule never accepts a
	// value it cannot order.
	if math.IsNaN(f) {
		return fmt.Errorf("%s: NaN is not allowed", q)
	}
	if r.Min != nil && f < *r.Min {
		return fmt.Errorf("%s: value %v less than min %v", q, f, *r.Min)
	}
	if r.Max != nil && f > *r.Max {
		return fmt.Errorf("%s: value %v greater than max %v", q, f, *r.Max)
	}
	return nil
}

// Bool matches a boolean value.
type Bool struct{}

// Validate implements Rule.
func (Bool) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsBoolValue() {
		return fmt.Errorf("%s: expected bool, got %s", q, n.Type())
	}
	return nil
}

// Array matches an array with optional length bounds. Element rules
// are expressed as separate entries in QueryRules (e.g. ".arr[]").
type Array struct {
	MinLen *int
	MaxLen *int
}

// Validate implements Rule.
func (r Array) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsArray() {
		return fmt.Errorf("%s: expected array, got %s", q, n.Type())
	}
	length := len(n.Array())
	if r.MinLen != nil && length < *r.MinLen {
		return fmt.Errorf("%s: length %d less than min %d", q, length, *r.MinLen)
	}
	if r.MaxLen != nil && length > *r.MaxLen {
		return fmt.Errorf("%s: length %d greater than max %d", q, length, *r.MaxLen)
	}
	return nil
}

// Map matches a map. When Keys is non-empty, any key outside the
// allow-list is reported (one error per unknown key, keys sorted for
// stable output).
type Map struct {
	Keys []string
}

// Validate implements Rule.
func (r Map) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsMap() {
		return fmt.Errorf("%s: expected map, got %s", q, n.Type())
	}
	if len(r.Keys) == 0 {
		return nil
	}
	m := n.Map()
	var errs []error
	for _, k := range m.Keys() {
		if !slices.Contains(r.Keys, k) {
			errs = append(errs, fmt.Errorf("%s: unknown key %q", q, k))
		}
	}
	return errors.Join(errs...)
}
