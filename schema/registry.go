package schema

import (
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/mojatter/tree"
)

// Factory parses a rule-specification node (a tree.Map typically
// decoded from YAML or JSON) into a Rule. It receives the enclosing
// Registry so composite factories (And, Or, Not) can recurse into
// child specs.
type Factory func(spec tree.Node, reg *Registry) (Rule, error)

// Registry is a type-name -> Factory dispatch table. ParseQueryRules
// looks up the "type" field on each spec and invokes the corresponding
// Factory. Custom rule types are added via Register; use
// BuiltinRegistry() to start with the built-in set.
//
// MaxReportedErrors caps the number of errors surfaced by Parse and
// ParseQueryRulesWith; overflow is summarised as "and N more errors".
// Zero or negative means no cap.
type Registry struct {
	entries           map[string]entry
	MaxReportedErrors int
}

// entry bundles a factory with the list of type-specific fields it
// accepts. Registry.Parse uses fields to reject unknown keys on the
// spec ("type" and "required" are always implicitly accepted).
type entry struct {
	factory Factory
	fields  []string
}

// NewRegistry returns an empty Registry with MaxReportedErrors = 20.
// Use BuiltinRegistry() if you want the standard rule types
// preinstalled.
func NewRegistry() *Registry {
	return &Registry{
		entries:           map[string]entry{},
		MaxReportedErrors: 20,
	}
}

// BuiltinRegistry returns a Registry preloaded with the built-in rule
// types: require, string, int, float, bool, array, map, and, or,
// not, every.
func BuiltinRegistry() *Registry {
	r := NewRegistry()
	r.Register("require", parseRequireSpec)
	r.Register("string", parseStringSpec, "enum", "regex", "minLen", "maxLen")
	r.Register("int", parseIntSpec, "min", "max")
	r.Register("float", parseFloatSpec, "min", "max")
	r.Register("bool", parseBoolSpec)
	r.Register("array", parseArraySpec, "minLen", "maxLen")
	r.Register("map", parseMapSpec, "keys")
	r.Register("and", parseAndSpec, "of")
	r.Register("or", parseOrSpec, "of")
	r.Register("not", parseNotSpec, "rule")
	r.Register("every", parseEverySpec, "rules")
	return r
}

// Register adds or replaces the factory for name. fields lists the
// type-specific field names accepted on the spec; "type" and
// "required" are always accepted implicitly.
func (r *Registry) Register(name string, fac Factory, fields ...string) {
	r.entries[name] = entry{factory: fac, fields: fields}
}

// Parse dispatches spec to the factory registered under spec's "type"
// field and returns the resulting Rule. Unknown fields are reported
// as errors. When spec carries "required: true", the result is wrapped
// with Required(...).
func (r *Registry) Parse(spec tree.Node) (Rule, error) {
	if !spec.Type().IsMap() {
		return nil, fmt.Errorf("rule spec must be map, got %s", spec.Type())
	}
	m := spec.Map()

	tn := m.Get("type")
	if tn.IsNil() {
		return nil, fmt.Errorf("rule spec missing 'type'")
	}
	if !tn.Type().IsStringValue() {
		return nil, fmt.Errorf("'type' must be string, got %s", tn.Type())
	}
	typ := tn.Value().String()
	if typ == "" {
		return nil, fmt.Errorf("'type' must not be empty")
	}

	e, ok := r.entries[typ]
	if !ok {
		return nil, fmt.Errorf("unknown rule type %q", typ)
	}

	var errs []error

	for _, k := range m.Keys() {
		if k == "type" || k == "required" {
			continue
		}
		if !slices.Contains(e.fields, k) {
			errs = append(errs, fmt.Errorf("unknown field %q", k))
		}
	}

	reqPtr, reqErr := optBoolPtr(spec, "required")
	if reqErr != nil {
		errs = append(errs, reqErr)
	}

	if len(errs) > 0 {
		return nil, joinLimited(errs, r.MaxReportedErrors)
	}

	rule, err := e.factory(spec, r)
	if err != nil {
		return nil, err
	}

	if reqPtr != nil && *reqPtr {
		rule = Required(rule)
	}
	return rule, nil
}

// ParseQueryRules builds QueryRules from spec using BuiltinRegistry().
// spec must be a map whose keys are tree queries and whose values are
// rule specs (each a map containing at minimum a "type" field).
func ParseQueryRules(spec tree.Node) (QueryRules, error) {
	return ParseQueryRulesWith(spec, BuiltinRegistry())
}

// ParseQueryRulesWith is ParseQueryRules but dispatches through reg,
// allowing custom rule types to be resolved. Errors from individual
// queries are aggregated and capped at reg.MaxReportedErrors.
func ParseQueryRulesWith(spec tree.Node, reg *Registry) (QueryRules, error) {
	if !spec.Type().IsMap() {
		return nil, fmt.Errorf("query rules spec must be map, got %s", spec.Type())
	}
	m := spec.Map()
	out := QueryRules{}
	var errs []error
	for _, k := range m.Keys() {
		r, err := reg.Parse(m[k])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", k, err))
			continue
		}
		out[k] = r
	}
	if len(errs) > 0 {
		return nil, joinLimited(errs, reg.MaxReportedErrors)
	}
	return out, nil
}

// joinLimited joins up to max errors via errors.Join; overflow is
// collapsed into a trailing "and N more errors" entry. max <= 0
// disables the cap.
func joinLimited(errs []error, max int) error {
	if max <= 0 || len(errs) <= max {
		return errors.Join(errs...)
	}
	head := make([]error, 0, max+1)
	head = append(head, errs[:max]...)
	head = append(head, fmt.Errorf("and %d more errors", len(errs)-max))
	return errors.Join(head...)
}

// -- Built-in factories --

func parseRequireSpec(_ tree.Node, _ *Registry) (Rule, error) {
	return Require{}, nil
}

func parseStringSpec(spec tree.Node, _ *Registry) (Rule, error) {
	r := String{}
	var err error
	if r.Enum, err = optStringSlice(spec, "enum"); err != nil {
		return nil, err
	}
	pat, err := optStringVal(spec, "regex")
	if err != nil {
		return nil, err
	}
	if pat != "" {
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pat, err)
		}
		r.Regex = re
	}
	if r.MinLen, err = optIntPtr(spec, "minLen"); err != nil {
		return nil, err
	}
	if r.MaxLen, err = optIntPtr(spec, "maxLen"); err != nil {
		return nil, err
	}
	return r, nil
}

func parseIntSpec(spec tree.Node, _ *Registry) (Rule, error) {
	r := Int{}
	var err error
	if r.Min, err = optInt64Ptr(spec, "min"); err != nil {
		return nil, err
	}
	if r.Max, err = optInt64Ptr(spec, "max"); err != nil {
		return nil, err
	}
	return r, nil
}

func parseFloatSpec(spec tree.Node, _ *Registry) (Rule, error) {
	r := Float{}
	var err error
	if r.Min, err = optFloat64Ptr(spec, "min"); err != nil {
		return nil, err
	}
	if r.Max, err = optFloat64Ptr(spec, "max"); err != nil {
		return nil, err
	}
	return r, nil
}

func parseBoolSpec(_ tree.Node, _ *Registry) (Rule, error) {
	return Bool{}, nil
}

func parseArraySpec(spec tree.Node, _ *Registry) (Rule, error) {
	r := Array{}
	var err error
	if r.MinLen, err = optIntPtr(spec, "minLen"); err != nil {
		return nil, err
	}
	if r.MaxLen, err = optIntPtr(spec, "maxLen"); err != nil {
		return nil, err
	}
	return r, nil
}

func parseMapSpec(spec tree.Node, _ *Registry) (Rule, error) {
	r := Map{}
	ks, err := optStringSlice(spec, "keys")
	if err != nil {
		return nil, err
	}
	r.Keys = ks
	return r, nil
}

func parseAndSpec(spec tree.Node, reg *Registry) (Rule, error) {
	rs, err := parseOfArray("and", spec, reg)
	if err != nil {
		return nil, err
	}
	return And(rs), nil
}

func parseOrSpec(spec tree.Node, reg *Registry) (Rule, error) {
	rs, err := parseOfArray("or", spec, reg)
	if err != nil {
		return nil, err
	}
	return Or(rs), nil
}

func parseNotSpec(spec tree.Node, reg *Registry) (Rule, error) {
	inner := spec.Get("rule")
	if inner.IsNil() {
		return nil, fmt.Errorf("not: 'rule' is required")
	}
	r, err := reg.Parse(inner)
	if err != nil {
		return nil, fmt.Errorf("not.rule: %w", err)
	}
	return Not{Rule: r}, nil
}

func parseEverySpec(spec tree.Node, reg *Registry) (Rule, error) {
	rulesSpec := spec.Get("rules")
	if rulesSpec.IsNil() {
		return nil, fmt.Errorf("every: 'rules' is required")
	}
	rules, err := ParseQueryRulesWith(rulesSpec, reg)
	if err != nil {
		return nil, fmt.Errorf("every.rules: %w", err)
	}
	return Every{Rules: rules}, nil
}

func parseOfArray(kind string, spec tree.Node, reg *Registry) ([]Rule, error) {
	of := spec.Get("of")
	if of.IsNil() {
		return nil, fmt.Errorf("%s: 'of' is required", kind)
	}
	if !of.Type().IsArray() {
		return nil, fmt.Errorf("%s: 'of' must be array, got %s", kind, of.Type())
	}
	arr := of.Array()
	rs := make([]Rule, 0, len(arr))
	for i, child := range arr {
		r, err := reg.Parse(child)
		if err != nil {
			return nil, fmt.Errorf("%s.of[%d]: %w", kind, i, err)
		}
		rs = append(rs, r)
	}
	return rs, nil
}

// -- Field extractors --
//
// Each returns the zero value with nil error when the field is absent,
// the converted value with nil error when it is present and well-typed,
// or an error describing the type mismatch.

func optStringVal(spec tree.Node, key string) (string, error) {
	n := spec.Get(key)
	if n.IsNil() {
		return "", nil
	}
	if !n.Type().IsStringValue() {
		return "", fmt.Errorf("%q must be string, got %s", key, n.Type())
	}
	return n.Value().String(), nil
}

func optStringSlice(spec tree.Node, key string) ([]string, error) {
	n := spec.Get(key)
	if n.IsNil() {
		return nil, nil
	}
	if !n.Type().IsArray() {
		return nil, fmt.Errorf("%q must be array, got %s", key, n.Type())
	}
	arr := n.Array()
	out := make([]string, 0, len(arr))
	for i, v := range arr {
		if !v.Type().IsStringValue() {
			return nil, fmt.Errorf("%q[%d] must be string, got %s", key, i, v.Type())
		}
		out = append(out, v.Value().String())
	}
	return out, nil
}

func optBoolPtr(spec tree.Node, key string) (*bool, error) {
	n := spec.Get(key)
	if n.IsNil() {
		return nil, nil
	}
	if !n.Type().IsBoolValue() {
		return nil, fmt.Errorf("%q must be bool, got %s", key, n.Type())
	}
	v := n.Value().Bool()
	return &v, nil
}

func optIntPtr(spec tree.Node, key string) (*int, error) {
	n := spec.Get(key)
	if n.IsNil() {
		return nil, nil
	}
	if !n.Type().IsNumberValue() {
		return nil, fmt.Errorf("%q must be integer, got %s", key, n.Type())
	}
	f := n.Value().Float64()
	if f != float64(int64(f)) {
		return nil, fmt.Errorf("%q must be integer, got %v", key, f)
	}
	v := n.Value().Int()
	return &v, nil
}

func optInt64Ptr(spec tree.Node, key string) (*int64, error) {
	n := spec.Get(key)
	if n.IsNil() {
		return nil, nil
	}
	if !n.Type().IsNumberValue() {
		return nil, fmt.Errorf("%q must be integer, got %s", key, n.Type())
	}
	f := n.Value().Float64()
	if f != float64(int64(f)) {
		return nil, fmt.Errorf("%q must be integer, got %v", key, f)
	}
	v := n.Value().Int64()
	return &v, nil
}

func optFloat64Ptr(spec tree.Node, key string) (*float64, error) {
	n := spec.Get(key)
	if n.IsNil() {
		return nil, nil
	}
	if !n.Type().IsNumberValue() {
		return nil, fmt.Errorf("%q must be number, got %s", key, n.Type())
	}
	v := n.Value().Float64()
	return &v, nil
}
