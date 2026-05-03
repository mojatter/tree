package schema

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/mojatter/tree"
)

func intPtr(v int) *int         { return &v }
func int64Ptr(v int64) *int64   { return &v }
func f64Ptr(v float64) *float64 { return &v }

func TestAny(t *testing.T) {
	testCases := []struct {
		caseName string
		n        tree.Node
	}{
		{caseName: "nil", n: tree.Nil},
		{caseName: "string", n: tree.V("x")},
		{caseName: "number", n: tree.V(42)},
		{caseName: "bool", n: tree.V(true)},
		{caseName: "map", n: tree.Map{"a": tree.V(1)}},
		{caseName: "array", n: tree.A("x", "y")},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if err := (Any{}).Validate(tc.n, ".x"); err != nil {
				t.Errorf("Any: unexpected %v", err)
			}
		})
	}
}

func TestRequire(t *testing.T) {
	if err := (Require{}).Validate(tree.Nil, ".x"); err == nil {
		t.Error("Require on Nil: want error")
	}
	if err := (Require{}).Validate(tree.V("x"), ".x"); err != nil {
		t.Errorf("Require on present: unexpected %v", err)
	}
}

func TestRequired_Idempotent(t *testing.T) {
	// Required(Required(String{})) must still yield exactly one
	// "required" error on Nil, not two.
	r := Required(Required(String{}))
	err := r.Validate(tree.Nil, ".x")
	if err == nil {
		t.Fatal("want error")
	}
	if got := strings.Count(err.Error(), "required"); got != 1 {
		t.Errorf("required count = %d, want 1 (err=%q)", got, err.Error())
	}
}

func TestAnd(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     And
		n        tree.Node
		wantErrs []string
	}{
		{
			caseName: "all pass",
			rule:     And{Require{}, String{}},
			n:        tree.V("x"),
		},
		{
			caseName: "require fires, string skips on nil",
			rule:     And{Require{}, String{}},
			n:        tree.Nil,
			wantErrs: []string{".x: required"},
		},
		{
			caseName: "enum + length both fail",
			rule:     And{String{Enum: []string{"a"}}, String{MaxLen: intPtr(0)}},
			n:        tree.V("bb"),
			wantErrs: []string{"not in enum", "length 2 greater than max 0"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestOr(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Or
		n        tree.Node
		wantErrs []string
	}{
		{
			caseName: "nil skipped",
			rule:     Or{String{}, Int{}},
			n:        tree.Nil,
		},
		{
			caseName: "string branch passes",
			rule:     Or{String{}, Int{}},
			n:        tree.V("hi"),
		},
		{
			caseName: "int branch passes",
			rule:     Or{String{}, Int{}},
			n:        tree.V(42),
		},
		{
			caseName: "all branches fail",
			rule:     Or{String{}, Int{}},
			n:        tree.V(true),
			wantErrs: []string{"expected string", "expected number"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestNot(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Not
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skipped", rule: Not{Rule: String{}}, n: tree.Nil},
		{caseName: "inner fails so Not passes", rule: Not{Rule: String{}}, n: tree.V(42)},
		{caseName: "inner passes so Not fails", rule: Not{Rule: String{}}, n: tree.V("hi"), wantErrs: []string{".x: unexpected match"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestEvery(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Every
		n        tree.Node
		wantErrs []string
	}{
		{
			caseName: "nil skipped",
			rule:     Every{Rules: QueryRules{".name": Required(String{})}},
			n:        tree.Nil,
		},
		{
			caseName: "non-collection rejected",
			rule:     Every{Rules: QueryRules{".name": String{}}},
			n:        tree.V("x"),
			wantErrs: []string{".x: expected array or map, got string"},
		},
		{
			caseName: "array all pass",
			rule:     Every{Rules: QueryRules{".name": Required(String{})}},
			n: tree.A(
				tree.Map{"name": tree.V("alice")},
				tree.Map{"name": tree.V("bob")},
			),
		},
		{
			caseName: "array with missing field fires Required per element",
			rule:     Every{Rules: QueryRules{".name": Required(String{})}},
			n: tree.A(
				tree.Map{"name": tree.V("alice")},
				tree.Map{"count": tree.V(1)}, // no "name"
			),
			wantErrs: []string{".x[1].name: required"},
		},
		{
			caseName: "map values validated with concrete key path",
			rule:     Every{Rules: QueryRules{".name": Required(String{})}},
			n: tree.Map{
				"a": tree.Map{"name": tree.V("ok")},
				"b": tree.Map{"other": tree.V(1)},
			},
			wantErrs: []string{`.x["b"].name: required`},
		},
		{
			caseName: "nested Every",
			rule: Every{Rules: QueryRules{
				".items": Every{Rules: QueryRules{
					".id": Required(Int{}),
				}},
			}},
			n: tree.A(
				tree.Map{"items": tree.A(
					tree.Map{"id": tree.V(1)},
					tree.Map{"name": tree.V("x")}, // no "id"
				)},
			),
			wantErrs: []string{".x[0].items[1].id: required"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestString(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     String
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skip", rule: String{}, n: tree.Nil},
		{caseName: "plain string", rule: String{}, n: tree.V("x")},
		{caseName: "not string", rule: String{}, n: tree.V(1), wantErrs: []string{".x: expected string, got number"}},
		{caseName: "enum hit", rule: String{Enum: []string{"a", "b"}}, n: tree.V("a")},
		{caseName: "enum miss", rule: String{Enum: []string{"a", "b"}}, n: tree.V("c"), wantErrs: []string{`.x: value "c" not in enum`}},
		{caseName: "regex match", rule: String{Regex: regexp.MustCompile(`^\d+$`)}, n: tree.V("42")},
		{caseName: "regex miss", rule: String{Regex: regexp.MustCompile(`^\d+$`)}, n: tree.V("abc"), wantErrs: []string{`.x: value "abc" does not match`}},
		{caseName: "min len", rule: String{MinLen: intPtr(3)}, n: tree.V("ab"), wantErrs: []string{".x: length 2 less than min 3"}},
		{caseName: "max len", rule: String{MaxLen: intPtr(2)}, n: tree.V("abc"), wantErrs: []string{".x: length 3 greater than max 2"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestInt(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Int
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skip", rule: Int{}, n: tree.Nil},
		{caseName: "plain int", rule: Int{}, n: tree.V(42)},
		{caseName: "string rejected", rule: Int{}, n: tree.V("x"), wantErrs: []string{".x: expected number, got string"}},
		{caseName: "float rejected", rule: Int{}, n: tree.V(1.5), wantErrs: []string{".x: value 1.5 is not an integer"}},
		{caseName: "whole float ok", rule: Int{}, n: tree.V(3.0)},
		{caseName: "NaN rejected", rule: Int{}, n: tree.V(math.NaN()), wantErrs: []string{".x: value NaN is not an integer"}},
		{caseName: "+Inf rejected", rule: Int{}, n: tree.V(math.Inf(1)), wantErrs: []string{".x: value +Inf is not an integer"}},
		{caseName: "-Inf rejected", rule: Int{}, n: tree.V(math.Inf(-1)), wantErrs: []string{".x: value -Inf is not an integer"}},
		{caseName: "above int64 max rejected", rule: Int{}, n: tree.V(1e19), wantErrs: []string{".x: value 1e+19 exceeds int64 range"}},
		{caseName: "min ok", rule: Int{Min: int64Ptr(0)}, n: tree.V(1)},
		{caseName: "min violated", rule: Int{Min: int64Ptr(0)}, n: tree.V(-1), wantErrs: []string{".x: value -1 less than min 0"}},
		{caseName: "max violated", rule: Int{Max: int64Ptr(10)}, n: tree.V(11), wantErrs: []string{".x: value 11 greater than max 10"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestFloat(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Float
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skip", rule: Float{}, n: tree.Nil},
		{caseName: "float ok", rule: Float{}, n: tree.V(1.5)},
		{caseName: "int ok", rule: Float{}, n: tree.V(1)},
		{caseName: "string rejected", rule: Float{}, n: tree.V("x"), wantErrs: []string{".x: expected number, got string"}},
		{caseName: "NaN rejected", rule: Float{Min: f64Ptr(0.0)}, n: tree.V(math.NaN()), wantErrs: []string{".x: value NaN is not allowed"}},
		{caseName: "min violated", rule: Float{Min: f64Ptr(0.0)}, n: tree.V(-0.5), wantErrs: []string{"less than min"}},
		{caseName: "max violated", rule: Float{Max: f64Ptr(1.0)}, n: tree.V(1.5), wantErrs: []string{"greater than max"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestBool(t *testing.T) {
	testCases := []struct {
		caseName string
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skip", n: tree.Nil},
		{caseName: "true", n: tree.V(true)},
		{caseName: "false", n: tree.V(false)},
		{caseName: "string rejected", n: tree.V("true"), wantErrs: []string{".x: expected bool, got string"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := (Bool{}).Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestArray(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Array
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skip", rule: Array{}, n: tree.Nil},
		{caseName: "plain array", rule: Array{}, n: tree.A("x")},
		{caseName: "not array", rule: Array{}, n: tree.V("x"), wantErrs: []string{".x: expected array, got string"}},
		{caseName: "min len violated", rule: Array{MinLen: intPtr(2)}, n: tree.A("x"), wantErrs: []string{".x: length 1 less than min 2"}},
		{caseName: "max len violated", rule: Array{MaxLen: intPtr(1)}, n: tree.A("x", "y"), wantErrs: []string{".x: length 2 greater than max 1"}},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

func TestMap(t *testing.T) {
	testCases := []struct {
		caseName string
		rule     Map
		n        tree.Node
		wantErrs []string
	}{
		{caseName: "nil skip", rule: Map{}, n: tree.Nil},
		{caseName: "plain map", rule: Map{}, n: tree.Map{}},
		{caseName: "not map", rule: Map{}, n: tree.V("x"), wantErrs: []string{".x: expected map, got string"}},
		{caseName: "no known keys allows any", rule: Map{}, n: tree.Map{"anything": tree.V(1)}},
		{
			caseName: "unknown keys reported sorted",
			rule:     Map{Keys: []string{"a"}},
			n:        tree.Map{"a": tree.V(1), "c": tree.V(2), "b": tree.V(3)},
			wantErrs: []string{`.x: unknown key "b"`, `.x: unknown key "c"`},
		},
		{
			caseName: "keyed rules pass on valid values",
			rule: Map{KeyedRules: map[string]Rule{
				"name": String{},
				"age":  Int{Min: int64Ptr(0)},
			}},
			n: tree.Map{"name": tree.V("alice"), "age": tree.V(30)},
		},
		{
			caseName: "keyed rules: unknown key beyond declared rules",
			rule: Map{KeyedRules: map[string]Rule{
				"name": String{},
			}},
			n:        tree.Map{"name": tree.V("alice"), "bogus": tree.V(1)},
			wantErrs: []string{`.x: unknown key "bogus"`},
		},
		{
			caseName: "keyed rules: per-key violation reported with sub-path",
			rule: Map{KeyedRules: map[string]Rule{
				"name": String{},
				"age":  Int{Min: int64Ptr(0)},
			}},
			n:        tree.Map{"name": tree.V(42), "age": tree.V(-1)},
			wantErrs: []string{".x.age: value -1 less than min 0", ".x.name: expected string, got number"},
		},
		{
			caseName: "keyed rules: missing optional key passes",
			rule: Map{KeyedRules: map[string]Rule{
				"name": String{},
				"age":  Int{},
			}},
			n: tree.Map{"name": tree.V("alice")}, // age omitted, no Required
		},
		{
			caseName: "keyed rules: missing required key reported",
			rule: Map{KeyedRules: map[string]Rule{
				"name": Required(String{}),
				"age":  Int{},
			}},
			n:        tree.Map{"age": tree.V(30)},
			wantErrs: []string{".x.name: required"},
		},
		{
			caseName: "Keys + KeyedRules union becomes the allow-list",
			rule: Map{
				Keys: []string{"legacy"},
				KeyedRules: map[string]Rule{
					"name": String{},
				},
			},
			n: tree.Map{"name": tree.V("alice"), "legacy": tree.V("anything")},
		},
		{
			caseName: "Keys + KeyedRules: still rejects keys not in either",
			rule: Map{
				Keys: []string{"legacy"},
				KeyedRules: map[string]Rule{
					"name": String{},
				},
			},
			n:        tree.Map{"name": tree.V("alice"), "bogus": tree.V(1)},
			wantErrs: []string{`.x: unknown key "bogus"`},
		},
		{
			caseName: "Any inside KeyedRules: key allowed without per-value check",
			rule: Map{KeyedRules: map[string]Rule{
				"name": String{},
				"meta": Any{}, // any value type is fine
			}},
			n: tree.Map{"name": tree.V("alice"), "meta": tree.Map{"created": tree.V("2026")}},
		},
		{
			caseName: "nested KeyedRules: child Map runs its own validation",
			rule: Map{KeyedRules: map[string]Rule{
				"server": Map{KeyedRules: map[string]Rule{
					"port": Required(Int{Min: int64Ptr(1)}),
				}},
			}},
			n:        tree.Map{"server": tree.Map{}}, // missing port
			wantErrs: []string{".x.server.port: required"},
		},
		{
			caseName: "nested KeyedRules: parent Nil short-circuits children",
			rule: Map{KeyedRules: map[string]Rule{
				"server": Map{KeyedRules: map[string]Rule{
					"port": Required(Int{}),
				}},
			}},
			n: tree.Map{}, // server absent: child Required must NOT fire
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := tc.rule.Validate(tc.n, ".x")
			checkErrs(t, err, tc.wantErrs)
		})
	}
}
