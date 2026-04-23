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
		wantErrs []string // all substrings must appear; empty means no error
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
			wantErrs: []string{".a: expected string, got number", ".b: expected number, got bool"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := Validate(tc.n, tc.rules)
			checkErrs(t, err, tc.wantErrs)
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
				t.Error("Unwrap returned nil; want underlying tree.Find error")
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

// checkErrs asserts that err either is nil (when wantErrs is empty) or
// contains every substring in wantErrs. Shared by the test files in
// this package.
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
	for _, want := range wantErrs {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}
