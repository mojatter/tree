package schema

import (
	"strings"
	"testing"

	"github.com/mojatter/tree"
)

func TestBuiltinRegistry(t *testing.T) {
	r := BuiltinRegistry()
	wantTypes := []string{"require", "string", "int", "float", "bool", "array", "map", "and", "or", "not"}
	for _, name := range wantTypes {
		if _, ok := r.entries[name]; !ok {
			t.Errorf("BuiltinRegistry missing %q", name)
		}
	}
	if r.MaxReportedErrors != 20 {
		t.Errorf("MaxReportedErrors = %d, want 20", r.MaxReportedErrors)
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	r.Register("x", func(_ tree.Node, _ *Registry) (Rule, error) { return Require{}, nil }, "a", "b")
	e, ok := r.entries["x"]
	if !ok {
		t.Fatal("x not registered")
	}
	if len(e.fields) != 2 || e.fields[0] != "a" || e.fields[1] != "b" {
		t.Errorf("fields = %v, want [a b]", e.fields)
	}
}

func TestRegistry_Parse(t *testing.T) {
	testCases := []struct {
		caseName string
		spec     tree.Node
		wantErrs []string // substrings; empty means success
	}{
		{
			caseName: "spec not a map",
			spec:     tree.V("x"),
			wantErrs: []string{"rule spec must be map"},
		},
		{
			caseName: "missing type",
			spec:     tree.Map{"enum": tree.A("a")},
			wantErrs: []string{"rule spec missing 'type'"},
		},
		{
			caseName: "type not string",
			spec:     tree.Map{"type": tree.V(42)},
			wantErrs: []string{"'type' must be string"},
		},
		{
			caseName: "type empty",
			spec:     tree.Map{"type": tree.V("")},
			wantErrs: []string{"'type' must not be empty"},
		},
		{
			caseName: "unknown type",
			spec:     tree.Map{"type": tree.V("uuid")},
			wantErrs: []string{`unknown rule type "uuid"`},
		},
		{
			caseName: "unknown field",
			spec:     tree.Map{"type": tree.V("string"), "foo": tree.V(1)},
			wantErrs: []string{`unknown field "foo"`},
		},
		{
			caseName: "required not bool",
			spec:     tree.Map{"type": tree.V("string"), "required": tree.V("yes")},
			wantErrs: []string{`"required" must be bool, got string`},
		},
		{
			caseName: "valid string",
			spec:     tree.Map{"type": tree.V("string"), "enum": tree.A("a", "b")},
		},
		{
			caseName: "string regex compiled at parse time",
			spec:     tree.Map{"type": tree.V("string"), "regex": tree.V(`^\d+$`)},
		},
		{
			caseName: "string invalid regex fails parse",
			spec:     tree.Map{"type": tree.V("string"), "regex": tree.V(`[`)},
			wantErrs: []string{`invalid regex "["`},
		},
		{
			caseName: "string min len type mismatch",
			spec:     tree.Map{"type": tree.V("string"), "minLen": tree.V("zero")},
			wantErrs: []string{`"minLen" must be integer, got string`},
		},
		{
			caseName: "int min not integer",
			spec:     tree.Map{"type": tree.V("int"), "min": tree.V(1.5)},
			wantErrs: []string{`"min" must be integer`},
		},
		{
			caseName: "float min string",
			spec:     tree.Map{"type": tree.V("float"), "min": tree.V("zero")},
			wantErrs: []string{`"min" must be number, got string`},
		},
		{
			caseName: "map keys not array",
			spec:     tree.Map{"type": tree.V("map"), "keys": tree.V("name")},
			wantErrs: []string{`"keys" must be array`},
		},
		{
			caseName: "array minLen type mismatch",
			spec:     tree.Map{"type": tree.V("array"), "minLen": tree.V("one")},
			wantErrs: []string{`"minLen" must be integer, got string`},
		},
		{
			caseName: "array maxLen type mismatch",
			spec:     tree.Map{"type": tree.V("array"), "maxLen": tree.V(true)},
			wantErrs: []string{`"maxLen" must be integer, got bool`},
		},
		{
			caseName: "map keys element not string",
			spec:     tree.Map{"type": tree.V("map"), "keys": tree.A("a", 1)},
			wantErrs: []string{`"keys"[1] must be string`},
		},
		{
			caseName: "and missing of",
			spec:     tree.Map{"type": tree.V("and")},
			wantErrs: []string{"and: 'of' is required"},
		},
		{
			caseName: "or of not array",
			spec:     tree.Map{"type": tree.V("or"), "of": tree.V("nope")},
			wantErrs: []string{"or: 'of' must be array"},
		},
		{
			caseName: "not missing rule",
			spec:     tree.Map{"type": tree.V("not")},
			wantErrs: []string{"not: 'rule' is required"},
		},
		{
			caseName: "not rule nested error",
			spec:     tree.Map{"type": tree.V("not"), "rule": tree.Map{"type": tree.V("unknown")}},
			wantErrs: []string{`not.rule: unknown rule type "unknown"`},
		},
		{
			caseName: "every missing rules",
			spec:     tree.Map{"type": tree.V("every")},
			wantErrs: []string{"every: 'rules' is required"},
		},
		{
			caseName: "every nested error",
			spec: tree.Map{
				"type":  tree.V("every"),
				"rules": tree.Map{".x": tree.Map{"type": tree.V("bogus")}},
			},
			wantErrs: []string{`every.rules: .x: unknown rule type "bogus"`},
		},
		{
			caseName: "and of element error",
			spec: tree.Map{
				"type": tree.V("and"),
				"of":   tree.A(tree.Map{"type": tree.V("string")}, tree.Map{"type": tree.V("bogus")}),
			},
			wantErrs: []string{`and.of[1]: unknown rule type "bogus"`},
		},
		{
			caseName: "multiple unknown fields aggregated",
			spec: tree.Map{
				"type": tree.V("string"),
				"bogus1": tree.V(1),
				"bogus2": tree.V(2),
			},
			wantErrs: []string{`unknown field "bogus1"`, `unknown field "bogus2"`},
		},
	}
	r := BuiltinRegistry()
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			_, err := r.Parse(tc.spec)
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestRegistry_Parse_RequiredWraps(t *testing.T) {
	r := BuiltinRegistry()
	spec := tree.Map{"type": tree.V("string"), "required": tree.V(true)}
	rule, err := r.Parse(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Required(String{}) == And{Require{}, String{}}
	if err := rule.Validate(tree.Nil, ".x"); err == nil {
		t.Error("required wrap: want error on Nil")
	}
	if err := rule.Validate(tree.V("hi"), ".x"); err != nil {
		t.Errorf("required wrap: unexpected %v", err)
	}
}

func TestRegistry_Parse_RequiredFalseNotWrapped(t *testing.T) {
	r := BuiltinRegistry()
	spec := tree.Map{"type": tree.V("string"), "required": tree.V(false)}
	rule, err := r.Parse(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Not wrapped: Nil must be OK.
	if err := rule.Validate(tree.Nil, ".x"); err != nil {
		t.Errorf("required:false should not wrap; got %v", err)
	}
}

func TestRegistry_Parse_Require(t *testing.T) {
	r := BuiltinRegistry()
	rule, err := r.Parse(tree.Map{"type": tree.V("require")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := rule.(Require); !ok {
		t.Errorf("got %T; want Require", rule)
	}
	if err := rule.Validate(tree.Nil, ".x"); err == nil {
		t.Error("Require on Nil: want error")
	}
	if err := rule.Validate(tree.V("x"), ".x"); err != nil {
		t.Errorf("Require on present: unexpected %v", err)
	}
}

func TestRegistry_Parse_Array(t *testing.T) {
	r := BuiltinRegistry()
	rule, err := r.Parse(tree.Map{
		"type":   tree.V("array"),
		"minLen": tree.V(1),
		"maxLen": tree.V(3),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := rule.(Array)
	if !ok {
		t.Fatalf("got %T; want Array", rule)
	}
	if arr.MinLen == nil || *arr.MinLen != 1 {
		t.Errorf("MinLen = %v; want *1", arr.MinLen)
	}
	if arr.MaxLen == nil || *arr.MaxLen != 3 {
		t.Errorf("MaxLen = %v; want *3", arr.MaxLen)
	}
	// Round-trip: validator should enforce the bounds.
	if err := arr.Validate(tree.A(), ".x"); err == nil {
		t.Error("empty array: want MinLen violation")
	}
	if err := arr.Validate(tree.A("a", "b", "c", "d"), ".x"); err == nil {
		t.Error("4-element array: want MaxLen violation")
	}
	if err := arr.Validate(tree.A("a", "b"), ".x"); err != nil {
		t.Errorf("2-element array: unexpected %v", err)
	}
}

func TestParseQueryRules(t *testing.T) {
	testCases := []struct {
		caseName string
		spec     tree.Node
		wantErrs []string
	}{
		{
			caseName: "spec not a map",
			spec:     tree.V("x"),
			wantErrs: []string{"query rules spec must be map"},
		},
		{
			caseName: "valid mixed rules",
			spec: tree.Map{
				".name":   tree.Map{"type": tree.V("string"), "required": tree.V(true)},
				".age":    tree.Map{"type": tree.V("int"), "min": tree.V(0)},
				".tags[]": tree.Map{"type": tree.V("string")},
			},
		},
		{
			caseName: "per-query errors prefixed with query path",
			spec: tree.Map{
				".name": tree.Map{"type": tree.V("string"), "bogus": tree.V(1)},
				".age":  tree.Map{"type": tree.V("missing")},
			},
			wantErrs: []string{
				`.age: unknown rule type "missing"`,
				`.name: unknown field "bogus"`,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			rules, err := ParseQueryRules(tc.spec)
			if len(tc.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rules == nil {
					t.Error("rules == nil")
				}
				return
			}
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestParseQueryRules_RoundTrip(t *testing.T) {
	// spec → rules → run Validate → get expected errors.
	spec := tree.Map{
		".name":   tree.Map{"type": tree.V("string"), "required": tree.V(true)},
		".age":    tree.Map{"type": tree.V("int"), "min": tree.V(0), "max": tree.V(150)},
		".tags[]": tree.Map{"type": tree.V("string")},
		".role": tree.Map{
			"type": tree.V("or"),
			"of": tree.A(
				tree.Map{"type": tree.V("string"), "enum": tree.A("admin", "user")},
				tree.Map{"type": tree.V("bool")},
			),
		},
	}
	rules, err := ParseQueryRules(spec)
	if err != nil {
		t.Fatalf("ParseQueryRules: %v", err)
	}

	doc := tree.Map{
		"name": tree.V("alice"),
		"age":  tree.V(-1), // < min
		"tags": tree.A("a", 2, "b"), // element [1] not string
		"role": tree.V("guest"),     // not in enum, not bool
	}
	err = Validate(doc, rules)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	wants := []string{
		".age: value -1 less than min 0",
		".tags[1]: expected string",
		".role: value \"guest\" not in enum",
	}
	for _, w := range wants {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q missing %q", err.Error(), w)
		}
	}
}

func TestRegistry_Parse_Every(t *testing.T) {
	r := BuiltinRegistry()
	spec := tree.Map{
		"type": tree.V("every"),
		"rules": tree.Map{
			".name": tree.Map{"type": tree.V("string"), "required": tree.V(true)},
		},
	}
	rule, err := r.Parse(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := tree.A(
		tree.Map{"name": tree.V("alice")},
		tree.Map{"count": tree.V(1)}, // missing name
	)
	err = rule.Validate(doc, ".tags")
	if err == nil {
		t.Fatal("want error on missing name")
	}
	want := ".tags[1].name: required"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error %q missing %q", err, want)
	}
}

func TestParseQueryRulesWith_CustomType(t *testing.T) {
	reg := BuiltinRegistry()
	reg.Register("positive", func(_ tree.Node, _ *Registry) (Rule, error) {
		return positiveRule{}, nil
	})
	spec := tree.Map{
		".x": tree.Map{"type": tree.V("positive")},
	}
	rules, err := ParseQueryRulesWith(spec, reg)
	if err != nil {
		t.Fatalf("ParseQueryRulesWith: %v", err)
	}

	if err := Validate(tree.Map{"x": tree.V(5)}, rules); err != nil {
		t.Errorf("positive on 5: unexpected %v", err)
	}
	if err := Validate(tree.Map{"x": tree.V(-1)}, rules); err == nil {
		t.Error("positive on -1: want error")
	}
}

type positiveRule struct{}

func (positiveRule) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	if !n.Type().IsNumberValue() {
		return strErr(q + ": expected number")
	}
	if n.Value().Float64() <= 0 {
		return strErr(q + ": must be positive")
	}
	return nil
}

type strErr string

func (e strErr) Error() string { return string(e) }

func TestJoinLimited(t *testing.T) {
	mk := func(n int) []error {
		errs := make([]error, n)
		for i := range errs {
			errs[i] = strErr("e" + string(rune('0'+i)))
		}
		return errs
	}
	testCases := []struct {
		caseName string
		errs     []error
		max      int
		wantIn   []string
		wantNot  []string
	}{
		{caseName: "none", errs: nil, max: 2},
		{caseName: "under cap", errs: mk(2), max: 5, wantIn: []string{"e0", "e1"}, wantNot: []string{"more errors"}},
		{caseName: "equal cap", errs: mk(5), max: 5, wantIn: []string{"e0", "e4"}, wantNot: []string{"more errors"}},
		{caseName: "over cap", errs: mk(7), max: 3, wantIn: []string{"e0", "e1", "e2", "and 4 more errors"}, wantNot: []string{"e3", "e4"}},
		{caseName: "zero cap means unlimited", errs: mk(10), max: 0, wantIn: []string{"e0", "e9"}, wantNot: []string{"more errors"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := joinLimited(tc.errs, tc.max)
			if len(tc.errs) == 0 {
				if err != nil {
					t.Fatalf("empty: want nil, got %v", err)
				}
				return
			}
			s := err.Error()
			for _, w := range tc.wantIn {
				if !strings.Contains(s, w) {
					t.Errorf("missing %q in %q", w, s)
				}
			}
			for _, w := range tc.wantNot {
				if strings.Contains(s, w) {
					t.Errorf("unexpected %q in %q", w, s)
				}
			}
		})
	}
}

func TestParseQueryRulesWith_ErrorCap(t *testing.T) {
	reg := BuiltinRegistry()
	reg.MaxReportedErrors = 3
	spec := tree.Map{}
	// Build 5 broken rules; errors sort by key so ".e0" .. ".e4".
	for i := range 5 {
		key := "." + string(rune('a'+i))
		spec[key] = tree.Map{"type": tree.V("nope")}
	}
	_, err := ParseQueryRulesWith(spec, reg)
	if err == nil {
		t.Fatal("want error")
	}
	s := err.Error()
	if !strings.Contains(s, "and 2 more errors") {
		t.Errorf("missing cap summary; got %q", s)
	}
}
