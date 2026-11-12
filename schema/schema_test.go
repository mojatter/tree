package schema

import (
	"errors"
	"strings"
	"testing"

	"github.com/mojatter/tree"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		caseName string
		n        tree.Node
		rules    QueryRules
		wantErrs []string // one substring per error; empty means no error
	}{
		{
			caseName: "require fires on nil root",
			n:        tree.Nil,
			rules:    QueryRules{".": Require{}},
			wantErrs: []string{".: required"},
		},
		{
			caseName: "map known keys",
			n: tree.Map{
				"KNOWN":   tree.Nil,
				"UNKNOWN": tree.Nil,
			},
			rules:    QueryRules{".": Map{Keys: []string{"KNOWN"}}},
			wantErrs: []string{`.: unknown key "UNKNOWN"`},
		},
		{
			caseName: "leaf skips on nil",
			n:        tree.Map{},
			rules:    QueryRules{".missing": String{}},
		},
		{
			caseName: "[] in the middle names each element",
			n:        tree.Map{"arr": tree.A(tree.Map{}, tree.Map{}, tree.Map{"k": tree.V(1)})},
			rules:    QueryRules{".arr[].k": Required(String{})},
			wantErrs: []string{".arr[0].k: required", ".arr[1].k: required", ".arr[2].k: expected string, got number"},
		},
		{
			caseName: "[] in the middle on an empty array fires nothing",
			n:        tree.Map{"arr": tree.A()},
			rules:    QueryRules{".arr[].k": Required(String{})},
		},
		{
			caseName: "[] in the middle on an empty map fires nothing",
			n:        tree.Map{"m": tree.Map{}},
			rules:    QueryRules{".m[].k": Required(String{})},
		},
		{
			caseName: "filter after [] fires per element with no match",
			n:        tree.Map{"u": tree.A(tree.Map{"r": tree.A("admin")}, tree.Map{"r": tree.A("x")})},
			rules:    QueryRules{`.u[].r[. == "admin"]`: Required(String{})},
			wantErrs: []string{`.u[1].r[(. == "admin")]: required`},
		},
		{
			caseName: "walk after [] fires per element with no match",
			n:        tree.Map{"u": tree.A(tree.Map{"r": tree.A("admin")}, tree.Map{"r": tree.A("x")})},
			rules:    QueryRules{".u[]..k": Required(String{})},
			wantErrs: []string{".u[0]..k: required", ".u[1]..k: required"},
		},
		{
			caseName: "[] in the middle on a missing base fires once",
			n:        tree.Map{},
			rules:    QueryRules{".arr[].k": Required(String{})},
			wantErrs: []string{".arr[].k: required"},
		},
		{
			caseName: "[] in the middle on a null base fires once",
			n:        tree.Map{"arr": tree.Nil},
			rules:    QueryRules{".arr[].k": Required(String{})},
			wantErrs: []string{".arr[].k: required"},
		},
		{
			caseName: "[] under [] on an element without the key fires at the element",
			n:        tree.Map{"arr": tree.A(tree.Map{"k": tree.A("a")}, tree.Map{})},
			rules:    QueryRules{".arr[].k[]": Required(String{})},
			wantErrs: []string{".arr[1].k[]: required"},
		},
		{
			caseName: "[] after a key read as null on some elements keeps keyed paths",
			n:        tree.Map{"arr": tree.Array{tree.Map{"k": tree.Map{"a": tree.V(1)}}, tree.Map{}}},
			rules:    QueryRules{".arr[].k[]": String{}},
			wantErrs: []string{`.arr[0].k["a"]: expected string, got number`},
		},
		{
			caseName: "[] under [] keeps each parent's index",
			n:        tree.Map{"x": tree.A(tree.Map{"y": tree.A(1, "a")}, tree.Map{"y": tree.A("b", 2)})},
			rules:    QueryRules{".x[].y[]": String{}},
			wantErrs: []string{".x[0].y[0]: expected string", ".x[1].y[1]: expected string"},
		},
		{
			caseName: "[] after a walk indexes each match",
			n:        tree.Map{"x": tree.A(tree.Map{"y": tree.A(1, "a")}, tree.Map{"y": tree.A("b", 2)})},
			rules:    QueryRules{"..y[]": String{}},
			wantErrs: []string{"..y[0]: expected string", "..y[1]: expected string"},
		},
		{
			caseName: "[] after a pipe indexes the collected array",
			n:        tree.Map{"x": tree.V(1)},
			rules:    QueryRules{".x | .[]": String{}},
			wantErrs: []string{".x | .[0]: expected string"},
		},
		{
			caseName: "[] before a pipe does not expand",
			n:        tree.Map{"x": tree.A(1, 2)},
			rules:    QueryRules{".x[] | .": Array{MaxLen: IntPtr(1)}},
			wantErrs: []string{".x[] | .: "},
		},
		{
			caseName: "[] inside a group without a pipe expands",
			n:        tree.Map{"arr": tree.A(tree.Map{"k": tree.A(1)}, tree.Map{"k": tree.A(2)})},
			rules:    QueryRules{"(.arr[]).k[]": String{}},
			wantErrs: []string{".arr[0].k[0]: expected string", ".arr[1].k[0]: expected string"},
		},
		{
			caseName: "[] inside a group with a pipe does not expand",
			n:        tree.Map{"a": tree.Map{"k": tree.V(1)}},
			rules:    QueryRules{"(.a | .[]).k": String{}},
			wantErrs: []string{"(.a | .[]).k: expected string, got number"},
		},
		{
			caseName: "[] after a group with a pipe keeps the parentheses",
			n:        tree.Map{"a": tree.Map{"b": tree.Map{"c": tree.A(1)}}},
			rules:    QueryRules{"(.a | .[0].b).c[]": String{}},
			wantErrs: []string{"(.a | .[0].b).c[0]: expected string"},
		},
		{
			caseName: "[] after a slice keeps each element's own index",
			n:        tree.Map{"arr": tree.A(tree.A(1), tree.A(2))},
			rules:    QueryRules{".arr[0:2][]": String{}},
			wantErrs: []string{".arr[0:2][0]: expected string", ".arr[0:2][0]: expected string"},
		},
		{
			caseName: "[] on a nil Array fires once",
			n:        tree.Map{"arr": tree.Array(nil)},
			rules:    QueryRules{".arr[]": Required(String{})},
			wantErrs: []string{".arr[]: required"},
		},
		{
			caseName: "[] on a nil Map fires once",
			n:        tree.Map{"m": tree.Map(nil)},
			rules:    QueryRules{".m[]": Required(String{})},
			wantErrs: []string{".m[]: required"},
		},
		{
			caseName: "[] after a walk with only a null parent fires once",
			n:        tree.Map{"a": tree.Map{"tags": tree.Nil}},
			rules:    QueryRules{"..tags[]": Required(String{})},
			wantErrs: []string{"..tags[]: required"},
		},
		{
			caseName: "[] after a walk with a null parent beside a container fires nothing for it",
			n:        tree.Map{"a": tree.Map{"tags": tree.Nil}, "b": tree.Map{"tags": tree.A("x")}},
			rules:    QueryRules{"..tags[]": Required(String{})},
		},
		{
			caseName: "expanding rule uses one spelling when nothing expands",
			n:        tree.Map{},
			rules:    QueryRules{`."a"[]`: Required(String{})},
			wantErrs: []string{".a[]: required"},
		},
		{
			caseName: "[] inside a select does not expand",
			n:        tree.Map{"x": tree.A(tree.Map{"y": tree.A(1)})},
			rules:    QueryRules{".x[.y[] == 1]": String{}},
			wantErrs: []string{".x[.y[] == 1]: expected string"},
		},
		{
			caseName: "[][] with a null element leaves the document unchanged",
			n:        tree.Array{tree.Nil, tree.Array{tree.V("x")}},
			rules:    QueryRules{"[][]": String{}},
		},
		{
			caseName: "top-level [] with a leading space",
			n:        tree.A("ok", 1),
			rules:    QueryRules{" []": String{}},
			wantErrs: []string{"[1]: expected string"},
		},
		{
			caseName: "top-level [] on an array",
			n:        tree.A("ok", 1),
			rules:    QueryRules{"[]": String{}},
			wantErrs: []string{"[1]: expected string"},
		},
		{
			caseName: "top-level [] on a map",
			n:        tree.Map{"a": tree.V("x"), "b": tree.V(1)},
			rules:    QueryRules{"[]": String{}},
			wantErrs: []string{`["b"]: expected string`},
		},
		{
			caseName: "[] inside Every",
			n:        tree.Map{"arr": tree.A(tree.A("a", 1))},
			rules:    QueryRules{".arr": Every{Rules: QueryRules{"[]": String{}}}},
			wantErrs: []string{".arr[0][1]: expected string"},
		},
		{
			caseName: "key step on a scalar passes an optional rule",
			n:        tree.Map{"name": tree.V("alice")},
			rules:    QueryRules{".name.first": String{}},
		},
		{
			caseName: "key step on a scalar fires a required rule",
			n:        tree.Map{"name": tree.V("alice")},
			rules:    QueryRules{".name.first": Required(String{})},
			wantErrs: []string{".name.first: required"},
		},
		{
			caseName: "required string present",
			n:        tree.Map{"name": tree.V("alice")},
			rules:    QueryRules{".name": Required(String{})},
		},
		{
			caseName: "required string absent",
			n:        tree.Map{},
			rules:    QueryRules{".name": Required(String{})},
			wantErrs: []string{".name: required"},
		},
		{
			caseName: "array element errors carry concrete index",
			n:        tree.Map{"arr": tree.A("ok", 1, true)},
			rules:    QueryRules{".arr[]": String{}},
			wantErrs: []string{".arr[1]: expected string", ".arr[2]: expected string"},
		},
		{
			caseName: "map value errors carry concrete key",
			n: tree.Map{"labels": tree.Map{
				"admin": tree.V("x"),
				"guest": tree.V(1),
				"user":  tree.V(true),
			}},
			rules:    QueryRules{".labels[]": String{}},
			wantErrs: []string{`.labels["guest"]: expected string`, `.labels["user"]: expected string`},
		},
		{
			caseName: "map value error path is keyed not indexed",
			n:        tree.Map{"labels": tree.Map{"k": tree.V(1)}},
			rules:    QueryRules{".labels[]": String{}},
			wantErrs: []string{`.labels["k"]`},
		},
		{
			caseName: "errors collected across rules",
			n: tree.Map{
				"a": tree.V(1),
				"b": tree.V(true),
			},
			rules: QueryRules{
				".a": String{},
				".b": Int{},
			},
			wantErrs: []string{".a: expected string, got number", ".b: expected number, got boolean"},
		},
		{
			caseName: "nil map value under [] is null",
			n:        tree.Map{"m": tree.Map{"k": nil}},
			rules:    QueryRules{".m[]": Required(String{})},
			wantErrs: []string{`.m["k"]: required`},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			before := tree.CloneDeep(tc.n)
			err := Validate(tc.n, tc.rules)
			checkErrs(t, err, tc.wantErrs)
			if !tree.Equal(tc.n, before) {
				t.Errorf("Validate changed the document: got %v; want %v", tc.n, before)
			}
		})
	}
}

func TestValidateWithPrefix(t *testing.T) {
	testCases := []struct {
		caseName string
		n        tree.Node
		rules    QueryRules
		prefix   string
		wantErrs []string
	}{
		{
			caseName: "empty prefix matches Validate",
			n:        tree.Map{"name": tree.V(1)},
			rules:    QueryRules{".name": String{}},
			prefix:   "",
			wantErrs: []string{".name: expected string"},
		},
		{
			caseName: "prefix prepended to leaf error",
			n:        tree.Map{"name": tree.V(1)},
			rules:    QueryRules{".name": String{}},
			prefix:   `.handlers["c"]`,
			wantErrs: []string{`.handlers["c"].name: expected string`},
		},
		{
			caseName: "prefix prepended to required",
			n:        tree.Map{},
			rules:    QueryRules{".name": Required(String{})},
			prefix:   `.handlers["c"]`,
			wantErrs: []string{`.handlers["c"].name: required`},
		},
		{
			caseName: "prefix prepended to expanded array index",
			n:        tree.Map{"arr": tree.A("ok", 1)},
			rules:    QueryRules{".arr[]": String{}},
			prefix:   `.handlers["c"]`,
			wantErrs: []string{`.handlers["c"].arr[1]: expected string`},
		},
		{
			caseName: `self query "." under prefix drops to bare prefix`,
			n:        tree.V(1),
			rules:    QueryRules{".": String{}},
			prefix:   `.handlers["c"]`,
			wantErrs: []string{`.handlers["c"]: expected string`},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := ValidateWithPrefix(tc.n, tc.rules, tc.prefix)
			checkErrs(t, err, tc.wantErrs)
		})
	}
}

// dispatchRule mirrors the fasthttpd HandlerDispatch pattern: pick a
// QueryRules set based on the node's content, then recurse with
// ValidateWithPrefix so nested error paths are attributed to the
// enclosing query.
type dispatchRule struct {
	byType map[string]QueryRules
}

func (r dispatchRule) Validate(n tree.Node, q string) error {
	if n.IsNil() {
		return nil
	}
	typ := n.Get("type").Value().String()
	rules, ok := r.byType[typ]
	if !ok {
		return nil
	}
	return ValidateWithPrefix(n, rules, q)
}

func TestValidateWithPrefix_DispatchRule(t *testing.T) {
	rule := dispatchRule{byType: map[string]QueryRules{
		"http": {".port": Required(Int{})},
	}}
	n := tree.Map{"handlers": tree.Map{
		"c": tree.Map{"type": tree.V("http")},
	}}
	err := Validate(n, QueryRules{`.handlers[]`: rule})
	checkErrs(t, err, []string{`.handlers["c"].port: required`})
}

func TestValidate_ErrQuery(t *testing.T) {
	testCases := []struct {
		caseName  string
		n         tree.Node
		rules     QueryRules
		wantQuery string
	}{
		{
			caseName:  "malformed top-level query fails fast",
			n:         tree.Map{"ok": tree.V("x")},
			rules:     QueryRules{".ok": String{}, ".bad[": String{}},
			wantQuery: ".bad[",
		},
		{
			caseName:  "ErrQuery suppresses sibling validation errors",
			n:         tree.Map{"ok": tree.V(1)}, // would fail String{} normally
			rules:     QueryRules{".ok": String{}, ".bad[": String{}},
			wantQuery: ".bad[",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := Validate(tc.n, tc.rules)
			if err == nil {
				t.Fatalf("got nil; want *ErrQuery")
			}
			var eq *ErrQuery
			if !errors.As(err, &eq) {
				t.Fatalf("got %T (%v); want *ErrQuery", err, err)
			}
			if eq.Query != tc.wantQuery {
				t.Errorf("Query = %q; want %q", eq.Query, tc.wantQuery)
			}
			if eq.Err == nil {
				t.Error("Unwrap returned nil; want underlying tree error")
			}
			// Sibling String{} violation on .ok must NOT appear when ErrQuery fires.
			if strings.Contains(err.Error(), "expected string") {
				t.Errorf("ErrQuery should suppress sibling errors; got %q", err.Error())
			}
		})
	}
}

func TestValidateWithPrefix_ErrQueryCarriesPrefix(t *testing.T) {
	err := ValidateWithPrefix(tree.Map{}, QueryRules{".bad[": String{}}, `.handlers["c"]`)
	var eq *ErrQuery
	if !errors.As(err, &eq) {
		t.Fatalf("got %T (%v); want *ErrQuery", err, err)
	}
	if want := `.handlers["c"].bad[`; eq.Query != want {
		t.Errorf("Query = %q; want %q", eq.Query, want)
	}
}

func TestEvery_ErrQueryFailsFast(t *testing.T) {
	rule := Every{Rules: QueryRules{".bad[": String{}}}
	n := tree.Map{"arr": tree.A(
		tree.Map{"x": tree.V(1)},
		tree.Map{"x": tree.V(2)},
		tree.Map{"x": tree.V(3)},
	)}
	err := Validate(n, QueryRules{".arr": rule})
	var eq *ErrQuery
	if !errors.As(err, &eq) {
		t.Fatalf("got %T (%v); want *ErrQuery", err, err)
	}
	// Same malformed rule fires on every element; fail-fast must return only one.
	if got := strings.Count(err.Error(), "query "); got != 1 {
		t.Errorf("query-error count = %d; want 1 (fail-fast)", got)
	}
}

// checkErrs asserts that err is nil when wantErrs is empty, or reports exactly len(wantErrs) lines, one per violation, each containing its substring.
func checkErrs(t *testing.T, err error, wantErrs []string) {
	t.Helper()
	if len(wantErrs) == 0 {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("got nil; want error containing %v", wantErrs)
	}
	got := strings.Split(err.Error(), "\n")
	if len(got) != len(wantErrs) {
		t.Fatalf("got %d errors %q; want %d", len(got), got, len(wantErrs))
	}
	for i, want := range wantErrs {
		if !strings.Contains(got[i], want) {
			t.Errorf("error %d %q missing %q", i, got[i], want)
		}
	}
}
