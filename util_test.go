package tree

import (
	"reflect"
	"testing"
)

func TestV(t *testing.T) {
	// V is a thin alias for ToValue; sample a few representative
	// inputs to confirm they yield identical results.
	testCases := []struct {
		caseName string
		in       any
	}{
		{caseName: "nil", in: nil},
		{caseName: "string", in: "s"},
		{caseName: "bool", in: true},
		{caseName: "int", in: 1},
		{caseName: "int64", in: int64(2)},
		{caseName: "float64", in: 3.5},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got, want := V(tc.in), ToValue(tc.in); got != want {
				t.Errorf("V(%v) = %#v; want %#v", tc.in, got, want)
			}
		})
	}
}

func TestA(t *testing.T) {
	// A is a thin alias for ToArrayValues.
	got := A("a", 1, true)
	want := ToArrayValues("a", 1, true)
	if !Equal(got, want) {
		t.Errorf("A(...) = %#v; want %#v", got, want)
	}
}

func TestToValue(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     Node
	}{
		{caseName: "nil", v: nil, want: Nil},
		{caseName: "string", v: "string", want: StringValue("string")},
		{caseName: "bool", v: true, want: BoolValue(true)},
		{caseName: "int", v: 1, want: NumberValue(1)},
		{caseName: "int64", v: int64(2), want: NumberValue(2)},
		{caseName: "int32", v: int32(3), want: NumberValue(3)},
		{caseName: "float64", v: float64(4.4), want: NumberValue(4.4)},
		{caseName: "float32", v: float32(5.5), want: NumberValue(5.5)},
		{caseName: "uint64", v: uint64(6), want: NumberValue(uint64(6))},
		{caseName: "uint32", v: uint32(7), want: NumberValue(uint32(7))},
		{caseName: "BoolValue passthrough", v: BoolValue(true), want: BoolValue(true)},
		{caseName: "unknown struct fallback", v: struct{}{}, want: StringValue("struct {}{}")},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := ToValue(tc.v); !Equal(got, tc.want) {
				t.Errorf("for %v; got %#v; want %#v", tc.v, got, tc.want)
			}
		})
	}
}

func TestToNode(t *testing.T) {
	testCases := []struct {
		caseName string
		v        any
		want     Node
	}{
		{caseName: "nil", v: nil, want: Nil},
		{caseName: "StringValue", v: StringValue("a"), want: StringValue("a")},
		{
			caseName: "map",
			v:        map[string]any{"a": 1, "b": true},
			want:     Map{"a": NumberValue(1), "b": BoolValue(true)},
		},
		{
			caseName: "slice",
			v:        []any{"a", true, 1},
			want:     Array{StringValue("a"), BoolValue(true), NumberValue(1)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := ToNode(tc.v); !Equal(got, tc.want) {
				t.Errorf("for %v; got %v; want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestWalk(t *testing.T) {
	root := Array{
		Map{"ID": ToValue(1)},
		Map{"ID": ToValue(2), "Sub": Array{Map{"ID": ToValue(20)}}},
		Map{"ID": ToValue(3), "Sub": Array{Map{"ID": ToValue(30)}}},
	}

	expected := []struct {
		n    Node
		keys []any
		skip bool
	}{
		{
			n:    root,
			keys: []any{},
		}, {
			n:    root.Get(0),
			keys: []any{0},
		}, {
			n:    root.Get(0).Get("ID"),
			keys: []any{0, "ID"},
		}, {
			n:    root.Get(1),
			keys: []any{1},
			skip: true,
		}, {
			n:    root.Get(2),
			keys: []any{2},
		}, {
			n:    root.Get(2).Get("ID"),
			keys: []any{2, "ID"},
		}, {
			n:    root.Get(2).Get("Sub"),
			keys: []any{2, "Sub"},
		}, {
			n:    root.Get(2).Get("Sub").Get(0),
			keys: []any{2, "Sub", 0},
		}, {
			n:    root.Get(2).Get("Sub").Get(0).Get("ID"),
			keys: []any{2, "Sub", 0, "ID"},
		},
	}

	i := 0
	err := Walk(root, func(n Node, keys []any) error {
		if i >= len(expected) {
			t.Fatalf("fn is called too many times %d", i)
			return nil
		}
		want := expected[i]
		i++

		if !Equal(n, want.n) {
			t.Errorf("walk[%d] got %#v; want %#v", i, n, want.n)
		}
		if !reflect.DeepEqual(keys, want.keys) {
			t.Errorf("walk[%d] got %#v; want %#v", i, keys, want.keys)
		}
		if want.skip {
			return SkipWalk
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != i {
		t.Errorf("fn is called %d times; want %d", i, len(expected))
	}
}

func TestRegexpMatchString(t *testing.T) {
	testCases := []struct {
		caseName string
		expr     string
		value    string
		want     bool
		errstr   string
	}{
		{
			caseName: "substring match",
			expr:     `a`,
			value:    "abc",
			want:     true,
		}, {
			caseName: "anchored full match",
			expr:     `^[a-z]+$`,
			value:    "abc",
			want:     true,
		}, {
			caseName: "no match",
			expr:     `x`,
			value:    "abc",
			want:     false,
		}, {
			caseName: "invalid regex",
			expr:     `(`,
			value:    "abc",
			errstr:   "error parsing regexp: missing closing ): `(`",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := regexpMatchString(tc.expr, tc.value)
			if tc.errstr != "" {
				if err == nil {
					t.Fatalf("for %v; no error", tc.expr)
				}
				if err.Error() != tc.errstr {
					t.Errorf("for %v; got %v want %v", tc.expr, err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("for %v; %+v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("for %v; got %v; want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestClone(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     Node
	}{
		{caseName: "Value", n: ToValue(1), want: ToValue(1)},
		{
			caseName: "Map",
			n:        Map{"a": ToValue(1), "b": ToValue(2)},
			want:     Map{"a": ToValue(1), "b": ToValue(2)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := Clone(tc.n); !Equal(got, tc.want) {
				t.Errorf("unexpected %v; want %v", got, tc.want)
			}
		})
	}
}

func TestCloneDeep(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     Node
		update   func(n Node)
	}{
		{
			caseName: "Value",
			n:        ToValue(1),
			want:     ToValue(1),
			update:   func(n Node) {},
		}, {
			caseName: "Map of Arrays mutated post-clone",
			n:        Map{"a": ToArrayValues(1, 2), "b": ToArrayValues(3, 4)},
			want:     Map{"a": ToArrayValues(1, 2), "b": ToArrayValues(3, 4)},
			update: func(n Node) {
				n.Map().Get("a").Array()[0] = ToValue(5)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := CloneDeep(tc.n)
			if !Equal(got, tc.want) {
				t.Errorf("unexpected %v; want %v", got, tc.want)
			}
			tc.update(tc.n)
			if !Equal(got, tc.want) {
				t.Errorf("after update unexpected %v; want %v", got, tc.want)
			}
		})
	}
}
