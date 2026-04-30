package tree

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestType(t *testing.T) {
	testCases := []struct {
		caseName string
		is       func() bool
		want     bool
	}{
		{"TypeArray.IsArray", TypeArray.IsArray, true},
		{"TypeArray.IsMap", TypeArray.IsMap, false},
		{"TypeArray.IsValue", TypeArray.IsValue, false},
		{"TypeMap.IsArray", TypeMap.IsArray, false},
		{"TypeMap.IsMap", TypeMap.IsMap, true},
		{"TypeMap.IsValue", TypeMap.IsValue, false},
		{"TypeValue.IsArray", TypeValue.IsArray, false},
		{"TypeValue.IsMap", TypeValue.IsMap, false},
		{"TypeValue.IsValue", TypeValue.IsValue, true},
		{"TypeValue.IsNilValue", TypeValue.IsNilValue, false},
		{"TypeValue.IsStringValue", TypeValue.IsStringValue, false},
		{"TypeValue.IsBoolValue", TypeValue.IsBoolValue, false},
		{"TypeValue.IsNumberValue", TypeValue.IsNumberValue, false},
		{"TypeNilValue.IsArray", TypeNilValue.IsArray, false},
		{"TypeNilValue.IsMap", TypeNilValue.IsMap, false},
		{"TypeNilValue.IsValue", TypeNilValue.IsValue, true},
		{"TypeNilValue.IsNilValue", TypeNilValue.IsNilValue, true},
		{"TypeNilValue.IsStringValue", TypeNilValue.IsStringValue, false},
		{"TypeNilValue.IsBoolValue", TypeNilValue.IsBoolValue, false},
		{"TypeNilValue.IsNumberValue", TypeNilValue.IsNumberValue, false},
		{"TypeStringValue.IsArray", TypeStringValue.IsArray, false},
		{"TypeStringValue.IsMap", TypeStringValue.IsMap, false},
		{"TypeStringValue.IsValue", TypeStringValue.IsValue, true},
		{"TypeStringValue.IsNilValue", TypeStringValue.IsNilValue, false},
		{"TypeStringValue.IsStringValue", TypeStringValue.IsStringValue, true},
		{"TypeStringValue.IsBoolValue", TypeStringValue.IsBoolValue, false},
		{"TypeStringValue.IsNumberValue", TypeStringValue.IsNumberValue, false},
		{"TypeBoolValue.IsArray", TypeBoolValue.IsArray, false},
		{"TypeBoolValue.IsMap", TypeBoolValue.IsMap, false},
		{"TypeBoolValue.IsValue", TypeBoolValue.IsValue, true},
		{"TypeBoolValue.IsNilValue", TypeBoolValue.IsNilValue, false},
		{"TypeBoolValue.IsStringValue", TypeBoolValue.IsStringValue, false},
		{"TypeBoolValue.IsBoolValue", TypeBoolValue.IsBoolValue, true},
		{"TypeBoolValue.IsNumberValue", TypeBoolValue.IsNumberValue, false},
		{"TypeNumberValue.IsArray", TypeNumberValue.IsArray, false},
		{"TypeNumberValue.IsMap", TypeNumberValue.IsMap, false},
		{"TypeNumberValue.IsValue", TypeNumberValue.IsValue, true},
		{"TypeNumberValue.IsNilValue", TypeNumberValue.IsNilValue, false},
		{"TypeNumberValue.IsStringValue", TypeNumberValue.IsStringValue, false},
		{"TypeNumberValue.IsBoolValue", TypeNumberValue.IsBoolValue, false},
		{"TypeNumberValue.IsNumberValue", TypeNumberValue.IsNumberValue, true},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := tc.is(); got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestTypeString(t *testing.T) {
	testCases := []struct {
		caseName string
		typ      Type
		want     string
	}{
		{caseName: "array", typ: TypeArray, want: "array"},
		{caseName: "map", typ: TypeMap, want: "map"},
		{caseName: "value bitmask", typ: TypeValue, want: "value"},
		{caseName: "nil", typ: TypeNilValue, want: "nil"},
		{caseName: "string", typ: TypeStringValue, want: "string"},
		{caseName: "bool", typ: TypeBoolValue, want: "bool"},
		{caseName: "number", typ: TypeNumberValue, want: "number"},
		{caseName: "zero unknown", typ: Type(0), want: "unknown"},
		{caseName: "unknown bit pattern", typ: Type(0b1111), want: "unknown"},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := tc.typ.String(); got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestNode(t *testing.T) {
	testCases := []struct {
		caseName  string
		n         Node
		isNil     bool
		t         Type
		a         Array
		m         Map
		v         Value
		hasKeys   []any
		hasValue  bool
		getKeys   []any
		getValue  Node
		findExpr  string
		findValue []Node
	}{
		{
			caseName: "Map(nil)",
			n:        Map(nil),
			isNil:    true,
			t:        TypeMap,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			caseName: "Map empty",
			n:        Map{},
			t:        TypeMap,
			m:        Map{},
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			caseName: "Array(nil)",
			n:        Array(nil),
			isNil:    true,
			t:        TypeArray,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			caseName: "Array empty",
			n:        Array{},
			t:        TypeArray,
			m:        Map(nil),
			a:        Array{},
			v:        Nil,
			getValue: Nil,
		}, {
			caseName: "Nil",
			n:        Nil,
			isNil:    true,
			t:        TypeNilValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			caseName: "StringValue",
			n:        StringValue("a"),
			t:        TypeStringValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        StringValue("a"),
			getValue: Nil,
		}, {
			caseName: "BoolValue",
			n:        BoolValue(true),
			t:        TypeBoolValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        BoolValue(true),
			getValue: Nil,
		}, {
			caseName: "NumberValue",
			n:        NumberValue(1),
			t:        TypeNumberValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        NumberValue(1),
			getValue: Nil,
		}, {
			caseName: "Any wrapping Map(nil)",
			n:        Any{Map(nil)},
			isNil:    true,
			t:        TypeMap,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			n := tc.n
			if n.IsNil() != tc.isNil {
				t.Errorf("IsNil got %v; want %v", n.IsNil(), tc.isNil)
			}
			if tt := n.Type(); tt != tc.t {
				t.Errorf("Type got %v; want %v", tt, tc.t)
			}
			if aa := n.Array(); !Equal(aa, tc.a) {
				t.Errorf("Array got %v; want %v", aa, tc.a)
			}
			if mm := n.Map(); !Equal(mm, tc.m) {
				t.Errorf("Map got %v; want %v", mm, tc.m)
			}
			if vv := n.Value(); !Equal(vv, tc.v) {
				t.Errorf("Value got %v; want %v", vv, tc.v)
			}
			if had := n.Has(tc.hasKeys...); had != tc.hasValue {
				t.Errorf("Has got %v; want %v", had, tc.hasValue)
			}
			if got := n.Get(tc.getKeys...); !Equal(got, tc.getValue) {
				t.Errorf("Get got %v; want %v", got, tc.getValue)
			}
			found, err := n.Find(tc.findExpr)
			if err != nil {
				t.Errorf("failed to Find error %v", err)
			}
			if !Equal(Array(found), Array(tc.findValue)) {
				t.Errorf("Find got %v; want %v", found, tc.findValue)
			}
		})
	}
}

func TestNodeGet(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		keys     []any
		has      bool
		want     Node
	}{
		{
			caseName: "Array int key",
			n:        Array{StringValue("a"), StringValue("b")},
			keys:     []any{1},
			has:      true,
			want:     StringValue("b"),
		}, {
			caseName: "Array string-as-int key",
			n:        Array{StringValue("a"), StringValue("b")},
			keys:     []any{"1"},
			has:      true,
			want:     StringValue("b"),
		}, {
			caseName: "Array float key invalid",
			n:        Array{StringValue("a"), StringValue("b")},
			keys:     []any{1.0},
			want:     Nil,
		}, {
			caseName: "Array out of range",
			n:        Array{StringValue("a"), StringValue("b")},
			keys:     []any{2},
			want:     Nil,
		}, {
			caseName: "Array nil element",
			n:        Array{StringValue("a"), nil},
			keys:     []any{1},
			has:      true,
			want:     Nil,
		}, {
			caseName: "Map string key",
			n:        Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys:     []any{"1"},
			has:      true,
			want:     NumberValue(10),
		}, {
			caseName: "Map int key",
			n:        Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys:     []any{1},
			has:      true,
			want:     NumberValue(10),
		}, {
			caseName: "Map float key invalid",
			n:        Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys:     []any{1.0},
			want:     Nil,
		}, {
			caseName: "Map missing key",
			n:        Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys:     []any{"3"},
			want:     Nil,
		}, {
			caseName: "Map nil value",
			n:        Map{"1": NumberValue(10), "2": nil},
			keys:     []any{"2"},
			has:      true,
			want:     Nil,
		}, {
			caseName: "Map nested two keys",
			n:        Map{"a": Map{"b": StringValue("v")}},
			keys:     []any{"a", "b"},
			has:      true,
			want:     StringValue("v"),
		}, {
			caseName: "Map nested over-deep",
			n:        Map{"a": Map{"b": StringValue("v")}},
			keys:     []any{"a", "b", "c", "d"},
			want:     Nil,
		}, {
			caseName: "Map nested missing leaf",
			n:        Map{"a": Map{"b": StringValue("v")}},
			keys:     []any{"a", "c"},
			want:     Nil,
		}, {
			caseName: "Array nested two indices",
			n:        Array{Array{nil, StringValue("v")}},
			keys:     []any{0, 1},
			has:      true,
			want:     StringValue("v"),
		}, {
			caseName: "Array nested over-deep",
			n:        Array{Array{nil, StringValue("v")}},
			keys:     []any{0, 1, 2, 3},
			want:     Nil,
		}, {
			caseName: "mixed Array Map Array Map deep",
			n:        Array{Map{"a": Array{nil, Map{"b": StringValue("v")}}}},
			keys:     []any{0, "a", 1, "b"},
			has:      true,
			want:     StringValue("v"),
		}, {
			caseName: "StringValue no keys",
			n:        StringValue("str"),
			want:     Nil,
		}, {
			caseName: "BoolValue no keys",
			n:        BoolValue(true),
			want:     Nil,
		}, {
			caseName: "NumberValue no keys",
			n:        NumberValue(1),
			want:     Nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if tc.n.Has(tc.keys...) != tc.has {
				t.Errorf("Has got %v; want %v", !tc.has, tc.has)
			}
			got := tc.n.Get(tc.keys...)
			if !Equal(got, tc.want) {
				t.Errorf("Get got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestNodeEach(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		want     map[any]Node
	}{
		{
			caseName: "Array",
			n:        Array{StringValue("a"), StringValue("b")},
			want:     map[any]Node{0: StringValue("a"), 1: StringValue("b")},
		}, {
			caseName: "Map",
			n:        Map{"a": NumberValue(0), "b": NumberValue(1)},
			want:     map[any]Node{"a": NumberValue(0), "b": NumberValue(1)},
		}, {
			caseName: "StringValue",
			n:        StringValue("str"),
			want:     map[any]Node{nil: StringValue("str")},
		}, {
			caseName: "BoolValue",
			n:        BoolValue(true),
			want:     map[any]Node{nil: BoolValue(true)},
		}, {
			caseName: "NumberValue",
			n:        NumberValue(1),
			want:     map[any]Node{nil: NumberValue(1)},
		}, {
			caseName: "Any wrapping Map",
			n:        Any{Map{"a": NumberValue(0), "b": NumberValue(1)}},
			want:     map[any]Node{"a": NumberValue(0), "b": NumberValue(1)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got := map[any]Node{}
			err := tc.n.Each(func(key any, v Node) error {
				got[key] = v
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v; want %v", got, tc.want)
			}
			wantErr := fmt.Errorf("err for %s", tc.caseName)
			gotErr := tc.n.Each(func(key any, v Node) error {
				return wantErr
			})
			if wantErr != gotErr {
				t.Errorf("got %v; want %v", gotErr, wantErr)
			}
		})
	}
}

func TestNodeFind(t *testing.T) {
	testCases := []struct {
		caseName string
		n        Node
		expr     string
		want     []Node
	}{
		{
			caseName: "Array index",
			n:        Array{StringValue("a"), StringValue("b")},
			expr:     ".[0]",
			want:     []Node{StringValue("a")},
		}, {
			caseName: "Map key",
			n:        Map{"1": NumberValue(10), "2": NumberValue(20)},
			expr:     ".1",
			want:     []Node{NumberValue(10)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			got, err := tc.n.Find(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(Array(got), Array(tc.want)) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestEditorNodeAppend(t *testing.T) {
	testCases := []struct {
		caseName string
		n        EditorNode
		values   []Node
		want     EditorNode
		errstr   string
	}{
		{
			caseName: "Array",
			n:        &Array{NumberValue(1)},
			values:   []Node{StringValue("2"), BoolValue(true)},
			want:     &Array{NumberValue(1), StringValue("2"), BoolValue(true)},
		}, {
			caseName: "Map rejected",
			n:        Map{},
			values:   []Node{StringValue("2")},
			errstr:   "cannot append to map",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var err error
			for _, value := range tc.values {
				err = tc.n.Append(value)
				if err != nil {
					break
				}
			}
			if tc.errstr != "" {
				if err == nil {
					t.Fatal("no error")
				}
				if err.Error() != tc.errstr {
					t.Errorf("got %s; want %s", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := tc.n
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestEditorNodeSet(t *testing.T) {
	testCases := []struct {
		caseName string
		n        EditorNode
		entries  map[any]Node
		want     EditorNode
		errstr   string
	}{
		{
			caseName: "Array set existing and append",
			n:        &Array{NumberValue(0), StringValue("1")},
			entries: map[any]Node{
				0:   NumberValue(1),
				"1": StringValue("2"),
				2:   BoolValue(true),
			},
			want: &Array{NumberValue(1), StringValue("2"), BoolValue(true)},
		}, {
			caseName: "Array negative index rejected",
			n:        &Array{},
			entries:  map[any]Node{-2: StringValue("value")},
			errstr:   "cannot index array with -2",
		}, {
			caseName: "Map set and add new keys",
			n: Map{
				"1": NumberValue(1),
				"2": StringValue("2"),
				"3": BoolValue(true),
			},
			entries: map[any]Node{
				"1": NumberValue(10),
				"4": StringValue("40"),
				5:   BoolValue(true),
			},
			want: Map{
				"1": NumberValue(10),
				"2": StringValue("2"),
				"3": BoolValue(true),
				"4": StringValue("40"),
				"5": BoolValue(true),
			},
		}, {
			caseName: "Map bool key rejected",
			n:        Map{},
			entries:  map[any]Node{true: StringValue("value")},
			errstr:   "cannot index array with true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var err error
			for key, value := range tc.entries {
				err = tc.n.Set(key, value)
				if err != nil {
					break
				}
			}
			if tc.errstr != "" {
				if err == nil {
					t.Fatal("no error")
				}
				if err.Error() != tc.errstr {
					t.Errorf("got %s; want %s", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := tc.n
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestEditorNodeDelete(t *testing.T) {
	testCases := []struct {
		caseName string
		n        EditorNode
		keys     []any
		want     EditorNode
		errstr   string
	}{
		{
			caseName: "Array delete by int and string-as-int",
			n:        &Array{NumberValue(1), StringValue("1"), BoolValue(true)},
			keys:     []any{1, "1"},
			want:     &Array{NumberValue(1)},
		}, {
			caseName: "Array negative index rejected",
			n:        &Array{},
			keys:     []any{-1},
			errstr:   "cannot index array with -1",
		}, {
			caseName: "Map delete mixed keys",
			n: Map{
				"1": NumberValue(1),
				"2": StringValue("2"),
				"3": BoolValue(true),
				"4": StringValue("4"),
				"5": BoolValue(true),
			},
			keys: []any{"2", "4", 5, 7},
			want: Map{
				"1": NumberValue(1),
				"3": BoolValue(true),
			},
		}, {
			caseName: "Map bool key rejected",
			n:        Map{},
			keys:     []any{true},
			errstr:   "cannot index array with true",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			var err error
			for _, key := range tc.keys {
				err = tc.n.Delete(key)
				if err != nil {
					break
				}
			}
			if tc.errstr != "" {
				if err == nil {
					t.Fatal("no error")
				}
				if err.Error() != tc.errstr {
					t.Errorf("got %s; want %s", err.Error(), tc.errstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := tc.n
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	nan := NumberValue(math.NaN())

	testCases := []struct {
		caseName string
		a, b     Node
		want     bool
	}{
		// Untyped nil and Nil sentinel
		{"nil and nil", nil, nil, true},
		{"nil and Nil", nil, Nil, true},
		{"Nil and Nil", Nil, Nil, true},

		// Scalars: same type, same value
		{"StringValue same", StringValue("x"), StringValue("x"), true},
		{"StringValue different", StringValue("x"), StringValue("y"), false},
		{"NumberValue same", NumberValue(1), NumberValue(1), true},
		{"NumberValue different", NumberValue(1), NumberValue(2), false},
		{"NumberValue int and float same value", NumberValue(1), NumberValue(1.0), true},
		{"BoolValue true and true", BoolValue(true), BoolValue(true), true},
		{"BoolValue false and false", BoolValue(false), BoolValue(false), true},
		{"BoolValue different", BoolValue(true), BoolValue(false), false},

		// Type mismatches
		{"StringValue and NumberValue", StringValue("1"), NumberValue(1), false},
		{"BoolValue and StringValue", BoolValue(true), StringValue("true"), false},
		{"Nil and Map(nil)", Nil, Map(nil), false},
		{"Nil and Array(nil)", Nil, Array(nil), false},
		{"Map empty and Array empty", Map{}, Array{}, false},

		// NaN: IEEE 754 semantics
		{"NaN and NaN", nan, nan, false},
		{"NaN and Number", nan, NumberValue(1), false},

		// Map equality: order-independent, nil/empty equivalence
		{"Map empty and empty", Map{}, Map{}, true},
		{"Map nil and empty", Map(nil), Map{}, true},
		{"Map same single entry", Map{"a": NumberValue(1)}, Map{"a": NumberValue(1)}, true},
		{
			"Map same two entries same order",
			Map{"a": NumberValue(1), "b": NumberValue(2)},
			Map{"a": NumberValue(1), "b": NumberValue(2)},
			true,
		},
		{
			"Map same two entries different order",
			Map{"a": NumberValue(1), "b": NumberValue(2)},
			Map{"b": NumberValue(2), "a": NumberValue(1)},
			true,
		},
		{
			"Map different value",
			Map{"a": NumberValue(1)},
			Map{"a": NumberValue(2)},
			false,
		},
		{
			"Map different keys",
			Map{"a": NumberValue(1)},
			Map{"b": NumberValue(1)},
			false,
		},
		{
			"Map different size",
			Map{"a": NumberValue(1)},
			Map{"a": NumberValue(1), "b": NumberValue(2)},
			false,
		},

		// Array equality: order-dependent, nil/empty equivalence
		{"Array empty and empty", Array{}, Array{}, true},
		{"Array nil and empty", Array(nil), Array{}, true},
		{
			"Array same",
			Array{NumberValue(1), NumberValue(2)},
			Array{NumberValue(1), NumberValue(2)},
			true,
		},
		{
			"Array different order",
			Array{NumberValue(1), NumberValue(2)},
			Array{NumberValue(2), NumberValue(1)},
			false,
		},
		{
			"Array different size",
			Array{NumberValue(1)},
			Array{NumberValue(1), NumberValue(2)},
			false,
		},

		// Nested structures
		{
			"Map with nested Map equal",
			Map{"a": Map{"b": NumberValue(1)}},
			Map{"a": Map{"b": NumberValue(1)}},
			true,
		},
		{
			"Map with nested Map different",
			Map{"a": Map{"b": NumberValue(1)}},
			Map{"a": Map{"b": NumberValue(2)}},
			false,
		},
		{
			"Array of Map equal",
			Array{Map{"a": NumberValue(1)}},
			Array{Map{"a": NumberValue(1)}},
			true,
		},

		// nil child entries inside containers
		{"Map with nil value", Map{"a": nil}, Map{"a": nil}, true},
		{"Map nil value and Nil value", Map{"a": nil}, Map{"a": Nil}, true},
		{"Array with nil element", Array{nil}, Array{nil}, true},
		{"Array nil element and Nil element", Array{nil}, Array{Nil}, true},

		// Any wrapper unwrapping
		{
			"Any wraps Map equals raw Map",
			Any{Node: Map{"a": NumberValue(1)}},
			Map{"a": NumberValue(1)},
			true,
		},
		{
			"Any and Any same",
			Any{Node: NumberValue(1)},
			Any{Node: NumberValue(1)},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.caseName, func(t *testing.T) {
			if got := Equal(tc.a, tc.b); got != tc.want {
				t.Errorf("Equal(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.want)
			}
			if got := Equal(tc.b, tc.a); got != tc.want {
				t.Errorf("Equal(%v, %v) = %v; want %v (symmetry)", tc.b, tc.a, got, tc.want)
			}
		})
	}
}
