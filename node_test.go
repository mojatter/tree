package tree

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func Test_Type(t *testing.T) {
	tests := []struct {
		typ  Type
		is   func() bool
		want bool
	}{
		{typ: TypeArray, is: TypeArray.IsArray, want: true},
		{typ: TypeArray, is: TypeArray.IsMap, want: false},
		{typ: TypeArray, is: TypeArray.IsValue, want: false},
		{typ: TypeMap, is: TypeMap.IsArray, want: false},
		{typ: TypeMap, is: TypeMap.IsMap, want: true},
		{typ: TypeMap, is: TypeMap.IsValue, want: false},
		{typ: TypeValue, is: TypeValue.IsArray, want: false},
		{typ: TypeValue, is: TypeValue.IsMap, want: false},
		{typ: TypeValue, is: TypeValue.IsValue, want: true},
		{typ: TypeValue, is: TypeValue.IsNilValue, want: false},
		{typ: TypeValue, is: TypeValue.IsStringValue, want: false},
		{typ: TypeValue, is: TypeValue.IsBoolValue, want: false},
		{typ: TypeValue, is: TypeValue.IsNumberValue, want: false},
		{typ: TypeNilValue, is: TypeNilValue.IsArray, want: false},
		{typ: TypeNilValue, is: TypeNilValue.IsMap, want: false},
		{typ: TypeNilValue, is: TypeNilValue.IsValue, want: true},
		{typ: TypeNilValue, is: TypeNilValue.IsNilValue, want: true},
		{typ: TypeNilValue, is: TypeNilValue.IsStringValue, want: false},
		{typ: TypeNilValue, is: TypeNilValue.IsBoolValue, want: false},
		{typ: TypeNilValue, is: TypeNilValue.IsNumberValue, want: false},
		{typ: TypeStringValue, is: TypeStringValue.IsArray, want: false},
		{typ: TypeStringValue, is: TypeStringValue.IsMap, want: false},
		{typ: TypeStringValue, is: TypeStringValue.IsValue, want: true},
		{typ: TypeStringValue, is: TypeStringValue.IsNilValue, want: false},
		{typ: TypeStringValue, is: TypeStringValue.IsStringValue, want: true},
		{typ: TypeStringValue, is: TypeStringValue.IsBoolValue, want: false},
		{typ: TypeStringValue, is: TypeStringValue.IsNumberValue, want: false},
		{typ: TypeBoolValue, is: TypeBoolValue.IsArray, want: false},
		{typ: TypeBoolValue, is: TypeBoolValue.IsMap, want: false},
		{typ: TypeBoolValue, is: TypeBoolValue.IsValue, want: true},
		{typ: TypeBoolValue, is: TypeBoolValue.IsNilValue, want: false},
		{typ: TypeBoolValue, is: TypeBoolValue.IsStringValue, want: false},
		{typ: TypeBoolValue, is: TypeBoolValue.IsBoolValue, want: true},
		{typ: TypeBoolValue, is: TypeBoolValue.IsNumberValue, want: false},
		{typ: TypeNumberValue, is: TypeNumberValue.IsArray, want: false},
		{typ: TypeNumberValue, is: TypeNumberValue.IsMap, want: false},
		{typ: TypeNumberValue, is: TypeNumberValue.IsValue, want: true},
		{typ: TypeNumberValue, is: TypeNumberValue.IsNilValue, want: false},
		{typ: TypeNumberValue, is: TypeNumberValue.IsStringValue, want: false},
		{typ: TypeNumberValue, is: TypeNumberValue.IsBoolValue, want: false},
		{typ: TypeNumberValue, is: TypeNumberValue.IsNumberValue, want: true},
	}
	for i, test := range tests {
		if got := test.is(); got != test.want {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
	}
}

func Test_Type_String(t *testing.T) {
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

func Test_Node(t *testing.T) {
	tests := []struct {
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
			n:        Map(nil),
			isNil:    true,
			t:        TypeMap,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			n:        Map{},
			t:        TypeMap,
			m:        Map{},
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			n:        Array(nil),
			isNil:    true,
			t:        TypeArray,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			n:        Array{},
			t:        TypeArray,
			m:        Map(nil),
			a:        Array{},
			v:        Nil,
			getValue: Nil,
		}, {
			n:        Nil,
			isNil:    true,
			t:        TypeNilValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		}, {
			n:        StringValue("a"),
			t:        TypeStringValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        StringValue("a"),
			getValue: Nil,
		}, {
			n:        BoolValue(true),
			t:        TypeBoolValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        BoolValue(true),
			getValue: Nil,
		}, {
			n:        NumberValue(1),
			t:        TypeNumberValue,
			m:        Map(nil),
			a:        Array(nil),
			v:        NumberValue(1),
			getValue: Nil,
		}, {
			n:        Any{Map(nil)},
			isNil:    true,
			t:        TypeMap,
			m:        Map(nil),
			a:        Array(nil),
			v:        Nil,
			getValue: Nil,
		},
	}
	for i, test := range tests {
		n := test.n
		if n.IsNil() != test.isNil {
			t.Errorf("tests[%d] IsNil got %v; want %v", i, n.IsNil(), test.isNil)
		}
		if tt := n.Type(); tt != test.t {
			t.Errorf("tests[%d] Type got %v; want %v", i, tt, test.t)
		}
		if aa := n.Array(); !reflect.DeepEqual(aa, test.a) {
			t.Errorf("tests[%d] Array got %v; want %v", i, aa, test.a)
		}
		if mm := n.Map(); !reflect.DeepEqual(mm, test.m) {
			t.Errorf("tests[%d] Map got %v; want %v", i, mm, test.m)
		}
		if vv := n.Value(); !reflect.DeepEqual(vv, test.v) {
			t.Errorf("tests[%d] Value got %v; want %v", i, vv, test.v)
		}
		if had := n.Has(test.hasKeys...); had != test.hasValue {
			t.Errorf("tests[%d] Has got %v; want %v", i, had, test.hasValue)
		}
		if got := n.Get(test.getKeys...); !reflect.DeepEqual(got, test.getValue) {
			t.Errorf("tests[%d] Get got %v; want %v", i, got, test.getValue)
		}
		found, err := n.Find(test.findExpr)
		if err != nil {
			t.Errorf("tests[%d] failed to Find error %v", i, err)
		}
		if !reflect.DeepEqual(found, test.findValue) {
			t.Errorf("tests[%d] Find got %v; want %v", i, found, test.findValue)
		}
	}
}

func Test_Node_Get(t *testing.T) {
	tests := []struct {
		n    Node
		keys []any
		has  bool
		want Node
	}{
		{
			n:    Array{StringValue("a"), StringValue("b")},
			keys: []any{1},
			has:  true,
			want: StringValue("b"),
		}, {
			n:    Array{StringValue("a"), StringValue("b")},
			keys: []any{"1"},
			has:  true,
			want: StringValue("b"),
		}, {
			n:    Array{StringValue("a"), StringValue("b")},
			keys: []any{1.0},
			want: Nil,
		}, {
			n:    Array{StringValue("a"), StringValue("b")},
			keys: []any{2},
			want: Nil,
		}, {
			n:    Array{StringValue("a"), nil},
			keys: []any{1},
			has:  true,
			want: Nil,
		}, {
			n:    Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys: []any{"1"},
			has:  true,
			want: NumberValue(10),
		}, {
			n:    Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys: []any{1},
			has:  true,
			want: NumberValue(10),
		}, {
			n:    Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys: []any{1.0},
			want: Nil,
		}, {
			n:    Map{"1": NumberValue(10), "2": NumberValue(20)},
			keys: []any{"3"},
			want: Nil,
		}, {
			n:    Map{"1": NumberValue(10), "2": nil},
			keys: []any{"2"},
			has:  true,
			want: Nil,
		}, {
			n:    Map{"a": Map{"b": StringValue("v")}},
			keys: []any{"a", "b"},
			has:  true,
			want: StringValue("v"),
		}, {
			n:    Map{"a": Map{"b": StringValue("v")}},
			keys: []any{"a", "b", "c", "d"},
			want: Nil,
		}, {
			n:    Map{"a": Map{"b": StringValue("v")}},
			keys: []any{"a", "c"},
			want: Nil,
		}, {
			n:    Array{Array{nil, StringValue("v")}},
			keys: []any{0, 1},
			has:  true,
			want: StringValue("v"),
		}, {
			n:    Array{Array{nil, StringValue("v")}},
			keys: []any{0, 1, 2, 3},
			want: Nil,
		}, {
			n:    Array{Map{"a": Array{nil, Map{"b": StringValue("v")}}}},
			keys: []any{0, "a", 1, "b"},
			has:  true,
			want: StringValue("v"),
		}, {
			n:    StringValue("str"),
			want: Nil,
		}, {
			n:    BoolValue(true),
			want: Nil,
		}, {
			n:    NumberValue(1),
			want: Nil,
		},
	}
	for i, test := range tests {
		if test.n.Has(test.keys...) != test.has {
			t.Errorf("tests[%d] Has got %v; want %v", i, !test.has, test.has)
		}
		got := test.n.Get(test.keys...)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] Get got %v; want %v", i, got, test.want)
		}
	}
}

func Test_Node_Each(t *testing.T) {
	tests := []struct {
		n    Node
		want map[any]Node
	}{
		{
			n:    Array{StringValue("a"), StringValue("b")},
			want: map[any]Node{0: StringValue("a"), 1: StringValue("b")},
		}, {
			n:    Map{"a": NumberValue(0), "b": NumberValue(1)},
			want: map[any]Node{"a": NumberValue(0), "b": NumberValue(1)},
		}, {
			n:    StringValue("str"),
			want: map[any]Node{nil: StringValue("str")},
		}, {
			n:    BoolValue(true),
			want: map[any]Node{nil: BoolValue(true)},
		}, {
			n:    NumberValue(1),
			want: map[any]Node{nil: NumberValue(1)},
		}, {
			n:    Any{Map{"a": NumberValue(0), "b": NumberValue(1)}},
			want: map[any]Node{"a": NumberValue(0), "b": NumberValue(1)},
		},
	}
	for i, test := range tests {
		got := map[any]Node{}
		err := test.n.Each(func(key any, v Node) error {
			got[key] = v
			return nil
		})
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
		wantErr := fmt.Errorf("test%d", i)
		gotErr := test.n.Each(func(key any, v Node) error {
			return wantErr
		})
		if wantErr != gotErr {
			t.Errorf("tests[%d] got %v; want %v", i, gotErr, wantErr)
		}
	}
}

func Test_Node_Find(t *testing.T) {
	tests := []struct {
		n    Node
		expr string
		want []Node
	}{
		{
			n:    Array{StringValue("a"), StringValue("b")},
			expr: ".[0]",
			want: []Node{StringValue("a")},
		}, {
			n:    Map{"1": NumberValue(10), "2": NumberValue(20)},
			expr: ".1",
			want: []Node{NumberValue(10)},
		},
	}
	for i, test := range tests {
		got, err := test.n.Find(test.expr)
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %#v; want %#v", i, got, test.want)
		}
	}
}

func Test_EditorNode_Append(t *testing.T) {
	tests := []struct {
		n      EditorNode
		values []Node
		want   EditorNode
		errstr string
	}{
		{
			n:      &Array{NumberValue(1)},
			values: []Node{StringValue("2"), BoolValue(true)},
			want:   &Array{NumberValue(1), StringValue("2"), BoolValue(true)},
		}, {
			n:      Map{},
			values: []Node{StringValue("2")},
			errstr: "cannot append to map",
		},
	}
	for i, test := range tests {
		var err error
		for _, value := range test.values {
			err = test.n.Append(value)
			if err != nil {
				break
			}
		}
		if test.errstr != "" {
			if err == nil {
				t.Fatalf("tests[%d] no error", i)
			}
			if err.Error() != test.errstr {
				t.Errorf("tests[%d] got %s; want %s", i, err.Error(), test.errstr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		got := test.n
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
	}
}

func Test_EditorNode_Set(t *testing.T) {
	tests := []struct {
		n       EditorNode
		entries map[any]Node
		want    EditorNode
		errstr  string
	}{
		{
			n: &Array{NumberValue(0), StringValue("1")},
			entries: map[any]Node{
				0:   NumberValue(1),
				"1": StringValue("2"),
				2:   BoolValue(true),
			},
			want: &Array{NumberValue(1), StringValue("2"), BoolValue(true)},
		}, {
			n:       &Array{},
			entries: map[any]Node{-2: StringValue("value")},
			errstr:  "cannot index array with -2",
		}, {
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
			n:       Map{},
			entries: map[any]Node{true: StringValue("value")},
			errstr:  "cannot index array with true",
		},
	}
	for i, test := range tests {
		var err error
		for key, value := range test.entries {
			err = test.n.Set(key, value)
			if err != nil {
				break
			}
		}
		if test.errstr != "" {
			if err == nil {
				t.Fatalf("tests[%d] no error", i)
			}
			if err.Error() != test.errstr {
				t.Errorf(`tests[%d] got %s; want %s`, i, err.Error(), test.errstr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		got := test.n
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
	}
}

func Test_EditorNode_Delete(t *testing.T) {
	tests := []struct {
		n      EditorNode
		keys   []any
		want   EditorNode
		errstr string
	}{
		{
			n:    &Array{NumberValue(1), StringValue("1"), BoolValue(true)},
			keys: []any{1, "1"},
			want: &Array{NumberValue(1)},
		}, {
			n:      &Array{},
			keys:   []any{-1},
			errstr: "cannot index array with -1",
		}, {
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
			n:      Map{},
			keys:   []any{true},
			errstr: "cannot index array with true",
		},
	}
	for i, test := range tests {
		var err error
		for _, key := range test.keys {
			err = test.n.Delete(key)
			if err != nil {
				break
			}
		}
		if test.errstr != "" {
			if err == nil {
				t.Fatalf("tests[%d] no error", i)
			}
			if err.Error() != test.errstr {
				t.Errorf("tests[%d] got %s; want %s", i, err.Error(), test.errstr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("tests[%d] %v", i, err)
		}
		got := test.n
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("tests[%d] got %v; want %v", i, got, test.want)
		}
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
